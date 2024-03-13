//go:build !windows
// +build !windows

package tasks

import (
	"os"
	"syscall"
	"time"
)

func FileCreateTime(fi os.FileInfo) time.Time {
	return time.Unix(int64(fi.Sys().(*syscall.Stat_t).Birthtimespec.Sec), int64(fi.Sys().(*syscall.Stat_t).Birthtimespec.Nsec))
}
