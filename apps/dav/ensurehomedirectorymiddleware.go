package dav

import (
	"net/http"

	log "github.com/Sirupsen/logrus"
	"github.com/gowncloud/gowncloud/apps/dav/adapters"
	"github.com/gowncloud/gowncloud/core/identity"
	db "github.com/gowncloud/gowncloud/database"
)

func ensureHomeDirectoryMiddleware(adapter ocdavadapters.Adapter, handler http.HandlerFunc, w http.ResponseWriter, r *http.Request) {
	username := identity.CurrentSession(r).Username

	exists, err := db.NodeExists(username)
	if err != nil {
		log.Error("Failed to check if user home folder exists")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if !exists {
		err = MakeUserHomeDirectory(username)
	}

	if err != nil {
		log.Error("Failed to make user home directory")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	adapter(handler, w, r)

}
