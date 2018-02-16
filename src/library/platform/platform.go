package platform

import "runtime"

const (
	IS_WINDOWS   = "windows"
	IS_LINUX     = "linux"
	IS_MAC       = "darwin"
	IS_UNIX      = "freebsd" //freebsd
	IS_FREEBSD   = "freebsd"
	IS_ANDROID   = "android"
	IS_DRAGONFLY = "dragonfly"
	IS_NACL      = "ncal"
	IS_NETBSD    = "netbsd"
	IS_OPENBSD   = "openbsd"
	IS_PLAN9     = "plan9"
	IS_SOLARIS   = "solaris"
)

func System(par string) bool {
	return par == runtime.GOOS
}
