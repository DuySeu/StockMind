package rag

import (
	"strings"
	"testing"
)

func TestTXTParser(t *testing.T) {
	parser := NewTXTParser()
	content := "Hello World! This is a test file."
	r := strings.NewReader(content)

	got, err := parser.Parse(r)
	if err != nil {
		t.Fatalf("Parse() failed: %v", err)
	}

	if got != content {
		t.Errorf("got %q, want %q", got, content)
	}
}

func TestMDParser(t *testing.T) {
	parser := NewMDParser()
	md := `
# Header
This is [linked text](https://example.com).

| A | B |
|---|---|
| 1 | 2 |

- List item 1
- List item 2
`
	r := strings.NewReader(md)

	got, err := parser.Parse(r)
	if err != nil {
		t.Fatalf("Parse() failed: %v", err)
	}

	// Check if header is there
	if !strings.Contains(got, "Header") {
		t.Error("Header missed in MD output")
	}

	// Check link preservation
	if !strings.Contains(got, "linked text (https://example.com)") {
		t.Errorf("Link text context missed: %q", got)
	}

	// Check table preservation
	if !strings.Contains(got, "1 | 2 |") {
		t.Errorf("Table content missed: %q", got)
	}

	// Check list item
	if !strings.Contains(got, "- List item 1") {
		t.Error("List item missed in MD output")
	}
}

func TestValidator(t *testing.T) {
	v := NewValidator()

	tests := []struct {
		name    string
		text    string
		wantErr error
	}{
		{
			name:    "Too short",
			text:    "Too short to be a valid document.",
			wantErr: ErrTextTooShort,
		},
		{
			name:    "Valid English",
			text:    strings.Repeat("This is a valid English sentence for the purpose of testing the language validator confidence and length requirement. ", 5),
			wantErr: nil,
		},
		{
			name:    "Valid Vietnamese",
			text:    strings.Repeat("Đây là một câu tiếng Việt hợp lệ được dùng để kiểm tra khả năng nhận diện ngôn ngữ và độ dài yêu cầu của hệ thống. ", 5),
			wantErr: nil,
		},
		{
			name:    "Invalid language (Russian)",
			text:    strings.Repeat("Это предложение на русском языке, которое должно быть отклонено валидатором, так như hỗ trợ tiếng Việt và tiếng Anh. ", 5),
			wantErr: ErrInvalidLang,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := v.Validate(tt.text)
			
			if tt.wantErr != nil {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr.Error()) {
					t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
				}
			} else if err != nil {
				t.Errorf("Validate() unexpected error = %v", err)
			}
		})
	}
}

func TestPDFParser_Empty(t *testing.T) {
	parser := NewPDFParser()
	r := strings.NewReader("")

	_, err := parser.Parse(r)
	// Expecting error because size 0 is not a valid PDF
	if err == nil {
		t.Error("Parse() empty PDF should return error, got nil")
	}
}

func TestDOCXParser_Empty(t *testing.T) {
	parser := NewDOCXParser()
	r := strings.NewReader("")

	_, err := parser.Parse(r)
	// Zip reader (docx) should fail on empty file
	if err == nil {
		t.Error("Parse() empty DOCX should return error, got nil")
	}
}
