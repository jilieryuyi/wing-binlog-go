package std

import (
	"library/path"
	"os"
)

func Reset() error {
	dir := path.GetCurrentPath() + "/logs"
	logs_dir := &path.WPath{dir}

	if !logs_dir.Exists() {
		os.Mkdir(dir, 0755)
	}

	handle_stdout, err := os.OpenFile(dir+"/stdout.log", os.O_WRONLY|os.O_CREATE|os.O_SYNC|os.O_APPEND, 0755)
	os.Stdout = handle_stdout

	if err != nil {
		return err
	}

	handle_stderr, err := os.OpenFile(dir+"/stderr.log", os.O_WRONLY|os.O_CREATE|os.O_SYNC|os.O_APPEND, 0755)
	os.Stderr = handle_stderr

	if err != nil {
		return err
	}

	return nil
}
