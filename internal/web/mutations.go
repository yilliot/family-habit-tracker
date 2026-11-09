package web

import (
	"net/http"
	"strconv"

	"github.com/el/family-habit-tracker/internal/domain"
	"github.com/el/family-habit-tracker/internal/service"
	"github.com/el/family-habit-tracker/internal/web/views"
)

/* ——— Ticking ——————————————————————————————————————————————————— */

// toggleTick flips one habit on one day.
//
// From htmx it answers with the tapped row plus out-of-band updates for the
// count, week, points, month, family strip and celebration slot, all from one
// fresh read of the day, so nothing swapped in can drift from a reload.
// Without JavaScript the same POST falls through to a redirect and the page
// simply re-renders.
func (s *Server) toggleTick(w http.ResponseWriter, r *http.Request) {
	if !s.form(w, r) {
		return
	}

	habitID, err := parseInt64(field(r, "habit_id"))
	if err != nil {
		s.reportError(w, r, domain.Invalid("habit_id", "That habit could not be identified."), "/")

		return
	}

	day, err := dateField(r, "tick_date")
	if err != nil {
		s.reportError(w, r, err, "/")

		return
	}

	result, err := s.service.ToggleTick(ctx(r), habitID, day)
	if s.reportError(w, r, err, "/") {
		return
	}

	if !isHTMX(r) {
		if result.JustAchieved != nil {
			setFlash(w, "success", result.Person.Name+" earned "+result.JustAchieved.Name+"!")
		}

		s.redirectBack(w, r, views.DayHref(result.Person.ID, day, s.service.Today()))

		return
	}

	view, err := s.service.Day(ctx(r), result.Person.ID, day)
	if err != nil {
		s.fail(w, r, err)

		return
	}

	row, found := findRow(view, habitID)
	if !found {
		// The habit left this day between the tap and the re-read; reload
		// rather than swap in something inconsistent.
		w.Header().Set("HX-Refresh", "true")
		w.WriteHeader(http.StatusOK)

		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)

	if err := views.TickResponse(view, row).Render(ctx(r), w); err != nil {
		s.logger.Error("render tick", "error", err)

		return
	}

	celebration := views.ClearCelebration()
	if result.JustAchieved != nil {
		celebration = views.Celebration(result.Person, *result.JustAchieved)
	}

	if err := celebration.Render(ctx(r), w); err != nil {
		s.logger.Error("render celebration", "error", err)
	}
}

// findRow locates a habit's row in a day.
func findRow(view service.DayView, habitID int64) (service.DayHabit, bool) {
	for _, row := range view.Habits {
		if row.Habit.ID == habitID {
			return row, true
		}
	}

	return service.DayHabit{}, false
}

/* ——— A person's habits ————————————————————————————————————————— */

func (s *Server) addHabit(w http.ResponseWriter, r *http.Request) {
	personID, ok := s.personParam(w, r)
	if !ok || !s.form(w, r) {
		return
	}

	habit, err := s.service.AddHabit(ctx(r), personID, field(r, "name"))
	if s.reportError(w, r, err, views.PersonHref(personID)) {
		return
	}

	s.succeed(w, r, "Added “"+habit.Name+"”, starting today", views.PersonHref(personID))
}

func (s *Server) renameHabit(w http.ResponseWriter, r *http.Request) {
	id, ok := s.idParam(w, r, "habit")
	if !ok || !s.form(w, r) {
		return
	}

	habit, err := s.service.RenameHabit(ctx(r), id, field(r, "name"))
	if s.reportError(w, r, err, "/") {
		return
	}

	s.succeed(w, r, "Renamed to “"+habit.Name+"”", views.EditHref(habit.PersonID))
}

func (s *Server) moveHabit(w http.ResponseWriter, r *http.Request) {
	id, ok := s.idParam(w, r, "habit")
	if !ok || !s.form(w, r) {
		return
	}

	if err := s.service.MoveHabit(ctx(r), id, direction(r)); s.reportError(w, r, err, "/") {
		return
	}

	s.redirectBack(w, r, "/")
}

func (s *Server) stopHabit(w http.ResponseWriter, r *http.Request) {
	id, ok := s.idParam(w, r, "habit")
	if !ok {
		return
	}

	habit, err := s.service.StopHabit(ctx(r), id)
	if s.reportError(w, r, err, "/") {
		return
	}

	s.succeed(w, r, "Stopped “"+habit.Name+"”. Its ticks stay in the history.", views.EditHref(habit.PersonID))
}

func (s *Server) restartHabit(w http.ResponseWriter, r *http.Request) {
	id, ok := s.idParam(w, r, "habit")
	if !ok {
		return
	}

	habit, err := s.service.RestartHabit(ctx(r), id)
	if s.reportError(w, r, err, "/") {
		return
	}

	s.succeed(w, r, "“"+habit.Name+"” is running again", views.EditHref(habit.PersonID))
}

/* ——— Goals and points —————————————————————————————————————————— */

func (s *Server) setGoal(w http.ResponseWriter, r *http.Request) {
	personID, ok := s.personParam(w, r)
	if !ok || !s.form(w, r) {
		return
	}

	goal, achieved, err := s.service.SetGoal(ctx(r), personID, field(r, "name"), intField(r, "cost", 0))
	if s.reportError(w, r, err, views.PersonHref(personID)) {
		return
	}

	if achieved != nil {
		s.succeed(w, r, "Already earned: "+achieved.Name+" is yours!", views.PersonHref(personID))

		return
	}

	s.succeed(w, r, "Now saving for "+goal.Name, views.PersonHref(personID))
}

func (s *Server) updateReward(w http.ResponseWriter, r *http.Request) {
	id, ok := s.idParam(w, r, "reward")
	if !ok || !s.form(w, r) {
		return
	}

	reward, err := s.service.Store().Reward(ctx(r), id)
	if s.reportError(w, r, err, "/") {
		return
	}

	// An achieved goal's cost cannot change, so a form that leaves it out keeps
	// the stored cost rather than reading it as zero.
	cost := intField(r, "cost", reward.Cost)

	_, achieved, err := s.service.UpdateReward(ctx(r), id, field(r, "name"), cost)
	if s.reportError(w, r, err, views.EditHref(reward.PersonID)) {
		return
	}

	if achieved != nil {
		s.succeed(w, r, "Already earned: "+achieved.Name+" is yours!", views.EditHref(reward.PersonID))

		return
	}

	s.succeed(w, r, "Goal saved", views.EditHref(reward.PersonID))
}

func (s *Server) deleteReward(w http.ResponseWriter, r *http.Request) {
	id, ok := s.idParam(w, r, "reward")
	if !ok {
		return
	}

	reward, err := s.service.DeleteReward(ctx(r), id)
	if s.reportError(w, r, err, "/") {
		return
	}

	s.succeed(w, r, "Goal removed", views.EditHref(reward.PersonID))
}

// createAdjustment records a deduction. The form collects a positive number and
// it is stored as a negative adjustment, so nobody operating the tablet has to
// think about signs.
func (s *Server) createAdjustment(w http.ResponseWriter, r *http.Request) {
	personID, ok := s.personParam(w, r)
	if !ok || !s.form(w, r) {
		return
	}

	back := views.PersonHref(personID)

	pointsValue := intField(r, "points", 0)
	if pointsValue < 1 || pointsValue > domain.MaxAdjustment {
		s.reportError(w, r, domain.Invalid("points", "Enter between 1 and "+strconv.Itoa(domain.MaxAdjustment)+" points."), back)

		return
	}

	validator := &domain.Validator{}
	reason := validator.OptionalText("reason", field(r, "reason"), domain.MaxReasonLen)

	if err := validator.Err(); s.reportError(w, r, err, back) {
		return
	}

	if _, err := s.service.CreateAdjustment(ctx(r), personID, -pointsValue, reason); s.reportError(w, r, err, back) {
		return
	}

	s.succeed(w, r, "Deducted "+strconv.Itoa(pointsValue)+" "+views.Pluralise(pointsValue, "point", "points"), back)
}

/* ——— The family list ——————————————————————————————————————————— */

func (s *Server) createPerson(w http.ResponseWriter, r *http.Request) {
	if !s.form(w, r) {
		return
	}

	person, err := s.service.CreatePerson(ctx(r), field(r, "name"), nil)
	if s.reportError(w, r, err, "/family") {
		return
	}

	s.succeed(w, r, person.Name+" added", "/family")
}

func (s *Server) renamePerson(w http.ResponseWriter, r *http.Request) {
	id, ok := s.personParam(w, r)
	if !ok || !s.form(w, r) {
		return
	}

	person, err := s.service.RenamePerson(ctx(r), id, field(r, "name"))
	if s.reportError(w, r, err, "/family") {
		return
	}

	s.succeed(w, r, "Renamed to "+person.Name, "/family")
}

func (s *Server) movePerson(w http.ResponseWriter, r *http.Request) {
	id, ok := s.personParam(w, r)
	if !ok || !s.form(w, r) {
		return
	}

	if err := s.service.MovePerson(ctx(r), id, direction(r)); s.reportError(w, r, err, "/family") {
		return
	}

	s.redirectBack(w, r, "/family")
}

func (s *Server) archivePerson(w http.ResponseWriter, r *http.Request) {
	id, ok := s.personParam(w, r)
	if !ok {
		return
	}

	if err := s.service.ArchivePerson(ctx(r), id); s.reportError(w, r, err, "/family") {
		return
	}

	s.succeed(w, r, "Archived. Their history is kept.", "/family")
}

func (s *Server) restorePerson(w http.ResponseWriter, r *http.Request) {
	id, ok := s.personParam(w, r)
	if !ok {
		return
	}

	if err := s.service.RestorePerson(ctx(r), id); s.reportError(w, r, err, "/family") {
		return
	}

	s.succeed(w, r, "Back on the family list", "/family")
}

/* ——— Helpers ——————————————————————————————————————————————————— */

// idParam reads a numeric path value, answering 404 itself when it is not one.
func (s *Server) idParam(w http.ResponseWriter, r *http.Request, name string) (int64, bool) {
	id, err := pathID(r, name)
	if err != nil {
		s.notFound(w, r)

		return 0, false
	}

	return id, true
}

// direction reads the up/down buttons of a list as −1 or +1.
func direction(r *http.Request) int {
	if field(r, "dir") == "up" {
		return -1
	}

	return 1
}
