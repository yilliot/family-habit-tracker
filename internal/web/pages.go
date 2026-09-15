package web

import (
	"net/http"

	"github.com/el/family-habit-tracker/internal/web/views"
)

// tracker renders the home screen: the running sprint's ledger.
func (s *Server) tracker(w http.ResponseWriter, r *http.Request) {
	page, err := s.page(w, r, "Tracker")
	if err != nil {
		s.fail(w, r, err)

		return
	}

	grid, err := s.service.ActiveGrid(ctx(r))
	if err != nil {
		s.fail(w, r, err)

		return
	}

	s.render(w, r, http.StatusOK, views.Tracker(page, grid))
}

// historyIndex lists archived sprints.
func (s *Server) historyIndex(w http.ResponseWriter, r *http.Request) {
	page, err := s.page(w, r, "History")
	if err != nil {
		s.fail(w, r, err)

		return
	}

	entries, err := s.service.HistoryList(ctx(r))
	if err != nil {
		s.fail(w, r, err)

		return
	}

	s.render(w, r, http.StatusOK, views.HistoryIndex(page, entries))
}

// historyShow renders one archived sprint read-only. An active sprint is not
// reachable here; it lives on the tracker, where it can still be ticked.
func (s *Server) historyShow(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "sprint")
	if err != nil {
		s.notFound(w, r)

		return
	}

	sprint, err := s.service.Store().Sprint(ctx(r), id)
	if err != nil {
		s.fail(w, r, err)

		return
	}

	if !sprint.IsArchived() {
		http.Redirect(w, r, "/", http.StatusSeeOther)

		return
	}

	page, err := s.page(w, r, "Archived sprint")
	if err != nil {
		s.fail(w, r, err)

		return
	}

	// The masthead should print the sprint being read, not the running one.
	page.Sprint = &sprint

	grid, err := s.service.BuildGrid(ctx(r), sprint)
	if err != nil {
		s.fail(w, r, err)

		return
	}

	s.render(w, r, http.StatusOK, views.HistoryShow(page, grid))
}

// totals renders the family's lifetime record.
func (s *Server) totals(w http.ResponseWriter, r *http.Request) {
	page, err := s.page(w, r, "Totals")
	if err != nil {
		s.fail(w, r, err)

		return
	}

	records, err := s.service.LifetimeTotals(ctx(r))
	if err != nil {
		s.fail(w, r, err)

		return
	}

	s.render(w, r, http.StatusOK, views.Totals(page, records))
}

// settings renders the family-meeting page. The open tab comes from the query
// string so that a tab can be linked to and bookmarked.
func (s *Server) settings(w http.ResponseWriter, r *http.Request) {
	page, err := s.page(w, r, "Settings")
	if err != nil {
		s.fail(w, r, err)

		return
	}

	data, err := s.service.Settings(ctx(r))
	if err != nil {
		s.fail(w, r, err)

		return
	}

	view := views.SettingsView{Data: data, Tab: normaliseTab(r.URL.Query().Get("tab"))}

	if focus := r.URL.Query().Get("person"); focus != "" {
		if id, err := parseInt64(focus); err == nil {
			view.FocusID = id
		}
	}

	s.render(w, r, http.StatusOK, views.Settings(page, view))
}

// normaliseTab keeps an unknown or absent tab on the sprint panel, which is
// what a family meeting opens first.
func normaliseTab(tab string) string {
	switch tab {
	case "people", "rewards", "sprint":
		return tab
	default:
		return "sprint"
	}
}
