package service_test

import (
	"testing"

	"github.com/el/family-habit-tracker/internal/service"
)

func TestPersonNameIsRequired(t *testing.T) {
	svc, _, ctx := newTestService(t)

	_, err := svc.CreatePerson(ctx, "   ", nil)
	requireValidation(t, err, "name")
}

func TestPeopleAreOrderedForTheLedger(t *testing.T) {
	svc, _, ctx := newTestService(t)

	for _, name := range []string{"爸爸", "妈妈", "可遇", "奇乐"} {
		mustPerson(t, svc, ctx, name)
	}

	people, err := svc.Store().People(ctx)
	if err != nil {
		t.Fatalf("list people: %v", err)
	}

	want := []string{"爸爸", "妈妈", "可遇", "奇乐"}
	if len(people) != len(want) {
		t.Fatalf("got %d people, want %d", len(people), len(want))
	}

	for i, person := range people {
		if person.Name != want[i] {
			t.Errorf("position %d = %q, want %q", i, person.Name, want[i])
		}
	}
}

// Removing someone must never destroy what they earned. This is the rule that
// makes the archive safe to use during a sprint.
func TestArchivingSomeoneMidSprintKeepsEveryPointTheyEarned(t *testing.T) {
	svc, _, ctx := newTestService(t)

	person := mustPerson(t, svc, ctx, "可遇")
	_, participant := startSprint(t, svc, ctx, person, "阅读")
	habit := habitsOf(t, svc, ctx, participant)[0]

	for _, d := range []int{20, 21, 22} {
		if _, err := svc.ToggleTick(ctx, habit.ID, day(2026, 4, d)); err != nil {
			t.Fatalf("tick: %v", err)
		}
	}

	if err := svc.ArchivePerson(ctx, person.ID); err != nil {
		t.Fatalf("archive person: %v", err)
	}

	lifetime, err := svc.Points().LifetimeBalance(ctx, person.ID)
	if err != nil {
		t.Fatalf("lifetime balance: %v", err)
	}

	if lifetime != 3 {
		t.Errorf("lifetime balance after archiving = %d, want 3", lifetime)
	}

	earned, err := svc.Store().TicksEarnedByPerson(ctx, person.ID)
	if err != nil {
		t.Fatalf("count ticks: %v", err)
	}

	if earned != 3 {
		t.Errorf("ticks after archiving = %d, want 3", earned)
	}
}

func TestArchivingSomeoneRemovesThemFromTheActiveGrid(t *testing.T) {
	svc, _, ctx := newTestService(t)

	staying := mustPerson(t, svc, ctx, "爸爸")
	leaving := mustPerson(t, svc, ctx, "妈妈")

	sprint, _ := startSprint(t, svc, ctx, staying, "整齐")

	if _, err := svc.AddParticipant(ctx, sprint.ID, service.ParticipantInput{
		PersonID: leaving.ID,
		Habits:   []service.HabitInput{{Name: "聆听", DisplayOrder: 1}},
		Reward:   service.RewardChoice{Mode: service.RewardNone},
	}); err != nil {
		t.Fatalf("add participant: %v", err)
	}

	if err := svc.ArchivePerson(ctx, leaving.ID); err != nil {
		t.Fatalf("archive person: %v", err)
	}

	grid, err := svc.BuildGrid(ctx, sprint)
	if err != nil {
		t.Fatalf("build grid: %v", err)
	}

	for _, row := range grid.Participants {
		if row.Person.ID == leaving.ID {
			t.Error("an archived person should not appear in the active grid")
		}
	}

	if len(grid.Participants) != 1 {
		t.Errorf("grid has %d participants, want 1", len(grid.Participants))
	}
}

// An archived person can come back, and their history comes with them.
func TestAnArchivedPersonCanRejoinALaterSprint(t *testing.T) {
	svc, _, ctx := newTestService(t)

	person := mustPerson(t, svc, ctx, "可遇")
	sprint1, participant := startSprint(t, svc, ctx, person, "阅读")
	habit := habitsOf(t, svc, ctx, participant)[0]

	for _, d := range []int{20, 21} {
		if _, err := svc.ToggleTick(ctx, habit.ID, day(2026, 4, d)); err != nil {
			t.Fatalf("tick: %v", err)
		}
	}

	if err := svc.ArchivePerson(ctx, person.ID); err != nil {
		t.Fatalf("archive: %v", err)
	}

	if _, err := svc.EndSprint(ctx, sprint1.ID); err != nil {
		t.Fatalf("end sprint: %v", err)
	}

	// Bringing them back is a rename-free restore: create is rejected because
	// the row still exists, so the service exposes the person with trashed.
	restored, err := svc.Store().PersonWithTrashed(ctx, person.ID)
	if err != nil {
		t.Fatalf("load archived person: %v", err)
	}

	if !restored.IsArchived() {
		t.Fatal("expected the person to be archived")
	}

	lifetime, err := svc.Points().LifetimeBalance(ctx, person.ID)
	if err != nil {
		t.Fatalf("lifetime balance: %v", err)
	}

	if lifetime != 2 {
		t.Errorf("lifetime balance = %d, want 2", lifetime)
	}
}

func TestHabitsCannotBeChangedOnAnArchivedSprint(t *testing.T) {
	svc, _, ctx := newTestService(t)

	person := mustPerson(t, svc, ctx, "爸爸")
	sprint, participant := startSprint(t, svc, ctx, person, "整齐")
	habit := habitsOf(t, svc, ctx, participant)[0]

	if _, err := svc.EndSprint(ctx, sprint.ID); err != nil {
		t.Fatalf("end sprint: %v", err)
	}

	if _, err := svc.CreateHabit(ctx, participant.ID, "新习惯", nil); err == nil {
		t.Error("adding a habit to an archived sprint should be rejected")
	} else {
		requireValidation(t, err, "sprint")
	}

	if _, err := svc.UpdateHabit(ctx, habit.ID, "改名", 1); err == nil {
		t.Error("renaming a habit in an archived sprint should be rejected")
	} else {
		requireValidation(t, err, "sprint")
	}

	if err := svc.ArchiveHabit(ctx, habit.ID); err == nil {
		t.Error("removing a habit from an archived sprint should be rejected")
	} else {
		requireValidation(t, err, "sprint")
	}
}

// A habit retired mid-sprint disappears from the grid but keeps the points it
// already earned.
func TestRetiringAHabitKeepsItsPoints(t *testing.T) {
	svc, _, ctx := newTestService(t)

	person := mustPerson(t, svc, ctx, "爸爸")
	sprint, participant := startSprint(t, svc, ctx, person, "整齐", "带敬拜")
	habits := habitsOf(t, svc, ctx, participant)

	for _, d := range []int{20, 21} {
		if _, err := svc.ToggleTick(ctx, habits[0].ID, day(2026, 4, d)); err != nil {
			t.Fatalf("tick: %v", err)
		}
	}

	if err := svc.ArchiveHabit(ctx, habits[0].ID); err != nil {
		t.Fatalf("archive habit: %v", err)
	}

	if got := balanceOf(t, svc, ctx, sprint, participant); got != 2 {
		t.Errorf("balance after retiring a habit = %d, want 2", got)
	}

	grid, err := svc.BuildGrid(ctx, sprint)
	if err != nil {
		t.Fatalf("build grid: %v", err)
	}

	if len(grid.Participants[0].Habits) != 1 {
		t.Errorf("grid shows %d habits, want 1 after retiring one", len(grid.Participants[0].Habits))
	}
}

func TestHabitNameLengthIsEnforced(t *testing.T) {
	svc, _, ctx := newTestService(t)

	person := mustPerson(t, svc, ctx, "爸爸")
	_, participant := startSprint(t, svc, ctx, person, "整齐")

	tooLong := make([]byte, 0, 81)
	for range 81 {
		tooLong = append(tooLong, 'a')
	}

	_, err := svc.CreateHabit(ctx, participant.ID, string(tooLong), nil)
	requireValidation(t, err, "name")
}

// Names are counted in characters, not bytes, so a Chinese name close to the
// limit is not rejected for being multi-byte.
func TestChineseNamesAreCountedInCharacters(t *testing.T) {
	svc, _, ctx := newTestService(t)

	// 20 Chinese characters is 60 bytes but well inside the 50-character limit.
	name := "一二三四五六七八九十一二三四五六七八九十"

	if _, err := svc.CreatePerson(ctx, name, nil); err != nil {
		t.Fatalf("a 20-character name should be accepted: %v", err)
	}
}
