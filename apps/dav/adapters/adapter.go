package ocdavadapters

import (
	"net/http"
	"strings"

	log "github.com/Sirupsen/logrus"
	"github.com/gowncloud/gowncloud/core/identity"
	db "github.com/gowncloud/gowncloud/database"
)

// Adapter is an interface for the ocdavadapters
type Adapter func(handler http.HandlerFunc, w http.ResponseWriter, r *http.Request)

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
	path := strings.TrimPrefix(href, "/remote.php/webdav/")
	log.Debug("try to find shares for path ", path)
	nodes, err := db.GetSharedNamedNodesToTargets(path, user, groups)
	if err != nil {
		return nil, err
	}
	if len(nodes) > 0 {
		return nodes, nil
	}
	seperatorIndex := strings.LastIndex(path, "/")
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
