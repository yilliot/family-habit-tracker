package web_test

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/el/family-habit-tracker/internal/db"
	"github.com/el/family-habit-tracker/internal/domain"
	"github.com/el/family-habit-tracker/internal/service"
	"github.com/el/family-habit-tracker/internal/store"
	"github.com/el/family-habit-tracker/internal/web"
)

var fixedToday = time.Date(2026, 5, 1, 9, 30, 0, 0, time.UTC)

// newTestServer builds the full HTTP stack over a fresh database.
func newTestServer(t *testing.T) (*httptest.Server, *service.Service) {
	t.Helper()

	sqlDB, err := db.Open(t.Context(), filepath.Join(t.TempDir(), "web.sqlite"))
	if err != nil {
		t.Fatalf("open database: %v", err)
	}

	t.Cleanup(func() { sqlDB.Close() })

	dataStore := store.New(sqlDB).WithClock(func() time.Time { return fixedToday })
	svc := service.New(dataStore)

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	server := httptest.NewServer(web.NewServer(svc, logger).Routes())

	t.Cleanup(server.Close)

	return server, svc
}

// get fetches a page and returns its status and body.
func get(t *testing.T, server *httptest.Server, path string) (int, string) {
	t.Helper()

	resp, err := server.Client().Get(server.URL + path)
	if err != nil {
		t.Fatalf("GET %s: %v", path, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}

	return resp.StatusCode, string(body)
}

// post submits a form without following redirects, so the redirect itself can
// be asserted.
func post(t *testing.T, server *httptest.Server, path string, form url.Values, headers map[string]string) *http.Response {
	t.Helper()

	req, err := http.NewRequest(http.MethodPost, server.URL+path, strings.NewReader(form.Encode()))
	if err != nil {
		t.Fatalf("build request: %v", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	client := &http.Client{
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("POST %s: %v", path, err)
	}

	return resp
}

func TestEveryPageRenders(t *testing.T) {
	server, _ := newTestServer(t)

	paths := []string{"/", "/history", "/totals", "/settings", "/settings?tab=people", "/settings?tab=rewards", "/healthz"}

	for _, path := range paths {
		status, body := get(t, server, path)

		if status != http.StatusOK {
			t.Errorf("GET %s = %d, want 200", path, status)
		}

		if path != "/healthz" && !strings.Contains(body, "家庭习惯表") {
			t.Errorf("GET %s did not render the masthead", path)
		}
	}
}

func TestStaticAssetsAreServed(t *testing.T) {
	server, _ := newTestServer(t)

	for _, asset := range []string{"/static/app.css", "/static/app.js", "/static/htmx.min.js", "/static/favicon.svg"} {
		status, body := get(t, server, asset)

		if status != http.StatusOK {
			t.Errorf("GET %s = %d, want 200", asset, status)
		}

		if len(body) == 0 {
			t.Errorf("GET %s returned an empty body", asset)
		}
	}
}

func TestEmptyStateInvitesTheFirstSprint(t *testing.T) {
	server, _ := newTestServer(t)

	_, body := get(t, server, "/")

	if !strings.Contains(body, "Start your first sprint") {
		t.Error("an empty tracker should invite the family to start a sprint")
	}
}

// A full pass through the app as someone would actually use it: add the family,
// start a sprint, tick a box, and see the point land.
func TestFamilyCanRunASprintEndToEnd(t *testing.T) {
	server, svc := newTestServer(t)

	resp := post(t, server, "/people", url.Values{"name": {"可遇"}}, nil)
	resp.Body.Close()

	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("adding a person returned %d, want a redirect", resp.StatusCode)
	}

	people, err := svc.Store().People(t.Context())
	if err != nil || len(people) != 1 {
		t.Fatalf("expected one person, got %d (%v)", len(people), err)
	}

	personID := people[0].ID

	form := url.Values{
		"start_date":                      {"2026-04-19"},
		"end_date":                        {"2026-05-16"},
		"person_ids":                      {itoa64(personID)},
		"habits_" + itoa64(personID):      {"30分钟阅读时间\n整理书桌\n\n"},
		"reward_mode_" + itoa64(personID): {"new"},
		"reward_name_" + itoa64(personID): {"Lego Minecraft"},
		"reward_cost_" + itoa64(personID): {"3"},
	}

	resp = post(t, server, "/sprints", form, nil)
	resp.Body.Close()

	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("starting a sprint returned %d, want a redirect", resp.StatusCode)
	}

	sprint, err := svc.Store().ActiveSprint(t.Context())
	if err != nil || sprint == nil {
		t.Fatalf("expected an active sprint: %v", err)
	}

	grid, err := svc.BuildGrid(t.Context(), *sprint)
	if err != nil {
		t.Fatalf("build grid: %v", err)
	}

	if len(grid.Participants) != 1 {
		t.Fatalf("grid has %d participants, want 1", len(grid.Participants))
	}

	// The blank line in the habits box must not have become an empty habit.
	if got := len(grid.Participants[0].Habits); got != 2 {
		t.Fatalf("participant has %d habits, want 2", got)
	}

	habitID := grid.Participants[0].Habits[0].Habit.ID

	resp = post(t, server, "/ticks", url.Values{
		"habit_id":  {itoa64(habitID)},
		"tick_date": {"2026-04-20"},
	}, nil)
	resp.Body.Close()

	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("ticking returned %d, want a redirect", resp.StatusCode)
	}

	participant := grid.Participants[0].Participant

	balance, err := svc.Points().SprintBalance(t.Context(), *sprint, participant)
	if err != nil {
		t.Fatalf("balance: %v", err)
	}

	if balance != 1 {
		t.Errorf("balance after one tick = %d, want 1", balance)
	}

	// The tracker should now show the grid rather than the empty state.
	_, body := get(t, server, "/")

	if strings.Contains(body, "Start your first sprint") {
		t.Error("the tracker still shows the empty state after a sprint started")
	}

	if !strings.Contains(body, "Lego Minecraft") {
		t.Error("the reward should be printed on the grid")
	}
}

// htmx gets a fragment, not a redirect, so the cell can swap in place.
func TestHTMXTickReturnsAFragmentWithOutOfBandUpdates(t *testing.T) {
	server, svc := newTestServer(t)

	habitID, _ := seedSprint(t, svc)

	resp := post(t, server, "/ticks", url.Values{
		"habit_id":  {itoa64(habitID)},
		"tick_date": {"2026-04-20"},
	}, map[string]string{"HX-Request": "true"})
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("htmx tick returned %d, want 200", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	rendered := string(body)

	if !strings.Contains(rendered, `data-ticked="true"`) {
		t.Error("the returned cell should be ticked")
	}

	// The fragment is a whole <td>, so the form must swap its enclosing cell;
	// swapping the form itself nests a cell inside a cell and knocks the tick
	// off the grid.
	if !strings.HasPrefix(strings.TrimSpace(rendered), "<td") {
		t.Error("the returned fragment should be a whole day cell")
	}

	if !strings.Contains(rendered, `hx-target="closest td"`) {
		t.Error("the cell form should swap its enclosing td")
	}

	for _, fragment := range []string{`id="bal-`, `id="rw-`, `id="ps-`, `hx-swap-oob="true"`} {
		if !strings.Contains(rendered, fragment) {
			t.Errorf("response is missing the out-of-band fragment %q", fragment)
		}
	}
}

// Reaching a goal must deliver the celebration in the same response.
func TestHTMXTickDeliversTheCelebration(t *testing.T) {
	server, svc := newTestServer(t)

	habitID, _ := seedSprint(t, svc)

	resp := post(t, server, "/ticks", url.Values{
		"habit_id":  {itoa64(habitID)},
		"tick_date": {"2026-04-20"},
	}, map[string]string{"HX-Request": "true"})
	resp.Body.Close()

	resp = post(t, server, "/ticks", url.Values{
		"habit_id":  {itoa64(habitID)},
		"tick_date": {"2026-04-21"},
	}, map[string]string{"HX-Request": "true"})
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if !strings.Contains(string(body), "Goal hit") {
		t.Error("the tick that earns the goal should carry the celebration")
	}
}

// A rejected tick must not swap anything in; the page keeps the optimistic
// rollback and shows the reason.
func TestHTMXTickOnAFutureDayIsRejected(t *testing.T) {
	server, svc := newTestServer(t)

	habitID, _ := seedSprint(t, svc)

	resp := post(t, server, "/ticks", url.Values{
		"habit_id":  {itoa64(habitID)},
		"tick_date": {"2026-05-10"},
	}, map[string]string{"HX-Request": "true"})
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Errorf("future tick returned %d, want 422", resp.StatusCode)
	}

	if trigger := resp.Header.Get("HX-Trigger"); !strings.Contains(trigger, "future") {
		t.Errorf("HX-Trigger = %q, want it to explain the rejection", trigger)
	}
}

func TestValidationFailureFlashesAndRedirects(t *testing.T) {
	server, _ := newTestServer(t)

	resp := post(t, server, "/people", url.Values{"name": {"   "}}, nil)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("rejected form returned %d, want a redirect", resp.StatusCode)
	}

	var flashed bool
	for _, cookie := range resp.Cookies() {
		if cookie.Name == "ht_flash" && cookie.Value != "" {
			flashed = true
		}
	}

	if !flashed {
		t.Error("a rejected form should leave a flash message explaining why")
	}
}

// A form post carries a Referer; the redirect must not be steerable off-site
// through it.
func TestRedirectIgnoresAnOffsiteReferer(t *testing.T) {
	server, _ := newTestServer(t)

	resp := post(t, server, "/people", url.Values{"name": {"妈妈"}},
		map[string]string{"Referer": "https://evil.example.com/steal"})
	defer resp.Body.Close()

	location := resp.Header.Get("Location")

	if strings.Contains(location, "evil.example.com") {
		t.Errorf("Location = %q, want a same-origin path", location)
	}

	if !strings.HasPrefix(location, "/") {
		t.Errorf("Location = %q, want a relative path", location)
	}
}

func TestArchivedSprintIsReadOnly(t *testing.T) {
	server, svc := newTestServer(t)

	habitID, sprint := seedSprint(t, svc)

	if _, err := svc.EndSprint(t.Context(), sprint.ID); err != nil {
		t.Fatalf("end sprint: %v", err)
	}

	status, body := get(t, server, "/history/"+itoa64(sprint.ID))
	if status != http.StatusOK {
		t.Fatalf("history page = %d, want 200", status)
	}

	if !strings.Contains(body, `data-readonly="true"`) {
		t.Error("an archived grid should render its cells as read-only")
	}

	resp := post(t, server, "/ticks", url.Values{
		"habit_id":  {itoa64(habitID)},
		"tick_date": {"2026-04-20"},
	}, map[string]string{"HX-Request": "true"})
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Errorf("ticking an archived sprint returned %d, want 422", resp.StatusCode)
	}
}

// An active sprint has no history page; it belongs on the tracker where it can
// still be ticked.
func TestActiveSprintRedirectsAwayFromHistory(t *testing.T) {
	server, svc := newTestServer(t)

	_, sprint := seedSprint(t, svc)

	client := &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}

	resp, err := client.Get(server.URL + "/history/" + itoa64(sprint.ID))
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusSeeOther {
		t.Errorf("status = %d, want a redirect to the tracker", resp.StatusCode)
	}
}

func TestMissingSprintIsNotFound(t *testing.T) {
	server, _ := newTestServer(t)

	status, _ := get(t, server, "/history/9999")

	if status != http.StatusNotFound {
		t.Errorf("status = %d, want 404", status)
	}
}

// The settings tab lives in the URL, so a tab can be linked to and bookmarked.
func TestSettingsTabComesFromTheURL(t *testing.T) {
	server, _ := newTestServer(t)

	_, body := get(t, server, "/settings?tab=people")
	if !strings.Contains(body, "Add person") {
		t.Error("the people tab should render its own form")
	}

	_, body = get(t, server, "/settings?tab=rewards")
	if !strings.Contains(body, "Rewards") {
		t.Error("the rewards tab should render")
	}

	_, body = get(t, server, "/settings?tab=nonsense")
	if !strings.Contains(body, "sprint") {
		t.Error("an unknown tab should fall back to the sprint panel")
	}
}

// seedSprint sets up one person in a running sprint with a two-point goal,
// returning the habit to tick and the sprint.
func seedSprint(t *testing.T, svc *service.Service) (int64, domain.Sprint) {
	t.Helper()

	person, err := svc.CreatePerson(t.Context(), "可遇", nil)
	if err != nil {
		t.Fatalf("create person: %v", err)
	}

	sprint, err := svc.StartSprint(t.Context(),
		time.Date(2026, 4, 19, 0, 0, 0, 0, time.UTC),
		time.Date(2026, 5, 16, 0, 0, 0, 0, time.UTC),
		[]service.ParticipantInput{{
			PersonID: person.ID,
			Habits:   []service.HabitInput{{Name: "阅读", DisplayOrder: 1}},
			Reward:   service.RewardChoice{Mode: service.RewardNew, Name: "Lego Minecraft", Cost: 2},
		}},
	)
	if err != nil {
		t.Fatalf("start sprint: %v", err)
	}

	grid, err := svc.BuildGrid(t.Context(), sprint)
	if err != nil {
		t.Fatalf("build grid: %v", err)
	}

	return grid.Participants[0].Habits[0].Habit.ID, sprint
}

func itoa64(value int64) string {
	return strconv.FormatInt(value, 10)
}

// An uptime check must catch a dead database, not only a dead process.
func TestHealthCheckFailsWhenTheDatabaseIsUnavailable(t *testing.T) {
	sqlDB, err := db.Open(t.Context(), filepath.Join(t.TempDir(), "health.sqlite"))
	if err != nil {
		t.Fatalf("open database: %v", err)
	}

	svc := service.New(store.New(sqlDB))
	server := httptest.NewServer(web.NewServer(svc, slog.New(slog.NewTextHandler(io.Discard, nil))).Routes())
	defer server.Close()

	if status, _ := get(t, server, "/healthz"); status != http.StatusOK {
		t.Fatalf("healthy database: status = %d, want 200", status)
	}

	sqlDB.Close()

	if status, _ := get(t, server, "/healthz"); status != http.StatusServiceUnavailable {
		t.Errorf("closed database: status = %d, want 503", status)
	}
}
