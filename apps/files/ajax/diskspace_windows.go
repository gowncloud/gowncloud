package files

import log "github.com/Sirupsen/logrus"

// getFreeDiskSpace calculates the free space from the disk root. This is the windows
// implementation
func getFreeDiskSpace() int64 {
	log.Error("getFreeDiskSpace is not yet implemented on windows!")
	return -1
}
