package views

// Flash is a one-shot message shown as a toast after a redirect.
type Flash struct {
	// Type is "success" or "error".
	Type    string
	Message string
}

// Sections of the app, which the masthead's nav marks.
const (
	SectionToday   = "today"
	SectionHistory = "history"
	SectionFamily  = "family"
)

// Page carries what every page needs: the document title, which section the
// nav marks, whose pages the nav links point at, and any flash messages.
type Page struct {
	Title   string
	Section string
	// PersonID is the person the nav's Today and History links open; zero
	// sends them to the "Who's ticking?" picker.
	PersonID int64
	Flashes  []Flash
}

// AssetVersion is appended to static asset URLs so a deploy changes them, which
// lets the browser cache them for a year. The web package sets it at start-up
// from a hash of the embedded files.
var AssetVersion = "dev"

// asset returns a static file's versioned URL.
func asset(name string) string { return "/static/" + name + "?v=" + AssetVersion }

// todayHref is where the nav's Today link goes.
func (p Page) todayHref() string {
	if p.PersonID == 0 {
		return "/"
	}

	return PersonHref(p.PersonID)
}

// historyHref is where the nav's History link goes.
func (p Page) historyHref() string {
	if p.PersonID == 0 {
		return "/?next=history"
	}

	return PersonHref(p.PersonID) + "/history"
}

// documentTitle composes the browser title, keeping the app name as a suffix.
func (p Page) documentTitle() string {
	if p.Title == "" {
		return "家庭习惯表 · Family Habit Tracker"
	}

	return p.Title + " · 家庭习惯表"
}
