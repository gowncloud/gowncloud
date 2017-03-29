package files_sharing

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/gorilla/mux"

	"github.com/gowncloud/gowncloud/core/identity"
	db "github.com/gowncloud/gowncloud/database"
)

type meta struct {
	Status     string  `json:"status"`
	StatusCode int     `json:"statuscode"`
	Message    *string `json:"message"`
}

type shareesdata struct {
	Exact   exact    `json:"exact"`
	Groups  []ocuser `json:"groups"`
	Remotes []string `json:"remotes"`
	Users   []ocuser `json:"users"`
}

type sharedata struct {
	Displayname_file_owner string     `json:"displayname_file_owner"`
	Displayname_owner      string     `json:"displayname_owner"`
	Expiration             *time.Time `json:"expiration"` // null unless link?
	File_parent            float64    `json:"file_parent"`
	File_source            float64    `json:"file_source"` // nodeId?
	File_target            string     `json:"file_target"` // without username leading, start with slash
	Id                     string     `json:"id"`          // shareId?
	Item_source            float64    `json:"item_source"` // same as file source
	Item_type              string     `json:"item_type"`   // "file" or ...
	Mail_send              int        `json:"mail_send"`   // leave at 0 for now, could be bool
	Mimetype               string     `json:"mimetype"`
	Parent                 *string    `json:"parent"` // leave at null for now
	Path                   string     `json:"path"`   // same as file target? without leading username, start with slash
	Permissions            int        `json:"permissions"`
	Share_type             int        `json:"share_type"`
	Share_with             string     `json:"share_with"`             // sharee
	Share_with_displayname string     `json:"share_with_displayname"` // sharee
	Stime                  int64      `json:"stime"`                  // share time in seconds since epoch
	Storage                string     `json:"storage"`                // leave at "1" for now
	Storage_id             string     `json:"storage_id"`             // "home::user" where user is the system user?
	Token                  *string    `json:"token"`                  // null
	Uid_file_owner         string     `json:"uid_file_owner"`
	Uid_owner              string     `json:"uid_owner"`
}

type exact struct {
	Groups  []string `json:"groups"`
	Remotes []string `json:"remotes"`
	Users   []string `json:"users"`
}

type ocuser struct {
	Label string `json:"label"`
	Value value  `json:"value"`
}

type value struct {
	ShareType int    `json:"shareType"`
	ShareWith string `json:"shareWith"`
}

type ocs struct {
	Meta meta      `json:"meta"`
	Data sharedata `json:"data"`
}

type ocsinfo struct {
	Meta meta        `json:"meta"`
	Data []sharedata `json:"data"`
}

type ocsSharee struct {
	Meta meta        `json:"meta"`
	Data shareesdata `json:"data"`
}

// ShareInfo returns info on current shares for a given node
// It is the endpoint for GET /ocs/v2.php/apps/files_sharing/api/v1/shares
func ShareInfo(w http.ResponseWriter, r *http.Request) {
	ocsResponse := struct {
		Ocs ocsinfo `json:"ocs"`
	}{}
	data := make([]sharedata, 0)
	ocsResponse.Ocs.Meta.StatusCode = 200

	ocsResponse.Ocs.Meta.Status = "ok"
	ocsResponse.Ocs.Meta.Message = nil

	err := r.ParseForm()
	if err != nil {
		log.Error("Failed to parse POST form")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	sharedWithMe := r.URL.Query().Get("shared_with_me")

	// TODO: investigate reshares
	if sharedWithMe != "true" {
		shares, err := db.GetSharesByNodePath(identity.CurrentSession(r).Username + "/files" + r.FormValue("path"))

		if err != nil {
			log.Error("Failed to get shares")
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
			sd, err := makeShareData(node, share, share.Target)
			if err != nil {
				log.Warn("Failed to make share data")
				continue
			}
			data = append(data, *sd)
		}
	}

	ocsResponse.Ocs.Data = data

	w.Header().Add("Content-Type", "application/json")
	json.NewEncoder(w).Encode(&ocsResponse)
}

// CreateShare makes a new share on the given node
// It is the endpoint for POST /ocs/v2.php/apps/files_sharing/api/v1/shares
func CreateShare(w http.ResponseWriter, r *http.Request) {
	log.Debug("Creating share")
	err := r.ParseForm()
	if err != nil {
		log.Error("Failed to parse POST form")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	shareTypeString := r.FormValue("shareType")
	shareType, err := strconv.Atoi(shareTypeString)
	if err != nil {
		log.Warnf("Could not convert %v to integer", shareTypeString)
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}
	if !(shareType == db.GROUPSHARE || shareType == db.USERSHARE) {
		log.Warn("Invalid share type: ", shareType)
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}
	shareNode, err := db.GetNode(identity.CurrentSession(r).Username + "/files" + r.FormValue("path"))
	if err != nil {
		log.Error("Failed to get node from DB")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if shareNode == nil {
		http.Error(w, "No node at path "+r.FormValue("path"), http.StatusNotFound)
		return
	}
	permissionsString := r.FormValue("permissions")
	permissions, err := strconv.Atoi(permissionsString)
	if err != nil {
		log.Error("Failed to parse permissions")
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	target := r.FormValue("shareWith")
	share, err := db.CreateShare(shareNode.ID, permissions, target, shareType)
	if err != nil {
		log.Error("Failed to save share")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	response := struct {
		Ocs ocs `json:"ocs"`
	}{}

	response.Ocs.Meta.StatusCode = 200
	response.Ocs.Meta.Status = "ok"
	response.Ocs.Meta.Message = nil

	data, err := makeShareData(shareNode, share, target)
	if err != nil {
		log.Error("Failed to make share data")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	response.Ocs.Data = *data

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(&response)
}

// Shares is the endpoint the /ocs/v1.php/apps/files_sharing/api/v1/shares
func Sharees(w http.ResponseWriter, r *http.Request) {
	search := r.URL.Query().Get("search")

	// searchMatch keeps track whether we found the search input. If it isn't found,
	// add it to the groups so the UI doesn't throw a not found autmatically
	searchMatch := false

	usernames, err := db.SearchUserNames(search)
	if err != nil {
		log.Error("Error searching for usernames")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	response := struct {
		Ocs ocsSharee `json:"ocs"`
	}{}
	response.Ocs.Meta.Message = nil
	response.Ocs.Meta.StatusCode = 100
	response.Ocs.Meta.Status = "ok"

	//response.Ocs.Data.Groups = make([]string, 0)
	response.Ocs.Data.Remotes = make([]string, 0)

	response.Ocs.Data.Exact.Groups = make([]string, 0)
	response.Ocs.Data.Exact.Remotes = make([]string, 0)
	response.Ocs.Data.Exact.Users = make([]string, 0)

	users := make([]ocuser, 0)

	for _, username := range usernames {
		userstruct := ocuser{}
		userstruct.Label = username
		userstruct.Value = value{
			ShareType: 0,
			ShareWith: username,
		}
		users = append(users, userstruct)
		if username == search {
			searchMatch = true
		}
	}

	response.Ocs.Data.Users = users

	// Organization preview from the request.
	orgs := identity.CurrentSession(r).Organizations

	groups := make([]ocuser, 0)
	for _, org := range orgs {
		if strings.HasPrefix(org, search) {
			g := ocuser{
				Label: org,
				Value: value{
					ShareType: 1,
					ShareWith: org,
				},
			}
			groups = append(groups, g)
			if org == search {
				searchMatch = true
			}
		}
	}

	// Always return 1 result so we can share to groups we are not part of
	if !searchMatch {
		s := ocuser{
			Label: search,
			Value: value{
				ShareType: 1,
				ShareWith: search,
			},
		}
		groups = append(groups, s)
	}

	response.Ocs.Data.Groups = groups

	w.Header().Add("Content-Type", "application/json")
	json.NewEncoder(w).Encode(&response)
}

// DeleteShare deletes a share on a node
// It is the endpoint for DELETE /ocs/v2.php/apps/files_sharing/api/v1/shares/{shareid}
func DeleteShare(w http.ResponseWriter, r *http.Request) {
	shareIdString := mux.Vars(r)["shareid"]

	shareId, err := strconv.ParseFloat(shareIdString, 64)
	if err != nil {
		log.Error("Error parsing shareId: ", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	err = db.DeleteShare(shareId)
	if err != nil {
		log.Error("Error deleting share: ", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// makeShareData generates the shareData struct for a share
func makeShareData(shareNode *db.Node, share *db.Share, target string) (*sharedata, error) {
	item_type := "file"
	if shareNode.Isdir {
		item_type = "folder"
	}

	parent, err := db.GetNode(shareNode.Path[:strings.LastIndex(shareNode.Path, "/")])
	if err != nil {
		log.Error("Failed to get system user")
		return nil, err
	}

	storage_id := "home::" + shareNode.Owner

	data := sharedata{
		Displayname_file_owner: shareNode.Owner,
		Displayname_owner:      shareNode.Owner,
		Expiration:             nil,
		File_parent:            parent.ID,
		File_source:            shareNode.ID,
		File_target:            strings.TrimPrefix(shareNode.Path, shareNode.Owner+"/files"),
		Id:                     strconv.FormatFloat(share.ShareID, 'e', -1, 64),
		Item_source:            shareNode.ID,
		Item_type:              item_type,
		Mail_send:              0,
		Mimetype:               shareNode.MimeType,
		Parent:                 nil,
		Path:                   strings.TrimPrefix(shareNode.Path, shareNode.Owner+"/files"),
		Permissions:            share.Permissions,
		Share_type:             share.ShareType,
		Share_with:             target,
		Share_with_displayname: target,
		Stime:          share.Time.Unix(),
		Storage:        "1",
		Storage_id:     storage_id, // FIXME
		Token:          nil,
		Uid_file_owner: shareNode.Owner,
		Uid_owner:      shareNode.Owner,
	}

	return &data, nil
}

// parseIntToSize parses a string to an int and makes sure it fits in the specified
// amount of bits
func parseIntToSize(input string, bitsize int) (int, error) {
	for {
		i, err := strconv.ParseInt(input, 10, bitsize)
		if err != nil {
			nErr, ok := err.(*strconv.NumError)
			if !ok {
				return 0, err
			}
			if nErr.Err != strconv.ErrRange {
				return 0, err
			}
			err = nil
		} else {
			return int(i), nil
		}
		input = input[1:]
	}
}
