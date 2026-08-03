package main

import "testing"

func TestRewriteParentLinks(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "adr link from .claude resolves at repo root",
			in:   "Full rationale in [ADR-019](../docs/adr/ADR-019-Template-Scope-Boundary.md).",
			want: "Full rationale in [ADR-019](docs/adr/ADR-019-Template-Scope-Boundary.md).",
		},
		{
			name: "repo root file link",
			in:   "The cross-tool agent context lives in [`AGENTS.md`](../AGENTS.md) at the repo root.",
			want: "The cross-tool agent context lives in [`AGENTS.md`](AGENTS.md) at the repo root.",
		},
		{
			name: "multiple links on one line",
			in:   "[a](../docs/a.md) and [b](../docs/b.md)",
			want: "[a](docs/a.md) and [b](docs/b.md)",
		},
		{
			name: "root-relative link untouched",
			in:   "[ADR-003](docs/adr/ADR-003-Database-Access.md)",
			want: "[ADR-003](docs/adr/ADR-003-Database-Access.md)",
		},
		{
			name: "same-directory link untouched",
			in:   "[architect](roles/architect.md)",
			want: "[architect](roles/architect.md)",
		},
		{
			name: "doubled parent prefix collapses fully",
			in:   "[x](../../docs/x.md)",
			want: "[x](docs/x.md)",
		},
		{
			name: "parent path outside link syntax untouched",
			in:   "run `cd ../elsewhere` first",
			want: "run `cd ../elsewhere` first",
		},
		{
			name: "empty string",
			in:   "",
			want: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := rewriteParentLinks(tt.in); got != tt.want {
				t.Errorf("rewriteParentLinks(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}
