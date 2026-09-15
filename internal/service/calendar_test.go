package service_test

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/el/family-habit-tracker/internal/db"
	"github.com/el/family-habit-tracker/internal/service"
	"github.com/el/family-habit-tracker/internal/store"
)

// Calendar edge cases only surface once an app has been running long enough to
// meet them: a sprint across New Year, a leap day, a daylight-saving change.
// Each gets a clock pinned to the moment it happens.

func newServiceAt(t *testing.T, now time.Time) (*service.Service, *store.Store, context.Context) {
	t.Helper()

	sqlDB, err := db.Open(t.Context(), filepath.Join(t.TempDir(), "calendar.sqlite"))
	if err != nil {
		t.Fatalf("open: %v", err)
	}

	t.Cleanup(func() { sqlDB.Close() })

	dataStore := store.New(sqlDB).WithClock(func() time.Time { return now })

	return service.New(dataStore), dataStore, t.Context()
}

func TestSprintAcrossNewYear(t *testing.T) {
	svc, _, ctx := newServiceAt(t, time.Date(2028, 1, 10, 12, 0, 0, 0, time.UTC))

	person := mustPerson(t, svc, ctx, "可遇")

	sprint, err := svc.StartSprint(ctx, day(2027, 12, 20), day(2028, 1, 16), []service.ParticipantInput{{
		PersonID: person.ID,
		Habits:   []service.HabitInput{{Name: "阅读", DisplayOrder: 1}},
		Reward:   service.RewardChoice{Mode: service.RewardNone},
	}})
	if err != nil {
		t.Fatalf("start sprint: %v", err)
	}

	participant := participantFor(t, svc, ctx, sprint, person)
	habit := habitsOf(t, svc, ctx, participant)[0]

	for _, d := range []time.Time{day(2027, 12, 31), day(2028, 1, 1)} {
		if _, err := svc.ToggleTick(ctx, habit.ID, d); err != nil {
			t.Fatalf("tick %s: %v", d.Format("2006-01-02"), err)
		}
	}

	grid, err := svc.BuildGrid(ctx, sprint)
	if err != nil {
		t.Fatalf("build grid: %v", err)
	}

	if len(grid.Days) != 28 {
		t.Errorf("days = %d, want 28", len(grid.Days))
	}

	want := []service.MonthGroup{{Label: "December 2027", Span: 12}, {Label: "January 2028", Span: 16}}
	if len(grid.MonthGroups) != len(want) {
		t.Fatalf("month groups = %v, want %v", grid.MonthGroups, want)
	}
	for i := range want {
		if grid.MonthGroups[i] != want[i] {
			t.Errorf("month group %d = %v, want %v", i, grid.MonthGroups[i], want[i])
		}
	}

	if got := grid.Participants[0].Balance; got != 2 {
		t.Errorf("balance across New Year = %d, want 2", got)
	}

	ticks := grid.Participants[0].Habits[0]
	if !ticks.IsTicked("2027-12-31") || !ticks.IsTicked("2028-01-01") {
		t.Errorf("ticks on both sides of midnight should be kept, got %v", ticks.Ticked)
	}
}

// The same calendar month in two different years must not merge into one
// header, which would happen if months were compared without their year.
func TestSameMonthInDifferentYearsIsNotMerged(t *testing.T) {
	svc, _, ctx := newServiceAt(t, time.Date(2027, 1, 1, 12, 0, 0, 0, time.UTC))

	person := mustPerson(t, svc, ctx, "奇乐")

	sprint, err := svc.StartSprint(ctx, day(2026, 1, 30), day(2027, 1, 2), []service.ParticipantInput{
		{PersonID: person.ID, Reward: service.RewardChoice{Mode: service.RewardNone}},
	})
	if err != nil {
		t.Fatalf("start sprint: %v", err)
	}

	grid, err := svc.BuildGrid(ctx, sprint)
	if err != nil {
		t.Fatalf("build grid: %v", err)
	}

	first, last := grid.MonthGroups[0], grid.MonthGroups[len(grid.MonthGroups)-1]
	if first.Label != "January 2026" || last.Label != "January 2027" {
		t.Errorf("first/last month groups = %q / %q, want January 2026 / January 2027", first.Label, last.Label)
	}
}

func TestLeapDayCanBeTicked(t *testing.T) {
	svc, _, ctx := newServiceAt(t, time.Date(2028, 3, 5, 12, 0, 0, 0, time.UTC))

	person := mustPerson(t, svc, ctx, "妈妈")

	sprint, err := svc.StartSprint(ctx, day(2028, 2, 15), day(2028, 3, 13), []service.ParticipantInput{{
		PersonID: person.ID,
		Habits:   []service.HabitInput{{Name: "聆听", DisplayOrder: 1}},
		Reward:   service.RewardChoice{Mode: service.RewardNone},
	}})
	if err != nil {
		t.Fatalf("start sprint: %v", err)
	}

	participant := participantFor(t, svc, ctx, sprint, person)
	habit := habitsOf(t, svc, ctx, participant)[0]

	if _, err := svc.ToggleTick(ctx, habit.ID, day(2028, 2, 29)); err != nil {
		t.Fatalf("tick leap day: %v", err)
	}

	grid, err := svc.BuildGrid(ctx, sprint)
	if err != nil {
		t.Fatalf("build grid: %v", err)
	}

	if len(grid.Days) != 28 {
		t.Errorf("days = %d, want 28 (15 in February including the 29th, 13 in March)", len(grid.Days))
	}

	if grid.MonthGroups[0].Span != 15 {
		t.Errorf("February span = %d, want 15", grid.MonthGroups[0].Span)
	}

	if !grid.Participants[0].Habits[0].IsTicked("2028-02-29") {
		t.Error("the leap day tick should be stored and shown")
	}
}

// "Today" is the family's day, not UTC's. At 7am in Singapore it is still the
// previous day in UTC; the tablet must nevertheless allow today's cell.
func TestTodayFollowsTheConfiguredTimezone(t *testing.T) {
	singapore, err := time.LoadLocation("Asia/Singapore")
	if err != nil {
		t.Skipf("timezone data unavailable: %v", err)
	}

	newYork, err := time.LoadLocation("America/New_York")
	if err != nil {
		t.Skipf("timezone data unavailable: %v", err)
	}

	cases := []struct {
		name string
		now  time.Time
		want string
	}{
		{"early morning in Singapore is still yesterday in UTC", time.Date(2026, 9, 15, 7, 0, 0, 0, singapore), "2026-09-15"},
		{"just after the spring-forward gap", time.Date(2026, 3, 8, 3, 30, 0, 0, newYork), "2026-03-08"},
		{"inside the repeated fall-back hour", time.Date(2026, 11, 1, 1, 30, 0, 0, newYork), "2026-11-01"},
		{"one minute before midnight", time.Date(2026, 12, 31, 23, 59, 0, 0, newYork), "2026-12-31"},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			_, dataStore, _ := newServiceAt(t, testCase.now)

			if got := dataStore.Today().Format("2006-01-02"); got != testCase.want {
				t.Errorf("Today() = %s, want %s", got, testCase.want)
			}
		})
	}
}

// Across a daylight-saving change the grid must still have exactly one column
// per calendar day: no duplicated day in autumn, no missing one in spring.
func TestSprintAcrossDaylightSavingHasOneColumnPerDay(t *testing.T) {
	newYork, err := time.LoadLocation("America/New_York")
	if err != nil {
		t.Skipf("timezone data unavailable: %v", err)
	}

	svc, _, ctx := newServiceAt(t, time.Date(2026, 11, 20, 12, 0, 0, 0, newYork))

	person := mustPerson(t, svc, ctx, "爸爸")

	sprint, err := svc.StartSprint(ctx, day(2026, 10, 25), day(2026, 11, 21), []service.ParticipantInput{
		{PersonID: person.ID, Reward: service.RewardChoice{Mode: service.RewardNone}},
	})
	if err != nil {
		t.Fatalf("start sprint: %v", err)
	}

	grid, err := svc.BuildGrid(ctx, sprint)
	if err != nil {
		t.Fatalf("build grid: %v", err)
	}

	seen := map[string]bool{}
	for _, d := range grid.Days {
		if seen[d.ISO] {
			t.Errorf("day %s appears twice", d.ISO)
		}
		seen[d.ISO] = true
	}

	if len(grid.Days) != 28 {
		t.Errorf("days = %d, want 28", len(grid.Days))
	}
}
