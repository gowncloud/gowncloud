package files

import "fmt"

// getFreeDiskSpace calculates the free space from the disk root. This is the windows
// implementation
func getFreeDiskSpace() int64 {
	return 0, fmt.Errorf("getFreeDiskSpace is not yet implemented on windows!")
}
