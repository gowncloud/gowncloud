package ocdavadapters

import (
	"net/http"
	"strings"

	"github.com/gowncloud/gowncloud/core/identity"
)

// MkcolAdapter is the adapter for the WebDav MKCOL method
func MkcolAdapter(handler http.HandlerFunc, w http.ResponseWriter, r *http.Request) {

	r.URL.Path = strings.Replace(r.URL.Path, "/remote.php/webdav", "/remote.php/webdav/"+identity.CurrentSession(r).Username, 1)

	handler.ServeHTTP(w, r)
}
