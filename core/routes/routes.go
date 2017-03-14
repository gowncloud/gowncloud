package routes

import (
	"net/http"

	log "github.com/Sirupsen/logrus"
	"github.com/gowncloud/gowncloud/core"
)

func RegisterRoutes(protected *http.ServeMux, publicMux *http.ServeMux) {
	log.Debug("Registering core routes")

	publicMux.HandleFunc("/status.php", core.Status)
}
