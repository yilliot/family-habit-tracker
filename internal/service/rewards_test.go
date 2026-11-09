package service_test

import "testing"

func TestOneGoalAtATime(t *testing.T) {
	svc, _, ctx := newTestService(t)
	dad := mustPerson(t, svc, ctx, "爸爸")

	if _, _, err := svc.SetGoal(ctx, dad.ID, "Lego Ford T Model", 584); err != nil {
		t.Fatalf("set goal: %v", err)
	}

	_, _, err := svc.SetGoal(ctx, dad.ID, "Another", 10)
	requireValidation(t, err, "name")

	_, _, err = svc.SetGoal(ctx, dad.ID, "Free", 0)
	requireValidation(t, err, "cost")
}

func TestAGoalYouCanAlreadyAffordIsEarnedOnTheSpot(t *testing.T) {
	svc, clock, ctx := newTestService(t)
	qile := mustPerson(t, svc, ctx, "奇乐")

	clock.at(2026, 11, 7)
	bed := mustHabit(t, svc, ctx, qile, "整理床")
	clock.at(2026, 11, 9)
	for d := 7; d <= 9; d++ {
		mustTick(t, svc, ctx, bed, day(2026, 11, d))
	}

	_, achieved, err := svc.SetGoal(ctx, qile.ID, "Piano Lesson", 3)
	if err != nil || achieved == nil {
		t.Fatalf("a goal covered by 3 points should be earned at once (achieved=%v, err=%v)", achieved, err)
	}

	if got := balanceOf(t, svc, ctx, qile); got != 0 {
		t.Errorf("balance = %d, want 0", got)
	}
}

func TestLoweringTheCostCanEarnTheGoal(t *testing.T) {
	svc, _, ctx := newTestService(t)
	qile := mustPerson(t, svc, ctx, "奇乐")
	bed := mustHabit(t, svc, ctx, qile, "整理床")
	mustTick(t, svc, ctx, bed, day(2026, 11, 9))

	goal, _, err := svc.SetGoal(ctx, qile.ID, "Sticker", 5)
	if err != nil {
		t.Fatalf("set goal: %v", err)
	}

	_, achieved, err := svc.UpdateReward(ctx, goal.ID, "Sticker", 1)
	if err != nil || achieved == nil {
		t.Fatalf("lowering the cost to 1 should earn it (achieved=%v, err=%v)", achieved, err)
	}

	_, _, err = svc.UpdateReward(ctx, goal.ID, "Sticker", 2)
	requireValidation(t, err, "cost")

	if renamed, _, err := svc.UpdateReward(ctx, goal.ID, "Big sticker", 1); err != nil || renamed.Name != "Big sticker" {
		t.Errorf("an achieved goal's name can still be corrected: %q %v", renamed.Name, err)
	}

	_, err = svc.DeleteReward(ctx, goal.ID)
	requireValidation(t, err, "reward")
}

func TestAGoalNotYetEarnedCanBeRemoved(t *testing.T) {
	svc, _, ctx := newTestService(t)
	mum := mustPerson(t, svc, ctx, "妈妈")

	goal, _, err := svc.SetGoal(ctx, mum.ID, "Lego altar", 69)
	if err != nil {
		t.Fatalf("set goal: %v", err)
	}

	if _, err := svc.DeleteReward(ctx, goal.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}

	if _, _, err := svc.SetGoal(ctx, mum.ID, "Something else", 20); err != nil {
		t.Errorf("after removing the goal a new one can be set: %v", err)
	}
}
