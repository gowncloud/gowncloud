package files

import (
	"net/http"

	log "github.com/Sirupsen/logrus"
	files "github.com/gowncloud/gowncloud/apps/files/ajax"
)

func RegisterRoutes(protectedMux *http.ServeMux, publicMux *http.ServeMux) {
	log.Debug("Regestering files routes")

	protectedMux.HandleFunc("/index.php/apps/files/ajax/upload.php", files.Upload)

	protectedMux.HandleFunc("/index.php/apps/files/api/v1/files/", files.Favorite)

	protectedMux.HandleFunc("/index.php/apps/files/api/v1/tags/_$!<Favorite>!$_/files", files.ListFavorites)

	protectedMux.HandleFunc("/index.php/apps/files/ajax/getstoragestats.php", files.GetStorageStats)
	protectedMux.HandleFunc("/index.php/core/preview.png", files.GetPreview)

	protectedMux.HandleFunc("/index.php/apps/files/api/v1/thumbnail/", files.GetThumbnail)

	protectedMux.HandleFunc("/index.php/apps/files/ajax/download.php", files.Download)

}
