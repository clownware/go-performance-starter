package components

import (
	"strings"
	"testing"
)

// TestFormValidationXData_EscapesJSContext proves that hostile prop values
// cannot break out of the Alpine x-data JS string context (2026-07-06 audit).
func TestFormValidationXData_EscapesJSContext(t *testing.T) {
	props := FormValidationProps{
		Value:       `' ); alert(1); //`,
		ServerError: `</script><script>alert(2)</script>`,
		Pattern:     `\d'`,
	}

	got := formValidationXData(props)

	// The value must be a JSON (double-quoted) literal, never a single-quoted
	// one that an embedded ' could close.
	if strings.Contains(got, `value: '`) {
		t.Errorf("value emitted with single-quote delimiter (breakout risk):\n%s", got)
	}
	// json.Marshal HTML-escapes angle brackets, so neither a closing nor an
	// opening script tag can survive into the rendered attribute.
	if strings.Contains(got, "</script>") || strings.Contains(got, "<script>") {
		t.Errorf("serverError left an executable script tag unescaped:\n%s", got)
	}
	// A benign value still round-trips as a quoted JSON string.
	if !strings.Contains(formValidationXData(FormValidationProps{Value: "hi"}), `value: "hi"`) {
		t.Error("benign value not emitted as a JSON string literal")
	}
}
