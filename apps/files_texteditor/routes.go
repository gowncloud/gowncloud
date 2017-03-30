package files_texteditor

import (
	"net/http"

	log "github.com/Sirupsen/logrus"
)

func RegisterRoutes(protectedMux *http.ServeMux, publicMux *http.ServeMux) {
	log.Debug("Regestering texteditor routes")

	protectedMux.HandleFunc("/index.php/apps/files_texteditor/ajax/loadfile", LoadFile)
	protectedMux.HandleFunc("/index.php/apps/files_texteditor/ajax/savefile", SaveFile)

}
