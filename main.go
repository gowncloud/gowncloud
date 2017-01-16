package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"

	"golang.org/x/net/webdav"
)

func main() {
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
	http.HandleFunc("/index.php/apps/files/ajax/upload.php", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("called files/ajax/upload.php")
		w.WriteHeader(http.StatusOK)
		//try to mock some json responses
		body := UploadResponse{
			Directory:         "/",
			Etag:              "f384b2d8d0b5ce097ec3fe40dc45b799",
			Id:                95,
			MaxHumanFilesize:  "513 MB",
			Mimetype:          "image/png",
			Mtime:             1484566972000,
			Name:              "apple-touch-icon-57x57.png",
			Originalname:      "apple-touch-icon-57x57.png",
			ParentId:          2,
			Permissions:       27,
			Size:              2499,
			Status:            "success",
			Sort:              "file",
			UploadMaxFilesize: 537919488,
		}
		json.NewEncoder(w).Encode(body)
	})
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

type UploadResponse struct {
	Directory         string
	Etag              string
	Id                int
	MaxHumanFilesize  string
	Mimetype          string
	Mtime             int64
	Name              string
	Originalname      string
	ParentId          int
	Permissions       int
	Size              int
	Status            string
	Sort              string `json:"type"`
	UploadMaxFilesize int
}
