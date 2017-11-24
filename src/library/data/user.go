package data

import (
    "library/file"
    "database/sql"
    _ "github.com/mattn/go-sqlite3"
    log "library/log"
)
var user_data_path string
func init() {
    // 默认的数据目录
    data_path := file.GetCurrentPath() + "/data"

    // 如果不存在，尝试创建
    wpath := &file.WPath{data_path}
    wpath.Mkdir()

    // db文件
    user_data_path = data_path+"/wing.db"
    wfile := &file.WFile{user_data_path}

    // 如果不存在，尝试创建
    if !wfile.Exists() {
        create := "CREATE TABLE `userinfo` (`id` INTEGER PRIMARY KEY AUTOINCREMENT,`username` VARCHAR(64) NULL,`password` VARCHAR(64) NULL,`created` TIMESTAMP default (datetime('now', 'localtime')));"
        db, err := sql.Open("sqlite3", data_path + "/wing.db")
        defer db.Close()
        if err != nil {
            log.Println("sqlite3 open error", err)
            return
        }
        _, err = db.Exec(create)
        if err != nil {
            log.Println("sqlite3 exec error", err)
            return
        }
        // 添加默认用户
        u := User{"admin", "admin"}
        u.Add()
    }
}

type User struct{
    Name string
    Password string
}

// 添加用户
func (user *User) Add() bool {
    if user.Get() {
        log.Println("用户已存在", user.Name)
        return false
    }
    db, err := sql.Open("sqlite3", user_data_path)
    if err != nil {
        log.Println("2-open sqlite3 error", err)
        return false
    }
    defer db.Close()

    //插入数据
    stmt, err := db.Prepare("INSERT INTO userinfo(username, password) values(?,?)")
    if err != nil {
        log.Println("2-open sqlite3 error", err)
        return false
    }
    defer stmt.Close()

    res, err := stmt.Exec(user.Name, user.Password)
    if err != nil {
        log.Println("2-open sqlite3 error", err)
        return false
    }

    id, err := res.LastInsertId()
    if err != nil {
        log.Println("2-open sqlite3 error", err)
        return false
    }

    log.Println(id)
    return true
}

// 查询用户是否存在
// 存在返回true，不存在返回false
func (user *User) Get() bool {

    db, err := sql.Open("sqlite3", user_data_path)
    if err != nil {
        log.Println("2-open sqlite3 error", err)
        return false
    }
    defer db.Close()

    sql_str := "SELECT id, username, password, strftime('%Y-%m-%d %H:%M:%S',created) as created FROM userinfo WHERE username = ? AND password = ?"
    stmt, err:= db.Prepare(sql_str)
    if err != nil {
        log.Println(err)
        return false
    }

    defer stmt.Close()

    res := stmt.QueryRow(user.Name, user.Password)
    var (
        id int64
        username string
        password string
        daytime string
    )

    err = res.Scan(&id, &username, &password, &daytime)
    if err != nil {
        return false
    }

    log.Println(id, username, password, daytime)
    return true
}

// 更新用户
func (user *User) Update(id int64) bool {
    if !user.Get() {
        return false
    }

    db, err := sql.Open("sqlite3", user_data_path)
    if err != nil {
        log.Println("2-open sqlite3 error", err)
        return false
    }
    defer db.Close()

    //插入数据
    stmt, err := db.Prepare("update userinfo set username=?, password=? where id=?")
    if err != nil {
        log.Println("2-open sqlite3 error", err)
        return false
    }
    defer stmt.Close()

    res, err := stmt.Exec(user.Name, user.Password, id)
    if err != nil {
        log.Println("2-open sqlite3 error", err)
        return false
    }

    num, err := res.RowsAffected()
    if err != nil {
        log.Println("2-open sqlite3 error", err)
        return false
    }

    log.Println(num)
    return num > 0
}


// 删除用户
func (user *User) Delete() bool {
    if !user.Get() {
        return false
    }

    db, err := sql.Open("sqlite3", user_data_path)
    if err != nil {
        log.Println("2-open sqlite3 error", err)
        return false
    }
    defer db.Close()

    //插入数据
    stmt, err := db.Prepare("delete from userinfo where username=?")
    if err != nil {
        log.Println("2-open sqlite3 error", err)
        return false
    }
    defer stmt.Close()

    res, err := stmt.Exec(user.Name)
    if err != nil {
        log.Println("2-open sqlite3 error", err)
        return false
    }

    num, err := res.RowsAffected()
    if err != nil {
        log.Println("2-open sqlite3 error", err)
        return false
    }

    log.Println(num)
    return num > 0
}