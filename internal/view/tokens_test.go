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
		{regexp.MustCompile(`(bg|text|border|divide|ring)-gray-[0-9]`), "raw gray utilities — use role tokens like text-muted-foreground / border-border (ADR-029 §2)"},
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
