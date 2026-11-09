package views

import (
	"fmt"
	"strconv"
	"time"

	"github.com/a-h/templ"
	"github.com/el/family-habit-tracker/internal/domain"
	"github.com/el/family-habit-tracker/internal/service"
)

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
func widthStyle(percent int) templ.SafeCSS {
	return templ.SafeCSS("width:" + strconv.Itoa(min(max(percent, 0), 100)) + "%")
}

// dotStyle feeds a day's completion to the ring drawn around its dot.
func dotStyle(percent int) templ.SafeCSS {
	return templ.SafeCSS("--p:" + strconv.Itoa(min(max(percent, 0), 100)))
}

// ISODate renders a date for form values and URLs.
func ISODate(t time.Time) string { return t.Format("2006-01-02") }

// PersonHref is a person's today page.
func PersonHref(personID int64) string { return "/p/" + i64toa(personID) }

// DayHref is a person's page for one day; today's is the plain person URL.
func DayHref(personID int64, day, today time.Time) string {
	if day.Equal(today) {
		return PersonHref(personID)
	}

	return PersonHref(personID) + "/" + ISODate(day)
}

// MonthHref is a person's history for one month.
func MonthHref(personID int64, month time.Time) string {
	return PersonHref(personID) + "/history?m=" + month.Format("2006-01")
}

// EditHref is a person's edit page.
func EditHref(personID int64) string { return PersonHref(personID) + "/edit" }

// Pluralise returns singular or plural according to n.
func Pluralise(n int, singular, plural string) string {
	if n == 1 {
		return singular
	}

	return plural
}

// dayKicker is the small line above a day's date: Today, Yesterday, or the
// weekday.
func dayKicker(v service.DayView) string {
	switch {
	case v.IsToday():
		return "Today"
	case v.IsYesterday():
		return "Yesterday"
	default:
		return v.Day.Format("Monday")
	}
}

// dayTitle is a day's big heading: "Monday 9 Nov", with the year when it is
// not this one.
func dayTitle(day, today time.Time) string {
	if day.Year() != today.Year() {
		return day.Format("Mon 2 Jan 2006")
	}

	return day.Format("Monday 2 Jan")
}

// shortDate is "12 Aug", or "12 Aug 2025" in another year.
func shortDate(day, today time.Time) string {
	if day.Year() != today.Year() {
		return day.Format("2 Jan 2006")
	}

	return day.Format("2 Jan")
}

// timeDate is shortDate for a stored instant.
func timeDate(t *time.Time, today time.Time) string {
	if t == nil {
		return ""
	}

	return shortDate(*t, today)
}

// dowLetter is the one-letter weekday head of the week strip.
func dowLetter(day time.Time) string { return day.Format("Mon")[:1] }

// countText is "3 of 8 today" and its relatives.
func countText(done, total int, today bool) string {
	when := "that day"
	if today {
		when = "today"
	}

	return fmt.Sprintf("%d of %d %s", done, total, when)
}

// stripCount is the "3 / 8" under a name in the family strip.
func stripCount(entry service.StripEntry) string {
	if entry.AllDone() {
		return fmt.Sprintf("%d / %d ✓", entry.Done, entry.Total)
	}

	return fmt.Sprintf("%d / %d", entry.Done, entry.Total)
}

// streakText is the "4 days running" beside a habit, from two days up.
func streakText(days int) string {
	if days >= service.StreakLookback {
		return fmt.Sprintf("%d+ days running", service.StreakLookback)
	}

	return fmt.Sprintf("%d days running", days)
}

// pointsOf is "/ 249 points", or "points" with no goal.
func pointsOf(goal *domain.Reward) string {
	if goal == nil {
		return "points"
	}

	return fmt.Sprintf("/ %d points", goal.Cost)
}

// cellLabel describes a habit's box for a screen reader.
func cellLabel(name string, day time.Time, done bool) string {
	state := "not done"
	if done {
		state = "done"
	}

	return fmt.Sprintf("%s on %s: %s", name, day.Format("Monday 2 January"), state)
}

// spanText describes a stopped habit's history: "19 Apr – 19 Jul · 21 ticks".
func spanText(h service.StoppedHabit, today time.Time) string {
	return fmt.Sprintf("%s – %s · %d %s", shortDate(h.First, today), shortDate(h.Last, today), h.Ticks, Pluralise(h.Ticks, "tick", "ticks"))
}

// goalProgress is "Lego The pickaxes mine · 122 / 249".
func goalProgress(goal *domain.Reward, balance int) string {
	if goal == nil {
		return fmt.Sprintf("%d %s, no goal", balance, Pluralise(balance, "point", "points"))
	}

	return fmt.Sprintf("%s · %d / %d", goal.Name, balance, goal.Cost)
}

// reachedText is "30 Sep · 50 pts".
func reachedText(r domain.Reward, today time.Time) string {
	return fmt.Sprintf("%s · %d pts", timeDate(r.AchievedAt, today), r.Cost)
}

// deductionText is "15 Aug · −3".
func deductionText(a domain.PointAdjustment, today time.Time) string {
	return fmt.Sprintf("%s · %+d", shortDate(a.CreatedAt, today), a.Amount)
}

// barWidth scales one habit's count against the month's best.
func barWidth(count, top int) templ.SafeCSS {
	if top <= 0 {
		return widthStyle(0)
	}

	return widthStyle(count * 100 / top)
}

// Year is the current year, printed in the footer.
func Year() int { return time.Now().Year() }
