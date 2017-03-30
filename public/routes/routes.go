package routes

import (
	"bytes"
	"net/http"

	log "github.com/Sirupsen/logrus"
	"github.com/gowncloud/gowncloud/public/assets/components"
	core_assets "github.com/gowncloud/gowncloud/public/assets/core"
	federatedfilesharing_assets "github.com/gowncloud/gowncloud/public/assets/federatedfilesharing"
	files_assets "github.com/gowncloud/gowncloud/public/assets/files"
	files_sharing_assets "github.com/gowncloud/gowncloud/public/assets/files_sharing"
	files_texteditor_assets "github.com/gowncloud/gowncloud/public/assets/files_texteditor"
	files_trashbin_assets "github.com/gowncloud/gowncloud/public/assets/files_trashbin"
	files_videoplayer_assets "github.com/gowncloud/gowncloud/public/assets/files_videoplayer"
	gallery_assets "github.com/gowncloud/gowncloud/public/assets/gallery"
	settings_assets "github.com/gowncloud/gowncloud/public/assets/settings"

	"github.com/gowncloud/gowncloud/tools/assetfs"
)

func RegisterRoutes(protectedMux *http.ServeMux, publicMux *http.ServeMux) {
	log.Debug("Registering asset routes")

	protectedMux.Handle("/core/", http.FileServer(&assetfs.AssetFS{Asset: core_assets.Asset, AssetDir: core_assets.AssetDir, AssetInfo: core_assets.AssetInfo}))
	protectedMux.Handle("/index.php/core/", http.StripPrefix("/index.php/", http.FileServer(&assetfs.AssetFS{Asset: core_assets.Asset, AssetDir: core_assets.AssetDir, AssetInfo: core_assets.AssetInfo})))

	protectedMux.Handle("/apps/federatedfilesharing/", http.FileServer(&assetfs.AssetFS{Asset: federatedfilesharing_assets.Asset, AssetDir: federatedfilesharing_assets.AssetDir, AssetInfo: federatedfilesharing_assets.AssetInfo}))

	protectedMux.Handle("/apps/files/css/", http.FileServer(&assetfs.AssetFS{Asset: files_assets.Asset, AssetDir: files_assets.AssetDir, AssetInfo: files_assets.AssetInfo}))
	protectedMux.Handle("/apps/files/img/", http.FileServer(&assetfs.AssetFS{Asset: files_assets.Asset, AssetDir: files_assets.AssetDir, AssetInfo: files_assets.AssetInfo}))
	protectedMux.Handle("/apps/files/js/", http.FileServer(&assetfs.AssetFS{Asset: files_assets.Asset, AssetDir: files_assets.AssetDir, AssetInfo: files_assets.AssetInfo}))

	protectedMux.Handle("/apps/files_trashbin/css/", http.FileServer(&assetfs.AssetFS{Asset: files_trashbin_assets.Asset, AssetDir: files_trashbin_assets.AssetDir, AssetInfo: files_trashbin_assets.AssetInfo}))
	protectedMux.Handle("/apps/files_trashbin/img/", http.FileServer(&assetfs.AssetFS{Asset: files_trashbin_assets.Asset, AssetDir: files_trashbin_assets.AssetDir, AssetInfo: files_trashbin_assets.AssetInfo}))
	protectedMux.Handle("/apps/files_trashbin/js/", http.FileServer(&assetfs.AssetFS{Asset: files_trashbin_assets.Asset, AssetDir: files_trashbin_assets.AssetDir, AssetInfo: files_trashbin_assets.AssetInfo}))

	protectedMux.Handle("/settings/", http.FileServer(&assetfs.AssetFS{Asset: settings_assets.Asset, AssetDir: settings_assets.AssetDir, AssetInfo: settings_assets.AssetInfo}))

	protectedMux.Handle("/apps/files_sharing/", http.FileServer(&assetfs.AssetFS{Asset: files_sharing_assets.Asset, AssetDir: files_sharing_assets.AssetDir, AssetInfo: files_sharing_assets.AssetInfo}))

	protectedMux.Handle("/apps/files_videoplayer/", http.FileServer(&assetfs.AssetFS{Asset: files_videoplayer_assets.Asset, AssetDir: files_videoplayer_assets.AssetDir, AssetInfo: files_videoplayer_assets.AssetInfo}))

	protectedMux.Handle("/apps/files_texteditor/", http.FileServer(&assetfs.AssetFS{Asset: files_texteditor_assets.Asset, AssetDir: files_texteditor_assets.AssetDir, AssetInfo: files_texteditor_assets.AssetInfo}))

	protectedMux.Handle("/apps/gallery/", http.FileServer(&assetfs.AssetFS{Asset: gallery_assets.Asset, AssetDir: gallery_assets.AssetDir, AssetInfo: gallery_assets.AssetInfo}))

	protectedMux.HandleFunc("/index.php/apps/gallery/slideshow", func(w http.ResponseWriter, r *http.Request) {
		filename := "public/components/slideshow.html"
		file, err := components.Asset(filename)
		if err != nil {
			log.Error("Failed to get slideshow.html embedded asset: ", err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
		fi, err := components.AssetInfo(filename)
		if err != nil {
			log.Error("Failed to get info of embedded asset slideshow.html: ", err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
		http.ServeContent(w, r, filename, fi.ModTime(), bytes.NewReader(file))
	})
}
