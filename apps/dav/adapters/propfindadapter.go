package ocdavadapters

import (
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

// PropFindAdapter is the adapter for the PROPFIND method. It intercepts the response
// from the dav server, and then tries to modify it by adding responses stored in
// the datastore
func PropFindAdapter(handler http.HandlerFunc, w http.ResponseWriter, r *http.Request) {

	username := identity.CurrentSession(r).Username

	// Check the request path. If it points to the home direcotry we need to
	// include the shares later on
	isHomeDir := r.URL.Path == "/remote.php/webdav/"

	// Sinse home directories can't be shared, we don't need to check if we are going
	// into a shared folder
	if !isHomeDir {
		targetNode, err := db.GetNode(strings.Replace(r.URL.Path, "/remote.php/webdav", username, 1))
		if err != nil {
			log.Error("Error while searching for target node")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		// If o node was found, look for shared nodes
		if targetNode == nil {
			log.Debug("Looking for shares")
			// sharedNodes, err := db.GetSharedNamedNodesToUser(strings.TrimLeft(r.URL.Path, "/remote.php/webdav/"), username)
			sharedNodes, err := findShareRoot(r.URL.Path, username)
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
			propFindSharedDirectory(handler, w, r, sharedNodes[0])
			return
		}
	}

	r.URL.Path = strings.Replace(r.URL.Path, "/remote.php/webdav", "/remote.php/webdav/"+username, 1)

	rh := newResponseHijacker(w)
	handler.ServeHTTP(rh, r)

	xmldoc := etree.NewDocument()
	err := xmldoc.ReadFromBytes(rh.body)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Error(err)
		return
	}

	// Specify the OC namespace, else the parser and UI will whine
	multistatus := xmldoc.SelectElement("multistatus")
	multistatus.CreateAttr("xmlns:OC", "http://owncloud.org/ns")

	// TODO: right now selecting based on grandchildren being present is not supported
	// so parse the result manually. This should be changed at a later date to just selecting
	// the required nodes with one Xpath query
	responses := xmldoc.FindElements("//response")

	// Remove the user folder from the href nodes
	for _, response := range responses {
		for _, href := range response.SelectElements("href") {
			tmp := strings.Replace(href.Text(), "/remote.php/webdav/"+username, "/remote.php/webdav/", 1)
			href.SetText(strings.Replace(tmp, "//", "/", 1))
		}
	}

	// Seperate file and folder responses.
	folderResponses := []*etree.Element{}
	fileResponses := []*etree.Element{}
	for _, response := range responses {
		propstats := response.SelectElements("propstat")
		for _, propstat := range propstats {
			props := propstat.SelectElements("prop")
			for _, prop := range props {
				resourcetypes := prop.SelectElements("resourcetype")
				for _, resourcetype := range resourcetypes {
					collection := resourcetype.SelectElements("collection")
					if len(collection) != 0 {
						folderResponses = append(folderResponses, response)
						continue
					}
					fileResponses = append(fileResponses, response)
				}
			}
		}
	}

	if len(folderResponses)+len(fileResponses) != len(responses) {
		w.WriteHeader(http.StatusInternalServerError)
		log.Errorf("Total nodes (%v) doesn't match the amount of files (%v) and folders (%v)",
			len(responses), len(fileResponses), len(folderResponses))
		return
	}

	// Patch response for files.
	log.Debug("patch file responses")
	for _, fileResponse := range fileResponses {

		file, err := getNodeFromHref(fileResponse.SelectElement("href").Text(), username)
		if err != nil {
			log.Error("Error getting file from database")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		if file == nil {
			log.Error("Failed to get file from database")
			w.WriteHeader(http.StatusNotFound)
			return
		}
		shares, err := db.GetSharesByNodeId(file.ID)
		if err != nil {
			log.Error("Error getting possible shares from database")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		propstats := fileResponse.SelectElements("propstat")
		for _, propstat := range propstats {
			prop := propstat.SelectElement("prop")
			status := propstat.SelectElement("status")
			if status.Text() == STATUS_OK {
				// Patch attributes
				permissions := prop.CreateElement("OC:permissions")
				permissions.SetText("RDNVW") // This should set all permissions
				// Set fileid
				fileId := prop.CreateElement("OC:fileid")
				fileIdString := strconv.Itoa(file.ID)
				fileId.SetText(fileIdString)
				if len(shares) != 0 {
					shareTypes := prop.CreateElement("OC:share-types")
					shareType := shareTypes.CreateElement("OC:share-type")
					shareType.SetText("0")
				}
				continue
			}
			// Remove attributes we patchted from the not found section
			permissions := prop.SelectElement("permissions")
			removedChild := prop.RemoveChild(permissions)
			if removedChild == nil {
				log.Error("Failed to patch permissions")
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			fileId := prop.SelectElement("fileid")
			removedChild = prop.RemoveChild(fileId)
			if removedChild == nil {
				log.Error("Failed to patch fileid")
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			shareTypes := prop.SelectElement("share-types")
			removedChild = prop.RemoveChild(shareTypes)
			if removedChild == nil {
				log.Error("Failed to patch share-types")
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
		}
	}

	// Patch response for folders.
	log.Debug("patch folder responses")
	for _, folderResponse := range folderResponses {

		dir, err := getNodeFromHref(folderResponse.SelectElement("href").Text(), username)
		if err != nil {
			log.Error("Error getting directory from database")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		if dir == nil {
			log.Error("Failed to get directory from database")
			w.WriteHeader(http.StatusNotFound)
			return
		}
		shares, err := db.GetSharesByNodeId(dir.ID)
		if err != nil {
			log.Error("Error getting possible shares from database")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		propstats := folderResponse.SelectElements("propstat")
		for _, propstat := range propstats {
			prop := propstat.SelectElement("prop")
			status := propstat.SelectElement("status")
			if status.Text() == STATUS_OK {
				// Patch attributes
				// Set permissions
				permissions := prop.CreateElement("OC:permissions")
				permissions.SetText("RDNVCK") // This should set all permissions
				// Set fileid
				fileId := prop.CreateElement("OC:fileid")
				fileIdString := strconv.Itoa(dir.ID)
				fileId.SetText(fileIdString)
				// Set size
				byteSize, err := getDirSize(db.GetSetting(db.DAV_ROOT) + dir.Path)
				if err != nil {
					log.Error("Failed to calculate directory size: ", err)
					w.WriteHeader(http.StatusInternalServerError)
					return
				}
				size := prop.CreateElement("OC:size")
				sizeString := strconv.FormatInt(byteSize, 10)
				size.SetText(sizeString)
				if len(shares) != 0 {
					shareTypes := prop.CreateElement("OC:share-types")
					shareType := shareTypes.CreateElement("OC:share-type")
					shareType.SetText("0")
				}
				continue
			}
			// Remove attributes we patchted from the not found section
			permissions := prop.SelectElement("permissions")
			removedChild := prop.RemoveChild(permissions)
			if removedChild == nil {
				log.Error("Failed to patch permissions")
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			fileId := prop.SelectElement("fileid")
			removedChild = prop.RemoveChild(fileId)
			if removedChild == nil {
				log.Error("Failed to patch fileid")
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			size := prop.SelectElement("size")
			removedChild = prop.RemoveChild(size)
			if removedChild == nil {
				log.Error("Failed to patch size")
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			shareTypes := prop.SelectElement("share-types")
			removedChild = prop.RemoveChild(shareTypes)
			if removedChild == nil {
				log.Error("Failed to patch share-types")
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
		}
	}

	// We only care about shares if this is the users home directory
	if isHomeDir {

		// Load the shares and make their responses.
		shares, err := db.GetSharesToUser(username)
		if err != nil {
			log.Error("Failed to get shares")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		for _, share := range shares {
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
				for _, href := range response.SelectElements("href") {
					href.SetText("/remote.php/webdav/" + href.Text()[strings.LastIndex(href.Text(), "/")+1:])
				}

				for _, propstat := range response.SelectElements("propstat") {
					status := propstat.SelectElement("status")
					if status.Text() == STATUS_OK {
						for _, prop := range propstat.SelectElements("prop") {
							if sharedNode.Isdir {
								byteSize, err := getDirSize(db.GetSetting(db.DAV_ROOT) + sharedNode.Path)
								if err != nil {
									log.Error("Failed to calculate directory size: ", err)
									w.WriteHeader(http.StatusInternalServerError)
									return
								}
								size := prop.CreateElement("OC:size")
								sizeString := strconv.FormatInt(byteSize, 10)
								size.SetText(sizeString)

								fileId := prop.CreateElement("OC:fileid")
								fileIdString := strconv.Itoa(sharedNode.ID)
								fileId.SetText(fileIdString)

								permissions := prop.CreateElement("OC:permissions")
								permissions.SetText("SRDNVCK")

								owner := prop.CreateElement("OC:owner-display-name")
								owner.SetText(sharedNode.Owner)

								prop.CreateElement("OC:share-types")

								continue
							}
							fileId := prop.CreateElement("OC:fileid")
							fileIdString := strconv.Itoa(sharedNode.ID)
							fileId.SetText(fileIdString)

							permissions := prop.CreateElement("OC:permissions")
							permissions.SetText("SRDNVW")

							owner := prop.CreateElement("OC:owner-display-name")
							owner.SetText(sharedNode.Owner)

							prop.CreateElement("OC:share-types")
						}
					}
				}

				multistatus.AddChild(response)
			}

		}

	}

	for key, valuemap := range rh.headers {
		w.Header().Set(key, strings.Join(valuemap, " "))
	}

	w.WriteHeader(rh.status)
	xmldoc.WriteTo(w)
}

// propFindSharedDirectory returns a modified propfind response to allow users
// to enter shared directories.
// FIXME: lots of code copied from PropFindAdapter, needs merge
func propFindSharedDirectory(handler http.HandlerFunc, w http.ResponseWriter, r *http.Request, target *db.Node) {

	targetRoot := "/" + target.Path[:strings.LastIndex(target.Path, "/")]

	originalPath := r.URL.Path

	finalPath := target.Path[:strings.LastIndex(target.Path, "/")] + strings.TrimPrefix(originalPath, "/remote.php/webdav")

	r.URL.Path = "/remote.php/webdav/" + finalPath

	rh := newResponseHijacker(w)
	handler.ServeHTTP(rh, r)

	xmldoc := etree.NewDocument()
	err := xmldoc.ReadFromBytes(rh.body)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Error(err)
		return
	}

	// Specify the OC namespace, else the parser and UI will whine
	multistatus := xmldoc.SelectElement("multistatus")
	multistatus.CreateAttr("xmlns:OC", "http://owncloud.org/ns")

	// TODO: right now selecting based on grandchildren being present is not supported
	// so parse the result manually. This should be changed at a later date to just selecting
	// the required nodes with one Xpath query
	responses := xmldoc.FindElements("//response")

	responseMap := make(map[*etree.Element]string)

	// Patch to shared path
	for _, response := range responses {
		for _, href := range response.SelectElements("href") {
			responseMap[response] = href.Text()
			href.SetText(strings.Replace(href.Text(), targetRoot, "", 1))
		}
	}

	// Seperate file and folder responses.
	folderResponses := []*etree.Element{}
	fileResponses := []*etree.Element{}
	for _, response := range responses {
		propstats := response.SelectElements("propstat")
		for _, propstat := range propstats {
			props := propstat.SelectElements("prop")
			for _, prop := range props {
				resourcetypes := prop.SelectElements("resourcetype")
				for _, resourcetype := range resourcetypes {
					collection := resourcetype.SelectElements("collection")
					if len(collection) != 0 {
						folderResponses = append(folderResponses, response)
						continue
					}
					fileResponses = append(fileResponses, response)
				}
			}
		}
	}

	if len(folderResponses)+len(fileResponses) != len(responses) {
		w.WriteHeader(http.StatusInternalServerError)
		log.Errorf("Total nodes (%v) doesn't match the amount of files (%v) and folders (%v)",
			len(responses), len(fileResponses), len(folderResponses))
		return
	}

	// Patch response for files.
	log.Debug("patch file responses")
	for _, fileResponse := range fileResponses {

		file, err := getSharedNodeFromHref(responseMap[fileResponse])
		if err != nil {
			log.Error("Error getting file from database")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		if file == nil {
			log.Error("Failed to get file from database")
			w.WriteHeader(http.StatusNotFound)
			return
		}
		shares, err := db.GetSharesByNodeId(file.ID)
		if err != nil {
			log.Error("Error getting possible shares from database")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		propstats := fileResponse.SelectElements("propstat")
		for _, propstat := range propstats {
			prop := propstat.SelectElement("prop")
			status := propstat.SelectElement("status")
			if status.Text() == STATUS_OK {
				// Patch attributes
				permissions := prop.CreateElement("OC:permissions")
				permissions.SetText("RDNVW") // This should set all permissions
				// Set fileid
				fileId := prop.CreateElement("OC:fileid")
				fileIdString := strconv.Itoa(file.ID)
				fileId.SetText(fileIdString)
				if len(shares) != 0 {
					shareTypes := prop.CreateElement("OC:share-types")
					shareType := shareTypes.CreateElement("OC:share-type")
					shareType.SetText("0")
				}
				continue
			}
			// Remove attributes we patchted from the not found section
			permissions := prop.SelectElement("permissions")
			removedChild := prop.RemoveChild(permissions)
			if removedChild == nil {
				log.Error("Failed to patch permissions")
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			fileId := prop.SelectElement("fileid")
			removedChild = prop.RemoveChild(fileId)
			if removedChild == nil {
				log.Error("Failed to patch fileid")
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			shareTypes := prop.SelectElement("share-types")
			removedChild = prop.RemoveChild(shareTypes)
			if removedChild == nil {
				log.Error("Failed to patch share-types")
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
		}
	}

	// Patch response for folders.
	log.Debug("patch folder responses")
	for _, folderResponse := range folderResponses {

		//dir, err := db.GetNode(strings.Trim(responseMap[folderResponse], "/remote.php/webdav/"))
		dir, err := getSharedNodeFromHref(responseMap[folderResponse])
		if err != nil {
			log.Error("Error getting directory from database")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		if dir == nil {
			log.Error("Failed to get directory from database")
			w.WriteHeader(http.StatusNotFound)
			return
		}
		shares, err := db.GetSharesByNodeId(dir.ID)
		if err != nil {
			log.Error("Error getting possible shares from database")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		propstats := folderResponse.SelectElements("propstat")
		for _, propstat := range propstats {
			prop := propstat.SelectElement("prop")
			status := propstat.SelectElement("status")
			if status.Text() == STATUS_OK {
				// Patch attributes
				// Set permissions
				permissions := prop.CreateElement("OC:permissions")
				permissions.SetText("RDNVCK") // This should set all permissions
				// Set fileid
				fileId := prop.CreateElement("OC:fileid")
				fileIdString := strconv.Itoa(dir.ID)
				fileId.SetText(fileIdString)
				// Set size
				byteSize, err := getDirSize(db.GetSetting(db.DAV_ROOT) + dir.Path)
				if err != nil {
					log.Error("Failed to calculate directory size: ", err)
					w.WriteHeader(http.StatusInternalServerError)
					return
				}
				size := prop.CreateElement("OC:size")
				sizeString := strconv.FormatInt(byteSize, 10)
				size.SetText(sizeString)
				if len(shares) != 0 {
					shareTypes := prop.CreateElement("OC:share-types")
					shareType := shareTypes.CreateElement("OC:share-type")
					shareType.SetText("0")
				}
				continue
			}
			// Remove attributes we patchted from the not found section
			permissions := prop.SelectElement("permissions")
			removedChild := prop.RemoveChild(permissions)
			if removedChild == nil {
				log.Error("Failed to patch permissions")
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			fileId := prop.SelectElement("fileid")
			removedChild = prop.RemoveChild(fileId)
			if removedChild == nil {
				log.Error("Failed to patch fileid")
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			size := prop.SelectElement("size")
			removedChild = prop.RemoveChild(size)
			if removedChild == nil {
				log.Error("Failed to patch size")
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			shareTypes := prop.SelectElement("share-types")
			removedChild = prop.RemoveChild(shareTypes)
			if removedChild == nil {
				log.Error("Failed to patch share-types")
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
		}
	}
	log.Debug("Finished patching directory responses")

	for key, valuemap := range rh.headers {
		w.Header().Set(key, strings.Join(valuemap, " "))
	}

	w.WriteHeader(rh.status)
	xmldoc.WriteTo(w)
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
	path := strings.TrimSuffix(strings.Replace(href, "/remote.php/webdav", username, 1), "/")
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
func findShareRoot(href string, username string) ([]*db.Node, error) {
	path := strings.TrimLeft(href, "/remote.php/webdav/")
	nodes, err := db.GetSharedNamedNodesToUser(path, username)
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
		nodes, err = db.GetSharedNamedNodesToUser(path, username)
		if err != nil {
			return nil, err
		}
		if len(nodes) > 0 {
			break
		}
	}
	return nodes, nil
}
