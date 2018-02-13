// +build  linux,darwin,!windows

package app

import (
	"library/path"
	"os"
	log "github.com/sirupsen/logrus"
	"github.com/sevlyar/go-daemon"
)

var ctx *daemon.Context = nil
// run as daemon process
func DaemonProcess(d bool) bool {
	if d {
		log.Debugf("run as daemon process")
		params := []string{os.Args[0], "-daemon", "-config-path", ConfigPath}
		//os.Args[0] + " -daemon -config-path " +  ConfigPath
		log.Debugf("params: %v", params)
		ctx = &daemon.Context{
			PidFileName: Pid,
			PidFilePerm: 0644,
			LogFileName: LogPath + "/wing-binlog-go.log",
			LogFilePerm: 0640,
			WorkDir:     path.CurrentPath,
			Umask:       027,
			Args:        params,//[]string{params},
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

func Release() {
	// delete pid when exit
	file.Delete(Pid)
	if ctx != nil {
		// release process context when exit
		ctx.Release()
	}
}

