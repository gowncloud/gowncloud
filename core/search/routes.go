package search

import (
	"net/http"

	log "github.com/Sirupsen/logrus"
)

func RegisterRoutes(protectedMux *http.ServeMux, publicMux *http.ServeMux) {
	log.Debug("Regestering search routes")

	protectedMux.HandleFunc("/index.php/core/search", Search)
}
