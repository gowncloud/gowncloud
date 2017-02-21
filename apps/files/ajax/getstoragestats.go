package files

import (
	"encoding/json"
	"net/http"

	log "github.com/Sirupsen/logrus"
	"github.com/gowncloud/gowncloud/core/identity"
	db "github.com/gowncloud/gowncloud/database"
)

type Data struct {
	UploadMaxFileSize int    `json:"uploadMaxFilesize"`
	MaxHumanFilesize  string `json:"maxHumanFilesize"`
	FreeSpace         int64  `json:"freeSpace"`
	UsedSpacePercent  int    `json:"usedSpacePercent"`
	Owner             string `json:"owner"`
	OwnerDisplayName  string `json:"ownerDisplayName"`
}

type StorageStats struct {
	Data   Data   `json:"data"`
	Status string `json:"status"`
}

func GetStorageStats(w http.ResponseWriter, r *http.Request) {
	log.Debug("getting storage stats")

	username := identity.CurrentSession(r).Username
	user, err := db.GetUser(username)
	if err != nil {
		log.Error("Failed to get user from db: ", err)
		w.WriteHeader(http.StatusInternalServerError)
	}

	var freeSpace int64
	usedSpacePercent := 0
	if user.Allowedspace != 0 {
		size, err := getDirSize(db.GetSetting(db.DAV_ROOT) + username)
		if err != nil {
			log.Error("Failed to get directory size: ", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		// Allowedspace is stored as GB
		freeSpace = int64(user.Allowedspace)<<30 - size
		usedSpacePercent = int(100 * (freeSpace / int64(user.Allowedspace) << 30))
	}

	diskSpace := getFreeDiskSpace()
	if user.Allowedspace == 0 {
		freeSpace = diskSpace
	} else {
		if diskSpace < freeSpace {
			freeSpace = diskSpace
		}
	}

	data := &StorageStats{
		Data: Data{
			UploadMaxFileSize: 537919488,
			MaxHumanFilesize:  "Upload (max. 513 MB)",
			FreeSpace:         freeSpace,
			UsedSpacePercent:  usedSpacePercent,
			Owner:             username,
			OwnerDisplayName:  username,
		},
		Status: "success",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}
