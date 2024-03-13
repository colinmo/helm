//go:build windows
// +build windows

package tasks

import (
	"os"
	"syscall"
	"time"
)

func FileCreateTime(fi os.FileInfo) time.Time {
	return time.Unix(0, int64(fi.Sys().(*syscall.Win32FileAttributeData).CreationTime.Nanoseconds()))
}
