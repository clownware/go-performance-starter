package view

import (
	"fmt"
	"math"
	"os"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

// TestTokenContrast computes WCAG 2.1 contrast ratios for the load-bearing
// role-token pairs straight from input.css, in both modes, so a palette edit
// that regresses below AA fails task ci instead of waiting for the next
// audit (2026-07-17: --color-teal shipped at 4.41:1 under white button text
// because nothing measured it).
//
// Thresholds: 4.5:1 for normal text (1.4.3), 3.0:1 for large text, non-text
// UI boundaries, and graphical objects (1.4.3, 1.4.11).
func TestTokenContrast(t *testing.T) {
	light, dark := loadTokenPalettes(t, "../../web/static/css/input.css")

	tests := []struct {
		name string
		mode map[string]string
		fg   string
		bg   string
		min  float64
	}{
		// Body text
		{"light foreground on background", light, "foreground", "background", 4.5},
		{"light foreground on surface", light, "foreground", "surface", 4.5},
		{"light muted-foreground on surface", light, "muted-foreground", "surface", 4.5},
		{"light muted-foreground on background", light, "muted-foreground", "background", 4.5},
		{"dark foreground on background", dark, "foreground", "background", 4.5},
		{"dark foreground on surface", dark, "foreground", "surface", 4.5},
		{"dark muted-foreground on surface", dark, "muted-foreground", "surface", 4.5},

		// Primary buttons: white text on the teal constant (both modes)
		{"light white on primary (btn-primary)", light, "#ffffff", "primary", 4.5},
		{"light white on primary-strong (btn hover)", light, "#ffffff", "primary-strong", 4.5},
		{"dark white on primary (btn-primary)", dark, "#ffffff", "primary", 4.5},

		// Primary as text is allowed only at large-text/non-text sizes (3:1);
		// small text must use the link role instead (see templ sources).
		{"light primary as large text on surface", light, "primary", "surface", 3.0},
		{"dark primary as large text on surface", dark, "primary", "surface", 3.0},

		// Links
		{"light link on surface", light, "link", "surface", 4.5},
		{"light link on background", light, "link", "background", 4.5},
		{"dark link on surface", dark, "link", "surface", 4.5},
		{"dark link on background", dark, "link", "background", 4.5},

		// Status text on surface
		{"light danger on surface", light, "danger", "surface", 4.5},
		{"light success on surface", light, "success", "surface", 4.5},
		{"light warning on surface", light, "warning", "surface", 4.5},
		{"dark danger on surface", dark, "danger", "surface", 4.5},
		{"dark success on surface", dark, "success", "surface", 4.5},
		{"dark warning on surface", dark, "warning", "surface", 4.5},

		// Alert text on its 10% danger tint over surface
		{"light danger-emphasis on danger/10 tint", light, "danger-emphasis", "tint:danger:0.10:surface", 4.5},
		{"dark danger-emphasis on danger/10 tint", dark, "danger-emphasis", "tint:danger:0.10:surface", 4.5},

		// Form-control boundaries (WCAG 1.4.11 non-text)
		{"light input border vs surface", light, "border-input", "surface", 3.0},
		{"dark input border vs surface", dark, "border-input", "surface", 3.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fg := resolveTokenColor(t, tt.mode, tt.fg)
			bg := resolveTokenColor(t, tt.mode, tt.bg)
			got := contrastRatio(fg, bg)
			if got < tt.min {
				t.Errorf("contrast %.2f:1 (fg %s on bg %s), need %.1f:1", got, fg, bg, tt.min)
			}
		})
	}
}

// loadTokenPalettes parses the @theme block (light) and the .dark override
// block from input.css into name->hex maps with var() chains resolved.
func loadTokenPalettes(t *testing.T, path string) (light, dark map[string]string) {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading %s: %v", path, err)
	}
	css := string(raw)

	themeBlock := extractBlock(t, css, "@theme {")
	darkBlock := extractBlock(t, css, ".dark {")

	light = parseColorVars(themeBlock)
	resolveVarChains(t, light)

	// Dark inherits every light value, then applies its overrides (which may
	// reference base-palette names defined only in @theme).
	dark = map[string]string{}
	for k, v := range light {
		dark[k] = v
	}
	overrides := parseColorVars(darkBlock)
	for k, v := range overrides {
		dark[k] = v
	}
	resolveVarChains(t, dark)
	return light, dark
}

func extractBlock(t *testing.T, css, opener string) string {
	t.Helper()
	start := strings.Index(css, opener)
	if start < 0 {
		t.Fatalf("block %q not found in input.css", opener)
	}
	rest := css[start+len(opener):]
	end := strings.Index(rest, "}")
	if end < 0 {
		t.Fatalf("block %q is unterminated", opener)
	}
	return rest[:end]
}

var colorVarRe = regexp.MustCompile(`--color-([a-z-]+):\s*([^;]+);`)

func parseColorVars(block string) map[string]string {
	out := map[string]string{}
	for _, m := range colorVarRe.FindAllStringSubmatch(block, -1) {
		out[m[1]] = strings.TrimSpace(m[2])
	}
	return out
}

var varRefRe = regexp.MustCompile(`var\(--color-([a-z-]+)\)`)

func resolveVarChains(t *testing.T, palette map[string]string) {
	t.Helper()
	for i := 0; i < 5; i++ { // bounded; chains are shallow
		changed := false
		for k, v := range palette {
			if m := varRefRe.FindStringSubmatch(v); m != nil {
				ref, ok := palette[m[1]]
				if !ok {
					t.Fatalf("token %q references unknown --color-%s", k, m[1])
				}
				palette[k] = ref
				changed = true
			}
		}
		if !changed {
			return
		}
	}
	t.Fatal("var() chains did not resolve in 5 passes — circular reference?")
}

// resolveTokenColor turns a test-table reference into a hex color: a literal
// #hex, a token name, or "tint:<token>:<alpha>:<token>" for an alpha blend.
func resolveTokenColor(t *testing.T, palette map[string]string, ref string) string {
	t.Helper()
	if strings.HasPrefix(ref, "#") {
		return ref
	}
	if strings.HasPrefix(ref, "tint:") {
		parts := strings.Split(ref, ":")
		if len(parts) != 4 {
			t.Fatalf("bad tint ref %q", ref)
		}
		alpha, err := strconv.ParseFloat(parts[2], 64)
		if err != nil {
			t.Fatalf("bad tint alpha in %q: %v", ref, err)
		}
		fg := resolveTokenColor(t, palette, parts[1])
		bg := resolveTokenColor(t, palette, parts[3])
		return blendHex(fg, bg, alpha)
	}
	hex, ok := palette[ref]
	if !ok {
		t.Fatalf("token --color-%s not found", ref)
	}
	if !strings.HasPrefix(hex, "#") {
		t.Fatalf("token --color-%s resolved to non-hex %q", ref, hex)
	}
	return hex
}

func blendHex(fg, bg string, alpha float64) string {
	fr, fgc, fb := hexRGB(fg)
	br, bgc, bb := hexRGB(bg)
	mix := func(f, b int) int { return int(math.Round(float64(f)*alpha + float64(b)*(1-alpha))) }
	return fmt.Sprintf("#%02x%02x%02x", mix(fr, br), mix(fgc, bgc), mix(fb, bb))
}

func hexRGB(hex string) (r, g, b int) {
	hex = strings.TrimPrefix(hex, "#")
	rv, _ := strconv.ParseInt(hex[0:2], 16, 0)
	gv, _ := strconv.ParseInt(hex[2:4], 16, 0)
	bv, _ := strconv.ParseInt(hex[4:6], 16, 0)
	return int(rv), int(gv), int(bv)
}

// contrastRatio implements WCAG 2.1 relative luminance contrast.
func contrastRatio(a, b string) float64 {
	la, lb := relLuminance(a), relLuminance(b)
	hi, lo := math.Max(la, lb), math.Min(la, lb)
	return (hi + 0.05) / (lo + 0.05)
}

func relLuminance(hex string) float64 {
	r, g, b := hexRGB(hex)
	lin := func(c int) float64 {
		s := float64(c) / 255
		if s <= 0.04045 {
			return s / 12.92
		}
		return math.Pow((s+0.055)/1.055, 2.4)
	}
	return 0.2126*lin(r) + 0.7152*lin(g) + 0.0722*lin(b)
}
