package files

import (
	"bytes"
	"image"
	"net/http"
	"strconv"

	log "github.com/Sirupsen/logrus"
	"github.com/disintegration/imaging"
	"github.com/gowncloud/gowncloud/core/identity"
)

// GetPreview generates a preview for an image file and serves it to the client
func GetPreview(w http.ResponseWriter, r *http.Request) {
	username := identity.CurrentSession(r).Username

	query := r.URL.Query()

	filePath := query.Get("file")
	width, err := strconv.Atoi(query.Get("x"))
	if err != nil {
		log.Error("Failed to read width")
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	height, err := strconv.Atoi(query.Get("y"))
	if err != nil {
		log.Error("Failed to read height")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	file := "testdir/" + username + filePath

	var preview *image.NRGBA
	img, err := imaging.Open(file)
	if err != nil {
		log.Warn("Failed to open file as image")
		w.WriteHeader(http.StatusNotFound)
		return
	}
	preview = imaging.Thumbnail(img, width, height, imaging.Lanczos)

	var buffer bytes.Buffer
	err = imaging.Encode(&buffer, preview, imaging.GIF)
	if err != nil {
		log.Warn("Failed to encode preview")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-type", "image/gif")
	w.WriteHeader(http.StatusFound)
	w.Write(buffer.Bytes())

}
