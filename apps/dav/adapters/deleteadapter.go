package ocdavadapters

import (
	"net/http"
	"strings"

	log "github.com/Sirupsen/logrus"
	"github.com/gowncloud/gowncloud/core/identity"
	db "github.com/gowncloud/gowncloud/database"
)

// DeleteAdapter is the adapter for the WebDav DELETE method
func DeleteAdapter(handler http.HandlerFunc, w http.ResponseWriter, r *http.Request) {

	user := identity.CurrentSession(r).Username

	nodePath := strings.Replace(r.URL.Path, "/remote.php/webdav", user, 1)
	exists, err := db.NodeExists(nodePath)
	if err != nil {
		log.Error("Failed to check if node exists")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if !exists {
		nodePath = nodePath[strings.Index(nodePath, "/")+1:]
		var sharedNodes []*db.Node
		sharedNodes, err = findShareRoot(nodePath, user)
		if err != nil {
			log.Error("Error while searching for shared nodes")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		if len(sharedNodes) == 0 {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		// Log collisions
		if len(sharedNodes) > 1 {
			log.Warn("Shared folder collision")
		}

		target := sharedNodes[0]

		if strings.HasSuffix(target.Path, nodePath) {
			// This is the shared node, it should just be unshared and not deleted
			err = db.DeleteNodeShareToUserFromNodeId(target.ID, user)
			if err != nil {
				log.Error("Error deleting shared node")
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			w.WriteHeader(http.StatusNoContent)
			return
		}

		originalPath := r.URL.Path
		finalPath := target.Path[:strings.LastIndex(target.Path, "/")] + strings.TrimPrefix(originalPath, "/remote.php/webdav")
		r.URL.Path = "/remote.php/webdav/" + finalPath

	} else {

		r.URL.Path = strings.Replace(r.URL.Path,
			"/remote.php/webdav", "/remote.php/webdav/"+user,
			1)

	}

	err = db.DeleteNode(strings.Replace(r.URL.Path, "/remote.php/webdav/", "", 1))
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	handler.ServeHTTP(w, r)
}
