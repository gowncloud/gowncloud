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
	user := identity.CurrentSession(r).Username
	groups := identity.CurrentSession(r).Organizations

	nodePath := strings.Replace(r.URL.Path, "/remote.php/webdav", user+"/files", 1)
	parentNodePath := nodePath[:strings.LastIndex(nodePath, "/")]
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
		sharedNodes, err = findShareRoot(parentNodePath, append(groups, user))
		if err != nil {
			log.Error("Error while searching for shared nodes")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		if len(sharedNodes) == 0 {
			log.Debug("No parent node, and no shared parent nodes found")
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

	// fullname := r.URL.RequestURI()
	path := strings.TrimPrefix(r.URL.Path, "/remote.php/webdav/")
	parentPath := path[:strings.LastIndex(path, "/")]
	exists, err = db.NodeExists(parentPath)
	if err != nil {
		log.Error("Failed to check if parent node exists")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if !exists {
		log.Debug("No node at ", parentPath)
		w.WriteHeader(http.StatusNotFound)
		return
	}

	// Since put replaces any existing file, just remove the node.
	err = db.DeleteNode(path)
	if err != nil {
		log.Error("Failed to remove old node")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// TODO: check if this is the right header
	contentType := r.Header.Get("Content-Type")

	_, err = db.SaveNode(path, path[:strings.Index(path, "/")], false, contentType)
	if err != nil {
		log.Error("Failed to save node in database")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	handler.ServeHTTP(w, r)
}
