package tui

import (
	"golang.org/x/sys/windows"
)

func diskUsage(root string) (total, free uint64, ok bool) {
	rootPtr, err := windows.UTF16PtrFromString(root)
	if err != nil {
		return 0, 0, false
	}
	var freeBytes, totalBytes uint64
	err = windows.GetDiskFreeSpaceEx(rootPtr, &freeBytes, &totalBytes, nil)
	if err != nil || totalBytes == 0 {
		return 0, 0, false
	}
	return totalBytes, freeBytes, true
}
