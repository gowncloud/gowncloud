// +build !windows

package files

import (
	"syscall"

	log "github.com/Sirupsen/logrus"
)

// getFreeDiskSpace calculates the free space from the disk root
func getFreeDiskSpace() (int64, error) {
	log.Debug("Calculating free storage space")

	var stats syscall.Statfs_t

	// Get space of the root drive
	err := syscall.Statfs("/", &stats)
	if err != nil {
		return 0, err
	}
	space := int64(stats.Bavail) * stats.Bsize // unused blocks * the size of one block
	log.Debugf("available space: %v bytes", space)
	return space, nil
}
