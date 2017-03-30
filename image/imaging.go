package image

import (
	"bytes"
	"image"
	"net/http"
	"strconv"

	log "github.com/Sirupsen/logrus"
	"github.com/disintegration/imaging"
)

// generatePreview generates an image preview from a file
func GeneratePreview(widthString, heightString, filePath string, w http.ResponseWriter) {

	width, err := strconv.Atoi(widthString)
	if err != nil {
		log.Error("Failed to read width: ", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	height, err := strconv.Atoi(heightString)
	if err != nil {
		log.Error("Failed to read height: ", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	var preview *image.NRGBA
	img, err := imaging.Open(filePath)
	if err != nil {
		if err != image.ErrFormat {
			log.Error("Failed to open file as image: ", err)
			w.WriteHeader(http.StatusNotFound)
			return
		}
		log.Debug("Could not render preview: file is not a supported image format")
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
	w.WriteHeader(http.StatusOK)
	w.Write(buffer.Bytes())
}

// RenderImage renders the given image and resizes it to the designated size if it's
// too large. Images are encoded as JPEGs when they are send to the client
func RenderImage(maxWidthString string, maxHeightString string, path string, w http.ResponseWriter) {
	width, err := strconv.Atoi(maxWidthString)
	if err != nil {
		log.Error("Failed to read width: ", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	height, err := strconv.Atoi(maxHeightString)
	if err != nil {
		log.Error("Failed to read height: ", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	var preview *image.NRGBA
	img, err := imaging.Open(path)
	if err != nil {
		log.Warn("Failed to open file as image: ", err)
		w.WriteHeader(http.StatusNotFound)
		return
	}
	// Resize image if too large
	preview = imaging.Fit(img, width, height, imaging.Lanczos)

	var buffer bytes.Buffer
	// For now encode everything we send as a jpeg as this seems to be the most efficient way
	err = imaging.Encode(&buffer, preview, imaging.JPEG)
	if err != nil {
		log.Warn("Failed to encode preview")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-type", "image/jpeg")
	w.Header().Set("Content-length", strconv.Itoa(len(buffer.Bytes())))
	w.WriteHeader(http.StatusOK)
	w.Write(buffer.Bytes())
}
