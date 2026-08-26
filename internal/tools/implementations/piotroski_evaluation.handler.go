package implementations

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"slices"
	"sync"

	"stockmind/internal/common"
)

// fiscalPeriod identifies one reporting quarter. Periods are matched by value
// rather than by slice position: VietCap's history has holes (2021 Q2 is absent
// for every ticker checked), so "four rows back" is not "one year back".
type fiscalPeriod struct {
	Year    int
	Quarter int
}

// String renders a period the way a Vietnamese report cites it.
func (p fiscalPeriod) String() string {
	return fmt.Sprintf("Q%d/%d", p.Quarter, p.Year)
}

// ratioPeriod is the subset of a statistics-financial row the F-score reads.
// The comparison metrics are pointers so a metric VietCap does not report for a
// company type is distinguishable from a real zero, and cannot be scored as an
// improvement over nothing.
type ratioPeriod struct {
	YearReport        int      `json:"yearReport"`
	Quarter           int      `json:"quarter"`
	RatioType         string   `json:"ratioType"`
	ROA               *float64 `json:"roa"`
	CurrentRatio      *float64 `json:"currentRatio"`
	GrossMargin       *float64 `json:"grossMargin"`
	AssetTurnover     *float64 `json:"assetTurnover"`
	DebtToEquity      *float64 `json:"debtToEquity"`
	SharesOutstanding *float64 `json:"numberOfSharesMktCap"`
}

// statementRow carries the two raw statement figures the ratio table does not
// expose. isa20 is net profit after tax and cfa18 is net cash from operating
// activities; both are populated for corporates, banks and securities firms
// alike, since banks carry their isb*/cfb* codes in addition, not instead.
// lengthReport is the quarter for quarterly rows and 5 for annual ones.
type statementRow struct {
	YearReport   int     `json:"yearReport"`
	LengthReport int     `json:"lengthReport"`
	NetProfit    float64 `json:"isa20"`
	CashFromOps  float64 `json:"cfa18"`
}

type statementSections struct {
	Quarters []statementRow `json:"quarters"`
	Years    []statementRow `json:"years"`
}

// PiotroskiSignals is the nine-signal breakdown behind the score.
type PiotroskiSignals struct {
	PositiveROA            bool `json:"positive_roa"`
	PositiveOperatingCash  bool `json:"positive_operating_cash_flow"`
	ImprovingROA           bool `json:"improving_roa"`
	CashExceedsNetIncome   bool `json:"cash_flow_exceeds_net_income"`
	DecreasingLeverage     bool `json:"decreasing_leverage"`
	ImprovingCurrentRatio  bool `json:"improving_current_ratio"`
	NoShareDilution        bool `json:"no_share_dilution"`
	ImprovingGrossMargin   bool `json:"improving_gross_margin"`
	ImprovingAssetTurnover bool `json:"improving_asset_turnover"`
}

type PiotroskiEvaluation struct {
	Symbol      string           `json:"symbol"`
	Period      string           `json:"period"`
	PriorPeriod string           `json:"prior_period"`
	Score       int              `json:"score"`
	Signals     PiotroskiSignals `json:"signals"`
	// Unscorable names the signals VietCap reports no data for, so a bank's
	// missing current ratio reads as "not applicable" rather than as a failure.
	Unscorable []string `json:"unscorable_signals,omitempty"`
}

type PiotroskiInput struct {
	Symbol string `json:"symbol" jsonschema:"Stock symbol, e.g., HPG"`
}

// HandlePiotroskiEvaluation scores a symbol on the nine Piotroski F-score
// signals, comparing its latest quarter against the same quarter a year earlier.
func HandlePiotroskiEvaluation(ctx context.Context, input PiotroskiInput) (any, error) {
	if input.Symbol == "" {
		return nil, fmt.Errorf("symbol is required")
	}

	// Ratios, income statement and cash flow are three independent endpoints
	var (
		wg       sync.WaitGroup
		rawRatio []ratioPeriod
		income   statementSections
		cash     statementSections
		errs     [3]error
	)
	wg.Add(3)
	go func() {
		defer wg.Done()
		rawRatio, errs[0] = common.FetchIQInsight[[]ratioPeriod](ctx,
			fmt.Sprintf("%s/%s/statistics-financial", common.COMPANY_URL, input.Symbol))
	}()
	go func() {
		defer wg.Done()
		income, errs[1] = common.FetchIQInsight[statementSections](ctx,
			fmt.Sprintf("%s/%s/financial-statement?section=INCOME_STATEMENT", common.COMPANY_URL, input.Symbol))
	}()
	go func() {
		defer wg.Done()
		cash, errs[2] = common.FetchIQInsight[statementSections](ctx,
			fmt.Sprintf("%s/%s/financial-statement?section=CASH_FLOW", common.COMPANY_URL, input.Symbol))
	}()
	wg.Wait()

	if err := errors.Join(errs[0], errs[1], errs[2]); err != nil {
		return nil, fmt.Errorf("fetch financials for %s: %w", input.Symbol, err)
	}

	// Index all three sources by period so they can be matched by value
	ratios := make(map[fiscalPeriod]ratioPeriod, len(rawRatio))
	for _, row := range rawRatio {
		if row.RatioType != ratioBasisTTM {
			continue
		}

		// VietCap encodes "not applicable" as a literal 0 rather than as null for
		// the two ratios that mean nothing for a bank. No operating company has a
		// current ratio or an asset turnover of exactly zero, so read it as absent
		// and let the signal report unscorable instead of failing the bank on it.
		row.CurrentRatio = nilIfZero(row.CurrentRatio)
		row.AssetTurnover = nilIfZero(row.AssetTurnover)

		ratios[fiscalPeriod{Year: row.YearReport, Quarter: row.Quarter}] = row
	}
	netProfit := indexStatement(income.Quarters, func(r statementRow) float64 { return r.NetProfit })
	cashFromOps := indexStatement(cash.Quarters, func(r statementRow) float64 { return r.CashFromOps })

	current, prior, err := latestComparablePeriod(ratios, netProfit, cashFromOps)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", input.Symbol, err)
	}

	signals, unscorable := scorePiotroski(
		ratios[current], ratios[prior],
		netProfit[current], cashFromOps[current],
	)

	return PiotroskiEvaluation{
		Symbol:      input.Symbol,
		Period:      current.String(),
		PriorPeriod: prior.String(),
		Score:       countSignals(signals),
		Signals:     signals,
		Unscorable:  unscorable,
	}, nil
}

// nilIfZero treats an exact zero as an absent value.
func nilIfZero(value *float64) *float64 {
	if value == nil || *value == 0 {
		return nil
	}
	return value
}

// indexStatement keys quarterly statement rows by period, reading one figure out
// of each. Annual rows (lengthReport 5) are skipped.
func indexStatement(rows []statementRow, value func(statementRow) float64) map[fiscalPeriod]float64 {
	indexed := make(map[fiscalPeriod]float64, len(rows))
	for _, row := range rows {
		if row.LengthReport < 1 || row.LengthReport > 4 {
			continue
		}
		indexed[fiscalPeriod{Year: row.YearReport, Quarter: row.LengthReport}] = value(row)
	}
	return indexed
}

// latestComparablePeriod finds the newest quarter that all three sources cover
// and whose year-earlier quarter they also cover.
//
// It walks backwards rather than taking the newest period outright because a
// hole in VietCap's history would otherwise leave the comparison without a
// counterpart. Both chosen periods are reported in the output, so stepping back
// is visible to the caller rather than silent.
func latestComparablePeriod(
	ratios map[fiscalPeriod]ratioPeriod,
	netProfit, cashFromOps map[fiscalPeriod]float64,
) (current, prior fiscalPeriod, err error) {
	covered := func(p fiscalPeriod) bool {
		_, hasRatio := ratios[p]
		_, hasProfit := netProfit[p]
		_, hasCash := cashFromOps[p]
		return hasRatio && hasProfit && hasCash
	}

	// Newest first
	periods := make([]fiscalPeriod, 0, len(ratios))
	for period := range ratios {
		periods = append(periods, period)
	}
	slices.SortFunc(periods, func(a, b fiscalPeriod) int {
		if a.Year != b.Year {
			return cmp.Compare(b.Year, a.Year)
		}
		return cmp.Compare(b.Quarter, a.Quarter)
	})

	for _, period := range periods {
		yearEarlier := fiscalPeriod{Year: period.Year - 1, Quarter: period.Quarter}
		if covered(period) && covered(yearEarlier) {
			return period, yearEarlier, nil
		}
	}

	return current, prior, errors.New("no quarter has both ratio and statement data alongside its year-earlier counterpart")
}

// scorePiotroski evaluates the nine signals and names the ones that could not be
// scored because VietCap reports no value for them.
func scorePiotroski(current, prior ratioPeriod, netProfit, cashFromOps float64) (PiotroskiSignals, []string) {
	// A metric is only an improvement if both sides of the comparison exist
	improved := func(now, before *float64) bool {
		return now != nil && before != nil && *now > *before
	}

	signals := PiotroskiSignals{
		PositiveROA:           current.ROA != nil && *current.ROA > 0,
		PositiveOperatingCash: cashFromOps > 0,
		ImprovingROA:          improved(current.ROA, prior.ROA),
		CashExceedsNetIncome:  cashFromOps > netProfit,
		ImprovingCurrentRatio: improved(current.CurrentRatio, prior.CurrentRatio),
		ImprovingGrossMargin:  improved(current.GrossMargin, prior.GrossMargin),
		// Fewer shares outstanding than a year ago means no dilution
		NoShareDilution:        improved(prior.SharesOutstanding, current.SharesOutstanding) || equalShares(current, prior),
		ImprovingAssetTurnover: improved(current.AssetTurnover, prior.AssetTurnover),
	}

	// Leverage falls when debt/equity falls; the monotonic transform only
	// matters for reporting, so compare the ratio directly.
	signals.DecreasingLeverage = improved(prior.DebtToEquity, current.DebtToEquity)

	// Name the signals that had no data behind them, so a bank's absent current
	// ratio is not read as a genuine failure to improve
	both := func(now, before *float64) bool { return now != nil && before != nil }
	scorable := []struct {
		name      string
		available bool
	}{
		{"positive_roa", current.ROA != nil},
		{"improving_roa", both(current.ROA, prior.ROA)},
		{"decreasing_leverage", both(current.DebtToEquity, prior.DebtToEquity)},
		{"improving_current_ratio", both(current.CurrentRatio, prior.CurrentRatio)},
		{"no_share_dilution", both(current.SharesOutstanding, prior.SharesOutstanding)},
		{"improving_gross_margin", both(current.GrossMargin, prior.GrossMargin)},
		{"improving_asset_turnover", both(current.AssetTurnover, prior.AssetTurnover)},
	}

	var unscorable []string
	for _, signal := range scorable {
		if !signal.available {
			unscorable = append(unscorable, signal.name)
		}
	}

	return signals, unscorable
}

// equalShares reports whether the share count is unchanged, which counts as no
// dilution just as a reduction does.
func equalShares(current, prior ratioPeriod) bool {
	return current.SharesOutstanding != nil && prior.SharesOutstanding != nil &&
		*current.SharesOutstanding == *prior.SharesOutstanding
}

// countSignals sums the signals that passed.
func countSignals(s PiotroskiSignals) int {
	passed := []bool{
		s.PositiveROA, s.PositiveOperatingCash, s.ImprovingROA, s.CashExceedsNetIncome,
		s.DecreasingLeverage, s.ImprovingCurrentRatio, s.NoShareDilution,
		s.ImprovingGrossMargin, s.ImprovingAssetTurnover,
	}

	score := 0
	for _, ok := range passed {
		if ok {
			score++
		}
	}
	return score
}
