package guardrail

import (
	"log/slog"
	"regexp"
	"strings"
)

// removedInstruction replaces a sentence that reads as an instruction to the model.
const removedInstruction = "[REMOVED: possible instruction]"

// injections are phrasings that address the model rather than inform it. Each
// match removes its whole sentence; the rest of the document survives.
var injections = []struct {
	name string
	re   *regexp.Regexp
}{
	{"ignore_previous", regexp.MustCompile(`(?i)\b(?:ignore|disregard|forget)\s+(?:all\s+|any\s+)?(?:the\s+|your\s+)?(?:previous|prior|above|earlier)\s+(?:instructions?|prompts?|rules?|messages?)`)},
	{"disregard_above", regexp.MustCompile(`(?i)\bdisregard\s+(?:everything|all)\s+above\b`)},
	{"you_are_now", regexp.MustCompile(`(?i)\byou\s+are\s+now\b`)},
	{"system_prompt", regexp.MustCompile(`(?i)\bsystem\s+prompt\b`)},
	{"new_instructions", regexp.MustCompile(`(?i)\bnew\s+instructions?\s*:`)},
	{"vi_ignore", regexp.MustCompile(`(?i)bỏ\s+qua\s+(?:mọi\s+|tất\s+cả\s+)?(?:các\s+)?(?:hướng\s+dẫn|chỉ\s+dẫn|chỉ\s+thị)`)},
	{"vi_forget", regexp.MustCompile(`(?i)quên\s+(?:hết\s+|mọi\s+|tất\s+cả\s+)?(?:các\s+)?(?:hướng\s+dẫn|chỉ\s+dẫn|chỉ\s+thị)`)},
	{"role_token", regexp.MustCompile(`(?i)<\|?(?:system|im_start|im_end)\|?>|\[/?INST\]|###\s*(?:system|instruction)`)},
}

// delimiterTag matches an opening or closing tool_output tag inside content.
var delimiterTag = regexp.MustCompile(`(?i)<(/?tool_output)`)

// Sanitize cleans one tool result before the model reads it: personal data and
// credentials are masked and instruction-like sentences are removed. source is
// the tool name, used only for logging.
func Sanitize(source, text string) string {
	text, _ = Redact(text)
	for _, inj := range injections {
		for {
			loc := inj.re.FindStringIndex(text)
			if loc == nil {
				break
			}
			start, end := sentenceBounds(text, loc[0], loc[1])
			text = text[:start] + removedInstruction + text[end:]
			slog.Warn("guardrail: injection removed", "source", source, "pattern", inj.name)
		}
	}
	return text
}

// WrapToolOutput fences a tool result so the model can tell data from
// instructions. Tags already in content are neutralised first, so a document
// cannot close the fence and write outside it.
func WrapToolOutput(source, content string) string {
	content = delimiterTag.ReplaceAllString(content, "&lt;$1")
	source = strings.ReplaceAll(source, `"`, "")
	return `<tool_output source="` + source + `">` + "\n" + content + "\n</tool_output>"
}

// sentenceBounds widens [start, end) to the sentence around it. A quote counts
// as a boundary so a match inside a JSON string does not swallow its neighbours.
func sentenceBounds(text string, start, end int) (int, int) {
	const stops = ".!?\n\""
	if i := strings.LastIndexAny(text[:start], stops); i >= 0 {
		start = i + 1
	} else {
		start = 0
	}
	if i := strings.IndexAny(text[end:], stops); i >= 0 {
		end += i
	} else {
		end = len(text)
	}
	return start, end
}
