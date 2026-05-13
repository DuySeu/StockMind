package rag

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"unicode/utf8"

	"github.com/abadojack/whatlanggo"
	"github.com/ledongthuc/pdf"
	"github.com/nguyenthenguyen/docx"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	extast "github.com/yuin/goldmark/extension/ast"
	"github.com/yuin/goldmark/text"
)

// Parser defines the interface for extracting text from different document formats.
type Parser interface {
	// Parse reads a document from the provided reader and returns its raw text content.
	Parse(r io.Reader) (string, error)
}

// ---- PDF Parser ----

// PDFParser handles PDF documents.
type PDFParser struct{}

// NewPDFParser creates a new instance of PDFParser.
func NewPDFParser() *PDFParser {
	return &PDFParser{}
}

// Parse extracts text from a PDF document provided by an io.Reader.
func (p *PDFParser) Parse(r io.Reader) (string, error) {
	tempFile, err := os.CreateTemp("", "stockmind-*.pdf")
	if err != nil {
		return "", fmt.Errorf("failed to create temp file for PDF: %w", err)
	}
	defer os.Remove(tempFile.Name())
	defer tempFile.Close()

	size, err := io.Copy(tempFile, r)
	if err != nil {
		return "", fmt.Errorf("failed to buffer PDF content: %w", err)
	}

	pdfReader, err := pdf.NewReader(tempFile, size)
	if err != nil {
		return "", fmt.Errorf("failed to initialize PDF reader: %w", err)
	}

	var buf bytes.Buffer
	b, err := pdfReader.GetPlainText()
	if err != nil {
		return "", fmt.Errorf("failed to extract text from PDF: %w", err)
	}

	_, err = buf.ReadFrom(b)
	if err != nil {
		return "", fmt.Errorf("failed to read text buffer: %w", err)
	}

	return buf.String(), nil
}

// ---- DOCX Parser ----

// DOCXParser handles Microsoft Word documents.
type DOCXParser struct{}

// NewDOCXParser creates a new instance of DOCXParser.
func NewDOCXParser() *DOCXParser {
	return &DOCXParser{}
}

// Parse extracts text from a DOCX document provided by an io.Reader.
func (p *DOCXParser) Parse(r io.Reader) (string, error) {
	tempFile, err := os.CreateTemp("", "stockmind-*.docx")
	if err != nil {
		return "", fmt.Errorf("failed to create temp file for DOCX: %w", err)
	}
	defer os.Remove(tempFile.Name())
	defer tempFile.Close()

	if _, err := io.Copy(tempFile, r); err != nil {
		return "", fmt.Errorf("failed to buffer DOCX content: %w", err)
	}

	d, err := docx.ReadDocxFile(tempFile.Name())
	if err != nil {
		return "", fmt.Errorf("failed to read DOCX file: %w", err)
	}
	defer d.Close()

	rawXML := d.Editable().GetContent()
	return stripXMLTags(rawXML), nil
}

func stripXMLTags(xml string) string {
	var buf strings.Builder
	inTag := false
	for _, r := range xml {
		if r == '<' {
			inTag = true
			continue
		}
		if r == '>' {
			inTag = false
			continue
		}
		if !inTag {
			buf.WriteRune(r)
		}
	}
	return strings.Join(strings.Fields(buf.String()), " ")
}

// ---- Markdown Parser ----

// MDParser handles markdown documents by converting them into structured plaintext.
type MDParser struct {
	md goldmark.Markdown
}

// NewMDParser creates a new instance of MDParser with table support.
func NewMDParser() *MDParser {
	return &MDParser{
		md: goldmark.New(
			goldmark.WithExtensions(extension.Table),
		),
	}
}

// Parse extracts text from markdown and preserves structural elements like tables and link content.
func (p *MDParser) Parse(r io.Reader) (string, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return "", fmt.Errorf("failed to read markdown: %w", err)
	}

	doc := p.md.Parser().Parse(text.NewReader(data))
	var buf strings.Builder
	p.walk(doc, data, &buf)

	return buf.String(), nil
}

func (p *MDParser) walk(n ast.Node, source []byte, buf *strings.Builder) {
	switch n.Kind() {
	case ast.KindText:
		t := n.(*ast.Text)
		buf.Write(t.Segment.Value(source))
	case ast.KindParagraph:
		for c := n.FirstChild(); c != nil; c = c.NextSibling() {
			p.walk(c, source, buf)
		}
		buf.WriteString("\n\n")
	case ast.KindHeading:
		for c := n.FirstChild(); c != nil; c = c.NextSibling() {
			p.walk(c, source, buf)
		}
		buf.WriteString("\n")
	case ast.KindLink:
		l := n.(*ast.Link)
		for c := n.FirstChild(); c != nil; c = c.NextSibling() {
			p.walk(c, source, buf)
		}
		if l.Destination != nil {
			buf.WriteString(fmt.Sprintf(" (%s)", l.Destination))
		}
	case ast.KindList:
		for c := n.FirstChild(); c != nil; c = c.NextSibling() {
			p.walk(c, source, buf)
		}
		buf.WriteString("\n")
	case ast.KindListItem:
		buf.WriteString("- ")
		for c := n.FirstChild(); c != nil; c = c.NextSibling() {
			p.walk(c, source, buf)
		}
		buf.WriteString("\n")
	case extast.KindTable:
		for c := n.FirstChild(); c != nil; c = c.NextSibling() {
			p.walk(c, source, buf)
		}
		buf.WriteString("\n")
	case extast.KindTableRow:
		for c := n.FirstChild(); c != nil; c = c.NextSibling() {
			p.walk(c, source, buf)
		}
		buf.WriteString("\n")
	case extast.KindTableCell:
		for c := n.FirstChild(); c != nil; c = c.NextSibling() {
			p.walk(c, source, buf)
			buf.WriteString(" | ")
		}
	case ast.KindBlockquote:
		buf.WriteString("> ")
		for c := n.FirstChild(); c != nil; c = c.NextSibling() {
			p.walk(c, source, buf)
		}
	case ast.KindCodeBlock, ast.KindFencedCodeBlock:
		for i := 0; i < n.Lines().Len(); i++ {
			line := n.Lines().At(i)
			buf.Write(line.Value(source))
		}
		buf.WriteString("\n")
	default:
		for c := n.FirstChild(); c != nil; c = c.NextSibling() {
			p.walk(c, source, buf)
		}
	}
}

// ---- Text Parser ----

// TXTParser handles raw text files.
type TXTParser struct{}

// NewTXTParser creates a new instance of TXTParser.
func NewTXTParser() *TXTParser {
	return &TXTParser{}
}

// Parse extracts text from a raw text file reader and validates UTF-8 encoding.
func (p *TXTParser) Parse(r io.Reader) (string, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return "", fmt.Errorf("failed to read text file: %w", err)
	}

	if !utf8.Valid(data) {
		var cleaned strings.Builder
		for i := 0; i < len(data); {
			r, size := utf8.DecodeRune(data[i:])
			if r != utf8.RuneError || size > 1 {
				cleaned.WriteRune(r)
			}
			i += size
		}
		return cleaned.String(), nil
	}

	return string(data), nil
}

// ---- Validator ----

var (
	ErrTextTooShort = errors.New("text content is too short (min 100 characters)")
	ErrInvalidLang  = errors.New("invalid language (only Vietnamese and English are supported)")
)

// Validator handles quality checks for extracted text.
type Validator struct {
	MinLength int
}

// NewValidator creates a new instance of Validator with default settings.
func NewValidator() *Validator {
	return &Validator{
		MinLength: 100,
	}
}

// Validate checks if the text meets quality requirements (length, language).
func (v *Validator) Validate(text string) error {
	trimmed := strings.TrimSpace(text)
	if len(trimmed) < v.MinLength {
		return ErrTextTooShort
	}

	sampleLen := 1000
	if len(trimmed) < sampleLen {
		sampleLen = len(trimmed)
	}

	info := whatlanggo.Detect(trimmed[:sampleLen])
	if info.Lang != whatlanggo.Eng && info.Lang != whatlanggo.Vie {
		return fmt.Errorf("%w: detected %s", ErrInvalidLang, info.Lang.String())
	}

	return nil
}

func GetParser(fileType string) (Parser, error) {
	switch fileType {
	case "pdf":
		return NewPDFParser(), nil
	case "docx":
		return NewDOCXParser(), nil
	case "md", "markdown":
		return NewMDParser(), nil
	case "txt", "text":
		return NewTXTParser(), nil
	default:
		return nil, fmt.Errorf("unsupported file type: %s", fileType)
	}
}
