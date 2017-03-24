package files

import (
	"archive/zip"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/gowncloud/gowncloud/core/identity"
	db "github.com/gowncloud/gowncloud/database"
)

// Download serves the file or directory for downloading
func Download(w http.ResponseWriter, r *http.Request) {
	log.Debug("Starting download")
	query := r.URL.Query()

	rawFiles := strings.TrimSuffix(strings.TrimPrefix(query.Get("files"), "["), "]")
	fileList := strings.Split(rawFiles, ",")
	singleFile := len(fileList) == 1 && rawFiles != ""

	dls := query.Get("downloadStartSecret")
	setDownloadCookie(dls, w)

	dir := query.Get("dir")

	if rawFiles == "" && dir == "/" {
		serveHomeDir(w, r)
		return
	}

	files := make([]string, len(fileList))
	// If multiple files are specified the strings will contain quote characters
	if !singleFile && rawFiles != "" {
		for i, f := range fileList {
			file, err := strconv.Unquote(f)
			if err != nil {
				log.Errorf("Failed to unquote %v: %v", f, err)
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
				return
			}
			files[i] = file
		}
	}

	if singleFile {
		serveSingle(rawFiles, dir, w, r)
		return
	}
	serveMultiple(files, dir, w, r)
	return
}

// serveSingle serves a single file or directory for download
func serveSingle(file string, dir string, w http.ResponseWriter, r *http.Request) {
	log.Debug("Serving single file or directory for download")
	id := identity.CurrentSession(r)
	filePath, err := getNodePath(file, dir, id)
	if err != nil {
		log.Error("Failed to serve file or directory: ", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	if filePath == "" {
		log.Debug("The requested file could not be served - not found")
		http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		return
	}
	node, err := db.GetNode(filePath)
	if err != nil {
		log.Error("Failed to get node from database: ", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	if node == nil {
		log.Warn("The file path was found but no node exists in the database at ", filePath)
		http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		return
	}
	if node.Isdir {
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%v.zip\"", file))
		w.Header().Set("Content-Type", "application/zip")
		w.WriteHeader(http.StatusOK)
		zipper := zip.NewWriter(w)
		defer zipper.Close()
		err = serveDir(db.GetSetting(db.DAV_ROOT)+filePath, zipper)
		if err != nil {
			log.Error("Failed to serve file or directory: ", err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}
		return
	}
	w.Header().Set("Content-Disposition", "attachment; filename="+file)
	w.WriteHeader(http.StatusOK)
	http.ServeFile(w, r, db.GetSetting(db.DAV_ROOT)+filePath)
	return
}

// serveMultiple serves multiple files or directories for download
func serveMultiple(files []string, dir string, w http.ResponseWriter, r *http.Request) {
	log.Debug("Serving multiple files or directories for download")
	w.Header().Set("Content-Disposition", "attachment; filename=\"download.zip\"")
	w.Header().Set("Content-Type", "application/zip")
	w.WriteHeader(http.StatusOK)
	id := identity.CurrentSession(r)
	zipper := zip.NewWriter(w)
	defer zipper.Close()
	for _, file := range files {
		filePath, err := getNodePath(file, dir, id)
		if err != nil {
			log.Error("Failed to serve file or directory: ", err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
		if filePath == "" {
			log.Debug("The requested file could not be served - not found")
			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
			return
		}
		// No need to retrieve the node from the database as we don't need any info from it
		err = serveDir(db.GetSetting(db.DAV_ROOT)+filePath, zipper)
		if err != nil {
			log.Error("Error while writing zip file: ", err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
	}
}

// serveHomeDir serves all the users files and directories and all of the incomming shares
// If an error occurs while loading the shares, it is logged, but the zip file will
// still be served and the request is considered successfull if no further errors occur
func serveHomeDir(w http.ResponseWriter, r *http.Request) {
	log.Debug("Serving all files and shares from user home directory")
	id := identity.CurrentSession(r)
	davroot := db.GetSetting(db.DAV_ROOT)
	dirPath := id.Username + "/files"
	info, err := ioutil.ReadDir(davroot + dirPath)
	if err != nil {
		log.Error("Failed to read directory content: ", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	files := make([]string, 0)
	for _, fi := range info {
		log.Warn(fi.Name())
		files = append(files, dirPath+"/"+fi.Name())
	}

	// Load the shares
	shares, err := db.GetAllSharesToUser(id.Username, id.Organizations)
	if err != nil {
		log.Error("Failed to get shares: ", err)
		// Set the shares as an empty list so the next loop skips the shares entirely
		shares = make([]*db.Share, 0)
	}
	for _, share := range shares {
		var sharedNode *db.Node
		sharedNode, err = db.GetSharedNode(share.ShareID)
		if err != nil {
			log.Error("Failed to get node from share: ", err)
			continue
		}
		alreadyFound := false
		for _, path := range files {
			if sharedNode.Path == path {
				alreadyFound = true
				break
			}
		}
		if alreadyFound {
			continue
		}
		files = append(files, sharedNode.Path)
	}

	// finally, serve all the files and directories
	zipper := zip.NewWriter(w)
	defer zipper.Close()
	for _, path := range files {
		err = serveDir(davroot+path, zipper)
		if err != nil {
			log.Error("Error while writing zip file: ", err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
	}
}

// serveDir walks the directory tree and serves all the files. If the path points to a file,
// only said file is served
func serveDir(dirPath string, zipper *zip.Writer) error {
	var total int64
	return filepath.Walk(dirPath, func(path string, fi os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if fi.IsDir() {
			// skip directories
			return nil
		}
		zw, err := zipper.Create(strings.TrimPrefix(path, dirPath[:strings.LastIndex(dirPath, "/")+1]))
		if err != nil {
			return err
		}
		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()
		written, err := io.Copy(zw, file)
		if err != nil {
			return err
		}
		total += written
		log.Debugf("Written %v bytes to zip file, %v bytes total", written, total)
		return nil
	})
}

// getNodePath finds a possible node for a user from a given web path
func getNodePath(webPath string, dir string, id identity.Session) (string, error) {
	username := id.Username
	groups := id.Organizations

	if !strings.HasSuffix(dir, "/") {
		dir += "/"
	}

	nodePath := username + "/files" + dir + webPath
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
		sharedNodes, err = findShareRoot(nodePath, append(groups, username))
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
		filePath = target.Path[:strings.LastIndex(target.Path, "/")] + "/" + webPath

	} else {

		filePath = nodePath
	}
	return filePath, nil
}

// setDownloadCookie sets a cookie to be read by the javascript so it knows when the
// download has started to update the UI
func setDownloadCookie(code string, w http.ResponseWriter) {
	cookie := http.Cookie{
		Name:    "ocDownloadStarted",
		Value:   code,
		Path:    "/",
		Expires: time.Now().Add(time.Second * 20),
	}
	http.SetCookie(w, &cookie)
}
