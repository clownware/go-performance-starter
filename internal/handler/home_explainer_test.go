package handler

import (
	"strings"
	"testing"

	"github.com/clownware/go-performance-starter/internal/performance"
)

// TestExplainerNodes pins the teachable spine (ADR-024 surface 1, #67): the
// explainer narrates a request's journey in five fixed steps, each anchored,
// linked to its ADR, backed by a real source peek, and keyed to the quiz
// topic that draws from it — so read → quiz → flashcards closes the loop.
func TestExplainerNodes(t *testing.T) {
	nodes := ExplainerNodes()

	wantOrder := []struct{ anchor, topic, adr string }{
		{"routing", "routing", "ADR-014"},
		{"handlers", "handlers", "ADR-017"},
		{"database", "database", "ADR-003"},
		{"frontend", "frontend", "ADR-007"},
		{"performance", "performance", "ADR-000"},
	}
	if len(nodes) != len(wantOrder) {
		t.Fatalf("got %d nodes, want %d", len(nodes), len(wantOrder))
	}
	seenAnchors := map[string]bool{}
	for i, want := range wantOrder {
		n := nodes[i]
		if n.Step != i+1 {
			t.Errorf("node %d Step = %d, want %d", i, n.Step, i+1)
		}
		if n.Anchor != want.anchor {
			t.Errorf("node %d Anchor = %q, want %q", i, n.Anchor, want.anchor)
		}
		if seenAnchors[n.Anchor] {
			t.Errorf("duplicate anchor %q", n.Anchor)
		}
		seenAnchors[n.Anchor] = true
		if n.QuizTopic != want.topic {
			t.Errorf("node %d QuizTopic = %q, want %q (must match the seeded quiz_questions.topic)", i, n.QuizTopic, want.topic)
		}
		if !strings.Contains(n.ADR.Label, want.adr) || !strings.Contains(n.ADR.Href, want.adr) {
			t.Errorf("node %d ADR = %+v, want it to name %s", i, n.ADR, want.adr)
		}
		if n.Title == "" || n.Summary == "" || len(n.Prose) == 0 {
			t.Errorf("node %d is missing title/summary/prose: %+v", i, n)
		}
		if n.Source.File == "" || !strings.Contains(n.Source.Snippet, "\n") {
			t.Errorf("node %d needs a source peek with a file path and a multi-line snippet", i)
		}
		if !strings.HasPrefix(n.Source.File, "internal/") && !strings.HasPrefix(n.Source.File, "sql/") && !strings.HasPrefix(n.Source.File, "migrations/") {
			t.Errorf("node %d Source.File = %q should point into the repo tree", i, n.Source.File)
		}
	}
}

// TestPerfBudgetStats pins the "render from constants, never hardcode" rule
// (design brief §Live performance stats): every stat is derived from
// internal/performance at request time, so a budget change in ADR-000's
// constants changes the landing page without anyone editing a template.
func TestPerfBudgetStats(t *testing.T) {
	stats := PerfBudgetStats()

	want := map[string]string{
		"P50 response":  performance.MaxP50ResponseTime.String(),
		"P95 response":  performance.MaxP95ResponseTime.String(),
		"P99 response":  performance.MaxP99ResponseTime.String(),
		"Binary size":   "20 MB",
		"Memory":        "128 MB",
		"Startup":       performance.MaxStartupTime.String(),
		"JavaScript":    "50 KB",
		"CSS":           "30 KB",
		"Total page":    "500 KB",
		"Docker image":  "", // not a performance constant — must NOT appear
		"Peak memory":   "256 MB",
		"Response time": "", // generic label must not appear; the three percentiles do
	}
	got := map[string]string{}
	for _, s := range stats {
		if s.Label == "" || s.Value == "" {
			t.Errorf("stat with empty label/value: %+v", s)
		}
		got[s.Label] = s.Value
	}
	for label, value := range want {
		if value == "" {
			if _, ok := got[label]; ok {
				t.Errorf("stat %q should not be rendered (no constant backs it)", label)
			}
			continue
		}
		if got[label] != value {
			t.Errorf("stat %q = %q, want %q (from internal/performance)", label, got[label], value)
		}
	}
	if len(stats) != 10 {
		t.Errorf("got %d stats, want 10 (three latency percentiles, four resource, three frontend)", len(stats))
	}
}

// TestFormatBudgetBytes pins the human units the stats use.
func TestFormatBudgetBytes(t *testing.T) {
	tests := []struct {
		in   int64
		want string
	}{
		{512, "512 B"},
		{50 * 1024, "50 KB"},
		{30 * 1024, "30 KB"},
		{500 * 1024, "500 KB"},
		{20 * 1024 * 1024, "20 MB"},
		{128 * 1024 * 1024, "128 MB"},
		{1536, "1.5 KB"},
	}
	for _, tt := range tests {
		if got := formatBudgetBytes(tt.in); got != tt.want {
			t.Errorf("formatBudgetBytes(%d) = %q, want %q", tt.in, got, tt.want)
		}
	}
}
