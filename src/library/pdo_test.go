package library

import (
    "testing"
    "reflect"
    "log"
    _ "github.com/go-sql-driver/mysql"
)

func TestPDO_NewPDO(t *testing.T) {
    instance := NewPDO("root", "123456", "127.0.0.1", 3306, "wordpress", "utf8")
    v := reflect.ValueOf(instance)

    if !v.IsValid() {
        t.Error("pdo init error")
    }
}

func TestPDO_GetColumns(t *testing.T) {
    instance := NewPDO("root", "123456", "127.0.0.1", 3306, "wordpress", "utf8")

    columns, err:= instance.GetColumns("xsl", "x_fee")

    if err != nil {
        t.Error("get columns error happened")
    }

    log.Println(columns)
    for _, v := range columns {
        log.Println(v.TABLE_SCHEMA, v.TABLE_NAME, v.COLUMN_NAME)
    }

    columns, err = instance.GetColumns("xsl", "x_fee")

    if err != nil {
        t.Error("get columns error happened - 2")
    }

    log.Println(columns)
    for _, v := range columns {
        log.Println(v.TABLE_SCHEMA, v.TABLE_NAME, v.COLUMN_NAME)
    }

}
