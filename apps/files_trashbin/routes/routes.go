package routes

import (
	"net/http"

	log "github.com/Sirupsen/logrus"
	trash "github.com/gowncloud/gowncloud/apps/files_trashbin/ajax"
)

func RegisterRoutes(protectedMux *http.ServeMux, publicMux *http.ServeMux) {
	log.Debug("Registering trash routes")

	protectedMux.HandleFunc("/index.php/apps/files_trashbin/ajax/list.php", trash.GetTrash)
	protectedMux.HandleFunc("/index.php/apps/files_trashbin/ajax/delete.php", trash.DeleteTrash)
	protectedMux.HandleFunc("/index.php/apps/files_trashbin/ajax/undelete.php", trash.UndeleteTrash)
}
