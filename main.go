package main

import (
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/codegangsta/cli"

	"github.com/gorilla/mux"
	"github.com/gowncloud/gowncloud/apps/dav"
	"github.com/gowncloud/gowncloud/apps/files/ajax"
	files_sharing "github.com/gowncloud/gowncloud/apps/files_sharing/api"
	trash "github.com/gowncloud/gowncloud/apps/files_trashbin/ajax"
	"github.com/gowncloud/gowncloud/core"
	"github.com/gowncloud/gowncloud/core/identity"
	"github.com/gowncloud/gowncloud/core/logging"

	log "github.com/Sirupsen/logrus"

	db "github.com/gowncloud/gowncloud/database"
)

var version string

func main() {
	if version == "" {
		version = "Dev"
	}
	app := cli.NewApp()
	app.Name = "gowncloud"
	app.Version = version

	var debugLogging bool
	var bindAddress string
	var clientID, clientSecret string
	var dburl string
	var davroot string

	app.Flags = []cli.Flag{
		cli.BoolFlag{
			Name:        "debug, d",
			Usage:       "Enable debug logging",
			Destination: &debugLogging,
		},
		cli.StringFlag{
			Name:        "bind, b",
			Usage:       "Bind address",
			Value:       ":8080",
			Destination: &bindAddress,
		},
		cli.StringFlag{
			Name:        "clientid, c",
			Usage:       "OAuth2 clientid (required)",
			Destination: &clientID,
		},
		cli.StringFlag{
			Name:        "clientsecret, s",
			Usage:       "OAuth2 client secret (required)",
			Destination: &clientSecret,
		},
		cli.StringFlag{
			Name:        "databaseurl, db",
			Usage:       "Database connection url",
			Destination: &dburl,
		},
		cli.StringFlag{
			Name:        "dav-directory, dir",
			Usage:       "Dav root directory",
			Destination: &davroot,
		},
	}

	app.Before = func(c *cli.Context) error {
		if debugLogging {
			log.SetLevel(log.DebugLevel)
			log.Debug("Debug logging enabled")
		}
		return nil
	}

	app.Action = func(c *cli.Context) {

		if clientID == "" || clientSecret == "" {
			cli.ShowAppHelp(c)
			return
		}

		log.Infoln(app.Name, "version", app.Version)

		// init database connection
		parsedDbUrl, err := url.Parse(dburl)
		if err != nil {
			log.Fatal("failed to parse database url: ", err)
		}

		db.Connect("postgres", parsedDbUrl.String())
		defer db.Close()
		db.Initialize()

		// If the data-directory flag isn't set, use the previous or default directory
		if davroot == "" {
			davroot = db.GetSetting(db.DAV_ROOT)
		}

		// If the data-directory flag is set, but the user didn't end with a '/', append it
		// to maintain consistency
		if !strings.HasSuffix(davroot, "/") {
			davroot += "/"
		}

		// If the data-directory flag specifies another directory than the previously
		// used one or the default directory on first run, update the database to point
		// to this new directory
		if db.GetSetting(db.DAV_ROOT) != davroot {
			db.UpdateSetting(db.DAV_ROOT, davroot)
		}

		// Update the versionstring in the database if it changed
		if db.GetSetting(db.VERSION) != version {
			db.UpdateSetting(db.VERSION, version)
		}

		// make the dav root dir
		err = os.MkdirAll(davroot, os.ModePerm)
		if err != nil {
			log.Fatal("Failed to create dav root directory")
		}

		server := dav.NewCustomOCDav(davroot)

		http.Handle("/remote.php/webdav/", server.DispatchRequest())

		http.HandleFunc("/index.php", func(w http.ResponseWriter, r *http.Request) {
			s := identity.CurrentSession(r)
			renderTemplate(w, "index.html", &s)
		})
		r := mux.NewRouter()
		r.HandleFunc("/ocs/v2.php/apps/files_sharing/api/v1/shares", files_sharing.ShareInfo).Methods("GET")
		r.HandleFunc("/ocs/v2.php/apps/files_sharing/api/v1/shares", files_sharing.CreateShare).Methods("POST")
		r.HandleFunc("/ocs/v2.php/apps/files_sharing/api/v1/shares/{shareid}", files_sharing.DeleteShare).Methods("DELETE")
		http.Handle("/ocs/v2.php/apps/files_sharing/api/v1/shares", r)
		// FIXME: small hack for now to enale the shareid variable in the url for share deletes
		http.Handle("/ocs/v2.php/apps/files_sharing/api/v1/shares/", r)
		http.HandleFunc("/ocs/v1.php/apps/files_sharing/api/v1/sharees", files_sharing.Sharees)
		http.HandleFunc("/logout", func(w http.ResponseWriter, r *http.Request) {
			identity.ClearSession(w)
			//TODO: make a decent logged out page since now you will be redirected to itsyou.online for login again
			http.Redirect(w, r, "/", http.StatusFound)
		})
		http.Handle("/core/", http.StripPrefix("/core/", http.FileServer(http.Dir("core"))))
		http.Handle("/apps/dav/", http.StripPrefix("/apps/dav/", http.FileServer(http.Dir("apps/dav"))))
		http.Handle("/apps/federatedfilesharing/", http.StripPrefix("/apps/federatedfilesharing/", http.FileServer(http.Dir("apps/federatedfilesharing"))))
		http.Handle("/apps/files/css/", http.StripPrefix("/apps/files/css/", http.FileServer(http.Dir("apps/files/css"))))
		http.Handle("/apps/files/img/", http.StripPrefix("/apps/files/img/", http.FileServer(http.Dir("apps/files/img"))))
		http.Handle("/apps/files/js/", http.StripPrefix("/apps/files/js/", http.FileServer(http.Dir("apps/files/js"))))
		http.Handle("/apps/files_trashbin/css/", http.StripPrefix("/apps/files_trashbin/css/", http.FileServer(http.Dir("apps/files_trashbin/css"))))
		http.Handle("/apps/files_trashbin/img/", http.StripPrefix("/apps/files_trashbin/img/", http.FileServer(http.Dir("apps/files_trashbin/img"))))
		http.Handle("/apps/files_trashbin/js/", http.StripPrefix("/apps/files_trashbin/js/", http.FileServer(http.Dir("apps/files_trashbin/js"))))
		http.Handle("/settings/", http.StripPrefix("/settings/", http.FileServer(http.Dir("settings"))))
		http.Handle("/apps/files_sharing/", http.StripPrefix("/apps/files_sharing/", http.FileServer(http.Dir("apps/files_sharing"))))
		http.Handle("/index.php/", http.StripPrefix("/index.php/", http.FileServer(http.Dir("."))))
		http.HandleFunc("/index.php/apps/files/ajax/upload.php", files.Upload)

		http.HandleFunc("/index.php/apps/files/ajax/getstoragestats.php", files.GetStorageStats)
		http.HandleFunc("/index.php/core/preview.png", files.GetPreview)

		http.HandleFunc("/index.php/apps/files_trashbin/ajax/list.php", trash.GetTrash)
		http.HandleFunc("/index.php/apps/files_trashbin/ajax/delete.php", trash.DeleteTrash)
		http.HandleFunc("/index.php/apps/files_trashbin/ajax/undelete.php", trash.UndeleteTrash)

		http.HandleFunc("/status.php", core.Status)

		log.Infoln("Start listening on", bindAddress)
		if err := http.ListenAndServe(bindAddress, identity.AddIdentity(logging.Handler(os.Stdout, identity.Protect(clientID, clientSecret, http.DefaultServeMux)), clientID)); err != nil {
			log.Fatalf("server error: %v", err)
		}
	}

	app.Run(os.Args)
}
