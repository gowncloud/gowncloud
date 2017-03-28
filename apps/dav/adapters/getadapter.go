package ocdavadapters

import (
	"net/http"
	"strings"

	log "github.com/Sirupsen/logrus"
	"github.com/gowncloud/gowncloud/core/identity"
)

// GetAdapter is the adapter for the GET method. It adds the correct header to the
// response so the file can be downloaded by the client
func GetAdapter(handler http.HandlerFunc, w http.ResponseWriter, r *http.Request) {

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

	fullname := r.URL.RequestURI()
	filename := fullname[strings.LastIndex(fullname, "/")+1:]

	w.Header().Set("Content-Disposition", "attachment; filename="+filename)

	handler.ServeHTTP(w, r)
}
