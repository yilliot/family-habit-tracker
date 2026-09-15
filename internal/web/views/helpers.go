package views

import (
	"strconv"
	"strings"

	"github.com/el/family-habit-tracker/internal/domain"
)

// documentTitle composes the browser title, keeping the app name as a suffix.
func (p Page) documentTitle() string {
	if p.Title == "" {
		return "Family Habit Tracker"
	}

	return p.Title + " · Family Habit Tracker"
}

// boolAttr renders a boolean as the "true"/"false" string that the CSS
// attribute selectors in app.css match on.
func boolAttr(value bool) string {
	if value {
		return "true"
	}

	return "false"
}

// itoa renders an int for use in an attribute value.
func itoa(value int) string { return strconv.Itoa(value) }

// i64toa renders an int64 identifier for use in an attribute value.
func i64toa(value int64) string { return strconv.FormatInt(value, 10) }

// widthStyle renders the inline width of a progress bar fill.
func widthStyle(percent int) string {
	return "width:" + strconv.Itoa(percent) + "%"
}

// percentOf renders a balance against a cost as a bar percentage, clamped to
// the 0–100 the track can draw.
func percentOf(balance, cost int) int {
	if cost <= 0 {
		return 0
	}

	return min(max(balance*100/cost, 0), 100)
}

// habitLines renders a participant's habits as newline-separated text for the
// "one per line" editor, which is far quicker to revise at a family meeting
// than a column of individual inputs.
func habitLines(habits []domain.Habit) string {
	names := make([]string, 0, len(habits))
	for _, habit := range habits {
		names = append(names, habit.Name)
	}

	return strings.Join(names, "\n")
}

// editionText renders the masthead eyebrow for the page's sprint.
func (p Page) editionText() string {
	if p.Sprint == nil {
		return EditionText(nil)
	}

	return EditionText(&p.Sprint.ID)
}
