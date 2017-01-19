package dav

import (
	"net/http"
	"strings"

	log "github.com/Sirupsen/logrus"
)

func GetAdapter(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Debug("Method: ", r.Method)

		if r.Method == "GET" {
			fullname := r.URL.RequestURI()
			filename := fullname[strings.LastIndex(fullname, "/")+1:]
			w.Header().Set("Content-Disposition", "attachment; filename="+filename)
		}

		handler.ServeHTTP(w, r)
	})
}
