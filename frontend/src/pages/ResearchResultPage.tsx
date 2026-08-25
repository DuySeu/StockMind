import { Button } from "@/components/ui/button";
import { ArrowLeft, Download, FileSearch } from "lucide-react";
import { useLocation, useNavigate, useParams } from "react-router-dom";
import { Select, SelectContent, SelectGroup, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { PDFGenerator } from "@/lib/pdf-export";
import ResearchReport from "@/components/containers/ResearchReport";

// Render one ticker's research report, with a ticker switcher and PDF export
const ResearchResultPage = () => {
  const navigate = useNavigate();
  const { id } = useParams();
  const location = useLocation();

  const response = (location.state as { response: any } | null)?.response;
  const ticker = id?.toUpperCase() ?? "";
  const data: any | undefined = response?.reports?.[ticker];

  if (!response || !data) {
    return (
      <div className="mx-auto flex w-full max-w-lg flex-1 flex-col items-center justify-center gap-5 px-4 py-24 text-center">
        <span className="flex size-12 items-center justify-center rounded-lg bg-secondary">
          <FileSearch className="size-6 text-muted-foreground" aria-hidden="true" />
        </span>
        <div className="space-y-1.5">
          <h1 className="text-xl font-bold tracking-tight">This report is no longer loaded</h1>
          {/* Honest about the cause: the report lives in router state, so a
              refresh or a pasted link arrives with nothing to show. */}
          <p className="text-sm leading-relaxed text-muted-foreground">
            Results are held for the session that generated them, so a refreshed or shared link arrives empty. Run the
            digest again to see it.
          </p>
        </div>
        <Button onClick={() => navigate("/research")}>
          <ArrowLeft aria-hidden="true" /> Back to research
        </Button>
      </div>
    );
  }

  const tickers = Object.keys(response.reports);

  const handleExportPDF = () => {
    const generator = new PDFGenerator("p");

    // --- Header ---
    generator.addHeader(data.ticker ?? ticker, data.company_name ?? "");

    // --- Market Metrics ---
    generator.addMetricsBoard(data.market_cap, data.pe_ratio);

    // --- Stock Overview ---
    generator.addSectionTitle("Stock Overview");
    if (data.summary) {
      generator.addWrappedText(data.summary);
    }

    // --- Key Insights ---
    if (data.key_insights?.length) {
      generator.addSectionTitle("Key Insights");
      data.key_insights.forEach((insight: string, i: number) => {
        generator.addWrappedText(`${i + 1}. ${insight}`);
      });
    }

    // --- Current Performance ---
    if (data.current_performance) {
      generator.addSectionTitle("Current Performance");
      generator.addWrappedText(data.current_performance);
    }

    // --- Risk Assessment ---
    if (data.risk_assessment) {
      generator.addSectionTitle("Risk Assessment");
      generator.addWrappedText(data.risk_assessment);
    }

    // --- Price Outlook ---
    if (data.price_outlook) {
      generator.addSectionTitle("Price Outlook");
      generator.addWrappedText(data.price_outlook);
    }

    // --- Final Recommendation ---
    if (data.recommendation) {
      generator.addSectionTitle("Final Recommendation");
      generator.addWrappedText(data.recommendation, 11, "bold");
    }

    // --- Sources ---
    generator.addSourceLinks(data.sources);

    // --- Footer ---
    generator.addFooter();

    // --- Save ---
    generator.save(`${ticker}_Research_Report_${new Date().toISOString().slice(0, 10)}.pdf`);
  };

  return (
    <div className="flex w-full flex-1 flex-col">
      {/* Title left, actions right. The old three-part justify-between put the
          heading between two buttons, which collapsed on any narrow screen. */}
      <header className="w-full border-b border-border bg-background/85 backdrop-blur-md">
        <div className="mx-auto flex max-w-5xl flex-col gap-4 px-4 py-5 sm:px-6 lg:flex-row lg:items-center lg:justify-between lg:px-8">
          <div className="flex min-w-0 flex-col gap-1">
            <Button
              variant="link"
              onClick={() => navigate(-1)}
              className="h-auto w-fit gap-1.5 p-0 text-sm text-muted-foreground hover:text-foreground"
            >
              <ArrowLeft className="size-4" aria-hidden="true" /> Back
            </Button>
            <h1 className="text-xl font-bold tracking-tight">
              {ticker} research
              <span className="ml-2 font-mono text-sm font-medium tabular-nums text-muted-foreground">
                {new Date().toLocaleDateString("vi-VN")}
              </span>
            </h1>
          </div>

          <div className="flex shrink-0 items-center gap-2">
            <Select
              defaultValue={ticker}
              onValueChange={(val) => navigate(`/research/${val.toLowerCase()}`, { state: { response } })}
            >
              <SelectTrigger className="w-[140px]" aria-label="Switch ticker">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectGroup>
                  {tickers.map((t) => (
                    <SelectItem key={t} value={t}>
                      {t}
                    </SelectItem>
                  ))}
                </SelectGroup>
              </SelectContent>
            </Select>
            <Button variant="outline" onClick={handleExportPDF}>
              <Download aria-hidden="true" /> Export PDF
            </Button>
          </div>
        </div>
      </header>

      <div className="mx-auto w-full max-w-5xl px-4 py-6 sm:px-6 lg:px-8">
        <ResearchReport data={data} />
      </div>
    </div>
  );
};

export default ResearchResultPage;
