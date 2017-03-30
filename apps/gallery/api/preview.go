package gallery

import (
	"net/http"
	"strconv"

	log "github.com/Sirupsen/logrus"
	"github.com/gorilla/mux"
	db "github.com/gowncloud/gowncloud/database"
	"github.com/gowncloud/gowncloud/image"
)

// Preview renders the image identified by the id
func Preview(w http.ResponseWriter, r *http.Request) {
	fileId, err := strconv.ParseFloat(mux.Vars(r)["id"], 64)
	if err != nil {
		log.Error("Failed to parse file id: ", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	q := r.URL.Query()
	widthString := q.Get("width")
	heightString := q.Get("height")

	node, err := db.GetNodeById(fileId)
	if err != nil {
		log.Error("Failed to get node: ", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	path := node.Path

	renderImage(widthString, heightString, path, w)
}

func renderImage(maxWidthString string, maxHeightString string, path string, w http.ResponseWriter) {

	file := db.GetSetting(db.DAV_ROOT) + path

	image.RenderImage(maxWidthString, maxHeightString, file, w)
}
