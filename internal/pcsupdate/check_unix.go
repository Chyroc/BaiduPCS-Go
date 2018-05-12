// +build !windows

package pcsupdate

import (
	"syscall"

	"github.com/iikira/BaiduPCS-Go/pcsutil"
)

func checkWritable() bool {
	return syscall.Access(pcsutil.ExecutablePath(), syscall.O_RDWR) == nil
}
