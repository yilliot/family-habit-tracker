package service_test

import (
	"context"
	"testing"

	"github.com/el/family-habit-tracker/internal/domain"
	"github.com/el/family-habit-tracker/internal/service"
)

func TestOnlyOneActiveRewardPerPerson(t *testing.T) {
	svc, _, ctx := newTestService(t)

	person := mustPerson(t, svc, ctx, "可遇")

	if _, err := svc.CreateReward(ctx, person.ID, "Lego Minecraft", 250); err != nil {
		t.Fatalf("create first reward: %v", err)
	}

	_, err := svc.CreateReward(ctx, person.ID, "A bicycle", 400)
	requireValidation(t, err, "name")
}

func TestRewardCostMustBeSensible(t *testing.T) {
	svc, _, ctx := newTestService(t)

	person := mustPerson(t, svc, ctx, "可遇")

	_, err := svc.CreateReward(ctx, person.ID, "Free stuff", 0)
	requireValidation(t, err, "cost")
}

func TestRewardNameIsRequired(t *testing.T) {
	svc, _, ctx := newTestService(t)

	person := mustPerson(t, svc, ctx, "可遇")

	_, err := svc.CreateReward(ctx, person.ID, "   ", 10)
	requireValidation(t, err, "name")
}

// Lowering the cost at a family meeting should settle immediately, not wait for
// an unrelated tick to notice.
func TestLoweringACostEarnsTheRewardImmediately(t *testing.T) {
	svc, _, ctx := newTestService(t)

	person := mustPerson(t, svc, ctx, "奇乐")

	sprint, err := svc.StartSprint(ctx, day(2026, 4, 19), day(2026, 5, 16), []service.ParticipantInput{{
		PersonID: person.ID,
		Habits:   []service.HabitInput{{Name: "整理床", DisplayOrder: 1}},
		Reward:   service.RewardChoice{Mode: service.RewardNew, Name: "Water gun", Cost: 10},
	}})
	if err != nil {
		t.Fatalf("start sprint: %v", err)
	}

	participant := participantFor(t, svc, ctx, sprint, person)
	habit := habitsOf(t, svc, ctx, participant)[0]

	for _, d := range []int{20, 21} {
		if _, err := svc.ToggleTick(ctx, habit.ID, day(2026, 4, d)); err != nil {
			t.Fatalf("tick: %v", err)
		}
	}

	rewardID := *participant.ActiveRewardID

	updated, err := svc.UpdateReward(ctx, rewardID, "Water gun", 2)
	if err != nil {
		t.Fatalf("lower cost: %v", err)
	}

	if !updated.IsAchieved() {
		t.Error("lowering the cost to the current balance should have earned the reward")
	}
}

func TestAchievedRewardCostIsFrozen(t *testing.T) {
	svc, _, ctx := newTestService(t)

	person := mustPerson(t, svc, ctx, "奇乐")
	reward := earnReward(t, svc, ctx, person, "Ice cream", 1)

	_, err := svc.UpdateReward(ctx, reward.ID, "Ice cream", 5)
	requireValidation(t, err, "cost")
}

func TestAchievedRewardNameCanStillBeCorrected(t *testing.T) {
	svc, _, ctx := newTestService(t)

	person := mustPerson(t, svc, ctx, "奇乐")
	reward := earnReward(t, svc, ctx, person, "Ice cream", 1)

	updated, err := svc.UpdateReward(ctx, reward.ID, "Ice cream sundae", reward.Cost)
	if err != nil {
		t.Fatalf("rename achieved reward: %v", err)
	}

	if updated.Name != "Ice cream sundae" {
		t.Errorf("name = %q, want the corrected name", updated.Name)
	}

	if !updated.IsAchieved() {
		t.Error("renaming must not clear the achievement")
	}
}

func TestAchievedRewardCannotBeDeleted(t *testing.T) {
	svc, _, ctx := newTestService(t)

	person := mustPerson(t, svc, ctx, "奇乐")
	reward := earnReward(t, svc, ctx, person, "Ice cream", 1)

	err := svc.DeleteReward(ctx, reward.ID)
	requireValidation(t, err, "reward")
}

func TestRewardInUseCannotBeDeleted(t *testing.T) {
	svc, _, ctx := newTestService(t)

	person := mustPerson(t, svc, ctx, "可遇")

	sprint, err := svc.StartSprint(ctx, day(2026, 4, 19), day(2026, 5, 16), []service.ParticipantInput{{
		PersonID: person.ID,
		Reward:   service.RewardChoice{Mode: service.RewardNew, Name: "Lego Minecraft", Cost: 250},
	}})
	if err != nil {
		t.Fatalf("start sprint: %v", err)
	}

	participant := participantFor(t, svc, ctx, sprint, person)

	err = svc.DeleteReward(ctx, *participant.ActiveRewardID)
	requireValidation(t, err, "reward")
}

func TestUnusedRewardCanBeDeleted(t *testing.T) {
	svc, _, ctx := newTestService(t)

	person := mustPerson(t, svc, ctx, "可遇")

	reward, err := svc.CreateReward(ctx, person.ID, "A bicycle", 400)
	if err != nil {
		t.Fatalf("create reward: %v", err)
	}

	if err := svc.DeleteReward(ctx, reward.ID); err != nil {
		t.Fatalf("delete unused reward: %v", err)
	}
}

func TestCannotWorkTowardSomeoneElsesReward(t *testing.T) {
	svc, _, ctx := newTestService(t)

	mine := mustPerson(t, svc, ctx, "可遇")
	theirs := mustPerson(t, svc, ctx, "奇乐")

	sprint, participant := startSprint(t, svc, ctx, mine, "阅读")

	other, err := svc.CreateReward(ctx, theirs.ID, "Their bike", 100)
	if err != nil {
		t.Fatalf("create reward: %v", err)
	}

	_ = sprint

	_, err = svc.SetActiveReward(ctx, participant.ID, &other.ID)
	requireValidation(t, err, "reward_id")
}

func TestCannotWorkTowardAnAlreadyEarnedReward(t *testing.T) {
	svc, _, ctx := newTestService(t)

	person := mustPerson(t, svc, ctx, "奇乐")
	earned := earnReward(t, svc, ctx, person, "Ice cream", 1)

	sprint, err := svc.Store().ActiveSprint(ctx)
	if err != nil || sprint == nil {
		t.Fatalf("active sprint: %v", err)
	}

	participant := participantFor(t, svc, ctx, *sprint, person)

	_, err = svc.SetActiveReward(ctx, participant.ID, &earned.ID)
	requireValidation(t, err, "reward_id")
}

// Picking a goal that banked surplus already covers should settle on the spot.
func TestPickingAnAffordableRewardEarnsItAtOnce(t *testing.T) {
	svc, _, ctx := newTestService(t)

	person := mustPerson(t, svc, ctx, "奇乐")
	sprint, participant := startSprint(t, svc, ctx, person, "整理床")
	habit := habitsOf(t, svc, ctx, participant)[0]

	for _, d := range []int{20, 21, 22} {
		if _, err := svc.ToggleTick(ctx, habit.ID, day(2026, 4, d)); err != nil {
			t.Fatalf("tick: %v", err)
		}
	}

	cheap, err := svc.CreateReward(ctx, person.ID, "Sticker", 2)
	if err != nil {
		t.Fatalf("create reward: %v", err)
	}

	achieved, err := svc.SetActiveReward(ctx, participant.ID, &cheap.ID)
	if err != nil {
		t.Fatalf("set active reward: %v", err)
	}

	if achieved == nil {
		t.Fatal("three banked points should have earned a two-point goal at once")
	}

	if got := balanceOf(t, svc, ctx, sprint, participant); got != 1 {
		t.Errorf("remaining surplus = %d, want 1", got)
	}
}

func TestClearingTheActiveRewardIsAllowed(t *testing.T) {
	svc, _, ctx := newTestService(t)

	person := mustPerson(t, svc, ctx, "可遇")

	sprint, err := svc.StartSprint(ctx, day(2026, 4, 19), day(2026, 5, 16), []service.ParticipantInput{{
		PersonID: person.ID,
		Reward:   service.RewardChoice{Mode: service.RewardNew, Name: "Lego", Cost: 250},
	}})
	if err != nil {
		t.Fatalf("start sprint: %v", err)
	}

	participant := participantFor(t, svc, ctx, sprint, person)

	if _, err := svc.SetActiveReward(ctx, participant.ID, nil); err != nil {
		t.Fatalf("clear reward: %v", err)
	}

	reloaded := participantFor(t, svc, ctx, sprint, person)

	if reloaded.ActiveRewardID != nil {
		t.Error("the active reward should have been cleared")
	}
}

// earnReward runs a person through a sprint until they earn a cheap goal,
// returning it in its achieved state.
func earnReward(t *testing.T, svc *service.Service, ctx context.Context, person domain.Person, name string, cost int) domain.Reward {
	t.Helper()

	sprint, err := svc.StartSprint(ctx, day(2026, 4, 19), day(2026, 5, 16), []service.ParticipantInput{{
		PersonID: person.ID,
		Habits:   []service.HabitInput{{Name: "habit", DisplayOrder: 1}},
		Reward:   service.RewardChoice{Mode: service.RewardNew, Name: name, Cost: cost},
	}})
	if err != nil {
		t.Fatalf("start sprint: %v", err)
	}

	participant := participantFor(t, svc, ctx, sprint, person)
	habit := habitsOf(t, svc, ctx, participant)[0]

	for i := range cost {
		if _, err := svc.ToggleTick(ctx, habit.ID, day(2026, 4, 20+i)); err != nil {
			t.Fatalf("tick: %v", err)
		}
	}

	reward, err := svc.Store().Reward(ctx, *participant.ActiveRewardID)
	if err != nil {
		t.Fatalf("reload reward: %v", err)
	}

	if !reward.IsAchieved() {
		t.Fatalf("expected %q to be earned after %d ticks", name, cost)
	}

	return reward
}
