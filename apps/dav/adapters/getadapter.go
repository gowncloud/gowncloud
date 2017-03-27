package ocdavadapters

import (
	"net/http"
	"strings"

	log "github.com/Sirupsen/logrus"
	"github.com/gowncloud/gowncloud/core/identity"
	db "github.com/gowncloud/gowncloud/database"
)

// GetAdapter is the adapter for the GET method. It adds the correct header to the
// response so the file can be downloaded by the client
func GetAdapter(handler http.HandlerFunc, w http.ResponseWriter, r *http.Request) {

	user := identity.CurrentSession(r).Username
	groups := identity.CurrentSession(r).Organizations

	nodePath := strings.Replace(r.URL.Path, "/remote.php/webdav", user+"/files", 1)
	exists, err := db.NodeExists(nodePath)
	if err != nil {
		log.Error("Failed to check if node exists")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if !exists {
		nodePath = strings.TrimPrefix(nodePath, user+"/files")
		nodePath = nodePath[strings.Index(nodePath, "/")+1:]
		if nodePath == "" {
			nodePath = user + "/files"
		}
		var sharedNodes []*db.Node
		sharedNodes, err = findShareRoot(nodePath, user, groups)
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

	} else {

		r.URL.Path = strings.Replace(r.URL.Path,
			"/remote.php/webdav", "/remote.php/webdav/"+user+"/files",
			1)

	}

	fullname := r.URL.RequestURI()
	filename := fullname[strings.LastIndex(fullname, "/")+1:]

	w.Header().Set("Content-Disposition", "attachment; filename="+filename)

	handler.ServeHTTP(w, r)
}
