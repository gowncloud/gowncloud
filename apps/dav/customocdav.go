package dav

import (
	"net/http"
	"os"

	log "github.com/Sirupsen/logrus"
	"github.com/gowncloud/gowncloud/apps/dav/adapters"
	db "github.com/gowncloud/gowncloud/database"
	"golang.org/x/net/webdav"
)

// CustomOCDav is a wrapper around the stadard golang webdav implementation. It aims
// to mimic the standard owncloud webdav as much as possible, i.e. this implementation
// and the standard owncloud implementation should be interchangeable
// without an external client noticing
type CustomOCDav struct {
	dav          webdav.Handler
	filePathRoot string
}

// NewCustomOCDav initializes a new CustomOCDav. The root of the DAV server will
// be the given path.
func NewCustomOCDav(path string) *CustomOCDav {
	server := &CustomOCDav{
		filePathRoot: path,
		dav: webdav.Handler{
			Prefix:     "/remote.php/webdav",
			FileSystem: webdav.Dir(path),
			LockSystem: webdav.NewMemLS(),
			Logger: func(r *http.Request, err error) {
				log.Debug("Internal WEBDAV")
				if err != nil {
					log.Error("Method: ", r.Method)
					log.Error("Path: ", r.URL.Path)
					log.Errorf("WEBDAV ERROR: %v", err)
				}
			},
		},
	}
	return server
}

// DispatchRequest is the handler for incomming requests to the CustomOCDav. It checks
// the request method and then dispatches this request to the appropriate adapter.
// It is the responsibility of the adapter to make sure the generated response
// is compliant with the existing owncloud interfaces.
func (dav *CustomOCDav) DispatchRequest() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Debug("Method: ", r.Method)
		switch r.Method {
		case "DELETE":
			ocdavadapters.DeleteAdapter(dav.dav.ServeHTTP, w, r)
			break
		case "GET":
			ocdavadapters.GetAdapter(dav.dav.ServeHTTP, w, r)
			break
		case "MKCOL":
			ocdavadapters.MkcolAdapter(dav.dav.ServeHTTP, w, r)
			break
		case "MOVE":
			ocdavadapters.MoveAdapter(dav.dav.ServeHTTP, w, r)
		case "PROPFIND":
			ensureHomeDirectoryMiddleware(ocdavadapters.PropFindAdapter, dav.dav.ServeHTTP, w, r)
			break
		default:
			dav.dav.ServeHTTP(w, r)
			break
		}
	})
}

// MakeUserHomeDirectory creates the home directory for a user. The folder name is
// the username, and its parent folder is the webdavroot. It also creates the user
// in the database
func MakeUserHomeDirectory(username string) error {
	_, err := db.CreateUser(username)
	if err != nil {
		log.Errorf("Failed to create user %v: %v", username, err)
		return err
	}
	_, err = db.SaveNode(username, username, true, "dir")
	if err != nil {
		log.Errorf("Failed to make base directory for user %v: %v", username, err)
		return err
	}
	err = os.Mkdir(db.GetSetting(db.DAV_ROOT)+username, os.ModePerm)
	if err != nil {
		log.Errorf("Failed to make base directory for user %v: %v", username, err)
		return err
	}
	_, err = db.SaveNode(username+"/files", username, true, "dir")
	if err != nil {
		log.Errorf("Failed to make files directory for user %v: %v", username, err)
		return err
	}
	err = os.Mkdir(db.GetSetting(db.DAV_ROOT)+username+"/files", os.ModePerm)
	if err != nil {
		log.Errorf("Failed to make files directory for user %v: %v", username, err)
		return err
	}
	_, err = db.SaveNode(username+"/files_trash", username, true, "dir")
	if err != nil {
		log.Errorf("Failed to make trash directory for user %v: %v", username, err)
		return err
	}
	err = os.Mkdir(db.GetSetting(db.DAV_ROOT)+username+"/files_trash", os.ModePerm)
	if err != nil {
		log.Errorf("Failed to make trash directory for user %v: %v", username, err)
		return err
	}
	return nil
}
