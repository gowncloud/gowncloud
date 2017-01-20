package ocdavadapters

import (
	"net/http"
	"strings"
)

// GetAdapter is the adapter for the GET method. It adds the correct header to the
// response so the file can be downloaded by the client
func GetAdapter(handler http.HandlerFunc, w http.ResponseWriter, r *http.Request) {

	if r.Method == "GET" {
		fullname := r.URL.RequestURI()
		filename := fullname[strings.LastIndex(fullname, "/")+1:]
		w.Header().Set("Content-Disposition", "attachment; filename="+filename)
	}

	handler.ServeHTTP(w, r)
}
