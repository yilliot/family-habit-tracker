package web

import (
	"encoding/base64"
	"encoding/json"
	"net/http"

	"github.com/el/family-habit-tracker/internal/web/views"
)

// flashCookie carries one-shot messages across a redirect.
const flashCookie = "ht_flash"

// setFlash queues a message to be shown on the next page the browser loads.
func setFlash(w http.ResponseWriter, kind, message string) {
	payload, err := json.Marshal(views.Flash{Type: kind, Message: message})
	if err != nil {
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     flashCookie,
		Value:    base64.RawURLEncoding.EncodeToString(payload),
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   30,
	})
}

// takeFlash reads and immediately expires any queued message, so a flash is
// shown exactly once even if the page is reloaded.
func takeFlash(w http.ResponseWriter, r *http.Request) []views.Flash {
	cookie, err := r.Cookie(flashCookie)
	if err != nil || cookie.Value == "" {
		return nil
	}

	http.SetCookie(w, &http.Cookie{
		Name:     flashCookie,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	})

	raw, err := base64.RawURLEncoding.DecodeString(cookie.Value)
	if err != nil {
		return nil
	}

	var flash views.Flash
	if err := json.Unmarshal(raw, &flash); err != nil || flash.Message == "" {
		return nil
	}

	return []views.Flash{flash}
}
