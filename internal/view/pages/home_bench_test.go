package pages_test

import (
	"context"
	"io"
	"testing"

	"github.com/clownware/go-performance-starter/internal/handler"
	"github.com/clownware/go-performance-starter/internal/performance"
	"github.com/clownware/go-performance-starter/internal/view"
	"github.com/clownware/go-performance-starter/internal/view/pages"
)

// BenchmarkHomePage_Render measures the templ render of the landing page —
// the heaviest page the demo serves — with no HTTP or middleware around it.
// "props+render" also rebuilds the explainer spine and budget stats, as the
// home handler does on every request. Uses b.Loop (Go 1.24).
func BenchmarkHomePage_Render(b *testing.B) {
	props := func() pages.HomePageProps {
		return pages.HomePageProps{
			BaseProps: view.NewBaseProps("Go Performance Starter"),
			Nodes:     handler.ExplainerNodes(),
			Stats:     handler.PerfBudgetStats(performance.Default.Snapshot()),
		}
	}
	ctx := context.Background()

	b.Run("render", func(b *testing.B) {
		page := pages.HomePage(props())
		b.ReportAllocs()
		for b.Loop() {
			if err := page.Render(ctx, io.Discard); err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("props+render", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			if err := pages.HomePage(props()).Render(ctx, io.Discard); err != nil {
				b.Fatal(err)
			}
		}
	})
}
