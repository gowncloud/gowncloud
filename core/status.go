package core

import (
	"encoding/json"
	"net/http"

	db "github.com/gowncloud/gowncloud/database"
)

type status struct {
	Installed     bool   `json:"installed"`
	Maintenance   bool   `json:"maintenance"`
	Version       string `json:"version"`
	VersionString string `json:"versionstring"`
	Edition       string `json:"edition"`
}

func Status(w http.ResponseWriter, r *http.Request) {
	version := db.GetSetting(db.VERSION)
	response := &status{
		Installed:     true,
		Maintenance:   false,
		Edition:       "",
		Version:       version,
		VersionString: version,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}
