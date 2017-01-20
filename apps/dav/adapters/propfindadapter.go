package ocdavadapters

import (
	"net/http"
	"strings"

	log "github.com/Sirupsen/logrus"
	"github.com/beevik/etree"
	"github.com/gowncloud/gowncloud/core/identity"
)

const (
	STATUS_OK       = "HTTP/1.1 200 OK"
	STATUS_NOTFOUND = "HTTP/1.1 404 Not Found"
)

// PropFindAdapter is the adapter for the PROPFIND method. It intercepts the response
// from the dav server, and then tries to modify it by adding responses stored in
// the datastore
func PropFindAdapter(handler http.HandlerFunc, w http.ResponseWriter, r *http.Request) {

	if r.Method != "PROPFIND" {
		handler.ServeHTTP(w, r)
		return
	}

	_, ok := r.Context().Value("session").(identity.Session)
	if !ok {
		log.Error("could not get the session")
	}

	log.Debug("request body: ", r.Body)
	log.Debug("request headers: ", r.Header)
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

	// Patch response for files. For now just patch the permissions
	log.Debug("patch file responses")
	for _, fileResponse := range fileResponses {
		propstats := fileResponse.SelectElements("propstat")
		for _, propstat := range propstats {
			prop := propstat.SelectElement("prop")
			status := propstat.SelectElement("status")
			if status.Text() == STATUS_OK {
				// Patch attributes
				permissions := prop.CreateElement("OC:permissions")
				permissions.SetText("RDNVCK") // This should set all permissions
				continue
			}
			// Remove attributes we patchted from the not found section
			permissions := prop.SelectElement("permissions")
			removedChild := prop.RemoveChild(permissions)
			if removedChild == nil {
				log.Error("failed to patch permissions")
			}
		}
	}

	// Patch response for folders. For now just patch the permissions
	log.Debug("patch folder responses")
	for _, folderResponse := range folderResponses {
		propstats := folderResponse.SelectElements("propstat")
		for _, propstat := range propstats {
			prop := propstat.SelectElement("prop")
			status := propstat.SelectElement("status")
			if status.Text() == STATUS_OK {
				// Patch attributes
				permissions := prop.CreateElement("OC:permissions")
				permissions.SetText("RDNVCK") // This should set all permissions
				continue
			}
			// Remove attributes we patchted from the not found section
			permissions := prop.SelectElement("permissions")
			removedChild := prop.RemoveChild(permissions)
			if removedChild == nil {
				log.Error("failed to patch permissions")
			}
		}
	}

	for key, valuemap := range rh.headers {
		w.Header().Set(key, strings.Join(valuemap, " "))
	}

	w.WriteHeader(rh.status)
	xmldoc.WriteTo(w)
}
