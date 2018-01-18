package ini

import (
	_ "fmt"
	//"library/platform"
	"testing"
)

func TestParse(t *testing.T) {

	/*var config_path string = ""
	if platform.System(platform.IS_WINDOWS) {
		config_path = "C:\\__test_mysql.ini"
	} else {
		config_path = "/tmp/__test_mysql.ini"
	}

	file := &WFile{config_path}
	if !file.Exists() {
		t.Error("file does not exists: " + config_path)
		return
	}

	ini := Ini{config_path}
	data := ini.Parse()

	if data == nil {
		t.Error("ini config Parse error - 1")
		return
	}

	slave_id := string(data["client"]["slave_id"].(string))
	if slave_id != "9999" {
		t.Error("get client slave_id error")
	}*/
}
