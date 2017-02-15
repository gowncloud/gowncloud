package trash

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	log "github.com/Sirupsen/logrus"
	"github.com/gowncloud/gowncloud/core/identity"
	db "github.com/gowncloud/gowncloud/database"
)

// UndeleteTrash tries to restore nodes to their previous location before being deleted
// If the parent directory is no longer available, it will restore them to the
// root directory instead
func UndeleteTrash(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	fd := r.Form
	allFiles := fd.Get("allfiles")
	dir := fd.Get("dir")
	rawfiles := fd.Get("files")
	username := identity.CurrentSession(r).Username

	filePrefix := username + FILES_DIR

	filePaths, files, err := generateFileList(rawfiles, dir, username, allFiles)
	if err != nil {
		log.Error("Failed to generate file paths: ", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	nodeResponses := make([]nodeResponse, 0)

	for i, path := range filePaths {
		info, err := os.Stat(db.GetSetting(db.DAV_ROOT) + path)
		if err != nil {
			log.Errorf("Node %v not found in trash: %v", db.GetSetting(db.DAV_ROOT)+path, err)
			w.WriteHeader(http.StatusNotFound)
			return
		}
		modTime := info.ModTime()
		trashNode, err := db.GetTrashNode(path)
		if err != nil {
			log.Error("Could not get trash node: ", err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		// Check if the parent exists in the none trash nodes
		parentPath := trashNode.Path[:strings.LastIndex(trashNode.Path, "/")]
		parentPath = strings.Replace(parentPath, username+TRASH_DIR, filePrefix, 1)
		parentExists, err := db.NodeExists(parentPath)
		if err != nil {
			log.Error("Could not verify that parent path exists: ", err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		skippedPath := ""
		if !parentExists {
			skippedPath = strings.TrimPrefix(parentPath, username+FILES_DIR)
		}

		filepath.Walk(db.GetSetting(db.DAV_ROOT)+path, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				log.Debug("Walking with none nill error")
				return err
			}
			path = strings.TrimPrefix(path, db.GetSetting(db.DAV_ROOT))
			trashSubNode, err := db.GetTrashNode(path)
			if err != nil {
				log.Error("Could not get trash node: ", err)
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
				return err
			}
			restorePath := strings.Replace(trashSubNode.Path, skippedPath, "", 1)
			err = db.MoveNode(path, restorePath)
			if err != nil {
				log.Error("Failed to restore node in database: ", err)
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
				return err
			}
			err = db.DeleteTrashNode(trashSubNode.Path)
			if err != nil {
				log.Error("Could not delete trash node: ", err)
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
				return err
			}
			return nil
		})
		if err != nil {
			log.Error("Failed to restore node structure: ", err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		targetPath := db.GetSetting(db.DAV_ROOT) + trashNode.Path
		targetPath = strings.Replace(targetPath, skippedPath, "", 1)
		err = os.Rename(db.GetSetting(db.DAV_ROOT)+path, targetPath)
		if err != nil {
			log.Error("Failed to restore node: ", err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		nodeResponses = append(nodeResponses, nodeResponse{
			// Make sure to remove quotes from the filename because it is quoted
			// when we take it from the form values
			Filename:  strings.TrimPrefix(strings.TrimSuffix(files[i], "\""), "\""),
			Timestamp: strconv.FormatInt(modTime.Unix(), 10),
		})
	}

	resp := &deleteResponse{
		Data: deleteData{
			Success: nodeResponses,
		},
		Status: "success",
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)

}
