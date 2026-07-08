//go:build !windows

package tui

import (
	"golang.org/x/sys/unix"
)

func diskUsage(root string) (total, free uint64, ok bool) {
	var stat unix.Statfs_t
	err := unix.Statfs(root, &stat)
	if err != nil {
		return 0, 0, false
	}
	total = stat.Blocks * uint64(stat.Bsize)
	free = stat.Bavail * uint64(stat.Bsize)
	if total == 0 {
		return 0, 0, false
	}
	return total, free, true
}
