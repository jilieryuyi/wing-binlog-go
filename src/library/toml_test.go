package library

import (
    "testing"
    "reflect"
    "log"
)

func TestWConfig_Parse(t *testing.T) {
    config_file := "/tmp/__test_mysql.toml"
    config := &WConfig{config_file}

    app_config, err:= config.Parse()

    if err != nil {
        t.Error(err)
        return
    }

    /**
     slave_id     = 9999
    ignore_table = ["Test.abc", "Test.123"]
    bin_file     = ""
    bin_pos      = 0
     */
    if app_config.Client.Slave_id != 9999 {
        t.Error("config parse error - 1")
    }

    log.Println("===>", reflect.TypeOf(app_config.Client.Ignore_tables).String(), "<===")
    //switch  {
    //case []string:
        for _, v := range  app_config.Client.Ignore_tables {
            //log.Println(v)
            if v != "Test.abc" && v != "Test.123" {
                t.Error("config parse error - 2")
            }
        }
   // default:
      //  t.Error("config parse error - 3")
    //}

    if app_config.Client.Bin_file != "" {
        t.Error("config parse error - 4")
    }

    if app_config.Client.Bin_pos != 0 {
        t.Error("config parse error - 5")
    }


    /**
        host     = "127.0.0.1"
        user     = "root"
        password = "123456"
        port     = 3306
        charset  = "utf8"
     */
    if app_config.Mysql.Host != "127.0.0.1" {
        t.Error("config parse error - 6")
    }
    if app_config.Mysql.User != "root" {
        t.Error("config parse error - 7")
    }
    if app_config.Mysql.Password != "123456" {
        t.Error("config parse error - 8")
    }
    if app_config.Mysql.Port != 3306 {
        t.Error("config parse error - 9")
    }
    if app_config.Mysql.Charset != "utf8" {
        t.Error("config parse error - 10")
    }

    if app_config.Mysql.DbName != "wordpress" {
        t.Error("config parse error - 11")
    }
}
