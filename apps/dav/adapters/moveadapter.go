package ocdavadapters

import (
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	log "github.com/Sirupsen/logrus"
	"github.com/gowncloud/gowncloud/core/identity"
	db "github.com/gowncloud/gowncloud/database"
)

// MoveAdapter is the adapter for the MOVE method. It patches the request url and payload
// before it gets send to the internal webdav.
func MoveAdapter(handler http.HandlerFunc, w http.ResponseWriter, r *http.Request) {
	user := identity.CurrentSession(r).Username
	groups := identity.CurrentSession(r).Organizations

	destination := r.Header.Get("Destination")
	destinationUrl, err := url.Parse(destination)
	if err != nil {
		log.Debug("Could not parse destination: ", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// Don't move folders inside themselfs
	if destinationUrl.Path[:strings.LastIndex(destinationUrl.Path, "/")] == r.URL.Path {
		log.Debug("Trying to move node to the same location")
		http.Error(w, "The destination may not be part of the same subtree as the source path.", http.StatusConflict)
		return
	}

	nodePath := strings.Replace(r.URL.Path, "/remote.php/webdav", user+"/files", 1)
	destinationNodePath := strings.Replace(destinationUrl.Path, "/remote.php/webdav", user+"/files", 1)
	originExists, err := db.NodeExists(nodePath)
	if err != nil {
		log.Error("Failed to check if node exists")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if !originExists {
		nodePath = strings.TrimPrefix(nodePath, user+"/files")
		nodePath = nodePath[strings.Index(nodePath, "/")+1:]
		if nodePath == "" {
			nodePath = user + "/files"
		}

		var sharedNodes []*db.Node
		sharedNodes, err = findShareRoot(nodePath, append(groups, user))
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

	log.Debugf("check if %v exists", destinationNodePath[:strings.LastIndex(destinationNodePath, "/")])
	targetExists, err := db.NodeExists(destinationNodePath[:strings.LastIndex(destinationNodePath, "/")])
	if !targetExists {
		log.Debug("target node does not exist")
		destinationNodePath = strings.TrimLeft(destinationNodePath, "/files")
		destinationNodePath = destinationNodePath[strings.Index(destinationNodePath, "/")+1:]
		if destinationNodePath == "" {
			destinationNodePath = user + "/files"
		}

		var sharedNodes []*db.Node
		sharedNodes, err = findShareRoot(destinationNodePath, append(groups, user))
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
		log.Debug("found shares")

		target := sharedNodes[0]
		destinationPath := destinationUrl.Path
		destinationFinalPath := target.Path[:strings.LastIndex(target.Path, "/")] + strings.TrimPrefix(destinationPath, "/remote.php/webdav")
		destinationUrl.Path = "/remote.php/webdav/" + destinationFinalPath
		log.Debug("Destination path: ", destinationUrl.Path)
	} else {
		destinationUrl.Path = strings.Replace(destinationUrl.Path,
			"/remote.php/webdav", "/remote.php/webdav/"+user+"/files",
			1)
	}

	oldDbPath := strings.TrimPrefix(r.URL.Path, "/remote.php/webdav/")
	diskPath := db.GetSetting(db.DAV_ROOT) + oldDbPath

	newDbPath := strings.TrimPrefix(destinationUrl.Path, "/remote.php/webdav/")
	newOwner := newDbPath[:strings.Index(newDbPath, "/")]

	err = filepath.Walk(diskPath, func(path string, _ os.FileInfo, err error) error {
		if err != nil {
			log.Error("Move - walk called with non-nil error: ", err)
			return err
		}
		dbPath := strings.TrimPrefix(path, db.GetSetting(db.DAV_ROOT))
		target := strings.Replace(dbPath, oldDbPath, newDbPath, 1)

		node, err := db.GetNode(dbPath)
		if err != nil {
			log.Error("Error getting node: ", err)
			return err
		}
		if node == nil {
			log.Warn("Node found on disk but not in database: ", dbPath)
			return nil
		}

		err = db.TransferNode(dbPath, target, newOwner)
		if err != nil {
			log.Error("Could not update node location: ", err)
			log.Info("Original path: ", dbPath)
			log.Info("Target path: ", target)
			return err

		}

		return nil
	})

	if err != nil {
		log.Error("Failed to move nodes: ", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	destination = destinationUrl.String()

	r.Header.Set("Destination", destination)

	handler.ServeHTTP(w, r)
}
