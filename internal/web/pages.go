package web

import (
	"net/http"
	"strconv"
	"time"

	"github.com/el/family-habit-tracker/internal/web/views"
)

// personCookie remembers, per device, whose page it last showed, so a child's
// own tablet opens straight onto their today.
const personCookie = "ht_person"

// rememberedPerson returns the person this device last used, or zero.
func (s *Server) rememberedPerson(r *http.Request) int64 {
	cookie, err := r.Cookie(personCookie)
	if err != nil {
		return 0
	}

	id, err := strconv.ParseInt(cookie.Value, 10, 64)
	if err != nil {
		return 0
	}

	return id
}

// remember stores the person whose page is being shown.
func remember(w http.ResponseWriter, personID int64) {
	http.SetCookie(w, &http.Cookie{
		Name:     personCookie,
		Value:    strconv.FormatInt(personID, 10),
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   400 * 24 * 60 * 60,
	})
}

// home opens the person this device last used, on today. A device that has not
// been used yet, or whose person has since been archived, is asked who is
// ticking.
func (s *Server) home(w http.ResponseWriter, r *http.Request) {
	next := ""
	if r.URL.Query().Get("next") == "history" {
		next = "/history"
	}

	if id := s.rememberedPerson(r); id != 0 && r.URL.Query().Get("pick") == "" {
		if _, err := s.service.Store().Person(ctx(r), id); err == nil {
			http.Redirect(w, r, views.PersonHref(id)+next, http.StatusSeeOther)

			return
		}
	}

	strip, err := s.service.Strip(ctx(r))
	if err != nil {
		s.fail(w, r, err)

		return
	}

	page := s.page(w, r, "Who's ticking?", views.SectionToday, 0)
	page.PersonID = 0

	s.render(w, r, http.StatusOK, views.PickerPage(page, strip, next))
}

// dayPage renders one person's day: today at /p/{id}, another day at
// /p/{id}/{yyyy-mm-dd}. A future date, or today's own date, redirects to the
// plain URL so each day has exactly one address.
func (s *Server) dayPage(w http.ResponseWriter, r *http.Request) {
	personID, ok := s.personParam(w, r)
	if !ok {
		return
	}

	today := s.service.Today()
	day := today

	if raw := r.PathValue("date"); raw != "" {
		parsed, err := time.ParseInLocation("2006-01-02", raw, time.UTC)
		if err != nil {
			s.notFound(w, r)

			return
		}

		if !parsed.Before(today) {
			http.Redirect(w, r, views.PersonHref(personID), http.StatusSeeOther)

			return
		}

		day = parsed
	}

	view, err := s.service.Day(ctx(r), personID, day)
	if err != nil {
		s.fail(w, r, err)

		return
	}

	remember(w, personID)

	title := view.Person.Name
	if !view.IsToday() {
		title += " · " + views.ISODate(view.Day)
	}

	s.render(w, r, http.StatusOK, views.DayPage(s.page(w, r, title, views.SectionToday, personID), view))
}

// editPage renders a person's habits and goal for editing.
func (s *Server) editPage(w http.ResponseWriter, r *http.Request) {
	personID, ok := s.personParam(w, r)
	if !ok {
		return
	}

	view, err := s.service.Edit(ctx(r), personID)
	if err != nil {
		s.fail(w, r, err)

		return
	}

	remember(w, personID)

	s.render(w, r, http.StatusOK, views.EditPage(s.page(w, r, "Edit · "+view.Person.Name, views.SectionToday, personID), view))
}

// historyPage renders one month of a person's history; ?m=2026-08 picks the
// month, defaulting to this one.
func (s *Server) historyPage(w http.ResponseWriter, r *http.Request) {
	personID, ok := s.personParam(w, r)
	if !ok {
		return
	}

	month := s.service.Today()
	if raw := r.URL.Query().Get("m"); raw != "" {
		parsed, err := time.ParseInLocation("2006-01", raw, time.UTC)
		if err != nil {
			s.notFound(w, r)

			return
		}

		month = parsed
	}

	view, err := s.service.Month(ctx(r), personID, month)
	if err != nil {
		s.fail(w, r, err)

		return
	}

	remember(w, personID)

	page := s.page(w, r, view.Person.Name+" · "+view.Grid.Label(), views.SectionHistory, personID)

	s.render(w, r, http.StatusOK, views.HistoryPage(page, view))
}

// familyPage renders the family list.
func (s *Server) familyPage(w http.ResponseWriter, r *http.Request) {
	view, err := s.service.Family(ctx(r))
	if err != nil {
		s.fail(w, r, err)

		return
	}

	s.render(w, r, http.StatusOK, views.FamilyPage(s.page(w, r, "Family", views.SectionFamily, 0), view))
}

// personParam reads the {person} path value, answering 404 itself when it is
// not a number.
func (s *Server) personParam(w http.ResponseWriter, r *http.Request) (int64, bool) {
	id, err := pathID(r, "person")
	if err != nil {
		s.notFound(w, r)

		return 0, false
	}

	return id, true
}
