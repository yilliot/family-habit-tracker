package service_test

import (
	"slices"
	"testing"
)

func TestAHabitAddedTodayRunsFromToday(t *testing.T) {
	svc, _, ctx := newTestService(t)
	keyu := mustPerson(t, svc, ctx, "可遇")
	sing := mustHabit(t, svc, ctx, keyu, "  sing 30分钟  ")

	if sing.Name != "sing 30分钟" || sing.StartDate != day(2026, 11, 9) || !sing.IsRunning() {
		t.Errorf("new habit = %q from %s running=%v", sing.Name, sing.StartDate.Format("2006-01-02"), sing.IsRunning())
	}

	if got := names(dayView(t, svc, ctx, keyu, day(2026, 11, 9)).Habits); !slices.Equal(got, []string{"sing 30分钟"}) {
		t.Errorf("today's habits = %v", got)
	}

	if got := dayView(t, svc, ctx, keyu, day(2026, 11, 8)).Habits; len(got) != 0 {
		t.Errorf("yesterday should have no habits, got %v", names(got))
	}
}

func TestTheSameHabitCannotRunTwiceForOnePerson(t *testing.T) {
	svc, _, ctx := newTestService(t)
	keyu := mustPerson(t, svc, ctx, "可遇")
	dad := mustPerson(t, svc, ctx, "爸爸")
	mustHabit(t, svc, ctx, keyu, "Good Morning")

	_, err := svc.AddHabit(ctx, keyu.ID, "good morning")
	requireValidation(t, err, "name")

	// Someone else may have a habit of the same name.
	mustHabit(t, svc, ctx, dad, "Good Morning")
}

func TestRenamingKeepsTheTicks(t *testing.T) {
	svc, _, ctx := newTestService(t)
	keyu := mustPerson(t, svc, ctx, "可遇")
	sing := mustHabit(t, svc, ctx, keyu, "sing 30分种")
	mustTick(t, svc, ctx, sing, day(2026, 11, 9))

	renamed, err := svc.RenameHabit(ctx, sing.ID, "sing 30分钟")
	if err != nil || renamed.Name != "sing 30分钟" {
		t.Fatalf("rename: %q %v", renamed.Name, err)
	}

	if view := dayView(t, svc, ctx, keyu, day(2026, 11, 9)); !view.Habits[0].Done {
		t.Error("the renamed habit lost today's tick")
	}
}

func TestMovingReordersAndRenumbers(t *testing.T) {
	svc, _, ctx := newTestService(t)
	keyu := mustPerson(t, svc, ctx, "可遇")
	mustHabit(t, svc, ctx, keyu, "A")
	b := mustHabit(t, svc, ctx, keyu, "B")
	c := mustHabit(t, svc, ctx, keyu, "C")

	if err := svc.MoveHabit(ctx, c.ID, -1); err != nil {
		t.Fatalf("move: %v", err)
	}
	if err := svc.MoveHabit(ctx, b.ID, 1); err != nil {
		t.Fatalf("move: %v", err)
	}
	if err := svc.MoveHabit(ctx, b.ID, 1); err != nil { // already last: stays
		t.Fatalf("move: %v", err)
	}

	if got := names(dayView(t, svc, ctx, keyu, day(2026, 11, 9)).Habits); !slices.Equal(got, []string{"A", "C", "B"}) {
		t.Errorf("order = %v, want [A C B]", got)
	}
}

func TestStoppingEndsTheHabitButKeepsItsHistory(t *testing.T) {
	svc, clock, ctx := newTestService(t)
	keyu := mustPerson(t, svc, ctx, "可遇")

	clock.at(2026, 11, 5)
	piano := mustHabit(t, svc, ctx, keyu, "piano")
	mustTick(t, svc, ctx, piano, day(2026, 11, 5))

	clock.at(2026, 11, 9)
	stopped, err := svc.StopHabit(ctx, piano.ID)
	if err != nil {
		t.Fatalf("stop: %v", err)
	}

	if stopped.EndDate == nil || *stopped.EndDate != day(2026, 11, 8) {
		t.Errorf("not ticked today, so it should end yesterday; ended %v", stopped.EndDate)
	}

	if got := dayView(t, svc, ctx, keyu, day(2026, 11, 9)).Habits; len(got) != 0 {
		t.Errorf("a stopped habit still shows today: %v", names(got))
	}

	if view := dayView(t, svc, ctx, keyu, day(2026, 11, 5)); len(view.Habits) != 1 || !view.Habits[0].Done {
		t.Errorf("5 Nov should still show piano done, got %+v", view.Habits)
	}

	edit, err := svc.Edit(ctx, keyu.ID)
	if err != nil || len(edit.Stopped) != 1 || edit.Stopped[0].Ticks != 1 {
		t.Errorf("edit page stopped list = %+v (%v), want piano with 1 tick", edit.Stopped, err)
	}
}

func TestStoppingAfterTodaysTickKeepsToday(t *testing.T) {
	svc, clock, ctx := newTestService(t)
	keyu := mustPerson(t, svc, ctx, "可遇")

	clock.at(2026, 11, 1)
	piano := mustHabit(t, svc, ctx, keyu, "piano")
	clock.at(2026, 11, 9)
	mustTick(t, svc, ctx, piano, day(2026, 11, 9))

	stopped, err := svc.StopHabit(ctx, piano.ID)
	if err != nil || stopped.EndDate == nil || *stopped.EndDate != day(2026, 11, 9) {
		t.Fatalf("stop after today's tick should end today; got %v, %v", stopped.EndDate, err)
	}

	if view := dayView(t, svc, ctx, keyu, day(2026, 11, 9)); len(view.Habits) != 1 || !view.Habits[0].Done {
		t.Error("today's tick should still show on today's list")
	}
}

func TestAHabitAddedByMistakeTodayIsSimplyRemoved(t *testing.T) {
	svc, _, ctx := newTestService(t)
	keyu := mustPerson(t, svc, ctx, "可遇")
	oops := mustHabit(t, svc, ctx, keyu, "typo")

	if _, err := svc.StopHabit(ctx, oops.ID); err != nil {
		t.Fatalf("stop: %v", err)
	}

	if _, err := svc.Store().Habit(ctx, oops.ID); err == nil {
		t.Error("a habit added today with no ticks should be removed outright")
	}

	if edit, _ := svc.Edit(ctx, keyu.ID); len(edit.Stopped) != 0 {
		t.Errorf("nothing should be listed as stopped, got %+v", edit.Stopped)
	}
}

func TestRestartingSoonResumesWithoutAGap(t *testing.T) {
	svc, clock, ctx := newTestService(t)
	keyu := mustPerson(t, svc, ctx, "可遇")

	clock.at(2026, 11, 1)
	piano := mustHabit(t, svc, ctx, keyu, "piano")
	clock.at(2026, 11, 9)

	if _, err := svc.StopHabit(ctx, piano.ID); err != nil {
		t.Fatalf("stop: %v", err)
	}

	restarted, err := svc.RestartHabit(ctx, piano.ID)
	if err != nil {
		t.Fatalf("restart: %v", err)
	}

	if restarted.ID != piano.ID || !restarted.IsRunning() {
		t.Errorf("a mistaken stop should be undone on the same row, got #%d running=%v", restarted.ID, restarted.IsRunning())
	}
}

func TestRestartingLaterStartsANewRunThatHistoryJoins(t *testing.T) {
	svc, clock, ctx := newTestService(t)
	keyu := mustPerson(t, svc, ctx, "可遇")

	clock.at(2026, 10, 1)
	piano := mustHabit(t, svc, ctx, keyu, "piano")
	mustTick(t, svc, ctx, piano, day(2026, 10, 1))
	clock.at(2026, 10, 2)
	mustTick(t, svc, ctx, piano, day(2026, 10, 2))
	if _, err := svc.StopHabit(ctx, piano.ID); err != nil {
		t.Fatalf("stop: %v", err)
	}

	clock.at(2026, 11, 9)
	again, err := svc.RestartHabit(ctx, piano.ID)
	if err != nil {
		t.Fatalf("restart: %v", err)
	}

	if again.ID == piano.ID || again.StartDate != day(2026, 11, 9) {
		t.Errorf("an old habit should restart as a new run from today, got #%d from %s", again.ID, again.StartDate.Format("2006-01-02"))
	}

	_, err = svc.RestartHabit(ctx, piano.ID)
	requireValidation(t, err, "habit")

	if edit, _ := svc.Edit(ctx, keyu.ID); len(edit.Stopped) != 0 {
		t.Errorf("a name that is running again should leave the stopped list, got %+v", edit.Stopped)
	}
}
