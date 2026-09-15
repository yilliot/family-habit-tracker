// Package domain holds the habit-tracker entities and the validation rules
// that guard them. It has no dependencies on storage or transport.
package domain

import "time"

// Sprint statuses.
const (
	StatusActive   = "active"
	StatusArchived = "archived"
)

// Person is a family member. People are global and outlive any single sprint,
// so a person who sits one out keeps their history and can rejoin later.
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

// Sprint is a bounded, family-meeting-delimited period. Exactly one sprint may
// be active at a time.
type Sprint struct {
	ID        int64
	StartDate time.Time
	EndDate   time.Time
	Status    string
	StartedAt *time.Time
	EndedAt   *time.Time
	CreatedAt *time.Time
	UpdatedAt *time.Time
}

// IsActive reports whether the sprint still accepts writes.
func (s Sprint) IsActive() bool { return s.Status == StatusActive }

// IsArchived reports whether the sprint is read-only.
func (s Sprint) IsArchived() bool { return s.Status == StatusArchived }

// Days returns every day in the sprint range, inclusive of both ends.
func (s Sprint) Days() []time.Time {
	var days []time.Time

	for cursor := s.StartDate; !cursor.After(s.EndDate); cursor = cursor.AddDate(0, 0, 1) {
		days = append(days, cursor)
	}

	return days
}

// Reward is a freeform goal with a point cost. A person has at most one active
// (un-achieved) reward at a time, and earns toward it across as many sprints as
// it takes.
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

// SprintParticipant is one person's enrolment in one sprint. It carries the
// opening point balance for the sprint and the reward being earned toward.
type SprintParticipant struct {
	ID                  int64
	SprintID            int64
	PersonID            int64
	CarryForwardBalance int
	ActiveRewardID      *int64
	DisplayOrder        int
	CreatedAt           *time.Time
	UpdatedAt           *time.Time
}

// Habit is a behaviour tracked for one participant within one sprint.
type Habit struct {
	ID                  int64
	SprintParticipantID *int64
	Name                string
	DisplayOrder        int
	CreatedAt           *time.Time
	UpdatedAt           *time.Time
	DeletedAt           *time.Time
}

// Tick is a single day's completion of one habit, worth one point.
type Tick struct {
	ID        int64
	HabitID   int64
	TickDate  time.Time
	CreatedAt time.Time
}

// PointAdjustment is a manual, signed correction to a participant's balance.
// Negative amounts are deductions; positive amounts are manual awards.
type PointAdjustment struct {
	ID                  int64
	SprintParticipantID int64
	Amount              int
	Reason              *string
	CreatedAt           time.Time
}
