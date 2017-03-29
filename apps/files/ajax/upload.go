package files

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/gowncloud/gowncloud/core/identity"
	db "github.com/gowncloud/gowncloud/database"
)

type UploadResponse struct {
	Directory         string  `json:"directory"`
	Etag              string  `json:"etag"`
	Id                float64 `json:"id"`
	MaxHumanFilesize  string  `json:"maxHumanFilesize"`
	Mimetype          string  `json:"mimetype"`
	Mtime             int64   `json:"mtime"`
	Name              string  `json:"name"`
	Originalname      string  `json:"originalname"`
	ParentId          float64 `json:"parentId"`
	Permissions       int     `json:"permissions"`
	Size              int     `json:"size"`
	Status            string  `json:"status"`
	Sort              string  `json:"type"`
	UploadMaxFilesize int     `json:"uploadMaxFilesize"`
}

// Upload uploads files to the server and stores data in the database
func Upload(w http.ResponseWriter, r *http.Request) {
	username := identity.CurrentSession(r).Username
	groups := identity.CurrentSession(r).Organizations
	log.Println("Current logged in user:", username)

	if r.Method != "POST" {
		log.Printf("Used the unsupported %v method", r.Method)
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	// TODO: is this required?
	err := r.ParseMultipartForm(1 << 29) // reserve 2^29 bytes = 536870912B / 512MB
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	fileDirectory := r.PostForm.Get("file_directory")
	if fileDirectory != "" {
		uploadDirectory(w, r)
		return
	}

	dir := r.PostForm.Get("dir")
	targetdir := username + "/files"
	if dir != "/" {
		targetdir += dir
	}
	log.Debug("target directory: ", targetdir)

	exists, err := db.NodeExists(targetdir)
	if err != nil {
		log.Error("Failed to check if node exists")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if !exists {
		nodePath := strings.TrimPrefix(targetdir, username+"/files")
		nodePath = nodePath[strings.Index(nodePath, "/")+1:]
		if nodePath == "" {
			nodePath = username + "/files"
		}
		var sharedNodes []*db.Node
		sharedNodes, err = findShareRoot(nodePath, username, groups)
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

		targetNode := sharedNodes[0]
		targetdir = targetNode.Path[:strings.LastIndex(targetNode.Path, "/")] + strings.TrimPrefix(targetdir, username+"/files")

		targetdir = db.GetSetting(db.DAV_ROOT) + targetdir

	} else {

		targetdir = db.GetSetting(db.DAV_ROOT) + targetdir

	}

	body := []UploadResponse{}

	for _, fileHeaders := range r.MultipartForm.File {
		for _, file := range fileHeaders {
			// Open the upload file
			upload, err := file.Open()
			if err != nil {
				log.Errorf("Failed to open upload file: %v", file.Filename)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			// Create the upload target
			target, err := os.Create(targetdir + "/" + file.Filename)
			if err != nil {
				log.Errorf("Failed to open target file: %v", targetdir+"/"+file.Filename)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			log.Debug("target file: ", target.Name())
			// Buffered copy
			written, err := io.Copy(target, upload)
			if err != nil {
				log.Error("Failed to copy upload file")
				// TODO: clean up target
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			log.Debugf("copied %v bytes", written)

			targetStats, err := target.Stat()
			if err != nil {
				log.Error("Failed to get stats")
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			dbFileName := strings.TrimPrefix(target.Name(), db.GetSetting(db.DAV_ROOT))
			node, err := db.SaveNode(dbFileName, dbFileName[:strings.Index(dbFileName, "/")], false, file.Header.Get("Content-Type"))
			if err != nil {
				log.Error("Failed to save node in database")
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			// Create the response
			data := UploadResponse{
				Directory:         dir,
				Etag:              "adfafdlasdfafdsaf", // TODO: send upload through webdav
				Id:                node.ID,
				MaxHumanFilesize:  "512MB",
				Mimetype:          file.Header.Get("Content-Type"),
				Mtime:             int64(time.Now().Unix()) * 1000, // the upload time aka Now
				Name:              file.Filename,
				Originalname:      file.Filename,
				ParentId:          2,
				Permissions:       31,
				Size:              int(targetStats.Size()),
				Status:            "success",
				Sort:              "file",
				UploadMaxFilesize: 1 << 29,
			}
			body = append(body, data)
		}
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(body)
}

// TODO: Merge with Upload
func uploadDirectory(w http.ResponseWriter, r *http.Request) {
	log.Debug("Uploading directory")
	username := identity.CurrentSession(r).Username
	groups := identity.CurrentSession(r).Organizations

	fileDirectory := strings.TrimSuffix(r.PostForm.Get("file_directory"), "/")
	dir := r.PostForm.Get("dir")
	fullDirectory := username + "/files"
	if dir != "/" {
		fullDirectory += dir
	}
	fullDirectory += "/" + fileDirectory

	log.Debug("target directory: ", fullDirectory)

	// small hack to fix uploads to user home directory
	exists := true
	var err error
	if dir != "/" {
		exists, err = parentExists(fullDirectory, username)
		if err != nil {
			log.Error("Failed to check if node exists")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	}

	if !exists {
		nodePath := strings.TrimPrefix(fullDirectory, username+"/files")
		nodePath = nodePath[strings.Index(nodePath, "/")+1:]
		if nodePath == "" {
			nodePath = username + "/files"
		}
		var sharedNodes []*db.Node
		sharedNodes, err = findShareRoot(nodePath, username, groups)
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

		targetNode := sharedNodes[0]
		fullDirectory = targetNode.Path[:strings.LastIndex(targetNode.Path, "/")] + strings.TrimPrefix(fullDirectory, username+"/files")

	}

	var nodesToCreate []string
	tmpDir := fullDirectory
	for {
		exists, err := db.NodeExists(tmpDir)
		if err != nil {
			log.Error("Failed to check if directory exists")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		if exists {
			break
		}
		nodesToCreate = append(nodesToCreate, tmpDir)
		tmpDir = tmpDir[:strings.LastIndex(tmpDir, "/")]
	}

	for i := len(nodesToCreate) - 1; i >= 0; i-- {
		nodePath := nodesToCreate[i]
		err := os.Mkdir(db.GetSetting(db.DAV_ROOT)+nodePath, os.ModePerm)
		if err != nil {
			log.Error("Failed to create directory")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		_, err = db.SaveNode(nodePath, nodePath[:strings.Index(nodePath, "/")], true, "httpd/unix-directory")
		if err != nil {
			log.Error("Failed to save directory info")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	}

	targetdir := db.GetSetting(db.DAV_ROOT) + fullDirectory
	log.Debug("target directory: ", targetdir)

	body := []UploadResponse{}

	for _, fileHeaders := range r.MultipartForm.File {
		for _, file := range fileHeaders {
			// Open the upload file
			upload, err := file.Open()
			if err != nil {
				log.Errorf("Failed to open upload file: %v", file.Filename)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			dbFileName := fullDirectory + "/" + file.Filename
			node, err := db.SaveNode(dbFileName, dbFileName[:strings.Index(dbFileName, "/")], false, file.Header.Get("Content-Type"))
			if err != nil {
				log.Error("Failed to save node in database")
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			// Create the upload target
			target, err := os.Create(targetdir + "/" + file.Filename)
			if err != nil {
				log.Errorf("Failed to open target file: %v", targetdir+"/"+file.Filename)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			log.Debug("target file: ", target.Name())
			// Buffered copy
			written, err := io.Copy(target, upload)
			if err != nil {
				log.Error("Failed to copy upload file")
				// TODO: clean up target
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			log.Debugf("copied %v bytes", written)

			targetStats, err := target.Stat()
			if err != nil {
				log.Error("Failed to get stats")
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			dirName := dir
			if dirName != "/" {
				dirName += "/"
			}
			dirName += fileDirectory
			// Create the response
			data := UploadResponse{
				Directory:         dirName,
				Etag:              "adfafdlasdfafdsaf", // TODO: send upload through webdav
				Id:                node.ID,
				MaxHumanFilesize:  "512MB",
				Mimetype:          file.Header.Get("Content-Type"),
				Mtime:             int64(time.Now().Unix()) * 1000, // the upload time aka Now
				Name:              file.Filename,
				Originalname:      file.Filename,
				ParentId:          2,
				Permissions:       31,
				Size:              int(targetStats.Size()),
				Status:            "success",
				Sort:              "file",
				UploadMaxFilesize: 1 << 29,
			}
			body = append(body, data)
		}
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(body)
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

// parentExists checks if the parent of a node exists
func parentExists(path string, username string) (bool, error) {
	exists, err := db.NodeExists(path)
	if err != nil {
		log.Error("Failed to check if node exists")
		return false, err
	}
	lastSeperatorIndex := strings.LastIndex(path, "/")
	for lastSeperatorIndex >= 0 && !exists {
		path = path[:lastSeperatorIndex]
		if path == username+"/files" {
			return false, nil
		}
		exists, err = db.NodeExists(path)
		if err != nil {
			log.Error("Failed to check if node exists")
			return false, err
		}
		lastSeperatorIndex = strings.LastIndex(path, "/")
		if exists && lastSeperatorIndex < 0 {
			// We found the user home directory. Set exists to false, callers should
			// already have checked if the upload is going to this directory
			exists = false
		}
	}
	return exists, nil
}
