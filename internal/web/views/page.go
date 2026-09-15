package views

import "github.com/el/family-habit-tracker/internal/domain"

// Flash is a one-shot message shown as a toast after a redirect.
type Flash struct {
	// Type is "success" or "error".
	Type    string
	Message string
}

// Page carries what every page needs: the document title, the path (so the nav
// can mark itself), the sprint the masthead prints, and any flash messages.
type Page struct {
	Title   string
	Path    string
	Sprint  *domain.Sprint
	Flashes []Flash
}

// NavItem is one entry in the primary navigation.
type NavItem struct {
	Label string
	Href  string
}

// navItems is the primary navigation, in the order the masthead prints it.
var navItems = []NavItem{
	{Label: "Tracker", Href: "/"},
	{Label: "History", Href: "/history"},
	{Label: "Totals", Href: "/totals"},
	{Label: "Settings", Href: "/settings"},
}

// isActive reports whether a nav entry matches the current path.
func (p Page) isActive(item NavItem) bool {
	if item.Href == "/" {
		return p.Path == "/"
	}

	return p.Path == item.Href || len(p.Path) > len(item.Href) && p.Path[:len(item.Href)+1] == item.Href+"/"
}
