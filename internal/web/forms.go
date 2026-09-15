package web

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/el/family-habit-tracker/internal/domain"
)

// parseInt64 reads a decimal identifier.
func parseInt64(raw string) (int64, error) { return strconv.ParseInt(raw, 10, 64) }

// form parses a request body, returning the values and whether parsing worked.
func (s *Server) form(w http.ResponseWriter, r *http.Request) bool {
	if err := r.ParseForm(); err != nil {
		s.reportError(w, r, domain.Invalid("form", "That form could not be read."), "/")

		return false
	}

	return true
}

// field reads a trimmed form value.
func field(r *http.Request, name string) string {
	return strings.TrimSpace(r.PostFormValue(name))
}

// intField reads an integer form value, falling back to a default when the
// field is absent or unparseable.
func intField(r *http.Request, name string, fallback int) int {
	raw := field(r, name)
	if raw == "" {
		return fallback
	}

	value, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}

	return value
}

// optionalIntField reads an integer form value, yielding nil when absent so the
// service layer can apply its own default.
func optionalIntField(r *http.Request, name string) *int {
	raw := field(r, name)
	if raw == "" {
		return nil
	}

	value, err := strconv.Atoi(raw)
	if err != nil {
		return nil
	}

	return &value
}

// dateField reads a date input as a day in UTC.
func dateField(r *http.Request, name string) (time.Time, error) {
	parsed, err := time.ParseInLocation("2006-01-02", field(r, name), time.UTC)
	if err != nil {
		return time.Time{}, domain.Invalid(name, "Enter a valid date.")
	}

	return parsed, nil
}

// habitLines turns the "one habit per line" editor into habit inputs, dropping
// blank lines and trimming stray whitespace so a trailing newline does not
// create an empty habit.
func habitLines(raw string) []string {
	var names []string

	for _, line := range strings.Split(raw, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" {
			names = append(names, trimmed)
		}
	}

	return names
}
