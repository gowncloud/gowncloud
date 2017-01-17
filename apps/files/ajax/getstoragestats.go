package files

import (
	"encoding/json"
	"net/http"

	log "github.com/Sirupsen/logrus"
)

type Data struct {
	UploadMaxFileSize int
	MaxHumanFilesize  string
	FreeSpace         int64
	UsedSpacePercent  int
	Owner             string
	OwnerDisplayName  string
}

type StorageStats struct {
	Data   Data
	Status string
}

func GetStorageStats(w http.ResponseWriter, r *http.Request) {
	log.Debug("getting storage stats")

	// use a fake response for now.
	data := &StorageStats{
		Data: Data{
			UploadMaxFileSize: 537919488,
			MaxHumanFilesize:  "Upload (max. 513 MB)",
			FreeSpace:         219945758720,
			UsedSpacePercent:  0,
			Owner:             "test",
			OwnerDisplayName:  "test",
		},
		Status: "success",
	}

	json.NewEncoder(w).Encode(data)
}
