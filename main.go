package main

import (
	"encoding/json"
	"net/http"
	"os"

	"github.com/codegangsta/cli"
	"github.com/gorilla/handlers"
	"github.com/gowncloud/gowncloud/apps/files/ajax"
	"github.com/gowncloud/gowncloud/core/oauth"
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
		os.Mkdir("testdir", os.ModePerm)
		server := webdav.Handler{
			Prefix:     "/remote.php/webdav",
			FileSystem: webdav.Dir("/dav"),
			LockSystem: webdav.NewMemLS(),
			Logger: func(r *http.Request, err error) {
				log.Printf("WEBDAV: %v, ERROR: %v", r, err)
				log.Println()
				log.Printf("additional info: %v", r.Context())
			},
		}
		server.FileSystem.Mkdir(nil, "test", os.ModeDir)
		server.FileSystem.OpenFile(nil, "test.txt", os.O_CREATE, os.ModeExclusive)
		http.HandleFunc("/remote.php/webdav", server.ServeHTTP)
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

		http.HandleFunc("/index.php/apps/files/ajax/getstoragestats.php", func(w http.ResponseWriter, r *http.Request) {
			log.Println("called storagestats")
			//try to mock some json responses
			data := &StorageStats{
				Data: Data{
					UploadMaxFileSize: 537919488,
					MaxHumanFilesize:  "Upload (max. 513 MB)",
					FreeSpace:         219945758720,
					UsedSpacePercent:  0,
					Owner:             "test",
					OwnerDisplayName:  "test",
				},
				Status: "success",
			}
			json.NewEncoder(w).Encode(data)
		})
		if err := http.ListenAndServe(":8080", handlers.LoggingHandler(os.Stdout, oauth.Protect(clientID, clientSecret, http.DefaultServeMux))); err != nil {
			log.Fatalf("server error: %v", nil)
		}
	}

	app.Run(os.Args)
}

type Data struct {
	UploadMaxFileSize int
	MaxHumanFilesize  string
	FreeSpace         int64
	UsedSpacePercent  int
	Owner             string
	OwnerDisplayName  string
}

type StorageStats struct {
	Data   Data
	Status string
}
