package ocdavadapters

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"

	log "github.com/Sirupsen/logrus"
	"github.com/gowncloud/gowncloud/core/identity"
	db "github.com/gowncloud/gowncloud/database"
)

// DeleteAdapter is the adapter for the WebDav DELETE method
func DeleteAdapter(handler http.HandlerFunc, w http.ResponseWriter, r *http.Request) {

	user := identity.CurrentSession(r).Username

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
			"/remote.php/webdav", "/remote.php/webdav/"+user+"/files",
			1)

	}

	rootPath := strings.TrimPrefix(r.URL.Path, "/remote.php/webdav/")
	exists, err = db.NodeExists(rootPath)
	if err != nil {
		log.Error("Couldn't verify if node exists: ", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if !exists {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	diskPath := db.GetSetting(db.DAV_ROOT) + rootPath

	err = filepath.Walk(diskPath, func(path string, _ os.FileInfo, err error) error {
		if err != nil {
			log.Debug("trashFile called with none-nil error")
			log.Info(err)
			return err
		}
		dbPath := strings.TrimPrefix(path, db.GetSetting(db.DAV_ROOT))
		node, err := db.GetNode(dbPath)
		if err != nil {
			log.Error("Error getting node: ", err)
			return err
		}
		if node == nil {
			log.Warn("Node found on disk but not in database: ", dbPath)
			return nil
		}
		// Save the original path in the trash node table
		_, err = db.CreateTrashNode(node.ID, node.Owner, node.Path, node.Isdir)
		if err != nil {
			log.Error("Could not create trash entry: ", err)
			return err
		}

		// check if the parent exists in the trash folder
		parentPath := dbPath
		parentPath = strings.TrimPrefix(parentPath, node.Owner+"/files")
		parentPath = parentPath[:strings.LastIndex(parentPath, "/")]
		trashPrefix := node.Owner + "/files_trash"
		pathPieces := strings.Split(parentPath, "/")

		exists, err := db.NodeExists(trashPrefix + "/" + parentPath)
		if err != nil {
			log.Errorf("Could not verify if node %v exists: %v", parentPath, err)
			return err
		}
		for !exists {
			// Remove the first piece of the path
			pathPieces = append(pathPieces[:0], pathPieces[1:]...)
			parentPath = strings.Join(pathPieces, "/")
			if parentPath != "" {
				parentPath = "/" + parentPath
			}

			exists, err = db.NodeExists(trashPrefix + parentPath)
			if err != nil {
				log.Errorf("Could not verify if node %v exists: %v", parentPath, err)
				return err
			}
		}

		parentPath = strings.Join(pathPieces, "/")

		if !strings.HasPrefix(parentPath, "/") && parentPath != "" {
			parentPath = "/" + parentPath
		}

		// FIXME: derive scheme from the original request
		destinationUrl := "http://" + r.Host + "/remote.php/webdav/" + trashPrefix + parentPath +
			"/" + node.Path[strings.LastIndex(node.Path, "/")+1:]

		// Patch the request to send to the webdav. Only set destination on the first node
		// This should be the root node, so this should be the target for the webdav
		if r.Header.Get("Destination") == "" {
			r.Header.Set("Destination", destinationUrl)
		}

		err = db.MoveNode(dbPath, trashPrefix+parentPath+"/"+node.Path[strings.LastIndex(node.Path, "/")+1:])
		if err != nil {
			log.Error("Could not update node location: ", err)
			log.Info("Original path: ", dbPath)
			log.Info("Target path: ", trashPrefix+parentPath+"/"+node.Path[strings.LastIndex(node.Path, "/")+1:])
			return err
		}

		return nil
	})

	if err != nil {
		log.Error("Failed to mark nodes as deleted: ", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// patch the request method to MOVE before sending it to webdav.
	r.Method = "MOVE"

	handler.ServeHTTP(w, r)

}
