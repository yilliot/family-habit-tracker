package service_test

import (
	"slices"
	"testing"
)

func TestADayShowsTheHabitsRunningThatDay(t *testing.T) {
	svc, clock, ctx := newTestService(t)
	keyu := mustPerson(t, svc, ctx, "可遇")

	clock.at(2026, 11, 2)
	desk := mustHabit(t, svc, ctx, keyu, "整理书桌")
	clock.at(2026, 11, 4)
	mustHabit(t, svc, ctx, keyu, "piano")
	clock.at(2026, 11, 6)
	if _, err := svc.StopHabit(ctx, desk.ID); err != nil {
		t.Fatalf("stop: %v", err)
	}
	clock.at(2026, 11, 9)

	cases := map[int][]string{
		1: nil,
		2: {"整理书桌"},
		4: {"整理书桌", "piano"},
		5: {"整理书桌", "piano"},
		6: {"piano"},
		9: {"piano"},
	}

	for d, want := range cases {
		if got := names(dayView(t, svc, ctx, keyu, day(2026, 11, d)).Habits); !slices.Equal(got, want) {
			t.Errorf("%d Nov habits = %v, want %v", d, got, want)
		}
	}

	view := dayView(t, svc, ctx, keyu, day(2026, 11, 2))
	if view.HasPrev() {
		t.Error("the first day with a habit should have no earlier day to go to")
	}
}

func TestTheWeekAndTheEditableWindow(t *testing.T) {
	svc, clock, ctx := newTestService(t)
	keyu := mustPerson(t, svc, ctx, "可遇")

	clock.at(2026, 10, 26)
	a := mustHabit(t, svc, ctx, keyu, "A")
	mustHabit(t, svc, ctx, keyu, "B")
	clock.at(2026, 11, 9)
	mustTick(t, svc, ctx, a, day(2026, 11, 4))

	view := dayView(t, svc, ctx, keyu, day(2026, 11, 4))

	if !view.Editable {
		t.Error("five days ago should be editable")
	}

	if view.Week[0].Date != day(2026, 11, 2) || !view.Week[2].IsSelected {
		t.Errorf("week should run Mon 2 Nov with Wed 4 Nov selected, got %s", view.Week[0].ISO)
	}

	if cell := view.Week[2]; cell.Done != 1 || cell.Total != 2 || cell.Percent() != 50 {
		t.Errorf("4 Nov = %d of %d (%d%%), want 1 of 2", cell.Done, cell.Total, cell.Percent())
	}

	if old := dayView(t, svc, ctx, keyu, day(2026, 10, 27)); old.Editable {
		t.Error("27 Oct is more than a week back and should be read-only")
	}

	if future := dayView(t, svc, ctx, keyu, day(2026, 12, 1)); !future.IsToday() {
		t.Error("a future day should be pulled back to today")
	}
}

func TestStreaksCountAcrossRunsOfTheSameName(t *testing.T) {
	svc, clock, ctx := newTestService(t)
	keyu := mustPerson(t, svc, ctx, "可遇")

	clock.at(2026, 11, 3)
	first := mustHabit(t, svc, ctx, keyu, "piano")
	mustTick(t, svc, ctx, first, day(2026, 11, 3))
	clock.at(2026, 11, 4)
	mustTick(t, svc, ctx, first, day(2026, 11, 4))
	if _, err := svc.StopHabit(ctx, first.ID); err != nil { // ticked today: ends 4 Nov
		t.Fatalf("stop: %v", err)
	}

	clock.at(2026, 11, 5)
	second := mustHabit(t, svc, ctx, keyu, "Piano")
	mustTick(t, svc, ctx, second, day(2026, 11, 5))

	view := dayView(t, svc, ctx, keyu, day(2026, 11, 5))
	if view.Habits[0].Streak != 3 {
		t.Errorf("streak = %d, want 3 across both runs", view.Habits[0].Streak)
	}
}

func TestAMonthOfHistory(t *testing.T) {
	svc, clock, ctx := newTestService(t)
	keyu := mustPerson(t, svc, ctx, "可遇")

	clock.at(2026, 10, 30)
	story := mustHabit(t, svc, ctx, keyu, "story 30分钟")
	sing := mustHabit(t, svc, ctx, keyu, "sing")
	mustTick(t, svc, ctx, story, day(2026, 10, 30))
	clock.at(2026, 11, 2)
	mustTick(t, svc, ctx, story, day(2026, 11, 1))
	mustTick(t, svc, ctx, story, day(2026, 11, 2))
	mustTick(t, svc, ctx, sing, day(2026, 11, 2))

	reason := "zoo"
	if _, err := svc.CreateAdjustment(ctx, keyu.ID, -2, &reason); err != nil {
		t.Fatalf("deduct: %v", err)
	}

	clock.at(2026, 11, 9)

	view, err := svc.Month(ctx, keyu.ID, day(2026, 11, 20))
	if err != nil {
		t.Fatalf("month: %v", err)
	}

	if view.Ticks != 3 || view.ActiveDays != 2 || view.Earned != 4 || view.Balance != 2 {
		t.Errorf("November = %d ticks on %d days, %d all-time, balance %d; want 3, 2, 4, 2",
			view.Ticks, view.ActiveDays, view.Earned, view.Balance)
	}

	if len(view.ByHabit) != 2 || view.ByHabit[0].Name != "story 30分钟" || view.ByHabit[0].Count != 2 {
		t.Errorf("by habit = %+v", view.ByHabit)
	}

	if len(view.Deductions) != 1 || view.Deductions[0].Amount != -2 {
		t.Errorf("deductions = %+v", view.Deductions)
	}

	if !view.HasPrev() || view.HasNext() {
		t.Errorf("November should link back to October and not forward (prev=%v next=%v)", view.HasPrev(), view.HasNext())
	}

	if first := view.Grid.Weeks[0][0]; first.Date != day(2026, 10, 26) || first.InMonth {
		t.Errorf("grid should open on Mon 26 Oct, outside the month; got %s in=%v", first.ISO, first.InMonth)
	}
}

func TestTheFamilyStripCountsToday(t *testing.T) {
	svc, _, ctx := newTestService(t)
	dad := mustPerson(t, svc, ctx, "爸爸")
	keyu := mustPerson(t, svc, ctx, "可遇")
	a := mustHabit(t, svc, ctx, keyu, "A")
	mustHabit(t, svc, ctx, keyu, "B")
	mustTick(t, svc, ctx, a, day(2026, 11, 9))

	strip, err := svc.Strip(ctx)
	if err != nil {
		t.Fatalf("strip: %v", err)
	}

	if len(strip) != 2 || strip[0].Person.ID != dad.ID || strip[1].Done != 1 || strip[1].Total != 2 || strip[1].AllDone() {
		t.Errorf("strip = %+v", strip)
	}

	if err := svc.ArchivePerson(ctx, dad.ID); err != nil {
		t.Fatalf("archive: %v", err)
	}

	if strip, _ := svc.Strip(ctx); len(strip) != 1 {
		t.Errorf("an archived person should leave the strip, got %d entries", len(strip))
	}

	if err := svc.RestorePerson(ctx, dad.ID); err != nil {
		t.Fatalf("restore: %v", err)
	}

	if strip, _ := svc.Strip(ctx); len(strip) != 2 {
		t.Errorf("a restored person should come back, got %d entries", len(strip))
	}
}
