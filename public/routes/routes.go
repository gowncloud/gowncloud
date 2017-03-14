package routes

import (
	"net/http"

	log "github.com/Sirupsen/logrus"
	core_assets "github.com/gowncloud/gowncloud/public/assets/core"
	federatedfilesharing_assets "github.com/gowncloud/gowncloud/public/assets/federatedfilesharing"
	files_assets "github.com/gowncloud/gowncloud/public/assets/files"
	files_sharing_assets "github.com/gowncloud/gowncloud/public/assets/files_sharing"
	files_trashbin_assets "github.com/gowncloud/gowncloud/public/assets/files_trashbin"
	files_videoplayer_assets "github.com/gowncloud/gowncloud/public/assets/files_videoplayer"
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
}
