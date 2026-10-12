// Package guardrail screens a chat turn: it masks personal data and credentials,
// rejects off-topic questions, and strips instructions planted in tool output.
package guardrail

import (
	"regexp"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"
)

// rules run in order on progressively masked text, so a broader rule never sees
// a value a narrower one already replaced. Card runs before phone and CCCD so a
// 16-digit card is never half-claimed as an ID number.
var rules = []rule{
	{kind: "API_KEY", re: regexp.MustCompile(`sk-(?:or-)?[A-Za-z0-9_\-]{16,}|AKIA[0-9A-Z]{16}|ghp_[A-Za-z0-9]{30,}`)},
	{kind: "TOKEN", re: regexp.MustCompile(`xox[bp]-[A-Za-z0-9\-]{10,}|eyJ[A-Za-z0-9_\-]+\.[A-Za-z0-9_\-]+\.[A-Za-z0-9_\-]+`)},
	{kind: "PASSWORD", re: regexp.MustCompile(`(?i)(?:^|[^\pL\pN])(?:password|passwd|pwd|pass|mật khẩu|mk)\s*(?::|=|là)\s*(\S{4,})`), valid: looksLikeSecret},
	{kind: "EMAIL", re: regexp.MustCompile(`[A-Za-z0-9._%+\-]+@[A-Za-z0-9.\-]+\.[A-Za-z]{2,}`)},
	{kind: "CARD", re: regexp.MustCompile(`\d(?:[ \-]?\d){15}(?:(?:[ \-]?\d){3})?`), numeric: true, valid: isCard},
	{kind: "PHONE", re: regexp.MustCompile(`(?:\+84|84)[ .\-]?[35789](?:[ .\-]?\d){8}|0[35789](?:[ .\-]?\d){8}`), numeric: true},
	{kind: "CCCD", re: regexp.MustCompile(`0\d{11}`), numeric: true},
	{kind: "CCCD", re: regexp.MustCompile(`(?i)(?:cmnd|cccd|căn cước|chứng minh)[^\d\n]{0,20}(\d{9})`), numeric: true},
	{kind: "BANK_ACCOUNT", re: regexp.MustCompile(`(?i)(?:stk|số tài khoản|tài khoản|account)[^\d\n]{0,25}(\d(?:[ \-]?\d){7,14})`), numeric: true},
	{kind: "OTP", re: regexp.MustCompile(`(?i)(?:otp|mã xác thực|mã xác nhận|verification code)[^\d\n]{0,15}(\d{4,8})`), numeric: true},
}

// cardPrefixes are the issuer prefixes seen in Vietnam: Visa, Mastercard, JCB and
// domestic Napas. Amex (15 digits) is left out because a 15-digit VND figure is
// an ordinary revenue line.
var cardPrefixes = []string{"4", "51", "52", "53", "54", "55", "22", "23", "24", "25", "26", "27", "35", "9704"}

type rule struct {
	kind string
	re   *regexp.Regexp
	// numeric rules must not sit inside a longer number or a thousand-separated
	// figure; tool output is full of both.
	numeric bool
	valid   func(value string) bool
}

// Redact replaces personal data and credentials in text with placeholders such
// as [PHONE], and returns the masked text with the kinds it replaced, in the
// order first seen.
func Redact(text string) (string, []string) {
	var kinds []string
	for _, r := range rules {
		masked, hit := r.apply(text)
		if hit && !slices.Contains(kinds, r.kind) {
			kinds = append(kinds, r.kind)
		}
		text = masked
	}
	return text, kinds
}

// apply masks every acceptable match of r in text. A rule with a capture group
// masks only the group, so the keyword that located the value stays readable.
func (r rule) apply(text string) (string, bool) {
	matches := r.re.FindAllStringSubmatchIndex(text, -1)
	if len(matches) == 0 {
		return text, false
	}

	var out strings.Builder
	last, hit := 0, false
	for _, m := range matches {
		start, end := m[0], m[1]
		if len(m) >= 4 && m[2] >= 0 {
			start, end = m[2], m[3]
		}
		value := text[start:end]
		if r.numeric && !isolated(text, start, end) {
			continue
		}
		if r.valid != nil && !r.valid(value) {
			continue
		}
		out.WriteString(text[last:start])
		out.WriteString("[" + r.kind + "]")
		last, hit = end, true
	}
	out.WriteString(text[last:])
	return out.String(), hit
}

// isolated reports whether text[start:end] stands alone: not glued to a letter or
// digit, and not one group of a figure like 1.234.567 or 1,234,567.
func isolated(text string, start, end int) bool {
	if start > 0 {
		prev, size := utf8.DecodeLastRuneInString(text[:start])
		if unicode.IsLetter(prev) || unicode.IsDigit(prev) {
			return false
		}
		if (prev == '.' || prev == ',') && start-size > 0 && isDigitByte(text[start-size-1]) {
			return false
		}
	}
	if end < len(text) {
		next, size := utf8.DecodeRuneInString(text[end:])
		if unicode.IsLetter(next) || unicode.IsDigit(next) {
			return false
		}
		if (next == '.' || next == ',') && end+size < len(text) && isDigitByte(text[end+size]) {
			return false
		}
	}
	return true
}

func isDigitByte(b byte) bool { return b >= '0' && b <= '9' }

// isCard reports whether value is a 16- or 19-digit number with a known issuer
// prefix that passes the Luhn check.
func isCard(value string) bool {
	digits := strings.Map(func(r rune) rune {
		if r >= '0' && r <= '9' {
			return r
		}
		return -1
	}, value)
	if len(digits) != 16 && len(digits) != 19 {
		return false
	}
	if !slices.ContainsFunc(cardPrefixes, func(p string) bool { return strings.HasPrefix(digits, p) }) {
		return false
	}

	// Luhn: double every second digit from the right
	sum := 0
	for i := range len(digits) {
		d := int(digits[len(digits)-1-i] - '0')
		if i%2 == 1 {
			d *= 2
			if d > 9 {
				d -= 9
			}
		}
		sum += d
	}
	return sum%10 == 0
}

// looksLikeSecret rejects the ordinary word after "mật khẩu là" ("mật khẩu là
// không nên chia sẻ") by requiring a digit or a symbol.
// ponytail: an all-letter password ("hunter") slips through; add an entropy check if that matters.
func looksLikeSecret(value string) bool {
	return strings.ContainsFunc(value, func(r rune) bool {
		return unicode.IsDigit(r) || (unicode.IsPunct(r) || unicode.IsSymbol(r)) && !strings.ContainsRune(".,;!?", r)
	})
}
