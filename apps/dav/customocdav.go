package dav

import (
	"net/http"

	log "github.com/Sirupsen/logrus"
	"github.com/gowncloud/gowncloud/apps/dav/adapters"
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
		// session, ok := r.Context().Value("session").(identity.Session)
		// if !ok {
		// 	log.Error("could not get the session")
		// }
		switch r.Method {
		case "DELETE":
			ocdavadapters.DeleteAdapter(dav.dav.ServeHTTP, w, r)
		case "GET":
			ocdavadapters.GetAdapter(dav.dav.ServeHTTP, w, r)
		case "MKCOL":
			ocdavadapters.MkcolAdapter(dav.dav.ServeHTTP, w, r)
		case "PROPFIND":
			ocdavadapters.PropFindAdapter(dav.dav.ServeHTTP, w, r)
		default:
			dav.dav.ServeHTTP(w, r)
		}
	})
}
