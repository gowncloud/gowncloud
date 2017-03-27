package search

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"

	log "github.com/Sirupsen/logrus"
	"github.com/gowncloud/gowncloud/core/identity"
	db "github.com/gowncloud/gowncloud/database"
)

// SearchResult contains information about a single result to be returned by a search
type SearchResult struct {
	Id          string `json:"id"`
	Link        string `json:"link"`
	Mime        string `json:"mime"`
	MimeType    string `json:"mime_type"`
	Modified    string `json:"modified"`
	Name        string `json:"name"`
	Path        string `json:"path"`
	Permissions string `json:"permissions"`
	Size        string `json:"size"`
	Type        string `json:"type"`
}

// Search looks for all nodes the user has access to and gathers info about them
func Search(w http.ResponseWriter, r *http.Request) {
	id := identity.CurrentSession(r)
	q := r.URL.Query()

	query := q.Get("query")
	log.Debug("Looking for nodes with query ", query)
	if strings.Contains(query, "/") {
		response := []SearchResult{}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(&response)
		return
	}

	nodes, err := db.SearchNodesByName(query, id.Username, id.Organizations)
	if err != nil {
		log.Error("Failed to search for nodes: ", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	response := make([]SearchResult, len(nodes))
	for i, node := range nodes {
		isShared := node.Path[:strings.Index(node.Path, "/")] == id.Username
		nodeInfo, err := os.Stat(db.GetSetting(db.DAV_ROOT) + node.Path)
		if err != nil {
			log.Error("Could not get node info: ", err)
			continue
		}
		var shareNode *db.Share
		if isShared {
			for _, target := range append(id.Organizations, id.Username) {
				shareNode, err = db.GetNodeShareToTarget(node.ID, target)
				if err != nil {
					log.Errorf("Failed to get share on node %v to target %v", node.ID, target)
					continue
				}
				if shareNode != nil {
					break
				}
			}
		}
		permissionString := "27"
		if node.Isdir {
			permissionString = "31"
		}
		if shareNode != nil {
			permissionString = strconv.Itoa(shareNode.Permissions)
		}
		typeString := "file"
		if node.Isdir {
			typeString = "folder"
		}
		nodePath := node.Path[strings.Index(node.Path, "/")+1:]
		nodePath = nodePath[strings.Index(nodePath, "/")+1:]

		var linkDir string
		if strings.Contains(nodePath, "/") {
			linkDir = nodePath[:strings.LastIndex(nodePath, "/")]
		}
		link := fmt.Sprintf("/index.php/apps/files/?dir=/%v&scrollto=%v", linkDir, node.Path[strings.LastIndex(node.Path, "/")+1:])

		sr := SearchResult{
			Id:          strconv.Itoa(node.ID),
			Link:        link,
			Mime:        node.MimeType,
			Modified:    strconv.Itoa(int(nodeInfo.ModTime().Unix())),
			Name:        nodeInfo.Name(),
			Path:        nodePath,
			Permissions: permissionString,
			Size:        strconv.Itoa(int(nodeInfo.Size())),
			Type:        typeString,
		}
		response[i] = sr
	}
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(&response)
}
