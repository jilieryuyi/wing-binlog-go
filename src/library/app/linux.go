// +build  linux,386 darwin,!windows

package app

import (
	"library/path"
	"library/file"
	log "github.com/sirupsen/logrus"
	"github.com/sevlyar/go-daemon"
	"os"
	"strings"
)

var ctx *daemon.Context = nil
// run as daemon process
func DaemonProcess(d bool) bool {
	if d {
		log.Debugf("run as daemon process")
		exeFile := strings.Replace(os.Args[0], "\\", "/", -1)
		fileName := exeFile
		lastIndex := strings.LastIndex(exeFile, "/")
		if lastIndex > -1 {
			fileName = exeFile[lastIndex + 1:]
		}
		params := []string{path.CurrentPath + "/" + fileName, "-daemon", "-config-path=" + ConfigPath}
		log.Debugf("params: %v", params)
		ctx = &daemon.Context{
			PidFileName: Pid,
			PidFilePerm: 0644,
			LogFileName: "",//LogPath + "/wing-binlog-go.log",
			LogFilePerm: 0640,
			WorkDir:     path.CurrentPath,
			Umask:       027,
			Args:        params,
		}
		d, err := ctx.Reborn()
		if err != nil {
			log.Fatal("Unable to run: ", err)
		}
		if d != nil {
			return true
		}
		return false
	}
	return false
}

// use for main func, defer app.Release
// release ctx resource
func Release() {
	// delete pid when exit
	file.Delete(Pid)
	if ctx != nil {
		// release process context when exit
		ctx.Release()
	}
}

