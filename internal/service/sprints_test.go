package service_test

import (
	"testing"

	"github.com/el/family-habit-tracker/internal/service"
)

func TestOnlyOneSprintCanBeActive(t *testing.T) {
	svc, _, ctx := newTestService(t)

	person := mustPerson(t, svc, ctx, "爸爸")
	startSprint(t, svc, ctx, person, "整齐")

	_, err := svc.StartSprint(ctx, day(2026, 6, 1), day(2026, 6, 28), []service.ParticipantInput{
		{PersonID: person.ID, Reward: service.RewardChoice{Mode: service.RewardNone}},
	})
	requireValidation(t, err, "sprint")
}

func TestSprintRejectsAnEndBeforeItsStart(t *testing.T) {
	svc, _, ctx := newTestService(t)

	person := mustPerson(t, svc, ctx, "爸爸")

	_, err := svc.StartSprint(ctx, day(2026, 5, 16), day(2026, 4, 19), []service.ParticipantInput{
		{PersonID: person.ID, Reward: service.RewardChoice{Mode: service.RewardNone}},
	})
	requireValidation(t, err, "end_date")
}

func TestSprintRequiresSomeone(t *testing.T) {
	svc, _, ctx := newTestService(t)

	_, err := svc.StartSprint(ctx, day(2026, 4, 19), day(2026, 5, 16), nil)
	requireValidation(t, err, "participants")
}

func TestEndingASprintTwiceIsRejected(t *testing.T) {
	svc, _, ctx := newTestService(t)

	person := mustPerson(t, svc, ctx, "爸爸")
	sprint, _ := startSprint(t, svc, ctx, person, "整齐")

	if _, err := svc.EndSprint(ctx, sprint.ID); err != nil {
		t.Fatalf("end sprint: %v", err)
	}

	_, err := svc.EndSprint(ctx, sprint.ID)
	requireValidation(t, err, "sprint")
}

// This is the story the whole points model exists to serve: a reward costing
// more than one sprint can deliver is saved toward across several sprints.
func TestPointsCarryAcrossSprintsTowardOneBigReward(t *testing.T) {
	svc, _, ctx := newTestService(t)

	person := mustPerson(t, svc, ctx, "可遇")

	// First sprint: bank three points toward a five-point goal.
	sprint1, err := svc.StartSprint(ctx, day(2026, 4, 19), day(2026, 5, 16), []service.ParticipantInput{{
		PersonID: person.ID,
		Habits:   []service.HabitInput{{Name: "阅读", DisplayOrder: 1}},
		Reward:   service.RewardChoice{Mode: service.RewardNew, Name: "Lego Minecraft", Cost: 5},
	}})
	if err != nil {
		t.Fatalf("start first sprint: %v", err)
	}

	participant1 := participantFor(t, svc, ctx, sprint1, person)
	habit1 := habitsOf(t, svc, ctx, participant1)[0]

	for _, d := range []int{20, 21, 22} {
		if _, err := svc.ToggleTick(ctx, habit1.ID, day(2026, 4, d)); err != nil {
			t.Fatalf("tick %d: %v", d, err)
		}
	}

	if got := balanceOf(t, svc, ctx, sprint1, participant1); got != 3 {
		t.Fatalf("first sprint balance = %d, want 3", got)
	}

	if _, err := svc.EndSprint(ctx, sprint1.ID); err != nil {
		t.Fatalf("end first sprint: %v", err)
	}

	// Second sprint: the three points come with them.
	sprint2, err := svc.StartSprint(ctx, day(2026, 4, 25), day(2026, 5, 16), []service.ParticipantInput{{
		PersonID: person.ID,
		Habits:   []service.HabitInput{{Name: "阅读", DisplayOrder: 1}},
		Reward:   service.RewardChoice{Mode: service.RewardKeepCurrent},
	}})
	if err != nil {
		t.Fatalf("start second sprint: %v", err)
	}

	participant2 := participantFor(t, svc, ctx, sprint2, person)

	if participant2.CarryForwardBalance != 3 {
		t.Errorf("carried forward %d, want 3", participant2.CarryForwardBalance)
	}

	if participant2.ActiveRewardID == nil {
		t.Fatal("the unearned goal should have carried into the new sprint")
	}

	// Two more points completes the five-point goal.
	habit2 := habitsOf(t, svc, ctx, participant2)[0]

	if _, err := svc.ToggleTick(ctx, habit2.ID, day(2026, 4, 26)); err != nil {
		t.Fatalf("tick: %v", err)
	}

	result, err := svc.ToggleTick(ctx, habit2.ID, day(2026, 4, 27))
	if err != nil {
		t.Fatalf("tick: %v", err)
	}

	if result.JustAchieved == nil {
		t.Fatal("the fifth point should have earned the goal")
	}

	if got := balanceOf(t, svc, ctx, sprint2, participant2); got != 0 {
		t.Errorf("balance after earning a 5pt goal with exactly 5 points = %d, want 0", got)
	}
}

// Points earned beyond the cost are not lost; they seed the next goal.
func TestSurplusRollsOntoTheNextReward(t *testing.T) {
	svc, _, ctx := newTestService(t)

	person := mustPerson(t, svc, ctx, "奇乐")

	sprint, err := svc.StartSprint(ctx, day(2026, 4, 19), day(2026, 5, 16), []service.ParticipantInput{{
		PersonID: person.ID,
		Habits:   []service.HabitInput{{Name: "整理床", DisplayOrder: 1}},
		Reward:   service.RewardChoice{Mode: service.RewardNew, Name: "Ice cream", Cost: 2},
	}})
	if err != nil {
		t.Fatalf("start sprint: %v", err)
	}

	participant := participantFor(t, svc, ctx, sprint, person)
	habit := habitsOf(t, svc, ctx, participant)[0]

	// Four points against a two-point goal leaves two in hand.
	for _, d := range []int{20, 21, 22, 23} {
		if _, err := svc.ToggleTick(ctx, habit.ID, day(2026, 4, d)); err != nil {
			t.Fatalf("tick %d: %v", d, err)
		}
	}

	if got := balanceOf(t, svc, ctx, sprint, participant); got != 2 {
		t.Errorf("surplus = %d, want 2", got)
	}

	lifetime, err := svc.Points().LifetimeBalance(ctx, person.ID)
	if err != nil {
		t.Fatalf("lifetime balance: %v", err)
	}

	if lifetime != 2 {
		t.Errorf("lifetime balance = %d, want 2", lifetime)
	}
}

// Someone joining after a sprint has begun keeps everything they had banked.
func TestJoiningMidSprintKeepsTheirPoints(t *testing.T) {
	svc, _, ctx := newTestService(t)

	first := mustPerson(t, svc, ctx, "爸爸")
	second := mustPerson(t, svc, ctx, "妈妈")

	// The second person banks points in an earlier sprint, then sits one out.
	sprintA, err := svc.StartSprint(ctx, day(2026, 4, 1), day(2026, 4, 10), []service.ParticipantInput{{
		PersonID: second.ID,
		Habits:   []service.HabitInput{{Name: "聆听", DisplayOrder: 1}},
		Reward:   service.RewardChoice{Mode: service.RewardNone},
	}})
	if err != nil {
		t.Fatalf("start sprint A: %v", err)
	}

	participantA := participantFor(t, svc, ctx, sprintA, second)
	habitA := habitsOf(t, svc, ctx, participantA)[0]

	for _, d := range []int{2, 3} {
		if _, err := svc.ToggleTick(ctx, habitA.ID, day(2026, 4, d)); err != nil {
			t.Fatalf("tick: %v", err)
		}
	}

	if _, err := svc.EndSprint(ctx, sprintA.ID); err != nil {
		t.Fatalf("end sprint A: %v", err)
	}

	sprintB, _ := startSprint(t, svc, ctx, first, "整齐")

	joined, err := svc.AddParticipant(ctx, sprintB.ID, service.ParticipantInput{
		PersonID: second.ID,
		Habits:   []service.HabitInput{{Name: "15分钟运动", DisplayOrder: 1}},
		Reward:   service.RewardChoice{Mode: service.RewardNone},
	})
	if err != nil {
		t.Fatalf("add participant: %v", err)
	}

	if joined.CarryForwardBalance != 2 {
		t.Errorf("carried forward %d, want 2", joined.CarryForwardBalance)
	}
}

func TestSomeoneCannotJoinTheSameSprintTwice(t *testing.T) {
	svc, _, ctx := newTestService(t)

	person := mustPerson(t, svc, ctx, "爸爸")
	sprint, _ := startSprint(t, svc, ctx, person, "整齐")

	_, err := svc.AddParticipant(ctx, sprint.ID, service.ParticipantInput{
		PersonID: person.ID,
		Reward:   service.RewardChoice{Mode: service.RewardNone},
	})
	requireValidation(t, err, "person_id")
}

func TestCannotJoinAnArchivedSprint(t *testing.T) {
	svc, _, ctx := newTestService(t)

	first := mustPerson(t, svc, ctx, "爸爸")
	second := mustPerson(t, svc, ctx, "妈妈")
	sprint, _ := startSprint(t, svc, ctx, first, "整齐")

	if _, err := svc.EndSprint(ctx, sprint.ID); err != nil {
		t.Fatalf("end sprint: %v", err)
	}

	_, err := svc.AddParticipant(ctx, sprint.ID, service.ParticipantInput{
		PersonID: second.ID,
		Reward:   service.RewardChoice{Mode: service.RewardNone},
	})
	requireValidation(t, err, "sprint")
}

// A negative lifetime balance must not follow someone into a new sprint as a
// debt; they start fresh at zero.
func TestCarryForwardIsFlooredAtZero(t *testing.T) {
	svc, _, ctx := newTestService(t)

	person := mustPerson(t, svc, ctx, "奇乐")
	sprint1, participant1 := startSprint(t, svc, ctx, person, "整理床")

	if _, err := svc.CreateAdjustment(ctx, participant1.ID, -5, nil); err != nil {
		t.Fatalf("deduct: %v", err)
	}

	if got := balanceOf(t, svc, ctx, sprint1, participant1); got != -5 {
		t.Fatalf("balance = %d, want -5", got)
	}

	if _, err := svc.EndSprint(ctx, sprint1.ID); err != nil {
		t.Fatalf("end sprint: %v", err)
	}

	sprint2, err := svc.StartSprint(ctx, day(2026, 5, 20), day(2026, 6, 16), []service.ParticipantInput{
		{PersonID: person.ID, Reward: service.RewardChoice{Mode: service.RewardNone}},
	})
	if err != nil {
		t.Fatalf("start second sprint: %v", err)
	}

	participant2 := participantFor(t, svc, ctx, sprint2, person)

	if participant2.CarryForwardBalance != 0 {
		t.Errorf("carried forward %d, want 0 — a deficit should not follow someone", participant2.CarryForwardBalance)
	}
}

// "None" must mean none. The implementation this replaces silently re-linked an
// existing goal here, contradicting the label on the control.
func TestChoosingNoRewardLeavesTheParticipantWithoutOne(t *testing.T) {
	svc, _, ctx := newTestService(t)

	person := mustPerson(t, svc, ctx, "妈妈")

	if _, err := svc.CreateReward(ctx, person.ID, "Spa day", 50); err != nil {
		t.Fatalf("create reward: %v", err)
	}

	sprint, err := svc.StartSprint(ctx, day(2026, 4, 19), day(2026, 5, 16), []service.ParticipantInput{
		{PersonID: person.ID, Reward: service.RewardChoice{Mode: service.RewardNone}},
	})
	if err != nil {
		t.Fatalf("start sprint: %v", err)
	}

	participant := participantFor(t, svc, ctx, sprint, person)

	if participant.ActiveRewardID != nil {
		t.Error("choosing 'none' should leave the participant with no active reward")
	}
}

func TestKeepCurrentLinksTheExistingReward(t *testing.T) {
	svc, _, ctx := newTestService(t)

	person := mustPerson(t, svc, ctx, "妈妈")

	reward, err := svc.CreateReward(ctx, person.ID, "Spa day", 50)
	if err != nil {
		t.Fatalf("create reward: %v", err)
	}

	sprint, err := svc.StartSprint(ctx, day(2026, 4, 19), day(2026, 5, 16), []service.ParticipantInput{
		{PersonID: person.ID, Reward: service.RewardChoice{Mode: service.RewardKeepCurrent}},
	})
	if err != nil {
		t.Fatalf("start sprint: %v", err)
	}

	participant := participantFor(t, svc, ctx, sprint, person)

	if participant.ActiveRewardID == nil || *participant.ActiveRewardID != reward.ID {
		t.Error("the existing unearned reward should have been carried over")
	}
}
