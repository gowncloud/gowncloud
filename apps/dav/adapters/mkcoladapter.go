package ocdavadapters

import (
	"net/http"
	"strings"

	log "github.com/Sirupsen/logrus"
	"github.com/gowncloud/gowncloud/core/identity"
	db "github.com/gowncloud/gowncloud/database"
)

// MkcolAdapter is the adapter for the WebDav MKCOL method
func MkcolAdapter(handler http.HandlerFunc, w http.ResponseWriter, r *http.Request) {

	id := identity.CurrentSession(r)

	inputPath := strings.TrimPrefix(r.URL.Path, "/remote.php/webdav/")
	path, err := getNodePath(inputPath, id)
	if err != nil {
		log.Errorf("Failed to get the node path (%v): %v", inputPath, err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	if path == "" {
		// if path is empty we are sure we're not in a shared node, but the parent may still exist
		var parentPath string
		parentInputPath := ""
		if strings.Contains(inputPath, "/") {
			parentInputPath = inputPath[:strings.LastIndex(inputPath, "/")]
		}
		parentPath, err = getNodePath(parentInputPath, id)
		if err != nil {
			log.Errorf("Failed to get the node path (%v): %v", parentInputPath, err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
		if parentPath == "" {
			// Try to use MKCOL to create node who's parent does not yet exists
			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
			return
		}
		// take the target node from the url - so we don't have to add an extra check
		// when we make a node in the user root directory
		path = parentPath + r.URL.Path[strings.LastIndex(r.URL.Path, "/"):]
	} else {
		// path wa found, either we are in a shared node or the target node already exists
		var exists bool
		exists, err = db.NodeExists(path)
		if err != nil {
			log.Error("Failed to check if node already exists: ", err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
		if exists {
			log.Debug("Trying to use MKCOL to create duplicate node")
			http.Error(w, http.StatusText(http.StatusConflict), http.StatusConflict)
			return
		}

		var parentExists bool
		parentExists, err = db.NodeExists(path[:strings.LastIndex(path, "/")])
		if err != nil {
			log.Error("Failed to check if parent node already exists: ", err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
		if !parentExists {
			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
			return
		}
	}

	nodeOwner := path[:strings.Index(path, "/")]
	r.URL.Path = "/remote.php/webdav/" + path

	_, err = db.SaveNode(path, nodeOwner, true, "httpd/unix-directory")
	if err != nil {
		log.Error("Failed to save node: ", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	handler.ServeHTTP(w, r)
}
