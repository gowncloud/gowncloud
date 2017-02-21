package ocdavadapters

import (
	"net/http"
	"strings"

	log "github.com/Sirupsen/logrus"
	"github.com/gowncloud/gowncloud/core/identity"
	db "github.com/gowncloud/gowncloud/database"
)

func HeadAdapter(handler http.HandlerFunc, w http.ResponseWriter, r *http.Request) {

	log.Debug("Headadapter")

	// Apps use this url to ping the server
	if r.URL.Path == "/remote.php/webdav/" {
		log.Debug("Request is a server ping")
		w.WriteHeader(http.StatusOK)
		return
	}

	user := identity.CurrentSession(r).Username

	nodePath := strings.Replace(r.URL.Path, "/remote.php/webdav", user+"/files", 1)
	nodePath = strings.TrimSuffix(nodePath, "/")
	exists, err := db.NodeExists(nodePath)
	if err != nil {
		log.Error("Failed to check if node exists")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if !exists {
		log.Info("So the node does not exists")
		nodePath = strings.TrimPrefix(nodePath, user+"/files")
		nodePath = nodePath[strings.Index(nodePath, "/")+1:]
		if nodePath == "" {
			nodePath = user + "/files"
		}
		var sharedNodes []*db.Node
		sharedNodes, err = findShareRoot(nodePath, user)
		if err != nil {
			log.Error("Error while searching for shared nodes")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		log.Info("Shared nodes: ", len(sharedNodes))
		if len(sharedNodes) == 0 {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		// Log collisions
		if len(sharedNodes) > 1 {
			log.Warn("Shared folder collision")
		}

		target := sharedNodes[0]
		originalPath := r.URL.Path
		finalPath := target.Path[:strings.LastIndex(target.Path, "/")] + strings.TrimPrefix(originalPath, "/remote.php/webdav")
		r.URL.Path = "/remote.php/webdav/" + finalPath

	} else {

		r.URL.Path = strings.Replace(r.URL.Path,
			"/remote.php/webdav", "/remote.php/webdav/"+user+"/files",
			1)

	}

	handler.ServeHTTP(w, r)
}
