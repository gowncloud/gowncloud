package files_texteditor

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"os"
	"strings"

	log "github.com/Sirupsen/logrus"
	"github.com/gowncloud/gowncloud/core/identity"
	db "github.com/gowncloud/gowncloud/database"
)

const errFileTooLarge = "This file is too big to be opened. Please download the file instead."

type errMsg struct {
	Message string `json:"message"`
}

// LoadFile loads a text file and returns the contents. An error is returned if the file is
// too large
func LoadFile(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	id := identity.CurrentSession(r)

	dir := q.Get("dir")
	fileName := q.Get("filename")

	path := dir
	if !strings.HasSuffix(path, "/") {
		path += "/"
	}
	path += fileName

	nodePath, err := getNodePath(path, id)
	if err != nil {
		log.Errorf("Failed to get the node path: %v", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	if nodePath == "" {
		http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		return
	}

	filePath := db.GetSetting(db.DAV_ROOT) + nodePath
	fi, err := os.Stat(filePath)
	if err != nil {
		log.Errorf("Failed to get the node info (%v): %v", nodePath, err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	if fi.Size() > 4<<20 { //if size is bigger than 4MB
		msg := errMsg{
			Message: errFileTooLarge,
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(&msg)
		return
	}

	file, err := os.Open(filePath)
	if err != nil {
		log.Errorf("Failed to open the file (%v): %v", filePath, err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	defer file.Close()
	fileContent, err := ioutil.ReadAll(file)
	if err != nil {
		log.Errorf("Failed to read file content: %v", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	fiBody := struct {
		FileContents string `json:"filecontents"`
		Mime         string `json:"mime"`
		Mtime        int64  `json:"mtime"`
		Writeable    bool   `json:"writeable"`
	}{
		FileContents: string(fileContent),
		Mime:         "text/plain",
		Mtime:        fi.ModTime().Unix(),
		Writeable:    true,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(&fiBody)

}

// getNodePath finds a possible node for a user from a given web path
func getNodePath(path string, id identity.Session) (string, error) {
	username := id.Username
	groups := id.Organizations

	if path != "" && !strings.HasPrefix(path, "/") {
		path = "/" + path
	}

	nodePath := username + "/files" + path
	// Remove trailing slash when looking for directories
	nodePath = strings.TrimSuffix(nodePath, "/")
	var filePath string
	log.Debug("Looking for node at path ", nodePath)
	exists, err := db.NodeExists(nodePath)
	if err != nil {
		log.Error("Failed to check if node exists")
		return "", err
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
			return "", err
		}
		if len(sharedNodes) == 0 {
			return "", nil
		}
		// Log collisions
		if len(sharedNodes) > 1 {
			log.Warn("Shared folder collision")
		}

		target := sharedNodes[0]
		filePath = target.Path[:strings.LastIndex(target.Path, "/")] + path

	} else {

		filePath = nodePath
	}
	return filePath, nil
}

// findShareRoot parses a path and tries to find a share
func findShareRoot(href string, user string, groups []string) ([]*db.Node, error) {
	path := strings.TrimLeft(href, "/remote.php/webdav/")
	nodes, err := db.GetSharedNamedNodesToTargets(path, user, groups)
	if err != nil {
		return nil, err
	}
	if len(nodes) > 0 {
		return nodes, nil
	}
	seperatorIndex := strings.Index(path, "/")
	for len(nodes) == 0 && seperatorIndex >= 0 {
		path = path[:seperatorIndex]
		seperatorIndex = strings.Index(path, "/")
		nodes, err = db.GetSharedNamedNodesToTargets(path, user, groups)
		if err != nil {
			return nil, err
		}
		if len(nodes) > 0 {
			break
		}
	}
	return nodes, nil
}
