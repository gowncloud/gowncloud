package files_texteditor

import (
	"encoding/json"
	"net/http"
	"os"
	"strconv"

	log "github.com/Sirupsen/logrus"
	"github.com/gowncloud/gowncloud/core/identity"
	db "github.com/gowncloud/gowncloud/database"
)

func SaveFile(w http.ResponseWriter, r *http.Request) {
	id := identity.CurrentSession(r)

	err := r.ParseForm()
	if err != nil {
		log.Error("Failed to parse form", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	fileIn := struct {
		FileContents string `json:"filecontents"`
		Path         string `json:"path"`
		Mtime        int64  `json:"mtime"`
	}{}

	fileIn.FileContents = r.FormValue("filecontents")
	fileIn.Path = r.FormValue("path")
	fileIn.Mtime, err = strconv.ParseInt(r.FormValue("mtime"), 10, 64)
	if err != nil {
		log.Debug("Failed to decode mtime: ", err)
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	nodePath, err := getNodePath(fileIn.Path, id)
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
	file, err := os.OpenFile(filePath, os.O_WRONLY, os.ModeAppend)
	if err != nil {
		log.Errorf("Failed to open the file (%v): %v", filePath, err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	defer file.Close()
	_, err = file.Write([]byte(fileIn.FileContents))
	if err != nil {
		log.Errorf("Failed to write to the file (%v): %v", filePath, err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	fi, err := file.Stat()
	if err != nil {
		log.Errorf("Failed to get file info (%v): %v", filePath, err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	resp := struct {
		Mtime int64 `json:"mtime"`
		Size  int64 `json:"size"`
	}{
		Mtime: fi.ModTime().Unix(),
		Size:  fi.Size(),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(&resp)
}
