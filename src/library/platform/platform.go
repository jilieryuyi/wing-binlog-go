package platform

import "runtime"

const (
	IS_WINDOWS   = 1
	IS_LINUX     = 2
	IS_MAC       = 3
	IS_UNIX      = 4 //freebsd
	IS_FREEBSD   = 4
	IS_ANDROID   = 5
	IS_DRAGONFLY = 6
	IS_NACL      = 7
	IS_NETBSD    = 8
	IS_OPENBSD   = 9
	IS_PLAN9     = 10
	IS_SOLARIS   = 11
)

func System(par int) bool {
	switch par {
	case IS_LINUX:
		return "linux" == runtime.GOOS
	case IS_MAC:
		return "darwin" == runtime.GOOS
	case IS_UNIX:
		return "freebsd" == runtime.GOOS
	case IS_WINDOWS:
		return "windows" == runtime.GOOS
	case IS_ANDROID:
		return "android" == runtime.GOOS
	case IS_DRAGONFLY:
		return "dragonfly" == runtime.GOOS
	case IS_NACL:
		return "ncal" == runtime.GOOS
	case IS_NETBSD:
		return "netbsd" == runtime.GOOS
	case IS_OPENBSD:
		return "openbsd" == runtime.GOOS
	case IS_PLAN9:
		return "plan9" == runtime.GOOS
	case IS_SOLARIS:
		return "solaris" == runtime.GOOS
	}

	return false
}
