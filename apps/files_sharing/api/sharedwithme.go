package files_sharing

import (
	"encoding/json"
	"net/http"

	log "github.com/Sirupsen/logrus"
	"github.com/gowncloud/gowncloud/core/identity"
	db "github.com/gowncloud/gowncloud/database"
)

// SharedWithMe gets info about all the nodes shared to this user
func SharedWithMe(w http.ResponseWriter, r *http.Request) {

	username := identity.CurrentSession(r).Username
	groups := identity.CurrentSession(r).Organizations

	ocsResponse := struct {
		Ocs ocsinfo `json:"ocs"`
	}{}
	data := make([]sharedata, 0)
	ocsResponse.Ocs.Meta.StatusCode = 100

	ocsResponse.Ocs.Meta.Status = "ok"
	ocsResponse.Ocs.Meta.Message = nil

	shares, err := db.GetAllSharesToUser(username, groups)
	if err != nil {
		log.Error("Could not get shares for user: ", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	for _, share := range shares {
		node, err := db.GetSharedNode(share.ShareID)
		if err != nil {
			log.Error("Failed to get shared node")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		// Exclude users own files when shared with group
		if node.Owner == username {
			continue
		}
		sd, err := makeShareData(node, share, share.Target)
		if err != nil {
			log.Error("Failed to make share data")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		data = append(data, *sd)
	}

	ocsResponse.Ocs.Data = data

	w.Header().Add("Content-Type", "application/json")
	json.NewEncoder(w).Encode(&ocsResponse)
}
