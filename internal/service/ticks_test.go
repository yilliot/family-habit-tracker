package service_test

import (
	"testing"
	"time"

	"github.com/el/family-habit-tracker/internal/service"
)

func TestToggleTickBanksAPoint(t *testing.T) {
	svc, _, ctx := newTestService(t)

	person := mustPerson(t, svc, ctx, "可遇")
	sprint, participant := startSprint(t, svc, ctx, person, "30分钟阅读时间")
	habit := habitsOf(t, svc, ctx, participant)[0]

	result, err := svc.ToggleTick(ctx, habit.ID, day(2026, 4, 20))
	if err != nil {
		t.Fatalf("toggle tick: %v", err)
	}

	if !result.Ticked {
		t.Error("expected the cell to be ticked after the first tap")
	}

	if result.Balance != 1 {
		t.Errorf("balance = %d, want 1", result.Balance)
	}

	if got := balanceOf(t, svc, ctx, sprint, participant); got != 1 {
		t.Errorf("persisted balance = %d, want 1", got)
	}
}

func TestToggleTickIsReversible(t *testing.T) {
	svc, _, ctx := newTestService(t)

	person := mustPerson(t, svc, ctx, "奇乐")
	_, participant := startSprint(t, svc, ctx, person, "整理床")
	habit := habitsOf(t, svc, ctx, participant)[0]

	if _, err := svc.ToggleTick(ctx, habit.ID, day(2026, 4, 20)); err != nil {
		t.Fatalf("first toggle: %v", err)
	}

	result, err := svc.ToggleTick(ctx, habit.ID, day(2026, 4, 20))
	if err != nil {
		t.Fatalf("second toggle: %v", err)
	}

	if result.Ticked {
		t.Error("expected the cell to be cleared by the second tap")
	}

	if result.Balance != 0 {
		t.Errorf("balance = %d, want 0 after untick", result.Balance)
	}
}

// A double tap must never bank two points for one day. The unique index makes
// this structurally impossible, and this test pins that guarantee.
func TestToggleTickNeverDoubleCounts(t *testing.T) {
	svc, _, ctx := newTestService(t)

	person := mustPerson(t, svc, ctx, "妈妈")
	sprint, participant := startSprint(t, svc, ctx, person, "聆听")
	habit := habitsOf(t, svc, ctx, participant)[0]

	for range 5 {
		if _, err := svc.ToggleTick(ctx, habit.ID, day(2026, 4, 21)); err != nil {
			t.Fatalf("toggle: %v", err)
		}
	}

	// Five taps is an odd number, so the cell ends ticked, worth exactly one.
	if got := balanceOf(t, svc, ctx, sprint, participant); got != 1 {
		t.Errorf("balance after five taps = %d, want 1", got)
	}
}

func TestToggleTickRejectsFutureDays(t *testing.T) {
	svc, _, ctx := newTestService(t)

	person := mustPerson(t, svc, ctx, "爸爸")
	_, participant := startSprint(t, svc, ctx, person, "整齐")
	habit := habitsOf(t, svc, ctx, participant)[0]

	// fixedToday is 1 May; 2 May is inside the sprint but has not happened.
	_, err := svc.ToggleTick(ctx, habit.ID, day(2026, 5, 2))
	requireValidation(t, err, "tick_date")
}

func TestToggleTickAllowsBackfillingYesterday(t *testing.T) {
	svc, _, ctx := newTestService(t)

	person := mustPerson(t, svc, ctx, "爸爸")
	_, participant := startSprint(t, svc, ctx, person, "整齐")
	habit := habitsOf(t, svc, ctx, participant)[0]

	// Forgetting to tap yesterday should not cost the point.
	if _, err := svc.ToggleTick(ctx, habit.ID, day(2026, 4, 30)); err != nil {
		t.Fatalf("backfill yesterday: %v", err)
	}
}

func TestToggleTickRejectsDatesOutsideTheSprint(t *testing.T) {
	svc, _, ctx := newTestService(t)

	person := mustPerson(t, svc, ctx, "爸爸")
	_, participant := startSprint(t, svc, ctx, person, "整齐")
	habit := habitsOf(t, svc, ctx, participant)[0]

	_, err := svc.ToggleTick(ctx, habit.ID, day(2026, 4, 1))
	requireValidation(t, err, "tick_date")
}

func TestToggleTickRejectsArchivedSprints(t *testing.T) {
	svc, _, ctx := newTestService(t)

	person := mustPerson(t, svc, ctx, "爸爸")
	sprint, participant := startSprint(t, svc, ctx, person, "整齐")
	habit := habitsOf(t, svc, ctx, participant)[0]

	if _, err := svc.EndSprint(ctx, sprint.ID); err != nil {
		t.Fatalf("end sprint: %v", err)
	}

	_, err := svc.ToggleTick(ctx, habit.ID, day(2026, 4, 20))
	requireValidation(t, err, "sprint")
}

func TestToggleTickAwardsTheRewardExactlyOnce(t *testing.T) {
	svc, _, ctx := newTestService(t)

	person := mustPerson(t, svc, ctx, "可遇")

	sprint, err := svc.StartSprint(ctx, day(2026, 4, 19), day(2026, 5, 16), []service.ParticipantInput{{
		PersonID: person.ID,
		Habits:   []service.HabitInput{{Name: "阅读", DisplayOrder: 1}},
		Reward:   service.RewardChoice{Mode: service.RewardNew, Name: "Lego Minecraft", Cost: 2},
	}})
	if err != nil {
		t.Fatalf("start sprint: %v", err)
	}

	participant := participantFor(t, svc, ctx, sprint, person)
	habit := habitsOf(t, svc, ctx, participant)[0]

	first, err := svc.ToggleTick(ctx, habit.ID, day(2026, 4, 20))
	if err != nil {
		t.Fatalf("first tick: %v", err)
	}
	if first.JustAchieved != nil {
		t.Error("one point should not have earned a two-point reward")
	}

	second, err := svc.ToggleTick(ctx, habit.ID, day(2026, 4, 21))
	if err != nil {
		t.Fatalf("second tick: %v", err)
	}
	if second.JustAchieved == nil {
		t.Fatal("reaching the cost should have earned the reward")
	}
	if second.JustAchieved.Name != "Lego Minecraft" {
		t.Errorf("earned %q, want Lego Minecraft", second.JustAchieved.Name)
	}

	// The celebration fires once. A further tick must not re-announce it.
	third, err := svc.ToggleTick(ctx, habit.ID, day(2026, 4, 22))
	if err != nil {
		t.Fatalf("third tick: %v", err)
	}
	if third.JustAchieved != nil {
		t.Error("the reward was already earned; it should not be announced again")
	}
}

// Un-ticking a day after a goal was reached must not revoke the reward. The
// family has already celebrated it.
func TestUntickingDoesNotRevokeAnEarnedReward(t *testing.T) {
	svc, _, ctx := newTestService(t)

	person := mustPerson(t, svc, ctx, "奇乐")

	sprint, err := svc.StartSprint(ctx, day(2026, 4, 19), day(2026, 5, 16), []service.ParticipantInput{{
		PersonID: person.ID,
		Habits:   []service.HabitInput{{Name: "华文练习", DisplayOrder: 1}},
		Reward:   service.RewardChoice{Mode: service.RewardNew, Name: "Ice cream", Cost: 1},
	}})
	if err != nil {
		t.Fatalf("start sprint: %v", err)
	}

	participant := participantFor(t, svc, ctx, sprint, person)
	habit := habitsOf(t, svc, ctx, participant)[0]

	if _, err := svc.ToggleTick(ctx, habit.ID, day(2026, 4, 20)); err != nil {
		t.Fatalf("earn reward: %v", err)
	}

	if _, err := svc.ToggleTick(ctx, habit.ID, day(2026, 4, 20)); err != nil {
		t.Fatalf("untick: %v", err)
	}

	reloaded := participantFor(t, svc, ctx, sprint, person)

	reward, err := svc.Store().RewardByIDPtr(ctx, reloaded.ActiveRewardID)
	if err != nil {
		t.Fatalf("reload reward: %v", err)
	}

	if reward == nil || !reward.IsAchieved() {
		t.Error("an earned reward must stay earned after a tick is corrected")
	}
}

func TestAdjustmentsReduceTheBalanceAndMayGoNegative(t *testing.T) {
	svc, _, ctx := newTestService(t)

	person := mustPerson(t, svc, ctx, "奇乐")
	sprint, participant := startSprint(t, svc, ctx, person, "收拾game room")
	habit := habitsOf(t, svc, ctx, participant)[0]

	if _, err := svc.ToggleTick(ctx, habit.ID, day(2026, 4, 20)); err != nil {
		t.Fatalf("tick: %v", err)
	}

	reason := "missed bedtime"
	if _, err := svc.CreateAdjustment(ctx, participant.ID, -3, &reason); err != nil {
		t.Fatalf("create adjustment: %v", err)
	}

	if got := balanceOf(t, svc, ctx, sprint, participant); got != -2 {
		t.Errorf("balance = %d, want -2 (1 earned, 3 deducted)", got)
	}
}

func TestAdjustmentRejectsZero(t *testing.T) {
	svc, _, ctx := newTestService(t)

	person := mustPerson(t, svc, ctx, "爸爸")
	_, participant := startSprint(t, svc, ctx, person, "整齐")

	_, err := svc.CreateAdjustment(ctx, participant.ID, 0, nil)
	requireValidation(t, err, "points")
}

func TestAdjustmentRejectedOnArchivedSprint(t *testing.T) {
	svc, _, ctx := newTestService(t)

	person := mustPerson(t, svc, ctx, "爸爸")
	sprint, participant := startSprint(t, svc, ctx, person, "整齐")

	if _, err := svc.EndSprint(ctx, sprint.ID); err != nil {
		t.Fatalf("end sprint: %v", err)
	}

	_, err := svc.CreateAdjustment(ctx, participant.ID, -1, nil)
	requireValidation(t, err, "sprint")
}

// Ticks are stored day-precision, so a clock offset within the day must not
// move a tick onto a neighbouring date.
func TestTickDateIsStoredAtDayPrecision(t *testing.T) {
	svc, _, ctx := newTestService(t)

	person := mustPerson(t, svc, ctx, "妈妈")
	_, participant := startSprint(t, svc, ctx, person, "睡前祷告")
	habit := habitsOf(t, svc, ctx, participant)[0]

	target := time.Date(2026, 4, 23, 23, 59, 59, 0, time.UTC)

	if _, err := svc.ToggleTick(ctx, habit.ID, target); err != nil {
		t.Fatalf("toggle: %v", err)
	}

	ticked, err := svc.Store().TickDatesByParticipant(ctx, participant.ID)
	if err != nil {
		t.Fatalf("read ticks: %v", err)
	}

	if !ticked[habit.ID]["2026-04-23"] {
		t.Errorf("expected 2026-04-23 to be ticked, got %v", ticked[habit.ID])
	}
}
