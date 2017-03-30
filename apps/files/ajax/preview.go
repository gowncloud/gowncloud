package files

import (
	"net/http"
	"strings"

	log "github.com/Sirupsen/logrus"
	"github.com/gowncloud/gowncloud/core/identity"
	db "github.com/gowncloud/gowncloud/database"
	"github.com/gowncloud/gowncloud/image"
)

// GetPreview generates a preview for an image file and serves it to the client
func GetPreview(w http.ResponseWriter, r *http.Request) {
	username := identity.CurrentSession(r).Username
	groups := identity.CurrentSession(r).Organizations

	query := r.URL.Query()

	filePath := query.Get("file")
	if !strings.HasPrefix(filePath, "/") {
		filePath = "/" + filePath
	}

	nodePath := username + "/files" + filePath
	exists, err := db.NodeExists(nodePath)
	if err != nil {
		log.Error("Failed to check if node exists")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if !exists {
		nodePath = strings.TrimPrefix(nodePath, username+"/files")
		nodePath = nodePath[strings.Index(nodePath, "/")+1:]
		if nodePath == "" {
			nodePath = username + "/files"
		}
		var sharedNodes []*db.Node
		sharedNodes, err = findShareRoot(nodePath, username, groups)
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
		filePath = target.Path[:strings.LastIndex(target.Path, "/")] + filePath

	} else {

		filePath = username + "/files" + filePath
	}

	generatePreview(query.Get("x"), query.Get("y"), filePath, w)
}

// generatePreview generates an image preview from a file
func generatePreview(widthString, heightString, filePath string, w http.ResponseWriter) {
	path := db.GetSetting(db.DAV_ROOT) + filePath
	image.GeneratePreview(widthString, heightString, path, w)
}
