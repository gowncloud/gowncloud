package trash

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"strings"

	log "github.com/Sirupsen/logrus"
	"github.com/gowncloud/gowncloud/core/identity"
	db "github.com/gowncloud/gowncloud/database"
)

const TRASH_DIR = "/files_trash"
const FILES_DIR = "/files"

type response struct {
	Data   data   `json:"data"`
	Status string `json:"status"`
}

type data struct {
	Directory   string `json:"directory"`
	Files       []file `json:"files"`
	Permissions int    `json:"permission"`
}

type file struct {
	Etag        int    `json:"etag"`
	Id          int    `json:"id"`
	MimeType    string `json:"mimetype"`
	Mtime       int64  `json:"mtime"`
	Name        string `json:"name"`
	ParentId    *int   `json:"parentId"`
	Permissions int    `json:"permissions"`
	Size        int64  `json:"size"`
	Type        string `json:"type"`
}

// GetTrash is the endpoint for /index.php/apps/files_trashbin/ajax/list.php
// It returns info on all nodes in the trashdirectory with subdirectory "dir"
// as speciefied in the query parameter
func GetTrash(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	dir := q.Get("dir")
	dirParts := strings.Split(dir, "/")
	// UI seems to append some random .d[UNIXTIMESTRING] to the first "real" part
	// Because dir always has a leading slash, the first actual part (part 0) is empty
	if strings.Contains(dirParts[1], ".") {
		dirParts[1] = dirParts[1][:strings.LastIndex(dirParts[1], ".")]
	}
	dir = strings.Join(dirParts, "/")
	basePath := db.GetSetting(db.DAV_ROOT) + identity.CurrentSession(r).Username +
		TRASH_DIR + dir

	// ensure basePath ends with a '/' character
	if !strings.HasSuffix(dir, "/") {
		basePath += "/"
	}

	entries, err := ioutil.ReadDir(basePath)
	if err != nil {
		log.Errorf("Directory %v not found in trash: %v", dir, err)
		w.WriteHeader(http.StatusNotFound)
		return
	}

	files := make([]file, 0)

	for i, entry := range entries {
		nodePath := strings.TrimPrefix(basePath, db.GetSetting(db.DAV_ROOT))
		node, err := db.GetNode(nodePath + entry.Name())
		if err != nil {
			log.Errorf("Failed to get node %v in trash: %v", entry.Name(), err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		// If a node is found on disk but not in the database, log an error and
		// ommit the information on said node altogether.
		if node == nil {
			log.Error("Node not found in database - database out of sync")
			continue
		}
		fileInfo := file{
			Etag:        0,
			Id:          i,
			MimeType:    node.MimeType,
			Mtime:       entry.ModTime().Unix() * 1000,
			Name:        entry.Name(),
			ParentId:    nil,
			Permissions: 1,
			Size:        entry.Size(),
			Type:        "file",
		}
		if entry.IsDir() {
			fileInfo.Type = "dir"
		}
		files = append(files, fileInfo)
	}

	response := &response{
		Data: data{
			Directory:   dir,
			Files:       files,
			Permissions: 0,
		},
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
