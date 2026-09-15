// Package views renders every page. Templates are compiled by templ; this file
// holds the formatting helpers they call.
package views

import (
	"fmt"
	"strings"
	"time"

	"github.com/el/family-habit-tracker/internal/service"
)

// dowInitials indexes the single-letter weekday heads used across the grid's
// date row, matching the printed sheet.
var dowInitials = [...]string{"S", "M", "T", "W", "T", "F", "S"}

// FormatLong renders a date the way the masthead prints it: "Sun · Apr 19, 2026".
func FormatLong(t time.Time) string {
	return fmt.Sprintf("%s · %s %d, %d", t.Format("Mon"), t.Format("Jan"), t.Day(), t.Year())
}

// FormatRange renders a sprint's span: "Apr 19 – May 16, 2026".
func FormatRange(start, end time.Time) string {
	return fmt.Sprintf("%s %d – %s %d, %d",
		start.Format("Jan"), start.Day(),
		end.Format("Jan"), end.Day(), end.Year(),
	)
}

// FormatShort renders a compact date: "Apr 19".
func FormatShort(t time.Time) string {
	return fmt.Sprintf("%s %d", t.Format("Jan"), t.Day())
}

// DowInitial is the single-letter weekday head for a grid column.
func DowInitial(t time.Time) string { return dowInitials[int(t.Weekday())] }

// DayNumber is the day-of-month shown in a grid column head.
func DayNumber(t time.Time) int { return t.Day() }

// ISODate renders a date for form values and DOM identifiers.
func ISODate(t time.Time) string { return t.Format("2006-01-02") }

// Pluralise returns singular or plural according to n.
func Pluralise(n int, singular, plural string) string {
	if n == 1 {
		return singular
	}

	return plural
}

// Year is the current year, printed in the footer.
func Year() int { return time.Now().Year() }

// cellLabel describes a grid square for a screen reader, which otherwise sees
// only an unlabelled button in a sea of identical ones.
func cellLabel(habit service.GridHabit, day service.GridDay, ticked bool) string {
	state := "not done"
	if ticked {
		state = "done"
	}

	return fmt.Sprintf("%s on %s: %s", habit.Habit.Name, FormatLong(day.Date), state)
}

// The helpers below compose complete strings in Go rather than letting a
// template concatenate several expressions. templ preserves the newline between
// adjacent expressions as a space, which would otherwise show up as a doubled
// gap in the rendered text.

// BalanceText renders a participant's running total, against their goal's cost
// when they have one.
func BalanceText(participant service.GridParticipant) string {
	if participant.ActiveReward == nil {
		return fmt.Sprintf("%d pts", participant.Balance)
	}

	return fmt.Sprintf("%d / %d", participant.Balance, participant.ActiveReward.Cost)
}

// RewardMetaText renders the line under a goal's name: progress, and either the
// gap remaining or the fact that it has been earned.
func RewardMetaText(participant service.GridParticipant) string {
	reward := participant.ActiveReward
	if reward == nil {
		return ""
	}

	if reward.IsAchieved() {
		return fmt.Sprintf("%d / %d pts", participant.Balance, reward.Cost)
	}

	return fmt.Sprintf("%d / %d pts · %d to go", participant.Balance, reward.Cost, participant.Remaining())
}

// EditionText renders the masthead eyebrow, naming the sprint's edition number
// when a sprint is on the page.
func EditionText(sprintID *int64) string {
	if sprintID == nil {
		return "The Family Ledger"
	}

	return fmt.Sprintf("The Family Ledger · No. %02d", *sprintID)
}

// HistorySummary renders the one-line summary under an archived sprint.
func HistorySummary(entry service.HistoryEntry) string {
	return fmt.Sprintf("%d %s · %d %s · %d %s achieved",
		entry.ParticipantCount, Pluralise(entry.ParticipantCount, "participant", "participants"),
		entry.TicksEarned, Pluralise(entry.TicksEarned, "tick", "ticks"),
		len(entry.Achievers), Pluralise(len(entry.Achievers), "reward", "rewards"),
	)
}

// AchieverSummary lists who earned what in an archived sprint.
func AchieverSummary(achievers []service.Achiever) string {
	parts := make([]string, 0, len(achievers))
	for _, achiever := range achievers {
		parts = append(parts, achiever.Name+" → "+achiever.RewardName)
	}

	return strings.Join(parts, " · ")
}

// TotalsSavingText renders a person's progress toward their goal on the totals
// page, or their banked surplus when they have not chosen one.
func TotalsSavingText(person service.PersonTotals) string {
	if person.ActiveReward == nil {
		return fmt.Sprintf("%d %s banked · no reward chosen",
			person.CurrentBalance, Pluralise(person.CurrentBalance, "point", "points"))
	}

	return fmt.Sprintf("Saving for %s · %d / %d pts",
		person.ActiveReward.Name, person.CurrentBalance, person.ActiveReward.Cost)
}

// SprintTallyText renders the running total printed under the grid.
func SprintTallyText(grid service.Grid) string {
	return fmt.Sprintf("%d of %d ticks banked this sprint", grid.TotalEarned(), grid.TotalPossible())
}

// RewardOptionText labels a goal in the "working toward" picker.
func RewardOptionText(name string, cost int) string {
	return fmt.Sprintf("%s · %dpt", name, cost)
}

// CarriedInText labels how many points someone brings into a new sprint.
func CarriedInText(balance int) string {
	return fmt.Sprintf("%d pts carried in", balance)
}

// KeepRewardText labels the "keep the current goal" option.
func KeepRewardText(name string, cost int) string {
	return fmt.Sprintf("Keep %s · %dpt", name, cost)
}

// SprintRangeText renders a sprint's span with an arrow, as the settings header
// prints it.
func SprintRangeText(start, end time.Time) string {
	return FormatLong(start) + "  →  " + FormatLong(end)
}
