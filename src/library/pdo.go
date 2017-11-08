package library

import (
    "database/sql"
    "os"
    "log"
    "reflect"
    "fmt"
    "time"
    _ "github.com/go-sql-driver/mysql"
)

type PDO struct {
    User         string
    Password     string
    Host         string
    Port         int
    DbName       string
    Charset      string

    db_handler    *sql.DB
    is_connected  bool
    columns_cache map[string] cloumns_cache_st//map[string] []Column
}

type cloumns_cache_st struct {
    time int64
    col []Column
}

type Column struct {
    TABLE_CATALOG            string
    TABLE_SCHEMA             string
    TABLE_NAME               string
    COLUMN_NAME              string
    ORDINAL_POSITION         int64
    COLUMN_DEFAULT           string
    IS_NULLABLE              bool
    DATA_TYPE                string
    CHARACTER_MAXIMUM_LENGTH int64
    CHARACTER_OCTET_LENGTH   int64
    NUMERIC_PRECISION        int64
    NUMERIC_SCALE            int64
    DATETIME_PRECISION       int64
    CHARACTER_SET_NAME       string
    COLLATION_NAME           string
    COLUMN_TYPE              string
    COLUMN_KEY               string
    EXTRA                    string
    PRIVILEGES               string
    COLUMN_COMMENT           string
    GENERATION_EXPRESSION    string
}

func NewPDO(user string, password string,
host string, port int, db_name string, charset string) *PDO {

    dns := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=%s", user, password, host, port, db_name, charset)
    db, err := sql.Open("mysql",dns)

    if err != nil {
        log.Println(err)
        os.Exit(1)
    }

    instance := &PDO{user, password, host, port, db_name, charset, db, true, make(map[string] cloumns_cache_st)}
    //instance.db_handler = db
    //instance.is_connected = true
    //instance.columns_cache = make(map[string] []Column)
    return instance
}

func (pdo *PDO) connect() {
    if pdo.is_connected {
        return
    }
}

func (pdo *PDO) setColumnsCache(key string, col []Column) {
    c := cloumns_cache_st{time.Now().Unix(), col}
    pdo.columns_cache[key] = c
}

func (pdo *PDO) getColumnsCache(key string) []Column {
    col, ok := pdo.columns_cache[key]
    if ok {
        //缓存60秒
        t := time.Now().Unix() - col.time//< 60
        if t < 60 {
            return col.col
        }
    }
    return nil
}

//获取表所有的字段
//返回Column类型的数组和err，如果有错误发生err则不为nil
func (pdo *PDO) GetColumns(db_name string, table_name string ) ([]Column, error)  {
    //使用缓存，避免频繁查询
    cache := pdo.getColumnsCache(db_name + "." + table_name)
    if cache != nil {
        return cache, nil
    }

    sql_str := "SELECT * FROM information_schema.columns WHERE table_schema = '" + db_name +
        "' AND table_name = '" + table_name + "'"
    rows, _:= pdo.db_handler.Query(sql_str)
    defer rows.Close()

    columns, err := rows.Columns()

    if err != nil {
        return nil, err
    }

    scan_args  := make([]interface{}, len(columns))
    values     := make([]interface{}, len(columns))
    for i := range values {
        scan_args[i] = &values[i]
    }

    columns_res := []Column{}
    for rows.Next() {
        if err := rows.Scan(scan_args...); err != nil {
            log.Println(err)
            return nil, err
        }

        column := Column{}
        column_v := reflect.ValueOf(column)
        for i, col := range values {
            if col != nil {
                switch col.(type) {
                case string:
                    val := string(col.(string))
                    if columns[i] == "IS_NULLABLE" {
                        if val == "YES" {
                            column_v.FieldByName(columns[i]).SetBool(true)
                        } else {
                            column_v.FieldByName(columns[i]).SetBool(false)
                        }

                    } else {
                        column_v.FieldByName(columns[i]).SetString(val)
                    }
                case []uint8:
                    column_v.FieldByName(columns[i]).SetString(string(col.([]byte)))
                case int:
                    column_v.FieldByName(columns[i]).SetInt(int64(col.(int)))
                case int64:
                    column_v.FieldByName(columns[i]).SetInt(int64(col.(int64)))
                }
            }
        }

        columns_res = append(columns_res, column)
    }

    if err := rows.Err(); err != nil {
        log.Println(err)
    }

    return columns_res, err
}


