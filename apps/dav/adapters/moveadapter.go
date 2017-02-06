package ocdavadapters

import (
	"net/http"
	"net/url"
	"strings"

	log "github.com/Sirupsen/logrus"
	"github.com/gowncloud/gowncloud/core/identity"
	db "github.com/gowncloud/gowncloud/database"
)

// MoveAdapter is the adapter for the MOVE method. It patches the request url and payload
// before it gets send to the internal webdav.
func MoveAdapter(handler http.HandlerFunc, w http.ResponseWriter, r *http.Request) {
	user := identity.CurrentSession(r).Username

	destination := r.Header.Get("Destination")
	destinationUrl, err := url.Parse(destination)
	if err != nil {
		log.Debug("Could not parse destination: ", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	nodePath := strings.Replace(r.URL.Path, "/remote.php/webdav", user, 1)
	destinationNodePath := strings.Replace(destinationUrl.Path, "/remote.php/webdav", user, 1)
	exists, err := db.NodeExists(nodePath)
	if err != nil {
		log.Error("Failed to check if node exists")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if !exists {
		nodePath = nodePath[strings.Index(nodePath, "/")+1:]
		destinationNodePath = destinationNodePath[strings.Index(destinationNodePath, "/")+1:]
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
		originalPath := r.URL.Path
		finalPath := target.Path[:strings.LastIndex(target.Path, "/")] + strings.TrimPrefix(originalPath, "/remote.php/webdav")
		r.URL.Path = "/remote.php/webdav/" + finalPath

		destinationPath := destinationUrl.Path
		destinationFinalPath := target.Path[:strings.LastIndex(target.Path, "/")] + strings.TrimPrefix(destinationPath, "/remote.php/webdav")
		destinationUrl.Path = "/remote.php/webdav/" + destinationFinalPath

	} else {

		r.URL.Path = strings.Replace(r.URL.Path,
			"/remote.php/webdav", "/remote.php/webdav/"+user,
			1)

		destinationUrl.Path = strings.Replace(destinationUrl.Path,
			"/remote.php/webdav", "/remote.php/webdav/"+user,
			1)

	}

	destination = destinationUrl.String()

	r.Header.Set("Destination", destination)

	db.MoveNode(strings.TrimPrefix(r.URL.Path, "/remote.php/webdav/"),
		strings.TrimPrefix(destinationUrl.Path, "/remote.php/webdav/"))

	handler.ServeHTTP(w, r)
}
