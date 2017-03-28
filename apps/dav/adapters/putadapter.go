package ocdavadapters

import (
	"net/http"
	"strings"

	log "github.com/Sirupsen/logrus"
	"github.com/gowncloud/gowncloud/core/identity"
	db "github.com/gowncloud/gowncloud/database"
)

// PutAdapter saves the uploaded node in the database, then pass it on to store it on disk
func PutAdapter(handler http.HandlerFunc, w http.ResponseWriter, r *http.Request) {
	id := identity.CurrentSession(r)

	inputPath := strings.TrimPrefix(r.URL.Path, "/remote.php/webdav/")
	parentNodePath := inputPath[:strings.LastIndex(inputPath, "/")]
	path, err := getNodePath(parentNodePath, id)
	if err != nil {
		log.Error("Failed to get node path for url ", strings.TrimPrefix(r.URL.Path, "/remote.php/webdav/"))
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	if path == "" {
		http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		return
	}

	path = path + "/" + inputPath[strings.LastIndex(inputPath, "/")+1:]
	r.URL.Path = "/remote.php/webdav/" + path

	// Since put replaces any existing file, just remove the node.
	err = db.DeleteNode(path)
	if err != nil {
		log.Error("Failed to remove old node")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	contentType := r.Header.Get("Content-Type")

	_, err = db.SaveNode(path, path[:strings.Index(path, "/")], false, contentType)
	if err != nil {
		log.Error("Failed to save node in database")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	handler.ServeHTTP(w, r)
}
