package view

import "testing"

func TestTokenScanIgnoresComments(t *testing.T) {
	tests := []struct {
		name     string
		src      string
		findings int
	}{
		{"raw palette class flagged", `<div class="bg-gray-100"></div>`, 1},
		{"dark variant flagged", `<div class="dark:bg-surface"></div>`, 1},
		{"go line comment naming a class is not a finding", "// was bg-gray-100 before ADR-029\n<div class=\"bg-surface\"></div>", 0},
		{"html comment naming a class is not a finding", "<!-- no dark:bg-black here -->\n<div class=\"bg-surface\"></div>", 0},
		{"multi-line html comment is not a finding", "<!--\n  text-blue-600\n-->\n<p></p>", 0},
		{"block comment is not a finding", "/* text-red-500 */\n<p></p>", 0},
		{"url double slash is not a comment", `<a class="text-blue-600" href="https://example.com">x</a>`, 1},
		{"finding keeps its line number after a comment", "<!-- a\nb -->\n<p class=\"bg-red-500\"></p>", 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := scanTemplColors(tt.src)
			if len(got) != tt.findings {
				t.Errorf("got %d findings %v, want %d", len(got), got, tt.findings)
			}
		})
	}
	got := scanTemplColors("<!-- a\nb -->\n<p class=\"bg-red-500\"></p>")
	if len(got) != 1 || got[0].line != 3 {
		t.Errorf("line number not preserved: %+v", got)
	}
}
