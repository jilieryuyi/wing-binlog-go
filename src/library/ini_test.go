package library

import (
    "testing"
    _ "fmt"
)

func TestParse(t *testing.T) {

    config_path := "/tmp/mysql.ini"

    file := &WFile{config_path}
    if !file.Exists() {
        t.Error("file does not exists: " + config_path)
        return
    }

    ini  := Ini{config_path}
    data := ini.Parse()

    if data == nil {
        t.Error("ini config Parse error - 1")
        return
    }

    slave_id := string(data["client"]["slave_id"].(string))
    if (slave_id != "9999") {
        t.Error("get client slave_id error")
    }
}
