package web

import "encoding/json"

// jsonEvent encodes an htmx HX-Trigger payload: a single named event carrying
// a detail object.
func jsonEvent(name string, detail any) (string, error) {
	encoded, err := json.Marshal(map[string]any{name: detail})
	if err != nil {
		return "", err
	}

	return string(encoded), nil
}
