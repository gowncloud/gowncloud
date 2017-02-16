package files

import (
	"encoding/json"
	"net/http"
	"strings"

	log "github.com/Sirupsen/logrus"
	"github.com/gowncloud/gowncloud/core/identity"
	db "github.com/gowncloud/gowncloud/database"
)

const (
	FAVORITE_TAG      = "_$!<Favorite>!$_"
	FAVORITE_ENDPOINT = "/index.php/apps/files/api/v1/files"
)

type favoriteBody struct {
	Tags []string `json:"tags"`
}

// Favorite is the endpoint for POST /index.php/apps/files/api/v1/files/
// It identifies the file from the URL Path and marks it as favorite by the
// Authenticated user
func Favorite(w http.ResponseWriter, r *http.Request) {

	if r.Method != "POST" {
		log.Warn("'Favorite' called with wrong method: ", r.Method)
		w.WriteHeader(http.StatusMethodNotAllowed)
	}

	user := identity.CurrentSession(r).Username

	path := strings.Replace(r.URL.Path, FAVORITE_ENDPOINT, user+"/files", 1)
	exists, err := db.NodeExists(path)
	if err != nil {
		log.Error("Failed to check if node exists")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if !exists {
		path = strings.TrimPrefix(path, user+"/files")
		path = path[strings.Index(path, "/")+1:]
		if path == "" {
			path = user + "/files"
		}
		var sharedNodes []*db.Node
		sharedNodes, err = findShareRoot(path, user)
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
		path = target.Path[:strings.LastIndex(target.Path, "/")] + strings.TrimPrefix(originalPath, FAVORITE_ENDPOINT)

	}

	body := &favoriteBody{}
	json.NewDecoder(r.Body).Decode(body)

	err = db.RemoveNodeAsFavorite(path, user)
	if err != nil {
		log.Error("Could not unmark node as favorite: ", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	for _, tag := range body.Tags {
		if tag == FAVORITE_TAG {
			err := db.MarkNodeAsFavorite(path, user)
			if err != nil {
				log.Error("Could not mark node as favorite: ", err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(body)

}
