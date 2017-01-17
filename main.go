package main

import (
	"net/http"
	"os"

	"github.com/codegangsta/cli"
	"github.com/gowncloud/gowncloud/apps/files/ajax"
	"github.com/gowncloud/gowncloud/core/identity"
	"github.com/gowncloud/gowncloud/core/logging"
	"golang.org/x/net/webdav"

	log "github.com/Sirupsen/logrus"
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

	app.Flags = []cli.Flag{
		cli.BoolFlag{
			Name:        "debug, d",
			Usage:       "Enable debug logging",
			Destination: &debugLogging,
		},
		cli.StringFlag{
			Name:        "bind, b",
			Usage:       "Bind address",
			Value:       ":8443",
			Destination: &bindAddress,
		},
		cli.StringFlag{
			Name:        "clientid, c",
			Usage:       "OAuth2 clientid",
			Destination: &clientID,
		},
		cli.StringFlag{
			Name:        "clientsecret, s",
			Usage:       "OAuth2 client secret",
			Destination: &clientSecret,
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

		log.Infoln(app.Name, "version", app.Version)
		// make the dir for uploaded files
		// TODO: check if directory exists first. If it doesnt exist, make it
		// TODO: use a better directory (at least not a relative path)
		os.Mkdir("testdir", os.ModePerm)
		server := webdav.Handler{
			Prefix:     "/remote.php/webdav",
			FileSystem: webdav.Dir("testdir"),
			LockSystem: webdav.NewMemLS(),
			Logger: func(r *http.Request, err error) {
				log.Debug("WEBDAV")
				if err != nil {
					log.Errorf("WEBDAV: %v, ERROR: %v", r, err)
					log.Warnf("additional info: %v", r.Context())
				}
			},
		}
		// TODO: Check if dav filesystem works as intended
		// server.FileSystem.Mkdir(nil, "test", os.ModeDir)
		// server.FileSystem.OpenFile(nil, "test/test.txt", os.O_CREATE, os.ModeAppend)
		http.HandleFunc("/remote.php/webdav/", server.ServeHTTP)
		http.HandleFunc("/index.php", func(w http.ResponseWriter, r *http.Request) {
			http.ServeFile(w, r, "index.html")
		})
		http.Handle("/core/", http.StripPrefix("/core/", http.FileServer(http.Dir("core"))))
		http.Handle("/apps/dav/", http.StripPrefix("/apps/dav/", http.FileServer(http.Dir("apps/dav"))))
		http.Handle("/apps/federatedfilesharing/", http.StripPrefix("/apps/federatedfilesharing/", http.FileServer(http.Dir("apps/federatedfilesharing"))))
		http.Handle("/apps/files/css/", http.StripPrefix("/apps/files/css/", http.FileServer(http.Dir("apps/files/css"))))
		http.Handle("/apps/files/img/", http.StripPrefix("/apps/files/img/", http.FileServer(http.Dir("apps/files/img"))))
		http.Handle("/apps/files/js/", http.StripPrefix("/apps/files/js/", http.FileServer(http.Dir("apps/files/js"))))
		http.Handle("/settings/", http.StripPrefix("/settings/", http.FileServer(http.Dir("settings"))))
		http.Handle("/index.php/", http.StripPrefix("/index.php/", http.FileServer(http.Dir("."))))
		http.HandleFunc("/index.php/apps/files/ajax/upload.php", files.Upload)

		http.HandleFunc("/index.php/apps/files/ajax/getstoragestats.php", files.GetStorageStats)
		if err := http.ListenAndServe(":8080", identity.AddIdentity(logging.Handler(os.Stdout, identity.Protect(clientID, clientSecret, http.DefaultServeMux)))); err != nil {
			log.Fatalf("server error: %v", nil)
		}
	}

	app.Run(os.Args)
}
