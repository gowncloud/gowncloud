package ocdavadapters

import (
	"net/http"
	"strings"

	"github.com/gowncloud/gowncloud/core/identity"
	db "github.com/gowncloud/gowncloud/database"
)

// MkcolAdapter is the adapter for the WebDav MKCOL method
func MkcolAdapter(handler http.HandlerFunc, w http.ResponseWriter, r *http.Request) {

	user := identity.CurrentSession(r).Username
	r.URL.Path = strings.Replace(r.URL.Path,
		"/remote.php/webdav", "/remote.php/webdav/"+user,
		1)

	_, err := db.SaveNode(strings.Replace(r.URL.Path, "/remote.php/webdav/", "", 1), user, true)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// TODO: use responsehijacker to intercept response and delete the node if an error occurred
	handler.ServeHTTP(w, r)
}
