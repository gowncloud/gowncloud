package trash

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"strings"

	log "github.com/Sirupsen/logrus"
	"github.com/gowncloud/gowncloud/core/identity"
	db "github.com/gowncloud/gowncloud/database"
)

type deleteResponse struct {
	Data   deleteData `json:"data"`
	Status string     `json:"status"`
}

type deleteData struct {
	Success []nodeResponse `json:"success"`
}

type nodeResponse struct {
	Filename  string `json:"filename"`
	Timestamp string `json:"timestamp"`
}

// DeleteTrash removes a node from the trashbin
func DeleteTrash(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	fd := r.Form
	allFiles := fd.Get("allfiles")
	dir := fd.Get("dir")
	rawfiles := fd.Get("files")
	rawfiles = strings.TrimSuffix(strings.TrimPrefix(rawfiles, "["), "]")

	files := strings.Split(rawfiles, ",")

	dirParts := strings.Split(dir, "/")
	// UI seems to append some random .d[UNIXTIMESTRING] to the first "real" part
	// Because dir always has a leading slash, the first actual part (part 0) is empty
	if strings.Contains(dirParts[1], ".") {
		dirParts[1] = dirParts[1][:strings.LastIndex(dirParts[1], ".")]
	}
	dir = strings.Join(dirParts, "/")

	basePath := identity.CurrentSession(r).Username + TRASH_DIR + dir

	// ensure basePath ends with a '/' character
	if !strings.HasSuffix(dir, "/") {
		basePath += "/"
	}

	if allFiles == "true" {
		entries, err := ioutil.ReadDir(db.GetSetting(db.DAV_ROOT) + basePath)
		if err != nil {
			log.Error("Failed to get contents of trash directory: ", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		for _, entry := range entries {
			files = append(files, entry.Name())
		}
	}

	filePaths := make([]string, 0)
	for _, file := range files {
		if file == "" {
			// Don't remove the base directory
			continue
		}
		file = strings.TrimPrefix(strings.TrimSuffix(file, "\""), "\"")
		if dir == "/" && allFiles != "true" {
			file = file[:strings.LastIndex(file, ".")]
		}
		filePaths = append(filePaths, basePath+file)
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
		err = os.RemoveAll(db.GetSetting(db.DAV_ROOT) + path)
		if err != nil {
			log.Error("Failed to remove node from trash: ", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		err = db.DeleteNode(path)
		if err != nil {
			log.Error("Failed to remove node form db: ", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		nodeResponses = append(nodeResponses, nodeResponse{
			// Make sure to remove quotes from the filename because it is quoted
			// when we take it from the form values
			Filename:  strings.TrimPrefix(strings.TrimSuffix(files[i], "\""), "\""),
			Timestamp: strconv.FormatInt(modTime.Unix(), 10),
		})
	}

	if allFiles == "true" {
		nodeResponses = make([]nodeResponse, 0)
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
