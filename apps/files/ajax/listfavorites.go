package files

import (
	"encoding/json"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	log "github.com/Sirupsen/logrus"
	"github.com/gowncloud/gowncloud/core/identity"
	db "github.com/gowncloud/gowncloud/database"
)

type favoriteList struct {
	Files []file `json:"files"`
}

type file struct {
	Etag        string   `json:"etag"`
	Id          int      `json:"id"`
	MimeType    string   `json:"mimetype"`
	Mtime       int64    `json:"mtime"`
	Name        string   `json:"name"`
	ParentId    int      `json:"parentId"`
	ParentPath  string   `json:"path"`
	Permissions int      `json:"permissions"`
	Sharetypes  []int    `json:"shareTypes,omitempty"`
	Size        int64    `json:"size"`
	Tags        []string `json:"tags"`
	Type        string   `json:"type"`
}

// ListFavorits returns information about all favorites for a user
func ListFavorites(w http.ResponseWriter, r *http.Request) {
	id := identity.CurrentSession(r)

	nodes, err := db.GetFavoritedNodes(id.Username, append(id.Organizations, id.Username))
	if err != nil {
		log.Error("Failed to get favorited nodes: ", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	files := make([]file, 0)

	davroot := db.GetSetting(db.DAV_ROOT)
	for _, node := range nodes {
		fi, err := os.Stat(davroot + node.Path)
		if err != nil {
			log.Error("Failed to get node information: ", err)
			err = nil
			continue
		}

		parentNode, err := db.GetNode(node.Path[:strings.LastIndex(node.Path, "/")])
		if err != nil {
			log.Error("Failed to get parent node: ", err)
			err = nil
			continue
		}

		homeNode, err := db.GetNode(id.Username + "/files")
		if err != nil {
			log.Error("Failed to get home node: ", err)
			err = nil
			continue
		}

		permissions := 27
		if node.Isdir {
			permissions = 31
		}

		share, err := db.GetNodeShareToTarget(node.ID, id.Username)
		if err != nil {
			log.Error("Failed to check for share on node: ", err)
			err = nil
			continue
		}

		parentPath := strings.Replace(parentNode.Path, parentNode.Owner+"/files", "", 1)
		if parentPath == "" {
			parentPath = "/"
		}

		fileData := file{
			Etag:        strconv.Itoa(rand.Int()), // Fake for now
			Id:          node.ID,
			MimeType:    node.MimeType,
			Mtime:       fi.ModTime().Unix() * 1000,
			Name:        fi.Name(),
			ParentId:    homeNode.ID,
			ParentPath:  parentPath,
			Permissions: permissions,
			Size:        fi.Size(),
			Type:        "file",
		}

		if node.Isdir {
			fileData.Type = "dir"

			size, err := getDirSize(db.GetSetting(db.DAV_ROOT) + node.Path)
			if err != nil {
				log.Error("Failed to get directory size: ", err)
				err = nil
				continue
			}
			fileData.Size = size
		}

		if share != nil {
			shareTypes := make([]int, 1)
			fileData.Sharetypes = shareTypes
		}

		tags := make([]string, 0)
		tags = append(tags, FAVORITE_TAG)
		fileData.Tags = tags

		files = append(files, fileData)
	}

	response := &favoriteList{
		Files: files,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

func getDirSize(path string) (int64, error) {
	var space int64
	err := filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if !info.IsDir() {
			space += info.Size()
		}
		return err
	})
	return space, err
}
