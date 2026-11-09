package service_test

import (
	"testing"
	"time"
)

func TestTickingTodayEarnsAPointAndUntickingTakesItBack(t *testing.T) {
	svc, _, ctx := newTestService(t)
	keyu := mustPerson(t, svc, ctx, "可遇")
	piano := mustHabit(t, svc, ctx, keyu, "piano 20分钟")

	if result := mustTick(t, svc, ctx, piano, day(2026, 11, 9)); !result.Ticked {
		t.Fatal("first tap should tick")
	}

	if got := balanceOf(t, svc, ctx, keyu); got != 1 {
		t.Errorf("balance after a tick = %d, want 1", got)
	}

	if result := mustTick(t, svc, ctx, piano, day(2026, 11, 9)); result.Ticked {
		t.Fatal("second tap should untick")
	}

	if got := balanceOf(t, svc, ctx, keyu); got != 0 {
		t.Errorf("balance after untick = %d, want 0", got)
	}
}

func TestTheLastSevenDaysCanBeTickedAndNothingOlder(t *testing.T) {
	svc, clock, ctx := newTestService(t)
	keyu := mustPerson(t, svc, ctx, "可遇")

	clock.at(2026, 10, 20)
	reading := mustHabit(t, svc, ctx, keyu, "30分钟阅读时间")
	clock.at(2026, 11, 9)

	mustTick(t, svc, ctx, reading, day(2026, 11, 3)) // six days back: still open

	_, err := svc.ToggleTick(ctx, reading.ID, day(2026, 11, 2))
	requireValidation(t, err, "tick_date")

	_, err = svc.ToggleTick(ctx, reading.ID, day(2026, 11, 10))
	requireValidation(t, err, "tick_date")
}

func TestAHabitCannotBeTickedOnADayItWasNotRunning(t *testing.T) {
	svc, _, ctx := newTestService(t)
	keyu := mustPerson(t, svc, ctx, "可遇")
	sing := mustHabit(t, svc, ctx, keyu, "sing 30分钟") // starts today

	_, err := svc.ToggleTick(ctx, sing.ID, day(2026, 11, 8))
	requireValidation(t, err, "tick_date")
}

func TestAnArchivedPersonCannotTick(t *testing.T) {
	svc, _, ctx := newTestService(t)
	keyu := mustPerson(t, svc, ctx, "可遇")
	sing := mustHabit(t, svc, ctx, keyu, "sing")

	if err := svc.ArchivePerson(ctx, keyu.ID); err != nil {
		t.Fatalf("archive: %v", err)
	}

	_, err := svc.ToggleTick(ctx, sing.ID, day(2026, 11, 9))
	requireValidation(t, err, "person")
}

func TestReachingTheGoalIsCelebratedOnceAndNeverRevoked(t *testing.T) {
	svc, clock, ctx := newTestService(t)
	qile := mustPerson(t, svc, ctx, "奇乐")

	clock.at(2026, 11, 7)
	bed := mustHabit(t, svc, ctx, qile, "整理床")
	clock.at(2026, 11, 9)

	if _, achieved, err := svc.SetGoal(ctx, qile.ID, "Spiderman 笔盒", 2); err != nil || achieved != nil {
		t.Fatalf("set goal: achieved=%v err=%v", achieved, err)
	}

	if result := mustTick(t, svc, ctx, bed, day(2026, 11, 7)); result.JustAchieved != nil {
		t.Fatal("one point of two should not reach the goal")
	}

	result := mustTick(t, svc, ctx, bed, day(2026, 11, 8))
	if result.JustAchieved == nil || result.JustAchieved.Name != "Spiderman 笔盒" {
		t.Fatalf("second point should reach the goal, got %+v", result.JustAchieved)
	}

	if got := balanceOf(t, svc, ctx, qile); got != 0 {
		t.Errorf("balance after spending 2 of 2 = %d, want 0", got)
	}

	mustTick(t, svc, ctx, bed, day(2026, 11, 8)) // untick

	goal, err := svc.Store().Reward(ctx, result.JustAchieved.ID)
	if err != nil || !goal.IsAchieved() {
		t.Errorf("unticking must not revoke a celebrated goal (achieved=%v, err=%v)", goal.IsAchieved(), err)
	}

	if got := balanceOf(t, svc, ctx, qile); got != -1 {
		t.Errorf("balance after the untick = %d, want -1", got)
	}
}

func TestDeductionsLowerTheBalanceAndMayTakeItBelowZero(t *testing.T) {
	svc, _, ctx := newTestService(t)
	qile := mustPerson(t, svc, ctx, "奇乐")
	bed := mustHabit(t, svc, ctx, qile, "整理床")
	mustTick(t, svc, ctx, bed, day(2026, 11, 9))

	reason := "misconduct"
	if _, err := svc.CreateAdjustment(ctx, qile.ID, -3, &reason); err != nil {
		t.Fatalf("deduct: %v", err)
	}

	if got := balanceOf(t, svc, ctx, qile); got != -2 {
		t.Errorf("balance = %d, want -2", got)
	}

	_, err := svc.CreateAdjustment(ctx, qile.ID, 0, nil)
	requireValidation(t, err, "points")
}

func TestTicksOnAStoppedHabitKeepTheirPoints(t *testing.T) {
	svc, clock, ctx := newTestService(t)
	dad := mustPerson(t, svc, ctx, "爸爸")

	clock.at(2026, 11, 1)
	rinse := mustHabit(t, svc, ctx, dad, "Daily Sinus Rinse")
	for d := 1; d <= 3; d++ {
		clock.at(2026, 11, d)
		mustTick(t, svc, ctx, rinse, day(2026, 11, d))
	}

	clock.at(2026, 11, 9)
	if _, err := svc.StopHabit(ctx, rinse.ID); err != nil {
		t.Fatalf("stop: %v", err)
	}

	if got := balanceOf(t, svc, ctx, dad); got != 3 {
		t.Errorf("balance after stopping = %d, want the 3 points kept", got)
	}
}

func TestTicksLandOnTheFamilysOwnDay(t *testing.T) {
	svc, clock, ctx := newTestService(t)
	keyu := mustPerson(t, svc, ctx, "可遇")

	// New Year's Eve into New Year's Day: the week strip and the editable
	// window both cross the year.
	clock.at(2026, 12, 30)
	sing := mustHabit(t, svc, ctx, keyu, "sing")
	clock.now = time.Date(2027, 1, 1, 0, 5, 0, 0, time.UTC)

	mustTick(t, svc, ctx, sing, day(2026, 12, 31))
	mustTick(t, svc, ctx, sing, day(2027, 1, 1))

	view := dayView(t, svc, ctx, keyu, day(2027, 1, 1))
	if view.Week[0].Date != day(2026, 12, 28) || view.Week[6].Date != day(2027, 1, 3) {
		t.Errorf("week = %s … %s, want Mon 28 Dec … Sun 3 Jan", view.Week[0].ISO, view.Week[6].ISO)
	}

	if view.Habits[0].Streak != 2 {
		t.Errorf("streak across the new year = %d, want 2", view.Habits[0].Streak)
	}
}
