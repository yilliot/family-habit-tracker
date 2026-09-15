// Package web wires the HTTP surface: routing, request parsing, and rendering.
package web

import (
	"context"
	"embed"
	"errors"
	"io/fs"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/a-h/templ"
	"github.com/el/family-habit-tracker/internal/domain"
	"github.com/el/family-habit-tracker/internal/service"
	"github.com/el/family-habit-tracker/internal/store"
	"github.com/el/family-habit-tracker/internal/web/views"
)

//go:embed static
var staticFS embed.FS

// Server handles every request the application serves.
type Server struct {
	service *service.Service
	logger  *slog.Logger
}

// NewServer builds the HTTP handler set.
func NewServer(svc *service.Service, logger *slog.Logger) *Server {
	return &Server{service: svc, logger: logger}
}

// Routes returns the application's router.
//
// Mutations are POST-only and every destructive one has its own explicit path,
// so the whole app works from plain HTML forms with no JavaScript at all.
func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()

	assets, err := fs.Sub(staticFS, "static")
	if err != nil {
		panic("web: static assets missing: " + err.Error())
	}

	mux.Handle("GET /static/", http.StripPrefix("/static/", cacheControl(http.FileServerFS(assets))))
	mux.HandleFunc("GET /healthz", s.health)

	// Pages.
	mux.HandleFunc("GET /{$}", s.tracker)
	mux.HandleFunc("GET /history", s.historyIndex)
	mux.HandleFunc("GET /history/{sprint}", s.historyShow)
	mux.HandleFunc("GET /totals", s.totals)
	mux.HandleFunc("GET /settings", s.settings)

	// Ticking.
	mux.HandleFunc("POST /ticks", s.toggleTick)

	// Sprint lifecycle.
	mux.HandleFunc("POST /sprints", s.startSprint)
	mux.HandleFunc("POST /sprints/{sprint}/end", s.endSprint)
	mux.HandleFunc("POST /sprints/{sprint}/participants", s.addParticipant)

	// People.
	mux.HandleFunc("POST /people", s.createPerson)
	mux.HandleFunc("POST /people/{person}", s.updatePerson)
	mux.HandleFunc("POST /people/{person}/delete", s.archivePerson)

	// Habits.
	mux.HandleFunc("POST /participants/{participant}/habits", s.createHabit)
	mux.HandleFunc("POST /habits/{habit}", s.updateHabit)
	mux.HandleFunc("POST /habits/{habit}/delete", s.archiveHabit)

	// Rewards.
	mux.HandleFunc("POST /people/{person}/rewards", s.createReward)
	mux.HandleFunc("POST /rewards/{reward}", s.updateReward)
	mux.HandleFunc("POST /rewards/{reward}/delete", s.deleteReward)
	mux.HandleFunc("POST /participants/{participant}/active-reward", s.setActiveReward)

	// Points.
	mux.HandleFunc("POST /participants/{participant}/adjustments", s.createAdjustment)

	return s.recoverPanic(withTimeout(mux))
}

// requestTimeout bounds how long any one request may run.
//
// SQLite is reached through a single connection, so a request stuck waiting on
// it stalls every request queued behind it. Without a deadline that stall is
// permanent: the process is hung rather than crashed, so a supervisor never
// restarts it, and the tablet simply stops responding. With one, the stuck
// request fails, its transaction rolls back, the connection is released, and
// everything else carries on. It sits well inside the server's write timeout.
const requestTimeout = 15 * time.Second

// withTimeout attaches requestTimeout to every request's context, which the
// database layer honours when waiting for the connection.
func withTimeout(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), requestTimeout)
		defer cancel()

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// cacheControl lets the browser hold onto assets. The binary is redeployed as a
// unit, so a cache-busting query is unnecessary at this scale, but a modest TTL
// keeps a reload from refetching the font-heavy stylesheet.
func cacheControl(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "public, max-age=3600")
		next.ServeHTTP(w, r)
	})
}

// recoverPanic keeps one bad request from taking down the tablet's app.
func (s *Server) recoverPanic(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if recovered := recover(); recovered != nil {
				s.logger.Error("panic serving request", "path", r.URL.Path, "panic", recovered)
				http.Error(w, "Something went wrong.", http.StatusInternalServerError)
			}
		}()

		next.ServeHTTP(w, r)
	})
}

// health reports whether the app can actually serve the family, not merely
// whether the process is up. It round-trips the database through the shared
// connection, so an uptime check catches a wedged database as well as a dead
// process.
func (s *Server) health(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")

	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	if err := s.service.Store().Ping(ctx); err != nil {
		s.logger.Error("health check failed", "error", err)
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write([]byte("database unavailable\n"))

		return
	}

	w.Write([]byte("ok\n"))
}

// page assembles the shared page state: the masthead's sprint and any flash.
func (s *Server) page(w http.ResponseWriter, r *http.Request, title string) (views.Page, error) {
	page := views.Page{
		Title:   title,
		Path:    r.URL.Path,
		Flashes: takeFlash(w, r),
	}

	sprint, err := s.service.Store().ActiveSprint(r.Context())
	if err != nil {
		return views.Page{}, err
	}

	page.Sprint = sprint

	return page, nil
}

// render writes a component as a full HTML response.
func (s *Server) render(w http.ResponseWriter, r *http.Request, status int, component templ.Component) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)

	if err := component.Render(r.Context(), w); err != nil {
		// The status line is already sent, so the response cannot be rewritten;
		// log it so a broken template is visible rather than silent.
		s.logger.Error("render failed", "path", r.URL.Path, "error", err)
	}
}

// fail reports an unexpected error. Validation failures never reach here; they
// are surfaced to the user as a flash or a toast.
func (s *Server) fail(w http.ResponseWriter, r *http.Request, err error) {
	if errors.Is(err, store.ErrNotFound) {
		s.notFound(w, r)

		return
	}

	s.logger.Error("request failed", "method", r.Method, "path", r.URL.Path, "error", err)
	http.Error(w, "Something went wrong.", http.StatusInternalServerError)
}

// notFound renders the standard missing-page response.
func (s *Server) notFound(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "Not found.", http.StatusNotFound)
}

// isHTMX reports whether the request came from htmx and therefore wants a
// fragment rather than a whole page.
func isHTMX(r *http.Request) bool { return r.Header.Get("HX-Request") == "true" }

// pathID reads a numeric path parameter.
func pathID(r *http.Request, name string) (int64, error) {
	return strconv.ParseInt(r.PathValue(name), 10, 64)
}

// redirectBack returns the user to the page they submitted from, falling back
// to the tracker when there is no usable referer.
func (s *Server) redirectBack(w http.ResponseWriter, r *http.Request, fallback string) {
	target := fallback

	if referer := r.Header.Get("Referer"); referer != "" {
		// Only same-origin paths are followed, so a crafted Referer cannot turn
		// a form post into an open redirect.
		if path, ok := samePath(referer); ok {
			target = path
		}
	}

	http.Redirect(w, r, target, http.StatusSeeOther)
}

// samePath extracts the path and query of a same-origin referer.
func samePath(referer string) (string, bool) {
	if strings.HasPrefix(referer, "/") && !strings.HasPrefix(referer, "//") {
		return referer, true
	}

	for _, scheme := range []string{"http://", "https://"} {
		if !strings.HasPrefix(referer, scheme) {
			continue
		}

		rest := referer[len(scheme):]
		slash := strings.Index(rest, "/")
		if slash < 0 {
			return "/", true
		}

		return rest[slash:], true
	}

	return "", false
}

// reportError turns an error from the service layer into user-facing feedback:
// a toast for htmx requests, a flash plus redirect for ordinary form posts. It
// reports whether the error was handled here.
func (s *Server) reportError(w http.ResponseWriter, r *http.Request, err error, fallback string) bool {
	if err == nil {
		return false
	}

	validation, ok := service.AsValidation(err)
	if !ok {
		if errors.Is(err, store.ErrNotFound) {
			validation = domain.Invalid("record", "That item no longer exists.")
		} else {
			s.fail(w, r, err)

			return true
		}
	}

	message := validation.First()

	if isHTMX(r) {
		s.triggerToast(w, "error", message)
		w.WriteHeader(http.StatusUnprocessableEntity)

		return true
	}

	setFlash(w, "error", message)
	s.redirectBack(w, r, fallback)

	return true
}

// succeed reports a completed mutation the same way: a toast in place for htmx,
// a flash and redirect otherwise.
func (s *Server) succeed(w http.ResponseWriter, r *http.Request, message, fallback string) {
	setFlash(w, "success", message)
	s.redirectBack(w, r, fallback)
}

// triggerToast asks the page to show a toast via htmx's event mechanism.
func (s *Server) triggerToast(w http.ResponseWriter, kind, message string) {
	payload, err := jsonEvent("ht:toast", map[string]string{"type": kind, "message": message})
	if err != nil {
		return
	}

	w.Header().Set("HX-Trigger", payload)
}

// ctx is a short alias used by handlers.
func ctx(r *http.Request) context.Context { return r.Context() }
