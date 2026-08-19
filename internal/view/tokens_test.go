package view

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// TestTemplTokenDiscipline enforces ADR-029 over the templ sources: dark
// mode flips role tokens in input.css, so components must not fork colors
// with dark: variants, and must speak roles (muted-foreground, border,
// surface-hover) rather than raw Tailwind grays or the deleted -dark twins.
func TestTemplTokenDiscipline(t *testing.T) {
	forbidden := []struct {
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

	root := "."
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(path, ".templ") {
			return err
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		for i, line := range strings.Split(string(content), "\n") {
			for _, f := range forbidden {
				if m := f.pattern.FindString(line); m != "" {
					t.Errorf("%s:%d uses %q — %s", path, i+1, m, f.reason)
				}
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walking templ sources: %v", err)
	}
}
