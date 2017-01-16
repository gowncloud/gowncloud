package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"

	files "github.com/gowncloud/gowncloud/apps/files/ajax"

	"golang.org/x/net/webdav"
)

func main() {
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
	if err := http.ListenAndServe(":8080", logger(http.DefaultServeMux)); err != nil {
		log.Fatalf("server error: %v", nil)
	}
}

func logger(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("request url: %v", r.URL)
		handler.ServeHTTP(w, r)
	})
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
