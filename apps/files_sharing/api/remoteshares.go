package files_sharing

import (
	"encoding/json"
	"net/http"
)

// RemoteShares is currently a stub to statisfy the UI
func RemoteShares(w http.ResponseWriter, r *http.Request) {
	ocsResponse := struct {
		Ocs ocsinfo `json:"ocs"`
	}{}
	data := make([]sharedata, 0)
	ocsResponse.Ocs.Meta.StatusCode = 100

	ocsResponse.Ocs.Meta.Status = "ok"
	ocsResponse.Ocs.Meta.Message = nil

	ocsResponse.Ocs.Data = data

	w.Header().Add("Content-Type", "application/json")
	json.NewEncoder(w).Encode(&ocsResponse)
}
