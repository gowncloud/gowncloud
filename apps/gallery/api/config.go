package gallery

import (
	"encoding/json"
	"net/http"
)

// Config returns a fake config for the UI gallery app
func Config(w http.ResponseWriter, r *http.Request) {
	response := struct {
		Features   []string `json:"features"`
		MediaTypes []string `json:"mediatypes"`
	}{
		Features: make([]string, 0),
		MediaTypes: []string{
			"image/png",
			"image/jpeg",
			"image/gif",
			"image/x-xbitmap",
			"image/bmp",
		},
	}
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(&response)
}
