package validate

import (
	"fmt"
	"net/mail"
	"regexp"
	"strings"
)

var slugPattern = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

// Required checks that a string is non-empty after trimming whitespace.
func Required(value, field string) error {
	if strings.TrimSpace(value) == "" {
		return fmt.Errorf("%s is required", field)
	}
	return nil
}

// MaxLength checks that a string does not exceed max characters.
func MaxLength(value string, max int, field string) error {
	if len(value) > max {
		return fmt.Errorf("%s must be at most %d characters", field, max)
	}
	return nil
}

// Email checks that a string is a valid email address.
// Uses net/mail.ParseAddress per ADR-014 (whitelist validation over regex).
// Rejects display-name format ("Name" <addr>) — only bare addresses accepted.
func Email(value string) error {
	addr, err := mail.ParseAddress(value)
	if err != nil || addr.Address != strings.TrimSpace(value) {
		return fmt.Errorf("invalid email address")
	}
	return nil
}

// Slug checks that a string is a valid URL slug (lowercase alphanumeric + hyphens).
func Slug(value string) error {
	if !slugPattern.MatchString(value) {
		return fmt.Errorf("slug must contain only lowercase letters, numbers, and hyphens")
	}
	return nil
}

// MinLength checks that a string has at least min characters.
func MinLength(value string, min int, field string) error {
	if len(value) < min {
		return fmt.Errorf("%s must be at least %d characters", field, min)
	}
	return nil
}
