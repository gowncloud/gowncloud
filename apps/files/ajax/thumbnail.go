package files

import (
	"net/http"
	"strings"

	log "github.com/Sirupsen/logrus"
	"github.com/gowncloud/gowncloud/core/identity"
	db "github.com/gowncloud/gowncloud/database"
)

// GetThumbnail renders a preview of image files. It seems to perform the same
// functionality of GetPreview, though it encodes the variables in the path
// instead of the query
func GetThumbnail(w http.ResponseWriter, r *http.Request) {

	fileName := strings.TrimPrefix(r.URL.Path, "/index.php/apps/files/api/v1/thumbnail/")
	// Store and remove width
	widthString := fileName[:strings.Index(fileName, "/")]
	fileName = fileName[strings.Index(fileName, "/")+1:]
	// Store and remove height
	heightString := fileName[:strings.Index(fileName, "/")]
	fileName = fileName[strings.Index(fileName, "/")+1:]

	username := identity.CurrentSession(r).Username

	nodePath := username + "/files/" + fileName
	filePath := nodePath
	filePath = strings.TrimPrefix(filePath, username+"/files")
	nodes, err := db.GetNodesForUserByName(fileName, username)
	if err != nil {
		log.Error("Failed to check if node exists")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	exists := len(nodes) > 0
	if !exists {
		nodePath = strings.TrimPrefix(nodePath, username+"/files")
		nodePath = nodePath[strings.Index(nodePath, "/")+1:]
		if nodePath == "" {
			nodePath = username + "/files"
		}
		var sharedNodes []*db.Node
		sharedNodes, err = findShareRoot(nodePath, username)
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

		filePath = nodes[0].Path
	}

	generatePreview(widthString, heightString, filePath, w)
}
