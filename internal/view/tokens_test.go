package view

import (
	"bytes"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

type colorFinding struct {
	line   int
	match  string
	reason string
}

var forbiddenColors = []struct {
	pattern *regexp.Regexp
	reason  string
}{
	{regexp.MustCompile(`dark:(bg|text|border|divide|ring|from|to)-`), "dark: color variants — the .dark token block flips roles instead (ADR-029 §1)"},
	// Any Tailwind palette family, not just gray: a raw green-100
	// success box or blue-600 button forks from the role layer exactly
	// like a raw gray does (#85). Brand constants (teal, bittersweet)
	// carry no numeric step, so `bg-teal-strong` passes and `bg-teal-500`
	// does not.
	{regexp.MustCompile(`(bg|text|border|divide|ring|outline|from|via|to|fill|stroke|decoration|placeholder|accent|caret|shadow)-(slate|gray|zinc|neutral|stone|red|orange|amber|yellow|lime|green|emerald|teal|cyan|sky|blue|indigo|violet|purple|fuchsia|pink|rose)-[0-9]{2,3}\b`), "raw palette utilities — use role tokens (text-muted-foreground, bg-success/10 text-success, btn-primary …) (ADR-029 §2, §4)"},
	{regexp.MustCompile(`-(text|background|surface|primary|accent|muted)-dark\b`), "deleted -dark token twins (ADR-029)"},
}

// stripTemplComments blanks comments, newlines kept, so a comment that
// names a forbidden class is not a finding and line numbers still hold:
// <!-- --> and /* */ anywhere, and // only when it opens a line (the // in
// a URL attribute is left alone). Same rule as scripts/adrcheck.
func stripTemplComments(content []byte) []byte {
	out := make([]byte, len(content))
	copy(out, content)
	blank := func(from, to int) {
		for i := from; i < to && i < len(out); i++ {
			if out[i] != '\n' {
				out[i] = ' '
			}
		}
	}
	lineStart := true // only whitespace seen so far on this line
	for i := 0; i < len(content); {
		rest := content[i:]
		switch {
		case bytes.HasPrefix(rest, []byte("<!--")):
			end := bytes.Index(rest, []byte("-->"))
			if end < 0 {
				end = len(rest) - 3
			}
			blank(i, i+end+3)
			i += end + 3
		case bytes.HasPrefix(rest, []byte("/*")):
			end := bytes.Index(rest, []byte("*/"))
			if end < 0 {
				end = len(rest) - 2
			}
			blank(i, i+end+2)
			i += end + 2
		case lineStart && bytes.HasPrefix(rest, []byte("//")):
			end := bytes.IndexByte(rest, '\n')
			if end < 0 {
				end = len(rest)
			}
			blank(i, i+end)
			i += end
		default:
			switch content[i] {
			case '\n':
				lineStart = true
			case ' ', '\t', '\r':
			default:
				lineStart = false
			}
			i++
		}
	}
	return out
}

// scanTemplColors reports forbidden color utilities in one templ source,
// comments excluded.
func scanTemplColors(src string) []colorFinding {
	var out []colorFinding
	for i, line := range strings.Split(string(stripTemplComments([]byte(src))), "\n") {
		for _, f := range forbiddenColors {
			if m := f.pattern.FindString(line); m != "" {
				out = append(out, colorFinding{line: i + 1, match: m, reason: f.reason})
			}
		}
	}
	return out
}

// TestTemplTokenDiscipline enforces ADR-029 over the templ sources: dark
// mode flips role tokens in input.css, so components must not fork colors
// with dark: variants, and must speak roles (muted-foreground, border,
// surface-hover) rather than raw Tailwind grays or the deleted -dark twins.
func TestTemplTokenDiscipline(t *testing.T) {
	err := filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(path, ".templ") {
			return err
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		for _, f := range scanTemplColors(string(content)) {
			t.Errorf("%s:%d uses %q — %s", path, f.line, f.match, f.reason)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walking templ sources: %v", err)
	}
}
