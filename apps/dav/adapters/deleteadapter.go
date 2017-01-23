package ocdavadapters

import (
	"net/http"
	"strings"

	"github.com/gowncloud/gowncloud/core/identity"
)

// DeleteAdapter is the adapter for the WebDav DELETE method
func DeleteAdapter(handler http.HandlerFunc, w http.ResponseWriter, r *http.Request) {

	r.URL.Path = strings.Replace(r.URL.Path, "/remote.php/webdav", "/remote.php/webdav/"+identity.CurrentSession(r).Username, 1)

	handler.ServeHTTP(w, r)
}
