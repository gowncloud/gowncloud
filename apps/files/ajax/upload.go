package files

import (
	"encoding/json"
	"io"
	"math/rand"
	"net/http"
	"os"
	"time"

	log "github.com/Sirupsen/logrus"
)

type UploadResponse struct {
	Directory         string `json:"directory"`
	Etag              string `json:"etag"`
	Id                int    `json:"id"`
	MaxHumanFilesize  string `json:"maxHumanFilesize"`
	Mimetype          string `json:"mimetype"`
	Mtime             int64  `json:"mtime"`
	Name              string `json:"name"`
	Originalname      string `json:"originalname"`
	ParentId          int    `json:"parentId"`
	Permissions       int    `json:"permissions"`
	Size              int    `json:"size"`
	Status            string `json:"status"`
	Sort              string `json:"type"`
	UploadMaxFilesize int    `json:"uploadMaxFilesize"`
}

func Upload(w http.ResponseWriter, r *http.Request) {
	log.Debug("called files/ajax/upload.php")

	if r.Method != "POST" {
		log.Printf("Used the unsupported %v method", r.Method)
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	// TODO: is this required?
	err := r.ParseMultipartForm(1 << 29) // reserve 2^29 bytes = 536870912B / 512MB
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	body := []UploadResponse{}

	for _, fileHeaders := range r.MultipartForm.File {
		for _, file := range fileHeaders {
			// Open the upload file
			upload, err := file.Open()
			if err != nil {
				log.Errorf("failed to open upload file: %v", file.Filename)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			// Create the upload target
			target, err := os.Create("testdir/" + file.Filename)
			if err != nil {
				log.Errorf("failed to open target file: %v", "testdir/"+file.Filename)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			// Buffered copy
			written, err := io.Copy(target, upload)
			if err != nil {
				log.Errorf("failed to copy upload file")
				// TODO: clean up target
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			log.Debug("copied %v bytes", written)

			targetStats, err := target.Stat()
			if err != nil {
				log.Error("failed to get stats")
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			// Create the response
			data := UploadResponse{
				Directory:         "/",
				Etag:              "adfafdlasdfafdsaf", // TODO: need database support?
				Id:                rand.Int(),          // TODO: need database support?
				MaxHumanFilesize:  "512MB",
				Mimetype:          file.Header.Get("mimetype"),
				Mtime:             int64(time.Now().Unix()) * 1000, // the upload time aka Now
				Name:              target.Name(),
				Originalname:      file.Filename,
				ParentId:          2,
				Permissions:       27,
				Size:              int(targetStats.Size()), // cast to int should be removed if we allow files bigger than 2GB
				Status:            "success",
				Sort:              "file",
				UploadMaxFilesize: 2 << 29,
			}
			body = append(body, data)
		}
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(body)
}
