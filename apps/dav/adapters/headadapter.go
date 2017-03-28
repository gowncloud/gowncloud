package ocdavadapters

import (
	"net/http"
	"strings"

	log "github.com/Sirupsen/logrus"
	"github.com/gowncloud/gowncloud/core/identity"
)

func HeadAdapter(handler http.HandlerFunc, w http.ResponseWriter, r *http.Request) {

	log.Debug("Headadapter")

	// Apps use this url to ping the server
	if r.URL.Path == "/remote.php/webdav/" {
		log.Debug("Request is a server ping")
		w.WriteHeader(http.StatusOK)
		return
	}

	id := identity.CurrentSession(r)

	inputPath := strings.TrimPrefix(r.URL.Path, "/remote.php/webdav/")
	path, err := getNodePath(inputPath, id)
	if err != nil {
		log.Errorf("Failed to get the node path (%v): %v", inputPath, err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	if path == "" {
		http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		return
	}

	r.URL.Path = "/remote.php/webdav/" + path

	handler.ServeHTTP(w, r)
}
