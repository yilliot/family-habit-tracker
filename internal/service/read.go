package service

import (
	"context"
	"sort"
	"strings"
	"time"

	"github.com/el/family-habit-tracker/internal/db"
	"github.com/el/family-habit-tracker/internal/domain"
)

// StreakLookback bounds how far back a streak is counted. A streak that long
// shows as its bound; nobody is checking whether it is 400 or 401.
const StreakLookback = 400

/* ——— The raw material: one person's habits and ticks ———————————— */

// ledger is one person's habits, with their ticks over a stretch of days.
type ledger struct {
	habits []domain.Habit
	names  map[int64]string
	ticked map[time.Time]map[int64]bool
}

func (s *Service) loadLedger(ctx context.Context, personID int64, from, to time.Time) (ledger, error) {
	habits, err := s.store.HabitsForPerson(ctx, personID)
	if err != nil {
		return ledger{}, err
	}

	ticks, err := s.store.TicksForPerson(ctx, personID, from, to)
	if err != nil {
		return ledger{}, err
	}

	l := ledger{
		habits: habits,
		names:  make(map[int64]string, len(habits)),
		ticked: make(map[time.Time]map[int64]bool, len(ticks)),
	}

	for _, habit := range habits {
		l.names[habit.ID] = habit.Name
	}

	for _, tick := range ticks {
		if l.ticked[tick.Day] == nil {
			l.ticked[tick.Day] = make(map[int64]bool, 4)
		}

		l.ticked[tick.Day][tick.HabitID] = true
	}

	return l, nil
}

// liveOn lists the habits running on a day, in display order.
func (l ledger) liveOn(day time.Time) []domain.Habit {
	var live []domain.Habit

	for _, habit := range l.habits {
		if habit.LiveOn(day) {
			live = append(live, habit)
		}
	}

	return live
}

// counts reports how many of a day's habits were done, out of how many ran.
func (l ledger) counts(day time.Time) (done, total int) {
	for _, habit := range l.liveOn(day) {
		total++

		if l.ticked[day][habit.ID] {
			done++
		}
	}

	return done, total
}

// streak counts the days in a row, ending on day, on which a habit of this name
// was ticked. Habits re-created under the same name, as every sprint used to
// do, count as one.
func (l ledger) streak(name string, day, floor time.Time) int {
	n := 0

	for d := day; !d.Before(floor); d = d.AddDate(0, 0, -1) {
		found := false

		for habitID := range l.ticked[d] {
			if sameName(l.names[habitID], name) {
				found = true

				break
			}
		}

		if !found {
			break
		}

		n++
	}

	return n
}

// earliest is the first day the person had any habit, or today if none.
func (l ledger) earliest(today time.Time) time.Time {
	first := today

	for _, habit := range l.habits {
		neverRan := habit.EndDate != nil && habit.EndDate.Before(habit.StartDate)
		if !neverRan && habit.StartDate.Before(first) {
			first = habit.StartDate
		}
	}

	return first
}

/* ——— Calendar cells ——————————————————————————————————————————— */

// DayCell is one day in a week strip or month calendar.
type DayCell struct {
	Date       time.Time
	ISO        string
	Done       int
	Total      int
	InMonth    bool
	IsToday    bool
	IsFuture   bool
	IsSelected bool
}

// Percent is how much of the day's list was done, 0–100.
func (c DayCell) Percent() int {
	if c.Total == 0 {
		return 0
	}

	return min(c.Done*100/c.Total, 100)
}

// Level buckets a day for the calendar's shading: "none" when the person had
// no habits that day, then 0–4 by how much was done.
func (c DayCell) Level() string {
	switch {
	case c.Total == 0:
		return "none"
	case c.Done == 0:
		return "0"
	case c.Done >= c.Total:
		return "4"
	case c.Done*2 > c.Total:
		return "3"
	case c.Done*4 > c.Total:
		return "2"
	default:
		return "1"
	}
}

func (l ledger) cell(day, today, selected time.Time) DayCell {
	cell := DayCell{
		Date:       day,
		ISO:        db.FormatDate(day),
		IsToday:    day.Equal(today),
		IsFuture:   day.After(today),
		IsSelected: day.Equal(selected),
	}

	if !cell.IsFuture {
		cell.Done, cell.Total = l.counts(day)
	}

	return cell
}

// MonthGrid is a month laid out Monday to Sunday, padded with the neighbouring
// months' days so every row is a full week.
type MonthGrid struct {
	Month time.Time
	Weeks [][]DayCell
}

// Label names the month: "August 2026".
func (m MonthGrid) Label() string { return m.Month.Format("January 2006") }

// monthBounds returns the first and last day drawn for a month's grid.
func monthBounds(month time.Time) (time.Time, time.Time) {
	first := mondayOf(month)
	last := month.AddDate(0, 1, -1)
	last = mondayOf(last).AddDate(0, 0, 6)

	return first, last
}

func (l ledger) month(month, today, selected time.Time) MonthGrid {
	grid := MonthGrid{Month: month}
	first, last := monthBounds(month)

	for weekStart := first; !weekStart.After(last); weekStart = weekStart.AddDate(0, 0, 7) {
		week := make([]DayCell, 0, 7)

		for d := weekStart; d.Before(weekStart.AddDate(0, 0, 7)); d = d.AddDate(0, 0, 1) {
			cell := l.cell(d, today, selected)
			cell.InMonth = d.Month() == month.Month()
			week = append(week, cell)
		}

		grid.Weeks = append(grid.Weeks, week)
	}

	return grid
}

// mondayOf returns the Monday starting the week a day falls in.
func mondayOf(day time.Time) time.Time {
	offset := (int(day.Weekday()) + 6) % 7

	return day.AddDate(0, 0, -offset)
}

// firstOfMonth returns the first day of a day's month.
func firstOfMonth(day time.Time) time.Time {
	return time.Date(day.Year(), day.Month(), 1, 0, 0, 0, 0, time.UTC)
}

/* ——— The family strip ——————————————————————————————————————————— */

// StripEntry is one name in the switcher at the top of every person's page,
// with how far they are today.
type StripEntry struct {
	Person domain.Person
	Done   int
	Total  int
}

// AllDone reports whether the person has done everything on today's list.
func (e StripEntry) AllDone() bool { return e.Total > 0 && e.Done >= e.Total }

// Strip lists everyone in the family with today's progress.
func (s *Service) Strip(ctx context.Context) ([]StripEntry, error) {
	today := s.store.Today()

	people, err := s.store.People(ctx)
	if err != nil {
		return nil, err
	}

	done, err := s.store.TickCountsOn(ctx, today)
	if err != nil {
		return nil, err
	}

	strip := make([]StripEntry, 0, len(people))

	for _, person := range people {
		habits, err := s.store.HabitsForPerson(ctx, person.ID)
		if err != nil {
			return nil, err
		}

		entry := StripEntry{Person: person, Done: done[person.ID]}
		for _, habit := range habits {
			if habit.LiveOn(today) {
				entry.Total++
			}
		}

		strip = append(strip, entry)
	}

	return strip, nil
}

/* ——— A person's day ———————————————————————————————————————————— */

// DayHabit is one row of a person's day.
type DayHabit struct {
	Habit domain.Habit
	Done  bool
	// Streak is the run of days, ending this day, the habit has been done.
	Streak int
}

// DayView is everything a person's day page shows.
type DayView struct {
	Person   domain.Person
	Day      time.Time
	Today    time.Time
	Strip    []StripEntry
	Habits   []DayHabit
	Week     []DayCell
	Month    MonthGrid
	Balance  int
	Goal     *domain.Reward
	Editable bool
	Earliest time.Time
}

// Done counts the habits ticked on the day.
func (v DayView) Done() int {
	n := 0
	for _, h := range v.Habits {
		if h.Done {
			n++
		}
	}

	return n
}

// IsToday reports whether the page shows today.
func (v DayView) IsToday() bool { return v.Day.Equal(v.Today) }

// IsYesterday reports whether the page shows yesterday.
func (v DayView) IsYesterday() bool { return v.Day.Equal(v.Today.AddDate(0, 0, -1)) }

// HasPrev reports whether there is an earlier day with anything to show.
func (v DayView) HasPrev() bool { return v.Day.After(v.Earliest) }

// Prev is the day before.
func (v DayView) Prev() time.Time { return v.Day.AddDate(0, 0, -1) }

// Next is the day after; HasPrev's counterpart is IsToday.
func (v DayView) Next() time.Time { return v.Day.AddDate(0, 0, 1) }

// GoalPercent is progress toward the goal, 0–100.
func (v DayView) GoalPercent() int { return goalPercent(v.Balance, v.Goal) }

func goalPercent(balance int, goal *domain.Reward) int {
	if goal == nil || goal.Cost <= 0 {
		return 0
	}

	return min(max(balance*100/goal.Cost, 0), 100)
}

// Day assembles one person's page for one day. A day in the future is pulled
// back to today.
func (s *Service) Day(ctx context.Context, personID int64, day time.Time) (DayView, error) {
	person, err := s.store.Person(ctx, personID)
	if err != nil {
		return DayView{}, err
	}

	today := s.store.Today()
	day = startOfDay(day)
	if day.After(today) {
		day = today
	}

	weekStart := mondayOf(day)
	monthStart := firstOfMonth(day)
	_, gridEnd := monthBounds(monthStart)

	to := gridEnd
	if weekEnd := weekStart.AddDate(0, 0, 6); weekEnd.After(to) {
		to = weekEnd
	}
	if to.After(today) {
		to = today
	}

	l, err := s.loadLedger(ctx, person.ID, day.AddDate(0, 0, -StreakLookback), to)
	if err != nil {
		return DayView{}, err
	}

	view := DayView{
		Person:   person,
		Day:      day,
		Today:    today,
		Editable: isEditable(day, today),
		Earliest: l.earliest(today),
		Month:    l.month(monthStart, today, day),
	}

	floor := day.AddDate(0, 0, -StreakLookback)

	for _, habit := range l.liveOn(day) {
		row := DayHabit{Habit: habit, Done: l.ticked[day][habit.ID]}
		if row.Done {
			row.Streak = l.streak(habit.Name, day, floor)
		}

		view.Habits = append(view.Habits, row)
	}

	for d := weekStart; d.Before(weekStart.AddDate(0, 0, 7)); d = d.AddDate(0, 0, 1) {
		view.Week = append(view.Week, l.cell(d, today, day))
	}

	if view.Balance, err = s.points.Balance(ctx, person.ID); err != nil {
		return DayView{}, err
	}

	if view.Goal, err = s.store.ActiveRewardForPerson(ctx, person.ID); err != nil {
		return DayView{}, err
	}

	if view.Strip, err = s.Strip(ctx); err != nil {
		return DayView{}, err
	}

	return view, nil
}

/* ——— History: a person's month ————————————————————————————————— */

// HabitCount is how often one habit, by name, was done.
type HabitCount struct {
	Name  string
	Count int
}

// MonthView is everything a person's history page shows for one month.
type MonthView struct {
	Person     domain.Person
	Today      time.Time
	Strip      []StripEntry
	Grid       MonthGrid
	Ticks      int
	ActiveDays int
	ByHabit    []HabitCount
	Earned     int
	Balance    int
	Goal       *domain.Reward
	Achieved   []domain.Reward
	Deductions []domain.PointAdjustment
	Earliest   time.Time
}

// HasPrev reports whether an earlier month has anything to show.
func (v MonthView) HasPrev() bool { return v.Grid.Month.After(firstOfMonth(v.Earliest)) }

// HasNext reports whether a later month has started yet.
func (v MonthView) HasNext() bool { return v.Grid.Month.Before(firstOfMonth(v.Today)) }

// Prev is the month before.
func (v MonthView) Prev() time.Time { return v.Grid.Month.AddDate(0, -1, 0) }

// Next is the month after.
func (v MonthView) Next() time.Time { return v.Grid.Month.AddDate(0, 1, 0) }

// MaxByHabit is the largest count, which scales the bars.
func (v MonthView) MaxByHabit() int {
	top := 0
	for _, c := range v.ByHabit {
		top = max(top, c.Count)
	}

	return top
}

// GoalPercent is progress toward the goal, 0–100.
func (v MonthView) GoalPercent() int { return goalPercent(v.Balance, v.Goal) }

// Month assembles one person's history for the month containing the given day.
func (s *Service) Month(ctx context.Context, personID int64, month time.Time) (MonthView, error) {
	person, err := s.store.Person(ctx, personID)
	if err != nil {
		return MonthView{}, err
	}

	today := s.store.Today()

	month = firstOfMonth(month)
	if month.After(today) {
		month = firstOfMonth(today)
	}

	first, last := monthBounds(month)

	l, err := s.loadLedger(ctx, person.ID, first, last)
	if err != nil {
		return MonthView{}, err
	}

	view := MonthView{
		Person:   person,
		Today:    today,
		Grid:     l.month(month, today, time.Time{}),
		Earliest: l.earliest(today),
	}

	type tally struct {
		name  string
		count int
		last  int64
	}

	byName := map[string]*tally{}

	for d := month; d.Month() == month.Month() && !d.After(today); d = d.AddDate(0, 0, 1) {
		ticked := l.ticked[d]
		if len(ticked) > 0 {
			view.ActiveDays++
		}

		for habitID := range ticked {
			view.Ticks++

			name := l.names[habitID]
			key := strings.ToLower(strings.TrimSpace(name))

			t := byName[key]
			if t == nil {
				t = &tally{name: name}
				byName[key] = t
			}

			t.count++
			if habitID > t.last {
				// The newest row's spelling wins, so a renamed habit shows as
				// it is called now.
				t.last, t.name = habitID, name
			}
		}
	}

	for _, t := range byName {
		view.ByHabit = append(view.ByHabit, HabitCount{Name: t.name, Count: t.count})
	}

	sort.Slice(view.ByHabit, func(i, j int) bool {
		if view.ByHabit[i].Count != view.ByHabit[j].Count {
			return view.ByHabit[i].Count > view.ByHabit[j].Count
		}

		return view.ByHabit[i].Name < view.ByHabit[j].Name
	})

	if view.Earned, err = s.store.TicksEarnedByPerson(ctx, person.ID); err != nil {
		return MonthView{}, err
	}

	if view.Balance, err = s.points.Balance(ctx, person.ID); err != nil {
		return MonthView{}, err
	}

	if view.Goal, view.Achieved, err = s.goals(ctx, person.ID); err != nil {
		return MonthView{}, err
	}

	if view.Deductions, err = s.store.AdjustmentsForPerson(ctx, person.ID, month, month.AddDate(0, 1, 0)); err != nil {
		return MonthView{}, err
	}

	if view.Strip, err = s.Strip(ctx); err != nil {
		return MonthView{}, err
	}

	return view, nil
}

// goals splits a person's rewards into the one being saved for and the ones
// already reached, newest first.
func (s *Service) goals(ctx context.Context, personID int64) (*domain.Reward, []domain.Reward, error) {
	rewards, err := s.store.RewardsForPerson(ctx, personID)
	if err != nil {
		return nil, nil, err
	}

	var (
		active   *domain.Reward
		achieved []domain.Reward
	)

	for i := range rewards {
		if rewards[i].IsAchieved() {
			achieved = append(achieved, rewards[i])
		} else if active == nil {
			active = &rewards[i]
		}
	}

	return active, achieved, nil
}

/* ——— Editing a person's habits and goal ——————————————————————————— */

// StoppedHabit is a habit no longer running, with the history of every run it
// had under its name.
type StoppedHabit struct {
	// Habit is the most recent run; restarting acts on it.
	Habit domain.Habit
	First time.Time
	Last  time.Time
	Ticks int
}

// EditView is everything the edit page shows for one person.
type EditView struct {
	Person   domain.Person
	Today    time.Time
	Strip    []StripEntry
	Running  []domain.Habit
	Stopped  []StoppedHabit
	Balance  int
	Goal     *domain.Reward
	Achieved []domain.Reward
}

// GoalPercent is progress toward the goal, 0–100.
func (v EditView) GoalPercent() int { return goalPercent(v.Balance, v.Goal) }

// Edit assembles a person's edit page.
func (s *Service) Edit(ctx context.Context, personID int64) (EditView, error) {
	person, err := s.store.Person(ctx, personID)
	if err != nil {
		return EditView{}, err
	}

	habits, err := s.store.HabitsForPerson(ctx, person.ID)
	if err != nil {
		return EditView{}, err
	}

	counts, err := s.store.TickCountsByHabit(ctx, person.ID)
	if err != nil {
		return EditView{}, err
	}

	view := EditView{Person: person, Today: s.store.Today()}

	stopped := map[string]*StoppedHabit{}

	for _, habit := range habits {
		if habit.IsRunning() {
			view.Running = append(view.Running, habit)

			continue
		}

		neverRan := habit.EndDate.Before(habit.StartDate)
		if neverRan && counts[habit.ID] == 0 {
			continue
		}

		key := strings.ToLower(strings.TrimSpace(habit.Name))

		entry := stopped[key]
		if entry == nil {
			entry = &StoppedHabit{Habit: habit, First: habit.StartDate, Last: *habit.EndDate}
			stopped[key] = entry
		}

		entry.Ticks += counts[habit.ID]

		if habit.StartDate.Before(entry.First) {
			entry.First = habit.StartDate
		}

		if habit.EndDate.After(entry.Last) || habit.EndDate.Equal(entry.Last) && habit.ID > entry.Habit.ID {
			entry.Last = *habit.EndDate
			entry.Habit = habit
		}
	}

	for key, entry := range stopped {
		if runningNamed(habits, entry.Habit.Name, 0) != nil {
			delete(stopped, key)

			continue
		}

		view.Stopped = append(view.Stopped, *entry)
	}

	sort.Slice(view.Stopped, func(i, j int) bool {
		if !view.Stopped[i].Last.Equal(view.Stopped[j].Last) {
			return view.Stopped[i].Last.After(view.Stopped[j].Last)
		}

		return view.Stopped[i].Habit.Name < view.Stopped[j].Habit.Name
	})

	if view.Balance, err = s.points.Balance(ctx, person.ID); err != nil {
		return EditView{}, err
	}

	if view.Goal, view.Achieved, err = s.goals(ctx, person.ID); err != nil {
		return EditView{}, err
	}

	if view.Strip, err = s.Strip(ctx); err != nil {
		return EditView{}, err
	}

	return view, nil
}

/* ——— The family page ——————————————————————————————————————————— */

// FamilyMember is one person on the family page.
type FamilyMember struct {
	Person  domain.Person
	Running int
	Balance int
	Goal    *domain.Reward
}

// FamilyView is the family page: who is on the list, and who was archived.
type FamilyView struct {
	Members  []FamilyMember
	Archived []domain.Person
}

// Family assembles the family page.
func (s *Service) Family(ctx context.Context) (FamilyView, error) {
	people, err := s.store.PeopleWithTrashed(ctx)
	if err != nil {
		return FamilyView{}, err
	}

	today := s.store.Today()

	var view FamilyView

	for _, person := range people {
		if person.IsArchived() {
			view.Archived = append(view.Archived, person)

			continue
		}

		member := FamilyMember{Person: person}

		habits, err := s.store.HabitsForPerson(ctx, person.ID)
		if err != nil {
			return FamilyView{}, err
		}

		for _, habit := range habits {
			if habit.LiveOn(today) {
				member.Running++
			}
		}

		if member.Balance, err = s.points.Balance(ctx, person.ID); err != nil {
			return FamilyView{}, err
		}

		if member.Goal, err = s.store.ActiveRewardForPerson(ctx, person.ID); err != nil {
			return FamilyView{}, err
		}

		view.Members = append(view.Members, member)
	}

	return view, nil
}
