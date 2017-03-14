package routes

import (
	"net/http"

	log "github.com/Sirupsen/logrus"
	"github.com/gorilla/mux"
	files_sharing "github.com/gowncloud/gowncloud/apps/files_sharing/api"
)

func RegisterRoutes(protectedMux *http.ServeMux, publicMux *http.ServeMux) {
	log.Debug("Regestering file sharing routes")

	r := mux.NewRouter()

	r.HandleFunc("/ocs/v2.php/apps/files_sharing/api/v1/shares", files_sharing.ShareInfo).Methods("GET")
	r.HandleFunc("/ocs/v2.php/apps/files_sharing/api/v1/shares", files_sharing.CreateShare).Methods("POST")
	r.HandleFunc("/ocs/v2.php/apps/files_sharing/api/v1/shares/{shareid}", files_sharing.DeleteShare).Methods("DELETE")

	r.HandleFunc("/ocs/v1.php/apps/files_sharing/api/v1/shares", files_sharing.SharedWithMe).Methods("GET").Queries("shared_with_me", "true")
	r.HandleFunc("/ocs/v1.php/apps/files_sharing/api/v1/shares", files_sharing.SharedWithOthers).Methods("GET").Queries("shared_with_me", "false")

	protectedMux.Handle("/ocs/v2.php/apps/files_sharing/api/v1/shares", r)
	protectedMux.Handle("/ocs/v2.php/apps/files_sharing/api/v1/shares/", r)
	protectedMux.Handle("/ocs/v1.php/apps/files_sharing/api/v1/shares", r)

	protectedMux.HandleFunc("/ocs/v1.php/apps/files_sharing/api/v1/sharees", files_sharing.Sharees)

	protectedMux.HandleFunc("/ocs/v1.php/apps/files_sharing/api/v1/remote_shares", files_sharing.RemoteShares)
}
