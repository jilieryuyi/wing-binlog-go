// +build  !linux,!darwin,windows

package app

import "library/file"

func DaemonProcess(d bool) bool {
	return false
}

func Release() {
	// delete pid when exit
	file.Delete(Pid)
}