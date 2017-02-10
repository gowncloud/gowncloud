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

	user := identity.CurrentSession(r).Username
	nodeOwner := user

	parentNodePath := strings.Replace(r.URL.Path, "/remote.php/webdav", user+"/files", 1)
	parentNodePath = parentNodePath[:strings.LastIndex(parentNodePath, "/")]
	exists, err := db.NodeExists(parentNodePath)
	if err != nil {
		log.Error("Failed to check if node exists")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if !exists {
		parentNodePath = strings.TrimPrefix(parentNodePath, user+"/files")
		parentNodePath = parentNodePath[strings.Index(parentNodePath, "/")+1:]
		if parentNodePath == "" {
			parentNodePath = user + "/files"
		}
		var sharedNodes []*db.Node
		sharedNodes, err = findShareRoot(parentNodePath, user)
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
		originalPath := r.URL.Path
		finalPath := target.Path[:strings.LastIndex(target.Path, "/")] + strings.TrimPrefix(originalPath, "/remote.php/webdav")
		r.URL.Path = "/remote.php/webdav/" + finalPath

		// The owner of the directory will be the original owner of the share
		nodeOwner = target.Owner

	} else {

		r.URL.Path = strings.Replace(r.URL.Path,
			"/remote.php/webdav", "/remote.php/webdav/"+user+"/files",
			1)

	}

	_, err = db.SaveNode(strings.Replace(r.URL.Path, "/remote.php/webdav/", "", 1), nodeOwner, true, "dir")
	if err != nil {
		log.Error("Failed to save node")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// TODO: use responsehijacker to intercept response and delete the node if an error occurred
	handler.ServeHTTP(w, r)
}
