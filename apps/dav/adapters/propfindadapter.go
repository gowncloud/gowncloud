package ocdavadapters

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	log "github.com/Sirupsen/logrus"
	"github.com/beevik/etree"
	"github.com/gowncloud/gowncloud/core/identity"
	db "github.com/gowncloud/gowncloud/database"
)

const (
	STATUS_OK       = "HTTP/1.1 200 OK"
	STATUS_NOTFOUND = "HTTP/1.1 404 Not Found"
)

// patchFunction is the signature of any function that patches an element from the propfind response.
// A patchFunction should remove the element from the not found section, and add it
// with the appropriate value to the found section
type patchFunction func(foundProps *etree.Element, notFoundProps *etree.Element, node *db.Node, shared []*db.Share, user string) error

// patchMap maps the possible requested tags to their correct function to add
// said tags. To add a new tag, a function should be written that matches the
// patchFunction signature, and be registered here. This way additional tags can be
// implemented without changing any of the propfind function. Also we can check
// and only patch the requested properties.
var patchMap map[string]patchFunction = map[string]patchFunction{
	"fileid":             patchFileId,
	"id":                 patchId,
	"permissions":        patchPermissions,
	"share-types":        patchShareTypes,
	"favorite":           patchFavorite,
	"size":               patchSize,
	"owner-display-name": patchOwnerDisplayName,
}

// PropFindAdapter is the adapter for the PROPFIND method. It intercepts the response
// from the dav server, and then tries to modify it by adding responses stored in
// the datastore
func PropFindAdapter(handler http.HandlerFunc, w http.ResponseWriter, r *http.Request) {

	username := identity.CurrentSession(r).Username
	groups := identity.CurrentSession(r).Organizations

	// Check the request path. If it points to the home direcotry we need to
	// include the shares later on
	isHomeDir := r.URL.Path == "/remote.php/webdav/"

	var inSharedNode bool
	var targetRoot string

	// Sinse home directories can't be shared, we don't need to check if we are going
	// into a shared folder
	if !isHomeDir {
		targetNode, err := db.GetNode(strings.Replace(r.URL.Path, "/remote.php/webdav", username+"/files", 1))
		if err != nil {
			log.Error("Error while searching for target node")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		// If no node was found, look for shared nodes
		if targetNode == nil {
			log.Debug("Looking for shares")

			sharedNodes, err := findShareRoot(r.URL.Path, append(groups, username))
			if err != nil {
				log.Error("Error while searching for shared nodes")
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			if len(sharedNodes) == 0 {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			// Go into the first shared directory. Log collisions
			if len(sharedNodes) > 1 {
				log.Warn("Shared folder collision")
			}

			inSharedNode = true

			target := sharedNodes[0]

			targetRoot = "/" + target.Path[:strings.LastIndex(target.Path, "/")]
			originalPath := r.URL.Path
			finalPath := target.Path[:strings.LastIndex(target.Path, "/")] + strings.TrimPrefix(originalPath, "/remote.php/webdav")
			r.URL.Path = "/remote.php/webdav/" + finalPath
		}
	}

	if !inSharedNode {
		r.URL.Path = strings.Replace(r.URL.Path, "/remote.php/webdav", "/remote.php/webdav/"+username+"/files", 1)
	}

	inputDoc := etree.NewDocument()

	// First buffer the request body, then duplicate it so we can pass it on
	bodyBytes, _ := ioutil.ReadAll(r.Body)
	newBody := ioutil.NopCloser(bytes.NewBuffer(bodyBytes))

	r.Body = newBody
	err := inputDoc.ReadFromBytes(bodyBytes)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Error(err)
		return
	}

	// Get all the requested props
	// TODO: get the all header
	requestedProps := make([]string, 0)
	for _, prop := range inputDoc.FindElements("//prop") {
		for _, tbf := range prop.ChildElements() {
			requestedProps = append(requestedProps, tbf.Tag)
		}
	}

	rh := newResponseHijacker(w)
	handler.ServeHTTP(rh, r)

	xmldoc := etree.NewDocument()
	err = xmldoc.ReadFromBytes(rh.body)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Error(err)
		return
	}

	// Propfind responses have a multistatus root element
	multistatus := xmldoc.SelectElement("multistatus")

	// The call was not successfull so we cant patch anything
	// Copy the response and return
	if multistatus == nil {
		for key, valuemap := range rh.headers {
			w.Header().Set(key, strings.Join(valuemap, " "))
		}
		w.WriteHeader(rh.status)
		w.Write(rh.body)
		return
	}

	// Use lowercase namespace
	attr := multistatus.SelectAttr("D")
	if attr != nil {
		attr.Key = "d"
	}

	// Specify the oc namespace, else the parser and UI will whine
	multistatus.CreateAttr("xmlns:oc", "http://owncloud.org/ns")

	responses := xmldoc.FindElements("//response")

	// Collect all the errors for debug reasons
	patchErrors := make([]error, 0)

	// Keep track of the node id's
	nodeIDs := make([]int, 0)

	// Remove the user folder from the href nodes and patch the responses
	for _, response := range responses {
		href := response.SelectElement("href")
		if href == nil {
			log.Error("Response doesn't have an href tag")
			continue
		}

		var hrefString, nodePath string
		if inSharedNode {
			nodePath = href.Text()
			hrefString = strings.Replace(href.Text(), targetRoot, "", 1)
		} else {
			hrefString = strings.Replace(href.Text(), "/remote.php/webdav/"+username+"/files", "/remote.php/webdav/", 1)
			hrefString = strings.Replace(hrefString, "//", "/", 1)
			nodePath = hrefString
		}

		foundProps, notFoundProps, err := getPropStats(response)
		if err != nil {
			log.Warn("Error while getting props: ", err)
			// Don't patch this response if there is an error but contine with the other responses
			continue
		}
		if notFoundProps == nil {
			log.Debug("No not found props, nothing to do here")
			continue
		}
		var node *db.Node
		if inSharedNode {
			node, err = getSharedNodeFromHref(nodePath)
		} else {
			node, err = getNodeFromHref(nodePath, username)
		}
		if err != nil {
			log.Error("Error getting node from database")
			continue
		}
		if node == nil {
			log.Error("Failed to get node from database")
			continue
		}
		nodeIDs = append(nodeIDs, node.ID)
		// Directory references should end with a '/'
		if node.Isdir {
			// But make sure they don't end with a double '/'
			if !strings.HasSuffix(hrefString, "/") {
				hrefString += "/"
			}
		}
		href.SetText(hrefString)
		shares, err := db.GetSharesByNodeId(node.ID)
		if err != nil {
			log.Error("Error getting possible shares from database")
			continue
		}
		for _, requestedProp := range requestedProps {
			if patchMap[requestedProp] != nil {
				err = patchMap[requestedProp](foundProps, notFoundProps, node, shares, username)
				if err != nil {
					patchErrors = append(patchErrors, err)
				}
			}
		}
	}

	// We only care about shares if this is the users home directory
	if isHomeDir {

		orgs := identity.CurrentSession(r).Organizations
		// Load the shares and make their responses.
		shares, err := db.GetAllSharesToUser(username, orgs)
		if err != nil {
			log.Error("Failed to get shares")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		for _, share := range shares {
			alreadyFound := false
			for _, nodeID := range nodeIDs {
				if share.NodeID == nodeID {
					alreadyFound = true
					break
				}
			}
			if alreadyFound {
				continue
			}
			nodeIDs = append(nodeIDs, share.NodeID)
			sharedNode, err := db.GetSharedNode(share.ShareID)
			if err != nil {
				log.Error("Failed to get node from share")
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			rhj := newResponseHijacker(w)
			// Set 'Depth' header to 0 since we only care for the shared node, and not the
			// children of said node
			r.Header.Set("Depth", "0")
			r.URL.Path = "/remote.php/webdav/" + sharedNode.Path

			bodyBuffered := bytes.NewBuffer(bodyBytes)
			r.Body = ioutil.NopCloser(bodyBuffered)

			handler.ServeHTTP(rhj, r)
			if rhj.status != http.StatusMultiStatus {
				log.Error("PROPFIND on shared node failed")
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			responsexmldoc := etree.NewDocument()
			err = responsexmldoc.ReadFromBytes(rhj.body)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				log.Error(err)
				return
			}

			newresponses := responsexmldoc.FindElements("//response")

			for _, response := range newresponses {
				var hrefString string
				href := response.SelectElement("href")
				if href == nil {
					log.Error("Response doesn't have an href tag")
					continue
				}
				hrefString = "/remote.php/webdav/" + href.Text()[strings.LastIndex(href.Text(), "/")+1:]
				if sharedNode.Isdir {
					// But make sure they don't end with a double '/'
					if !strings.HasSuffix(hrefString, "/") {
						hrefString += "/"
					}
				}
				href.SetText(hrefString)

				foundProps, notFoundProps, err := getPropStats(response)
				if err != nil {
					log.Warn("Error while getting props: ", err)
					// Don't patch this response if there is an error but contine with the other responses
					continue
				}
				if notFoundProps == nil {
					log.Debug("No not found props, nothing to do here")
					continue
				}
				s := []*db.Share{
					0: share,
				}
				for _, requestedProp := range requestedProps {
					if patchMap[requestedProp] != nil {
						err = patchMap[requestedProp](foundProps, notFoundProps, sharedNode, s, username)
						if err != nil {
							patchErrors = append(patchErrors, err)
						}
					}
				}

				multistatus.AddChild(response)
			}
		}
	}

	// Make sure all the namespaces are lowercase
	modifyNamespaceToLower(multistatus)

	for key, valuemap := range rh.headers {
		w.Header().Set(key, strings.Join(valuemap, " "))
	}

	log.Debugf("Propfind patching finished with %v errors", len(patchErrors))
	for i, e := range patchErrors {
		log.Debug("Error %v: %v", i, e)
	}

	w.WriteHeader(rh.status)
	xmldoc.WriteTo(w)
}

func patchFileId(foundProps *etree.Element, notFoundProps *etree.Element, node *db.Node, shared []*db.Share, user string) error {
	fileIdNotFound := notFoundProps.SelectElement("fileid")
	if fileIdNotFound == nil {
		return fmt.Errorf("Failed to get the fileid prop from the not found section")
	}
	fileId := foundProps.CreateElement("oc:fileid")
	fileIdString := strconv.Itoa(node.ID)
	fileId.SetText(fileIdString)

	removedChild := notFoundProps.RemoveChild(fileIdNotFound)
	if removedChild == nil {
		log.Warn("Failed to patch fileid")
		return fmt.Errorf("Failed to patch fileid")
	}
	return nil
}

func patchId(foundProps *etree.Element, notFoundProps *etree.Element, node *db.Node, shared []*db.Share, user string) error {
	idNotFound := notFoundProps.SelectElement("id")
	if idNotFound == nil {
		return fmt.Errorf("Failed to get the id prop from the not found section")
	}
	id := foundProps.CreateElement("oc:id")
	idString := strconv.Itoa(node.ID)
	id.SetText(idString)

	removedChild := notFoundProps.RemoveChild(idNotFound)
	if removedChild == nil {
		log.Warn("Failed to patch id")
		return fmt.Errorf("Failed to patch id")
	}
	return nil
}

func patchPermissions(foundProps *etree.Element, notFoundProps *etree.Element, node *db.Node, shared []*db.Share, user string) error {
	permissionString := "RDNVW"
	if node.Isdir {
		permissionString = "RDNVCK"
	}
	if node.Owner != user {
		permissionString = "S" + permissionString
	}
	permissionsNotFound := notFoundProps.SelectElement("permissions")
	if permissionsNotFound == nil {
		return fmt.Errorf("Failed to get the permissions prop from the not found section")
	}
	permissions := foundProps.CreateElement("oc:permissions")
	permissions.SetText(permissionString)

	removedChild := notFoundProps.RemoveChild(permissionsNotFound)
	if removedChild == nil {
		log.Warn("Failed to patch permissions")
		return fmt.Errorf("Failed to patch permissions")
	}
	return nil
}

func patchShareTypes(foundProps *etree.Element, notFoundProps *etree.Element, node *db.Node, shared []*db.Share, user string) error {
	if !(len(shared) > 0) {
		return nil
	}
	notFoundShareTypes := notFoundProps.SelectElement("share-types")
	if notFoundShareTypes == nil {
		return fmt.Errorf("Failed to get share-types prop from the not found section")
	}
	shareTypes := foundProps.CreateElement("oc:share-types")
	shareType := shareTypes.CreateElement("oc:share-type")
	shareType.SetText("0")

	removedChild := notFoundProps.RemoveChild(notFoundShareTypes)
	if removedChild == nil {
		log.Warn("Failed to patch share types")
		return fmt.Errorf("Failed to patch share types")
	}
	return nil
}

func patchFavorite(foundProps *etree.Element, notFoundProps *etree.Element, node *db.Node, shared []*db.Share, user string) error {
	isFavorite, err := db.IsFavoriteByNodeid(node.ID, user)
	if err != nil {
		log.Error("Failed to check if node is favorite: ", err)
		return fmt.Errorf("Database error")
	}
	favoriteString := "1"
	if !isFavorite {
		favoriteString = "0"
	}
	notFoundFavorite := notFoundProps.SelectElement("favorite")
	if notFoundFavorite == nil {
		return fmt.Errorf("Failed to get favorite prop from the not found section")
	}
	favorite := foundProps.CreateElement("oc:favorite")
	favorite.SetText(favoriteString)

	removedChild := notFoundProps.RemoveChild(notFoundFavorite)
	if removedChild == nil {
		log.Warn("Failed to patch favorite")
		return fmt.Errorf("Failed to patch favorite")
	}
	return nil
}

func patchSize(foundProps *etree.Element, notFoundProps *etree.Element, node *db.Node, shared []*db.Share, user string) error {
	notFoundSize := notFoundProps.SelectElement("size")
	if notFoundSize == nil {
		return fmt.Errorf("Failed to get size prop from the not found section")
	}
	byteSize, err := getDirSize(db.GetSetting(db.DAV_ROOT) + node.Path)
	if err != nil {
		log.Error("Failed to calculate directory size: ", err)
		return fmt.Errorf("Failed to calculate directory size: %v", err)
	}
	sizeString := strconv.FormatInt(byteSize, 10)
	size := foundProps.CreateElement("oc:size")
	size.SetText(sizeString)

	removedChild := notFoundProps.RemoveChild(notFoundSize)
	if removedChild == nil {
		log.Warn("Failed to patch size")
		return fmt.Errorf("Failed to patch size")
	}
	return nil
}

func patchOwnerDisplayName(foundProps *etree.Element, notFoundProps *etree.Element, node *db.Node, shared []*db.Share, user string) error {
	notFoundOwnerDisplayName := notFoundProps.SelectElement("owner-display-name")
	if notFoundOwnerDisplayName == nil {
		return fmt.Errorf("Failed to get owner display name prop from not found section")
	}
	ownerDisplayName := foundProps.CreateElement("oc:owner-display-name")
	ownerDisplayName.SetText(node.Owner)

	removedChild := notFoundProps.RemoveChild(notFoundOwnerDisplayName)
	if removedChild == nil {
		log.Warn("Failed to patch owner display name")
		return fmt.Errorf("Failed to patch owner display name")
	}
	return nil
}

func getPropStats(response *etree.Element) (foundProps, notFoundProps *etree.Element, err error) {
	propstats := response.SelectElements("propstat")
	var foundPstat, notFoundPstat *etree.Element
	for _, pstat := range propstats {
		if pstat.SelectElement("status").Text() == STATUS_NOTFOUND {
			notFoundPstat = pstat
			continue
		}
		foundPstat = pstat
	}
	if notFoundPstat == nil {
		log.Debug("notFoundPstat = nil")
		return
	}
	if foundPstat == nil {
		log.Debug("foundPstat = nil")
		return
	}
	notFoundProps = notFoundPstat.SelectElement("prop")
	if notFoundProps == nil {
		err = fmt.Errorf("Could not find not found props")
		return
	}
	foundProps = foundPstat.SelectElement("prop")
	if foundProps == nil {
		err = fmt.Errorf("No found props in the response")
		return
	}
	return
}

// modifyNamespaceToLower recursively parses the xml with root element and modifies
// all element namespaces to be lowercase
func modifyNamespaceToLower(element *etree.Element) {
	element.Space = strings.ToLower(element.Space)
	for _, child := range element.ChildElements() {
		modifyNamespaceToLower(child)
	}
}

// getDirSize gets the size of the directory (all of its descendants) at path.
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

// getNodeFromHref unescapes the href and returns the associated node
func getNodeFromHref(href string, username string) (*db.Node, error) {
	path := strings.TrimSuffix(strings.Replace(href, "/remote.php/webdav", username+"/files", 1), "/")
	// Monkey business to prevent '+' from being decoded
	pathPieces := strings.Split(path, "+")

	var err error
	for i, piece := range pathPieces {
		pathPieces[i], err = url.QueryUnescape(piece)
		if err != nil {
			log.Error("Error converting href to node path: ", err)
			return nil, err
		}
	}
	path = strings.Join(pathPieces, "+")

	return db.GetNode(path)
}

// getSharedNodeFromHref unescapes the href and returns the associated node
func getSharedNodeFromHref(href string) (*db.Node, error) {
	path := strings.TrimPrefix(href, "/remote.php/webdav/")
	// Monkey business to prevent '+' from being decoded
	pathPieces := strings.Split(path, "+")

	var err error
	for i, piece := range pathPieces {
		pathPieces[i], err = url.QueryUnescape(piece)
		if err != nil {
			log.Error("Error converting href to node path: ", err)
			return nil, err
		}
	}
	path = strings.Join(pathPieces, "+")

	return db.GetNode(path)
}

// findShareRoot parses a path and tries to find a share
func findShareRoot(href string, targets []string) ([]*db.Node, error) {
	path := strings.TrimLeft(href, "/remote.php/webdav/")
	nodes, err := db.GetSharedNamedNodesToTargets(path, targets)
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
		nodes, err = db.GetSharedNamedNodesToTargets(path, targets)
		if err != nil {
			return nil, err
		}
		if len(nodes) > 0 {
			break
		}
	}
	return nodes, nil
}
