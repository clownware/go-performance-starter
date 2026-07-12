package validate

import (
	"strings"
	"testing"
)

func TestRequired(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{"plain value passes", "hello", false},
		{"empty string fails", "", true},
		{"whitespace-only fails", " \t\n ", true},
		{"value with surrounding whitespace passes", "  x  ", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Required(tt.value, "field")
			if (err != nil) != tt.wantErr {
				t.Errorf("Required(%q) err = %v, wantErr %v", tt.value, err, tt.wantErr)
			}
			if err != nil && !strings.Contains(err.Error(), "field") {
				t.Errorf("error %q must name the field", err.Error())
			}
		})
	}
}

func TestMaxLength(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		max     int
		wantErr bool
	}{
		{"under the limit passes", "ab", 3, false},
		{"exactly the limit passes", "abc", 3, false},
		{"one over the limit fails", "abcd", 3, true},
		{"empty always passes", "", 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := MaxLength(tt.value, tt.max, "field")
			if (err != nil) != tt.wantErr {
				t.Errorf("MaxLength(%q, %d) err = %v, wantErr %v", tt.value, tt.max, err, tt.wantErr)
			}
		})
	}
}

func TestMinLength(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		min     int
		wantErr bool
	}{
		{"over the minimum passes", "abcd", 3, false},
		{"exactly the minimum passes", "abc", 3, false},
		{"one under the minimum fails", "ab", 3, true},
		{"empty fails a positive minimum", "", 1, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := MinLength(tt.value, tt.min, "field")
			if (err != nil) != tt.wantErr {
				t.Errorf("MinLength(%q, %d) err = %v, wantErr %v", tt.value, tt.min, err, tt.wantErr)
			}
		})
	}
}

func TestEmail(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{"bare address passes", "user@example.com", false},
		{"subdomain and plus tag pass", "user+tag@mail.example.co.uk", false},
		{"missing domain fails", "user@", true},
		{"missing local part fails", "@example.com", true},
		{"no at sign fails", "user.example.com", true},
		{"display-name format is rejected", `"User" <user@example.com>`, true},
		{"surrounding whitespace is tolerated", " user@example.com ", false},
		{"empty fails", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Email(tt.value)
			if (err != nil) != tt.wantErr {
				t.Errorf("Email(%q) err = %v, wantErr %v", tt.value, err, tt.wantErr)
			}
		})
	}
}

func TestSlug(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{"lowercase alphanumeric passes", "abc123", false},
		{"hyphenated segments pass", "my-org-2", false},
		{"single character passes", "a", false},
		{"uppercase fails", "Abc", true},
		{"leading hyphen fails", "-abc", true},
		{"trailing hyphen fails", "abc-", true},
		{"consecutive hyphens fail", "a--b", true},
		{"spaces fail", "a b", true},
		{"empty fails", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Slug(tt.value)
			if (err != nil) != tt.wantErr {
				t.Errorf("Slug(%q) err = %v, wantErr %v", tt.value, err, tt.wantErr)
			}
		})
	}
}
