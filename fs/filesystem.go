package fs

import (
	"path/filepath"

	"golang.org/x/net/webdav"
)

type FileSystem interface {
	webdav.FileSystem
	Walk(string, filepath.WalkFunc, error) error
}
