package components

import (
	"context"
	"strings"
	"testing"
)

// TestFormValidation_DataAttributes pins ADR-036 §4: FormValidation names the
// registered formField component and hands it the rules as data-* attributes,
// so prop values reach the browser through templ's attribute escaping rather
// than a hand-built JS object literal (the 2026-07-06 audit's breakout risk).
func TestFormValidation_DataAttributes(t *testing.T) {
	tests := []struct {
		name        string
		props       FormValidationProps
		contains    []string
		notContains []string
	}{
		{
			name: "rules ride data attributes on a registered component",
			props: FormValidationProps{
				ID: "email", Name: "email", Value: "hi", Required: true,
				Pattern: `\S+@\S+`, MinLength: 3, MaxLength: 40,
				RequiredMessage: "Email is required", PatternMessage: "Not an email",
			},
			contains: []string{
				`x-data="formField"`,
				`data-value="hi"`,
				`data-required="true"`,
				`data-pattern="\S+@\S+"`,
				`data-min-length="3"`,
				`data-max-length="40"`,
				`data-required-message="Email is required"`,
				`data-pattern-message="Not an email"`,
			},
			notContains: []string{"get isValid", "new RegExp"},
		},
		{
			name:     "defaults: optional field, default messages, no max length",
			props:    FormValidationProps{ID: "n", Name: "n"},
			contains: []string{`data-required="false"`, `data-max-length="0"`, `data-required-message="This field is required"`, `data-pattern-message="Please enter a valid value"`},
		},
		{
			name: "hostile values stay inside attribute escaping",
			props: FormValidationProps{
				ID: "x", Name: "x",
				Value:       `" onmouseover="alert(1)`,
				ServerError: `</script><script>alert(2)</script>`,
			},
			contains:    []string{`data-value="&#34; onmouseover=&#34;alert(1)"`},
			notContains: []string{`onmouseover="alert(1)"`, "<script>"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var sb strings.Builder
			if err := FormValidation(tt.props).Render(context.Background(), &sb); err != nil {
				t.Fatalf("render: %v", err)
			}
			got := sb.String()
			for _, want := range tt.contains {
				if !strings.Contains(got, want) {
					t.Errorf("output missing %s\n%s", want, got)
				}
			}
			for _, bad := range tt.notContains {
				if strings.Contains(got, bad) {
					t.Errorf("output contains %s\n%s", bad, got)
				}
			}
		})
	}
}
