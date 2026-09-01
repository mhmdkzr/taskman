package components

import (
	"bytes"

	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/formatters/html"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
	"github.com/pmezard/go-difflib/difflib"
	"github.com/yuin/goldmark"
	highlighting "github.com/yuin/goldmark-highlighting/v2"
)

// md is a markdown renderer with GitHub-flavoured tables and syntax-highlighted
// fenced code blocks, shared by every component that renders prose or code.
var md = goldmark.New(
	goldmark.WithExtensions(highlighting.NewHighlighting(
		highlighting.WithStyle("dracula"),
		highlighting.WithFormatOptions(
			html.WithClasses(true),
		),
	)),
)

// renderMarkdown renders markdown (or plain text) to HTML, safe to embed with
// @templ.Raw. Plain text is wrapped in a code block so code-looking tool
// output still gets syntax highlighting.
func renderMarkdown(src string) string {
	var buf bytes.Buffer
	if err := md.Convert([]byte(src), &buf); err != nil {
		return escapeHTML(src)
	}
	return buf.String()
}

// escapeHTML escapes raw text for safe HTML embedding when markdown parsing
// fails.
func escapeHTML(s string) string {
	var b bytes.Buffer
	for _, r := range s {
		switch r {
		case '<':
			b.WriteString("&lt;")
		case '>':
			b.WriteString("&gt;")
		case '&':
			b.WriteString("&amp;")
		case '"':
			b.WriteString("&#34;")
		case '\'':
			b.WriteString("&#39;")
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}

// unifiedDiff builds a unified diff between old and new text, so the edit
// tool's before/after can be shown as a real highlighted diff rather than two
// plain blocks.
func unifiedDiff(oldText, newText string) string {
	d, err := difflib.GetUnifiedDiffString(difflib.UnifiedDiff{
		A:        difflib.SplitLines(oldText),
		B:        difflib.SplitLines(newText),
		FromFile: "old",
		ToFile:   "new",
		Context:  3,
	})
	if err != nil {
		return oldText + "\n---\n" + newText
	}
	return d
}

// renderDiff renders a unified diff patch with syntax highlighting. The diff
// is fed through chroma's diff lexer so +/- lines and hunk headers are
// coloured.
func renderDiff(patch string) string {
	lexer := lexers.Get("diff")
	if lexer == nil {
		lexer = lexers.Fallback
	}
	lexer = chroma.Coalesce(lexer)

	it, err := lexer.Tokenise(nil, patch)
	if err != nil {
		return escapeHTML(patch)
	}
	var buf bytes.Buffer
	if err := html.New(html.WithClasses(true), html.WithLineNumbers(false)).Format(
		&buf, styles.Get("dracula"), it,
	); err != nil {
		return escapeHTML(patch)
	}
	return buf.String()
}
