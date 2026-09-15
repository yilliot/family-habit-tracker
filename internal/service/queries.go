package service

import (
	"context"
	"time"

	"github.com/el/family-habit-tracker/internal/db"
	"github.com/el/family-habit-tracker/internal/domain"
	"github.com/el/family-habit-tracker/internal/points"
)

// GridDay is one column of the sprint grid.
type GridDay struct {
	Date time.Time
	// ISO is the date in YYYY-MM-DD form, used as a stable DOM and form key.
	ISO          string
	IsWeekend    bool
	IsSunday     bool
	IsMonthStart bool
	IsToday      bool
	IsFuture     bool
}

// MonthGroup spans the columns belonging to one calendar month, so the grid can
// print a month name across them the way the paper sheet did.
type MonthGroup struct {
	Label string
	Span  int
}

// GridHabit is one row of the grid: a habit and the days it is ticked on.
type GridHabit struct {
	Habit  domain.Habit
	Ticked map[string]bool
	// Count is how many days this habit has been completed.
	Count int
}

// IsTicked reports whether the habit is complete on a given day.
func (h GridHabit) IsTicked(iso string) bool { return h.Ticked[iso] }

// GridParticipant is everything the grid and the progress strip need about one
// person in a sprint.
type GridParticipant struct {
	Participant  domain.SprintParticipant
	Person       domain.Person
	Balance      int
	ActiveReward *domain.Reward
	Habits       []GridHabit
}

// Percent is progress toward the active goal, capped at 100. A participant
// without a goal reports zero.
func (p GridParticipant) Percent() int {
	if p.ActiveReward == nil || p.ActiveReward.Cost <= 0 {
		return 0
	}

	pct := p.Balance * 100 / p.ActiveReward.Cost

	return min(max(pct, 0), 100)
}

// Remaining is how many more points the goal needs.
func (p GridParticipant) Remaining() int {
	if p.ActiveReward == nil {
		return 0
	}

	return max(0, p.ActiveReward.Cost-p.Balance)
}

// Grid is a whole sprint rendered as the people-by-days ledger.
type Grid struct {
	Sprint       domain.Sprint
	Days         []GridDay
	MonthGroups  []MonthGroup
	Participants []GridParticipant
	Today        time.Time
	// ReadOnly marks an archived sprint, whose cells cannot be tapped.
	ReadOnly bool
}

// TodayIndex is the column offset of today, or -1 when today is outside the
// sprint. The grid uses it to scroll the current day into view.
func (g Grid) TodayIndex() int {
	for i, day := range g.Days {
		if day.IsToday {
			return i
		}
	}

	return -1
}

// TotalPossible is how many ticks the whole family could bank in this sprint,
// used to frame the totals the way the printed sheet did.
func (g Grid) TotalPossible() int {
	total := 0
	for _, participant := range g.Participants {
		total += len(participant.Habits) * len(g.Days)
	}

	return total
}

// TotalEarned is how many ticks the family has actually banked this sprint.
func (g Grid) TotalEarned() int {
	total := 0
	for _, participant := range g.Participants {
		for _, habit := range participant.Habits {
			total += habit.Count
		}
	}

	return total
}

// BuildGrid assembles the full ledger for one sprint.
func (s *Service) BuildGrid(ctx context.Context, sprint domain.Sprint) (Grid, error) {
	today := s.store.Today()

	grid := Grid{
		Sprint:   sprint,
		Today:    today,
		ReadOnly: sprint.IsArchived(),
	}

	for _, day := range sprint.Days() {
		weekday := day.Weekday()

		grid.Days = append(grid.Days, GridDay{
			Date:         day,
			ISO:          db.FormatDate(day),
			IsWeekend:    weekday == time.Saturday || weekday == time.Sunday,
			IsSunday:     weekday == time.Sunday,
			IsMonthStart: day.Day() == 1 && !day.Equal(sprint.StartDate),
			IsToday:      day.Equal(today),
			IsFuture:     day.After(today),
		})
	}

	grid.MonthGroups = buildMonthGroups(grid.Days)

	participants, err := s.store.ParticipantsBySprint(ctx, sprint.ID)
	if err != nil {
		return Grid{}, err
	}

	for _, participant := range participants {
		row, err := s.buildGridParticipant(ctx, sprint, participant)
		if err != nil {
			return Grid{}, err
		}

		// Someone archived mid-sprint keeps their enrolment row, because that
		// row is what ties their ticks to them, but they come off the running
		// grid. An archived sprint still shows them: they really were there.
		if sprint.IsActive() && row.Person.IsArchived() {
			continue
		}

		grid.Participants = append(grid.Participants, row)
	}

	return grid, nil
}

// buildGridParticipant assembles one person's rows.
func (s *Service) buildGridParticipant(ctx context.Context, sprint domain.Sprint, participant domain.SprintParticipant) (GridParticipant, error) {
	person, err := s.store.PersonWithTrashed(ctx, participant.PersonID)
	if err != nil {
		return GridParticipant{}, err
	}

	habits, err := s.store.HabitsByParticipant(ctx, participant.ID)
	if err != nil {
		return GridParticipant{}, err
	}

	ticksByHabit, err := s.store.TickDatesByParticipant(ctx, participant.ID)
	if err != nil {
		return GridParticipant{}, err
	}

	balance, err := s.points.SprintBalance(ctx, sprint, participant)
	if err != nil {
		return GridParticipant{}, err
	}

	reward, err := s.store.RewardByIDPtr(ctx, participant.ActiveRewardID)
	if err != nil {
		return GridParticipant{}, err
	}

	row := GridParticipant{
		Participant:  participant,
		Person:       person,
		Balance:      balance,
		ActiveReward: reward,
	}

	for _, habit := range habits {
		ticked := ticksByHabit[habit.ID]
		if ticked == nil {
			ticked = map[string]bool{}
		}

		row.Habits = append(row.Habits, GridHabit{Habit: habit, Ticked: ticked, Count: len(ticked)})
	}

	return row, nil
}

// buildMonthGroups collapses consecutive days of the same month into spans.
func buildMonthGroups(days []GridDay) []MonthGroup {
	var groups []MonthGroup

	for i := 0; i < len(days); {
		month := days[i].Date.Month()
		year := days[i].Date.Year()

		j := i
		for j < len(days) && days[j].Date.Month() == month && days[j].Date.Year() == year {
			j++
		}

		groups = append(groups, MonthGroup{Label: days[i].Date.Format("January 2006"), Span: j - i})
		i = j
	}

	return groups
}

// ActiveGrid returns the ledger for the running sprint, or nil when the family
// is between sprints.
func (s *Service) ActiveGrid(ctx context.Context) (*Grid, error) {
	sprint, err := s.store.ActiveSprint(ctx)
	if err != nil || sprint == nil {
		return nil, err
	}

	grid, err := s.BuildGrid(ctx, *sprint)
	if err != nil {
		return nil, err
	}

	return &grid, nil
}

// Achiever names one person who earned their goal during a sprint.
type Achiever struct {
	PersonID   int64
	Name       string
	RewardName string
}

// HistoryEntry summarises one archived sprint for the history list.
type HistoryEntry struct {
	Sprint           domain.Sprint
	ParticipantCount int
	Achievers        []Achiever
	TicksEarned      int
}

// HistoryList summarises every archived sprint, most recent first.
func (s *Service) HistoryList(ctx context.Context) ([]HistoryEntry, error) {
	sprints, err := s.store.ArchivedSprints(ctx)
	if err != nil {
		return nil, err
	}

	entries := make([]HistoryEntry, 0, len(sprints))

	for _, sprint := range sprints {
		participants, err := s.store.ParticipantsBySprint(ctx, sprint.ID)
		if err != nil {
			return nil, err
		}

		entry := HistoryEntry{Sprint: sprint, ParticipantCount: len(participants)}

		windowStart := points.SprintStartInstant(sprint)
		windowEnd := points.SprintEndInstant(sprint)

		for _, participant := range participants {
			earned, err := s.store.TicksEarnedByParticipant(ctx, participant.ID)
			if err != nil {
				return nil, err
			}

			entry.TicksEarned += earned

			reward, err := s.store.RewardByIDPtr(ctx, participant.ActiveRewardID)
			if err != nil {
				return nil, err
			}

			// Only count a goal as this sprint's win if it was actually earned
			// inside the sprint's window; a goal carried in already achieved
			// belongs to whichever sprint earned it.
			if reward == nil || reward.AchievedAt == nil {
				continue
			}
			if reward.AchievedAt.Before(windowStart) || reward.AchievedAt.After(windowEnd) {
				continue
			}

			person, err := s.store.PersonWithTrashed(ctx, participant.PersonID)
			if err != nil {
				return nil, err
			}

			entry.Achievers = append(entry.Achievers, Achiever{
				PersonID:   person.ID,
				Name:       person.Name,
				RewardName: reward.Name,
			})
		}

		entries = append(entries, entry)
	}

	return entries, nil
}

// PersonTotals is one person's lifetime record.
type PersonTotals struct {
	Person          domain.Person
	TicksEarned     int
	RewardsAchieved int
	SprintCount     int
	CurrentBalance  int
	ActiveReward    *domain.Reward
}

// LifetimeTotals reports every active family member's running record.
func (s *Service) LifetimeTotals(ctx context.Context) ([]PersonTotals, error) {
	people, err := s.store.People(ctx)
	if err != nil {
		return nil, err
	}

	totals := make([]PersonTotals, 0, len(people))

	for _, person := range people {
		ticks, err := s.store.TicksEarnedByPerson(ctx, person.ID)
		if err != nil {
			return nil, err
		}

		rewards, err := s.store.CountAchievedRewards(ctx, person.ID)
		if err != nil {
			return nil, err
		}

		sprints, err := s.store.CountSprintsForPerson(ctx, person.ID)
		if err != nil {
			return nil, err
		}

		balance, err := s.points.LifetimeBalance(ctx, person.ID)
		if err != nil {
			return nil, err
		}

		reward, err := s.store.ActiveRewardForPerson(ctx, person.ID)
		if err != nil {
			return nil, err
		}

		totals = append(totals, PersonTotals{
			Person:          person,
			TicksEarned:     ticks,
			RewardsAchieved: rewards,
			SprintCount:     sprints,
			CurrentBalance:  balance,
			ActiveReward:    reward,
		})
	}

	return totals, nil
}
