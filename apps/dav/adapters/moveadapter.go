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
	id := identity.CurrentSession(r)

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

	inputPath := strings.TrimPrefix(r.URL.Path, "/remote.php/webdav/")
	path, err := getNodePath(inputPath, id)
	if err != nil {
		log.Errorf("Failed to get the node path (%v): %v", inputPath, err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	if path == "" {
		http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		return
	}

	r.URL.Path = "/remote.php/webdav/" + path

	targetParentPath := strings.TrimPrefix(destinationUrl.Path, "/remote.php/webdav/")
	if strings.Contains(targetParentPath, "/") {
		targetParentPath = targetParentPath[:strings.LastIndex(targetParentPath, "/")]
	} else {
		targetParentPath = ""
	}

	log.Debugf("check if %v exists", targetParentPath)

	parentPath, err := getNodePath(targetParentPath, id)
	if err != nil {
		log.Errorf("Failed to get the node path (%v): %v", targetParentPath, err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	if parentPath == "" {
		http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		return
	}

	targetPath := parentPath + path[strings.LastIndex(path, "/"):]
	destinationUrl.Path = "/remote.php/webdav/" + targetPath

	exists, err := db.NodeExists(targetPath)
	if err != nil {
		log.Errorf("Failed to verify if node exists at path %v: %v", targetPath, err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	if exists {
		http.Error(w, http.StatusText(http.StatusConflict), http.StatusConflict)
		return
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
