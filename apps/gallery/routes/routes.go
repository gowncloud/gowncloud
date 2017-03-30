package gallery

import (
	"net/http"

	log "github.com/Sirupsen/logrus"
	"github.com/gorilla/mux"
	gallery "github.com/gowncloud/gowncloud/apps/gallery/api"
)

func RegisterRoutes(protectedMux *http.ServeMux, publicMux *http.ServeMux) {
	log.Debug("Regestering gallery routes")

	protectedMux.HandleFunc("/index.php/apps/gallery/config", gallery.Config)

	r := mux.NewRouter()
	r.HandleFunc("/index.php/apps/gallery/preview/{id}", gallery.Preview).Methods("GET")
	protectedMux.Handle("/index.php/apps/gallery/preview/", r)

}
