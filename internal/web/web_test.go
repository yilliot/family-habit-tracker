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
	"github.com/el/family-habit-tracker/internal/web/views"
)

// fixedNow is Monday 9 November 2026, mid-morning.
var fixedNow = time.Date(2026, 11, 9, 9, 30, 0, 0, time.UTC)

// newTestServer builds the full HTTP stack over a fresh database.
func newTestServer(t *testing.T) (*httptest.Server, *service.Service) {
	t.Helper()

	sqlDB, err := db.Open(t.Context(), filepath.Join(t.TempDir(), "web.sqlite"))
	if err != nil {
		t.Fatalf("open database: %v", err)
	}

	t.Cleanup(func() { sqlDB.Close() })

	dataStore := store.New(sqlDB).WithClock(func() time.Time { return fixedNow })
	svc := service.New(dataStore)

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	server := httptest.NewServer(web.NewServer(svc, logger).Routes())

	t.Cleanup(server.Close)

	return server, svc
}

// family seeds 可遇 with two habits, one of them ticked today, and a goal one
// point away.
func family(t *testing.T, svc *service.Service) (domain.Person, domain.Habit, domain.Habit) {
	t.Helper()

	ctx := t.Context()

	person, err := svc.CreatePerson(ctx, "可遇", nil)
	if err != nil {
		t.Fatalf("person: %v", err)
	}

	piano, err := svc.AddHabit(ctx, person.ID, "piano 20分钟")
	if err != nil {
		t.Fatalf("habit: %v", err)
	}

	story, err := svc.AddHabit(ctx, person.ID, "story 30分钟")
	if err != nil {
		t.Fatalf("habit: %v", err)
	}

	if _, err := svc.ToggleTick(ctx, piano.ID, fixedNow); err != nil {
		t.Fatalf("tick: %v", err)
	}

	if _, _, err := svc.SetGoal(ctx, person.ID, "Lego The pickaxes mine", 2); err != nil {
		t.Fatalf("goal: %v", err)
	}

	return person, piano, story
}

// noRedirects is a client that stops at the first response, so redirects and
// the cookies set on them can be asserted.
func noRedirects() *http.Client {
	return &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
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

// do sends a request without following redirects.
func do(t *testing.T, server *httptest.Server, method, path string, form url.Values, headers map[string]string) (*http.Response, string) {
	t.Helper()

	var reader io.Reader
	if form != nil {
		reader = strings.NewReader(form.Encode())
	}

	req, err := http.NewRequest(method, server.URL+path, reader)
	if err != nil {
		t.Fatalf("build request: %v", err)
	}

	if form != nil {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}

	for key, value := range headers {
		req.Header.Set(key, value)
	}

	resp, err := noRedirects().Do(req)
	if err != nil {
		t.Fatalf("%s %s: %v", method, path, err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	return resp, string(body)
}

func id(value int64) string { return strconv.FormatInt(value, 10) }

func TestEveryPageRenders(t *testing.T) {
	server, svc := newTestServer(t)
	person, _, _ := family(t, svc)
	p := "/p/" + id(person.ID)

	for _, path := range []string{"/?pick=1", p, p + "/2026-11-05", p + "/2026-08-12", p + "/edit", p + "/history", p + "/history?m=2026-10", "/family"} {
		status, body := get(t, server, path)

		if status != http.StatusOK {
			t.Errorf("GET %s = %d, want 200", path, status)
		}

		if !strings.Contains(body, "家庭习惯表") {
			t.Errorf("GET %s did not render the masthead", path)
		}
	}
}

func TestAPersonsDayShowsTheirHabitsPointsAndGoal(t *testing.T) {
	server, svc := newTestServer(t)
	person, _, _ := family(t, svc)

	_, body := get(t, server, "/p/"+id(person.ID))

	for _, want := range []string{"Monday 9 Nov", "piano 20分钟", "story 30分钟", "Lego The pickaxes mine", `class="ht-num">1<`, "1 of 2 today", `aria-pressed="true"`} {
		if !strings.Contains(body, want) {
			t.Errorf("day page is missing %q", want)
		}
	}
}

func TestAnEmptyAppSendsTheFamilyToAddPeople(t *testing.T) {
	server, _ := newTestServer(t)

	_, body := get(t, server, "/")

	if !strings.Contains(body, `href="/family"`) || !strings.Contains(body, "Add family") {
		t.Error("with nobody in the family, home should point at the Family page")
	}
}

func TestADeviceRemembersItsPerson(t *testing.T) {
	server, svc := newTestServer(t)
	person, _, _ := family(t, svc)

	resp, body := do(t, server, http.MethodGet, "/", nil, nil)
	if resp.StatusCode != http.StatusOK || !strings.Contains(body, "Who's ticking?") {
		t.Fatalf("a new device should be asked who is ticking (status %d)", resp.StatusCode)
	}

	resp, _ = do(t, server, http.MethodGet, "/p/"+id(person.ID), nil, nil)

	var remembered *http.Cookie
	for _, cookie := range resp.Cookies() {
		if cookie.Name == "ht_person" {
			remembered = cookie
		}
	}

	if remembered == nil || remembered.Value != id(person.ID) {
		t.Fatalf("opening a person's page should remember them, got %+v", remembered)
	}

	cookie := map[string]string{"Cookie": "ht_person=" + id(person.ID)}

	resp, _ = do(t, server, http.MethodGet, "/", nil, cookie)
	if resp.StatusCode != http.StatusSeeOther || resp.Header.Get("Location") != "/p/"+id(person.ID) {
		t.Errorf("home should open the remembered person's today, got %d → %q", resp.StatusCode, resp.Header.Get("Location"))
	}

	resp, _ = do(t, server, http.MethodGet, "/?next=history", nil, cookie)
	if resp.Header.Get("Location") != "/p/"+id(person.ID)+"/history" {
		t.Errorf("History from the nav should open their history, got %q", resp.Header.Get("Location"))
	}

	if err := svc.ArchivePerson(t.Context(), person.ID); err != nil {
		t.Fatalf("archive: %v", err)
	}

	if resp, _ = do(t, server, http.MethodGet, "/", nil, cookie); resp.StatusCode != http.StatusOK {
		t.Errorf("a remembered person who was archived should fall back to the picker, got %d", resp.StatusCode)
	}
}

func TestEachDayHasOneAddress(t *testing.T) {
	server, svc := newTestServer(t)
	person, _, _ := family(t, svc)
	p := "/p/" + id(person.ID)

	for _, path := range []string{p + "/2026-11-09", p + "/2026-12-25"} {
		resp, _ := do(t, server, http.MethodGet, path, nil, nil)
		if resp.StatusCode != http.StatusSeeOther || resp.Header.Get("Location") != p {
			t.Errorf("GET %s = %d → %q, want a redirect to today", path, resp.StatusCode, resp.Header.Get("Location"))
		}
	}

	for _, path := range []string{"/p/999", "/p/abc", p + "/not-a-date", p + "/history?m=nope"} {
		if status, _ := get(t, server, path); status != http.StatusNotFound {
			t.Errorf("GET %s = %d, want 404", path, status)
		}
	}
}

func TestSprintEraAddressesStillLandSomewhere(t *testing.T) {
	server, _ := newTestServer(t)

	for path, target := range map[string]string{"/settings": "/family", "/totals": "/family", "/history": "/", "/history/4": "/"} {
		resp, _ := do(t, server, http.MethodGet, path, nil, nil)
		if resp.StatusCode != http.StatusMovedPermanently || resp.Header.Get("Location") != target {
			t.Errorf("GET %s = %d → %q, want %s", path, resp.StatusCode, resp.Header.Get("Location"), target)
		}
	}
}

func TestTickingWithoutJavaScriptRedirectsBackToTheDay(t *testing.T) {
	server, svc := newTestServer(t)
	person, _, story := family(t, svc)

	resp, _ := do(t, server, http.MethodPost, "/ticks", url.Values{
		"habit_id":  {id(story.ID)},
		"tick_date": {"2026-11-09"},
	}, nil)

	if resp.StatusCode != http.StatusSeeOther || resp.Header.Get("Location") != "/p/"+id(person.ID) {
		t.Errorf("plain tick = %d → %q, want a redirect to the day", resp.StatusCode, resp.Header.Get("Location"))
	}

	if balance, _ := svc.Points().Balance(t.Context(), person.ID); balance != 0 {
		t.Errorf("balance = %d: two ticks should have bought the 2-point goal", balance)
	}
}

// From htmx a tick answers with the tapped row plus everything else it changed,
// out of band. Reaching the goal brings the celebration.
func TestHTMXTickReturnsTheRowTheTotalsAndTheCelebration(t *testing.T) {
	server, svc := newTestServer(t)
	_, _, story := family(t, svc)

	resp, body := do(t, server, http.MethodPost, "/ticks", url.Values{
		"habit_id":  {id(story.ID)},
		"tick_date": {"2026-11-09"},
	}, map[string]string{"HX-Request": "true"})

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("htmx tick returned %d, want 200", resp.StatusCode)
	}

	// Only the tapped row is swapped, so a tap on another row that is still
	// queued keeps its form; everything else the tap changed comes out of band.
	if !strings.HasPrefix(strings.TrimSpace(body), `<li class="ht-habit" id="row-`+id(story.ID)+`"`) {
		t.Error("the response should open with the tapped row, which the form targets")
	}

	if !strings.Contains(body, `hx-target="closest li"`) || strings.Count(body, `<li class="ht-habit"`) != 1 {
		t.Error("the response should carry exactly the one row, targeting itself")
	}

	for _, want := range []string{
		`id="day-foot" hx-swap-oob="true"`, `id="week" aria-label="This week" hx-swap-oob="true"`,
		`id="points" aria-label="Points" hx-swap-oob="true"`, `id="day-month" aria-label="This month" hx-swap-oob="true"`,
		`id="strip" aria-label="Family" hx-swap-oob="true"`, "2 of 2 today", "Goal hit", `id="celebration" hx-swap-oob="true"`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("htmx tick response is missing %q", want)
		}
	}

	_, body = do(t, server, http.MethodPost, "/ticks", url.Values{
		"habit_id":  {id(story.ID)},
		"tick_date": {"2026-11-09"},
	}, map[string]string{"HX-Request": "true"})

	if strings.Contains(body, "Goal hit") || !strings.Contains(body, `<div id="celebration" hx-swap-oob="true"></div>`) {
		t.Error("a tick that earns nothing should clear the celebration slot")
	}
}

// A rejected tick must not swap anything in; the page keeps the optimistic
// rollback and shows the reason.
func TestHTMXTickOutsideTheWindowIsRejected(t *testing.T) {
	server, svc := newTestServer(t)
	_, piano, _ := family(t, svc)

	resp, _ := do(t, server, http.MethodPost, "/ticks", url.Values{
		"habit_id":  {id(piano.ID)},
		"tick_date": {"2026-11-10"},
	}, map[string]string{"HX-Request": "true"})

	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Errorf("future tick returned %d, want 422", resp.StatusCode)
	}

	if trigger := resp.Header.Get("HX-Trigger"); !strings.Contains(trigger, "future") {
		t.Errorf("HX-Trigger = %q, want it to explain the rejection", trigger)
	}
}

func TestOldDaysRenderReadOnly(t *testing.T) {
	server, svc := newTestServer(t)
	person, _, _ := family(t, svc)

	_, body := get(t, server, "/p/"+id(person.ID)+"/2026-10-01")

	if !strings.Contains(body, "kept as it was") {
		t.Error("a day more than a week back should say it is history")
	}

	_, body = get(t, server, "/p/"+id(person.ID)+"/2026-11-08")
	if !strings.Contains(body, "Forgot a tick?") {
		t.Error("yesterday should invite a catch-up tick")
	}
}

func TestAHabitCanBeAddedRenamedMovedStoppedAndRestarted(t *testing.T) {
	server, svc := newTestServer(t)
	person, piano, _ := family(t, svc)
	p := "/p/" + id(person.ID)

	resp, _ := do(t, server, http.MethodPost, p+"/habits", url.Values{"name": {"Good Night"}}, map[string]string{"Referer": server.URL + p})
	if resp.StatusCode != http.StatusSeeOther || resp.Header.Get("Location") != p {
		t.Fatalf("add habit = %d → %q", resp.StatusCode, resp.Header.Get("Location"))
	}

	do(t, server, http.MethodPost, "/habits/"+id(piano.ID), url.Values{"name": {"piano 30分钟"}}, nil)
	do(t, server, http.MethodPost, "/habits/"+id(piano.ID)+"/move", url.Values{"dir": {"down"}}, nil)

	_, body := get(t, server, p)
	if strings.Index(body, "story 30分钟") > strings.Index(body, "piano 30分钟") {
		t.Error("piano should have been renamed and moved below story")
	}

	if !strings.Contains(body, "Good Night") {
		t.Error("the added habit should show today")
	}

	resp, _ = do(t, server, http.MethodPost, "/habits/"+id(piano.ID)+"/stop", url.Values{}, nil)
	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("stop = %d", resp.StatusCode)
	}

	_, body = get(t, server, p+"/edit")
	if !strings.Contains(body, "Stopped earlier") {
		t.Error("the edit page should list the stopped habit")
	}

	do(t, server, http.MethodPost, "/habits/"+id(piano.ID)+"/restart", url.Values{}, nil)

	if _, body = get(t, server, p+"/edit"); strings.Contains(body, "Stopped earlier") {
		t.Error("a restarted habit should leave the stopped list")
	}
}

func TestDeductingFromAPersonsPage(t *testing.T) {
	server, svc := newTestServer(t)
	person, _, _ := family(t, svc)
	p := "/p/" + id(person.ID)

	resp, _ := do(t, server, http.MethodPost, p+"/adjustments", url.Values{"points": {"3"}, "reason": {"zoo"}}, nil)
	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("deduct = %d", resp.StatusCode)
	}

	if balance, _ := svc.Points().Balance(t.Context(), person.ID); balance != -2 {
		t.Errorf("balance = %d, want 1 − 3 = −2", balance)
	}

	if resp, _ = do(t, server, http.MethodPost, p+"/adjustments", url.Values{"points": {"0"}}, nil); resp.StatusCode != http.StatusSeeOther {
		t.Errorf("a zero deduction should bounce back with a message, got %d", resp.StatusCode)
	}
}

func TestTheFamilyListCanChange(t *testing.T) {
	server, svc := newTestServer(t)
	person, _, _ := family(t, svc)

	do(t, server, http.MethodPost, "/people", url.Values{"name": {"奇乐"}}, nil)
	do(t, server, http.MethodPost, "/people/"+id(person.ID), url.Values{"name": {"可遇 Keyu"}}, nil)
	do(t, server, http.MethodPost, "/people/"+id(person.ID)+"/delete", url.Values{}, nil)

	_, body := get(t, server, "/family")
	if !strings.Contains(body, "奇乐") || !strings.Contains(body, "Bring back") {
		t.Error("the family page should show the new member and the archived one")
	}

	if status, _ := get(t, server, "/p/"+id(person.ID)); status != http.StatusNotFound {
		t.Errorf("an archived person's page = %d, want 404", status)
	}

	do(t, server, http.MethodPost, "/people/"+id(person.ID)+"/restore", url.Values{}, nil)

	if _, body = get(t, server, "/p/"+id(person.ID)); !strings.Contains(body, "可遇 Keyu") {
		t.Error("a restored person should be back, renamed")
	}
}

func TestValidationFailureFlashesAndRedirects(t *testing.T) {
	server, _ := newTestServer(t)

	resp, _ := do(t, server, http.MethodPost, "/people", url.Values{"name": {"   "}}, nil)

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

	resp, _ := do(t, server, http.MethodPost, "/people", url.Values{"name": {"妈妈"}}, map[string]string{"Referer": "https://evil.example//elsewhere"})

	if location := resp.Header.Get("Location"); strings.Contains(location, "evil.example") || strings.HasPrefix(location, "//") {
		t.Errorf("redirect followed an off-site referer: %q", location)
	}
}

func TestVersionedAssetsAreCachedForAYear(t *testing.T) {
	server, _ := newTestServer(t)

	for _, asset := range []string{"app.css", "app.js", "htmx.min.js", "favicon.svg"} {
		resp, body := do(t, server, http.MethodGet, "/static/"+asset+"?v="+views.AssetVersion, nil, nil)

		if resp.StatusCode != http.StatusOK || len(body) == 0 {
			t.Errorf("GET %s = %d with %d bytes", asset, resp.StatusCode, len(body))
		}

		if cache := resp.Header.Get("Cache-Control"); !strings.Contains(cache, "immutable") {
			t.Errorf("versioned %s Cache-Control = %q, want a year, immutable", asset, cache)
		}
	}

	if resp, _ := do(t, server, http.MethodGet, "/static/app.css", nil, nil); strings.Contains(resp.Header.Get("Cache-Control"), "immutable") {
		t.Error("an unversioned asset must not be cached forever")
	}

	if _, body := get(t, server, "/family"); !strings.Contains(body, "/static/app.css?v="+views.AssetVersion) {
		t.Error("pages should link the versioned stylesheet")
	}
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
