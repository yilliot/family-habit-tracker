// Package domain holds the habit-tracker entities and the validation rules
// that guard them. It has no dependencies on storage or transport.
package domain

import "time"

// Person is a family member. Archiving one hides them and keeps their history.
type Person struct {
	ID           int64
	Name         string
	DisplayOrder int
	CreatedAt    *time.Time
	UpdatedAt    *time.Time
	DeletedAt    *time.Time
}

// IsArchived reports whether the person has been soft-deleted.
func (p Person) IsArchived() bool { return p.DeletedAt != nil }

// Reward is a freeform goal with a point cost. A person has at most one active
// (un-achieved) reward at a time, and earns toward it for as long as it takes.
type Reward struct {
	ID         int64
	PersonID   int64
	Name       string
	Cost       int
	AchievedAt *time.Time
	CreatedAt  *time.Time
	UpdatedAt  *time.Time
}

// IsAchieved reports whether the reward has been earned.
func (r Reward) IsAchieved() bool { return r.AchievedAt != nil }

// Habit is something one person does daily. It runs from StartDate until
// EndDate, inclusive, or for as long as EndDate is nil. Stopping a habit sets
// EndDate and keeps the row, so its ticks and points stay in the history.
//
// SprintParticipantID survives only on rows created while the app still ran in
// sprints; it is kept for the record and never read.
type Habit struct {
	ID                  int64
	PersonID            int64
	SprintParticipantID *int64
	Name                string
	DisplayOrder        int
	StartDate           time.Time
	EndDate             *time.Time
	CreatedAt           *time.Time
	UpdatedAt           *time.Time
	DeletedAt           *time.Time
}

// LiveOn reports whether the habit was running on a day.
func (h Habit) LiveOn(day time.Time) bool {
	return !day.Before(h.StartDate) && (h.EndDate == nil || !day.After(*h.EndDate))
}

// IsRunning reports whether the habit has not been stopped.
func (h Habit) IsRunning() bool { return h.EndDate == nil }

// Tick is a single day's completion of one habit, worth one point.
type Tick struct {
	ID        int64
	HabitID   int64
	TickDate  time.Time
	CreatedAt time.Time
}

// PointAdjustment is a manual, signed correction to a person's balance.
// Negative amounts are deductions; positive amounts are manual awards.
type PointAdjustment struct {
	ID        int64
	PersonID  int64
	Amount    int
	Reason    *string
	CreatedAt time.Time
}
