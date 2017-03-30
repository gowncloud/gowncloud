package main

import (
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/codegangsta/cli"

	"github.com/gowncloud/gowncloud/apps/dav"
	files_routes "github.com/gowncloud/gowncloud/apps/files/routes"
	sharing_routes "github.com/gowncloud/gowncloud/apps/files_sharing/routes"
	"github.com/gowncloud/gowncloud/apps/files_texteditor"
	trash_routes "github.com/gowncloud/gowncloud/apps/files_trashbin/routes"
	gallery_routes "github.com/gowncloud/gowncloud/apps/gallery/routes"
	core_routes "github.com/gowncloud/gowncloud/core/routes"
	"github.com/gowncloud/gowncloud/core/search"

	"github.com/gowncloud/gowncloud/core/identity"
	"github.com/gowncloud/gowncloud/core/logging"
	"github.com/gowncloud/gowncloud/public/routes"

	log "github.com/Sirupsen/logrus"

	db "github.com/gowncloud/gowncloud/database"
)

var version string

func main() {
	if version == "" {
		version = "0.0-Dev"
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

		defaultMux := http.NewServeMux()
		publicMux := http.NewServeMux()

		server := dav.NewCustomOCDav(davroot)

		defaultMux.Handle("/remote.php/webdav/", dav.NormalizePath(server.DispatchRequest()))

		defaultMux.HandleFunc("/index.php", func(w http.ResponseWriter, r *http.Request) {
			s := identity.CurrentSession(r)
			renderTemplate(w, "index.html", &s)
		})

		defaultMux.HandleFunc("/index.php/apps/files/", func(w http.ResponseWriter, r *http.Request) {
			s := identity.CurrentSession(r)
			renderTemplate(w, "index.html", &s)
		})

		defaultMux.HandleFunc("/logout", func(w http.ResponseWriter, r *http.Request) {
			identity.ClearSession(w)
			//TODO: make a decent logged out page since now you will be redirected to itsyou.online for login again
			http.Redirect(w, r, "/", http.StatusFound)
		})

		routes.RegisterRoutes(defaultMux, publicMux)
		files_routes.RegisterRoutes(defaultMux, publicMux)
		trash_routes.RegisterRoutes(defaultMux, publicMux)
		sharing_routes.RegisterRoutes(defaultMux, publicMux)
		core_routes.RegisterRoutes(defaultMux, publicMux)
		search.RegisterRoutes(defaultMux, publicMux)
		gallery_routes.RegisterRoutes(defaultMux, publicMux)
		files_texteditor.RegisterRoutes(defaultMux, publicMux)

		rootMux := http.NewServeMux()
		rootMux.Handle("/", identity.AddIdentity(logging.Handler(os.Stdout, identity.Protect(clientID, clientSecret, defaultMux)), clientID))
		rootMux.Handle("/status.php", publicMux)

		log.Infoln("Start listening on", bindAddress)
		if err := http.ListenAndServe(bindAddress, rootMux); err != nil {
			log.Fatalf("server error: %v", err)
		}
	}

	app.Run(os.Args)
}
