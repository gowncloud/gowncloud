package ocdavadapters

import (
	"net/http"
	"strings"

	"github.com/gowncloud/gowncloud/core/identity"
	db "github.com/gowncloud/gowncloud/database"
)

// DeleteAdapter is the adapter for the WebDav DELETE method
func DeleteAdapter(handler http.HandlerFunc, w http.ResponseWriter, r *http.Request) {

	user := identity.CurrentSession(r).Username
	r.URL.Path = strings.Replace(r.URL.Path, "/remote.php/webdav", "/remote.php/webdav/"+user, 1)

	err := db.DeleteNode(strings.Replace(r.URL.Path, "/remote.php/webdav/", "", 1))
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	handler.ServeHTTP(w, r)
}
