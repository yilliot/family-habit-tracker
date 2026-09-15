package domain

import (
	"fmt"
	"sort"
	"strings"
	"unicode/utf8"
)

// Field length limits, matching the columns and the rules they replace.
const (
	MaxPersonNameLen = 50
	MaxHabitNameLen  = 80
	MaxRewardNameLen = 80
	MaxReasonLen     = 120
	MaxRewardCost    = 9999
	MaxAdjustment    = 9999
)

// ValidationError reports one or more rejected fields, keyed by field name.
// Handlers render it back into the form that produced it; the zero value
// carries no errors.
type ValidationError struct {
	Fields map[string]string
}

// Error implements error, listing fields in a stable order so messages and
// tests do not depend on map iteration order.
func (v *ValidationError) Error() string {
	if v == nil || len(v.Fields) == 0 {
		return "validation failed"
	}

	keys := make([]string, 0, len(v.Fields))
	for k := range v.Fields {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, fmt.Sprintf("%s: %s", k, v.Fields[k]))
	}

	return strings.Join(parts, "; ")
}

// First returns one message, preferring the named field when present. It backs
// the single-line toast shown for requests that have nowhere to put per-field
// errors.
func (v *ValidationError) First(preferred ...string) string {
	if v == nil || len(v.Fields) == 0 {
		return ""
	}

	for _, name := range preferred {
		if msg, ok := v.Fields[name]; ok {
			return msg
		}
	}

	keys := make([]string, 0, len(v.Fields))
	for k := range v.Fields {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	return v.Fields[keys[0]]
}

// Invalid builds a ValidationError for a single field.
func Invalid(field, message string) *ValidationError {
	return &ValidationError{Fields: map[string]string{field: message}}
}

// Validator accumulates field errors so a whole form can be checked in one
// pass and reported at once, rather than failing on the first bad field.
type Validator struct {
	fields map[string]string
}

// Add records a message for field unless one is already present, keeping the
// first (most specific) failure for that field.
func (val *Validator) Add(field, message string) {
	if val.fields == nil {
		val.fields = make(map[string]string)
	}

	if _, exists := val.fields[field]; !exists {
		val.fields[field] = message
	}
}

// OK reports whether nothing has been rejected.
func (val *Validator) OK() bool { return len(val.fields) == 0 }

// Err returns the accumulated ValidationError, or nil when everything passed.
// The nil return is a true nil error, safe to compare against nil.
func (val *Validator) Err() error {
	if val.OK() {
		return nil
	}

	return &ValidationError{Fields: val.fields}
}

// Name trims and validates a required, length-limited name, returning the
// cleaned value. Length is counted in runes so Chinese names are not penalised
// by their byte length.
func (val *Validator) Name(field, raw string, max int) string {
	trimmed := strings.TrimSpace(raw)

	switch {
	case trimmed == "":
		val.Add(field, "This field is required.")
	case utf8.RuneCountInString(trimmed) > max:
		val.Add(field, fmt.Sprintf("Must be %d characters or fewer.", max))
	}

	return trimmed
}

// IntRange validates that value falls within [min, max].
func (val *Validator) IntRange(field string, value, min, max int) int {
	if value < min || value > max {
		val.Add(field, fmt.Sprintf("Must be between %d and %d.", min, max))
	}

	return value
}

// OptionalText trims an optional free-text value and enforces a maximum
// length, returning nil when the result is empty.
func (val *Validator) OptionalText(field, raw string, max int) *string {
	trimmed := strings.TrimSpace(raw)

	if trimmed == "" {
		return nil
	}

	if utf8.RuneCountInString(trimmed) > max {
		val.Add(field, fmt.Sprintf("Must be %d characters or fewer.", max))
	}

	return &trimmed
}
