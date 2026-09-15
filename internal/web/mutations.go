package web

import (
	"net/http"
	"strconv"
	"time"

	"github.com/el/family-habit-tracker/internal/domain"
	"github.com/el/family-habit-tracker/internal/service"
	"github.com/el/family-habit-tracker/internal/web/views"
)

/* ——— Ticking ——————————————————————————————————————————————————— */

// toggleTick flips one square of the ledger.
//
// From htmx it answers with just the cell plus out-of-band updates for the
// balance, reward panel and progress card. Without JavaScript the same POST
// falls through to a redirect and the page simply re-renders.
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

		s.redirectBack(w, r, "/")

		return
	}

	s.renderTickFragment(w, r, result, habitID, day)
}

// renderTickFragment answers an htmx tick with the freshly computed cell and
// everything else on the page that the tick changed.
func (s *Server) renderTickFragment(w http.ResponseWriter, r *http.Request, result service.ToggleResult, habitID int64, day time.Time) {
	sprint, err := s.service.Store().Sprint(ctx(r), result.Participant.SprintID)
	if err != nil {
		s.fail(w, r, err)

		return
	}

	// Rebuilding the grid keeps the fragment and the full page render on one
	// code path, so a cell swapped in place can never drift from a reload. At
	// family scale this is a handful of indexed queries.
	grid, err := s.service.BuildGrid(ctx(r), sprint)
	if err != nil {
		s.fail(w, r, err)

		return
	}

	participant, habit, ok := findHabit(grid, habitID)
	if !ok {
		// The habit vanished between the toggle and the re-read; ask the page
		// to reload rather than swapping in something inconsistent.
		w.Header().Set("HX-Refresh", "true")
		w.WriteHeader(http.StatusOK)

		return
	}

	gridDay, ok := findDay(grid, day.Format("2006-01-02"))
	if !ok {
		w.Header().Set("HX-Refresh", "true")
		w.WriteHeader(http.StatusOK)

		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)

	if err := views.TickCell(grid, habit, gridDay, participant).Render(ctx(r), w); err != nil {
		s.logger.Error("render tick fragment", "error", err)

		return
	}

	if result.JustAchieved != nil {
		component := views.Celebration(result.Person.ID, result.Person.Name, result.JustAchieved.Name)
		if err := component.Render(ctx(r), w); err != nil {
			s.logger.Error("render celebration", "error", err)
		}

		return
	}

	if err := views.ClearCelebration().Render(ctx(r), w); err != nil {
		s.logger.Error("clear celebration", "error", err)
	}
}

// findHabit locates a habit and its participant within a rebuilt grid.
func findHabit(grid service.Grid, habitID int64) (service.GridParticipant, service.GridHabit, bool) {
	for _, participant := range grid.Participants {
		for _, habit := range participant.Habits {
			if habit.Habit.ID == habitID {
				return participant, habit, true
			}
		}
	}

	return service.GridParticipant{}, service.GridHabit{}, false
}

// findDay locates a column by its ISO date.
func findDay(grid service.Grid, iso string) (service.GridDay, bool) {
	for _, day := range grid.Days {
		if day.ISO == iso {
			return day, true
		}
	}

	return service.GridDay{}, false
}

/* ——— Sprint lifecycle —————————————————————————————————————————— */

// startSprint opens a sprint from the family-meeting form.
func (s *Server) startSprint(w http.ResponseWriter, r *http.Request) {
	if !s.form(w, r) {
		return
	}

	start, err := dateField(r, "start_date")
	if err != nil {
		s.reportError(w, r, err, "/settings")

		return
	}

	end, err := dateField(r, "end_date")
	if err != nil {
		s.reportError(w, r, err, "/settings")

		return
	}

	var participants []service.ParticipantInput

	for _, raw := range r.PostForm["person_ids"] {
		personID, err := parseInt64(raw)
		if err != nil {
			continue
		}

		participants = append(participants, service.ParticipantInput{
			PersonID: personID,
			Habits:   habitInputs(habitLines(field(r, "habits_"+raw))),
			Reward:   rewardChoice(r, raw),
		})
	}

	if len(participants) == 0 {
		s.reportError(w, r, domain.Invalid("participants", "Pick at least one person for the sprint."), "/settings")

		return
	}

	if _, err := s.service.StartSprint(ctx(r), start, end, participants); s.reportError(w, r, err, "/settings") {
		return
	}

	setFlash(w, "success", "Sprint started")
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// habitInputs turns habit names into service inputs, numbering them in the
// order they were typed.
func habitInputs(names []string) []service.HabitInput {
	inputs := make([]service.HabitInput, 0, len(names))

	for index, name := range names {
		inputs = append(inputs, service.HabitInput{Name: name, DisplayOrder: index + 1})
	}

	return inputs
}

// rewardChoice reads one person's reward decision out of the sprint form.
func rewardChoice(r *http.Request, personKey string) service.RewardChoice {
	mode := field(r, "reward_mode_"+personKey)
	if mode == "" {
		mode = service.RewardKeepCurrent
	}

	return service.RewardChoice{
		Mode: mode,
		Name: field(r, "reward_name_"+personKey),
		Cost: intField(r, "reward_cost_"+personKey, 0),
	}
}

// endSprint archives the running sprint.
func (s *Server) endSprint(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "sprint")
	if err != nil {
		s.notFound(w, r)

		return
	}

	if _, err := s.service.EndSprint(ctx(r), id); s.reportError(w, r, err, "/settings") {
		return
	}

	setFlash(w, "success", "Sprint archived. You can start a new one.")
	http.Redirect(w, r, "/settings", http.StatusSeeOther)
}

// addParticipant enrols someone into a sprint already under way.
func (s *Server) addParticipant(w http.ResponseWriter, r *http.Request) {
	sprintID, err := pathID(r, "sprint")
	if err != nil {
		s.notFound(w, r)

		return
	}

	if !s.form(w, r) {
		return
	}

	personID, err := parseInt64(field(r, "person_id"))
	if err != nil {
		s.reportError(w, r, domain.Invalid("person_id", "Choose who is joining."), "/settings")

		return
	}

	input := service.ParticipantInput{
		PersonID: personID,
		Habits:   habitInputs(habitLines(field(r, "habits"))),
		Reward:   service.RewardChoice{Mode: service.RewardKeepCurrent},
	}

	if _, err := s.service.AddParticipant(ctx(r), sprintID, input); s.reportError(w, r, err, "/settings") {
		return
	}

	s.succeed(w, r, "Added to the sprint", "/settings")
}

/* ——— People ———————————————————————————————————————————————————— */

func (s *Server) createPerson(w http.ResponseWriter, r *http.Request) {
	if !s.form(w, r) {
		return
	}

	_, err := s.service.CreatePerson(ctx(r), field(r, "name"), optionalIntField(r, "display_order"))
	if s.reportError(w, r, err, "/settings?tab=people") {
		return
	}

	s.succeed(w, r, "Person added", "/settings?tab=people")
}

func (s *Server) updatePerson(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "person")
	if err != nil {
		s.notFound(w, r)

		return
	}

	if !s.form(w, r) {
		return
	}

	_, err = s.service.UpdatePerson(ctx(r), id, field(r, "name"), intField(r, "display_order", 0))
	if s.reportError(w, r, err, "/settings?tab=people") {
		return
	}

	s.succeed(w, r, "Saved", "/settings?tab=people")
}

func (s *Server) archivePerson(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "person")
	if err != nil {
		s.notFound(w, r)

		return
	}

	if err := s.service.ArchivePerson(ctx(r), id); s.reportError(w, r, err, "/settings?tab=people") {
		return
	}

	s.succeed(w, r, "Person removed. Their history is kept.", "/settings?tab=people")
}

/* ——— Habits ———————————————————————————————————————————————————— */

func (s *Server) createHabit(w http.ResponseWriter, r *http.Request) {
	participantID, err := pathID(r, "participant")
	if err != nil {
		s.notFound(w, r)

		return
	}

	if !s.form(w, r) {
		return
	}

	_, err = s.service.CreateHabit(ctx(r), participantID, field(r, "name"), optionalIntField(r, "display_order"))
	if s.reportError(w, r, err, "/settings") {
		return
	}

	s.succeed(w, r, "Habit added", "/settings")
}

func (s *Server) updateHabit(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "habit")
	if err != nil {
		s.notFound(w, r)

		return
	}

	if !s.form(w, r) {
		return
	}

	habit, err := s.service.Store().Habit(ctx(r), id)
	if s.reportError(w, r, err, "/settings") {
		return
	}

	order := intField(r, "display_order", habit.DisplayOrder)

	if _, err := s.service.UpdateHabit(ctx(r), id, field(r, "name"), order); s.reportError(w, r, err, "/settings") {
		return
	}

	s.succeed(w, r, "Habit saved", "/settings")
}

func (s *Server) archiveHabit(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "habit")
	if err != nil {
		s.notFound(w, r)

		return
	}

	if err := s.service.ArchiveHabit(ctx(r), id); s.reportError(w, r, err, "/settings") {
		return
	}

	s.succeed(w, r, "Habit removed. Its ticks still count.", "/settings")
}

/* ——— Rewards ——————————————————————————————————————————————————— */

func (s *Server) createReward(w http.ResponseWriter, r *http.Request) {
	personID, err := pathID(r, "person")
	if err != nil {
		s.notFound(w, r)

		return
	}

	if !s.form(w, r) {
		return
	}

	_, err = s.service.CreateReward(ctx(r), personID, field(r, "name"), intField(r, "cost", 0))
	if s.reportError(w, r, err, "/settings?tab=rewards") {
		return
	}

	s.succeed(w, r, "Reward added", "/settings?tab=rewards")
}

func (s *Server) updateReward(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "reward")
	if err != nil {
		s.notFound(w, r)

		return
	}

	if !s.form(w, r) {
		return
	}

	reward, err := s.service.Store().Reward(ctx(r), id)
	if s.reportError(w, r, err, "/settings?tab=rewards") {
		return
	}

	// The cost input is disabled for an achieved reward, so the browser omits
	// it; fall back to the stored cost rather than reading it as zero.
	cost := intField(r, "cost", reward.Cost)

	if _, err := s.service.UpdateReward(ctx(r), id, field(r, "name"), cost); s.reportError(w, r, err, "/settings?tab=rewards") {
		return
	}

	s.succeed(w, r, "Reward saved", "/settings?tab=rewards")
}

func (s *Server) deleteReward(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "reward")
	if err != nil {
		s.notFound(w, r)

		return
	}

	if err := s.service.DeleteReward(ctx(r), id); s.reportError(w, r, err, "/settings?tab=rewards") {
		return
	}

	s.succeed(w, r, "Reward deleted", "/settings?tab=rewards")
}

func (s *Server) setActiveReward(w http.ResponseWriter, r *http.Request) {
	participantID, err := pathID(r, "participant")
	if err != nil {
		s.notFound(w, r)

		return
	}

	if !s.form(w, r) {
		return
	}

	var rewardID *int64

	if raw := field(r, "reward_id"); raw != "" {
		parsed, err := parseInt64(raw)
		if err != nil {
			s.reportError(w, r, domain.Invalid("reward_id", "Choose a valid reward."), "/settings")

			return
		}

		rewardID = &parsed
	}

	achieved, err := s.service.SetActiveReward(ctx(r), participantID, rewardID)
	if s.reportError(w, r, err, "/settings") {
		return
	}

	if achieved != nil {
		s.succeed(w, r, "Already earned — "+achieved.Name+" is yours!", "/settings")

		return
	}

	s.succeed(w, r, "Reward updated", "/settings")
}

/* ——— Points ———————————————————————————————————————————————————— */

// createAdjustment records a deduction. The form collects a positive number and
// it is stored as a negative adjustment, so nobody operating the tablet has to
// think about signs.
func (s *Server) createAdjustment(w http.ResponseWriter, r *http.Request) {
	participantID, err := pathID(r, "participant")
	if err != nil {
		s.notFound(w, r)

		return
	}

	if !s.form(w, r) {
		return
	}

	pointsValue := intField(r, "points", 0)
	if pointsValue < 1 || pointsValue > domain.MaxAdjustment {
		s.reportError(w, r, domain.Invalid("points", "Enter between 1 and "+strconv.Itoa(domain.MaxAdjustment)+" points."), "/")

		return
	}

	validator := &domain.Validator{}
	reason := validator.OptionalText("reason", field(r, "reason"), domain.MaxReasonLen)

	if err := validator.Err(); s.reportError(w, r, err, "/") {
		return
	}

	if _, err := s.service.CreateAdjustment(ctx(r), participantID, -pointsValue, reason); s.reportError(w, r, err, "/") {
		return
	}

	s.succeed(w, r, "Deducted "+strconv.Itoa(pointsValue)+" "+views.Pluralise(pointsValue, "point", "points"), "/")
}
