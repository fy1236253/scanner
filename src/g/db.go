package g

import (
	"database/sql"
	"log"
	"sync"

	_ "github.com/go-sql-driver/mysql" //mysql 相关配置
)

var (
	dbLock    sync.RWMutex
	dbConnMap map[string]*sql.DB
	//DB mysql链接
	DB *sql.DB
)

// InitDB 初始化DB
func InitDB() {
	var err error
	DB, err = makeDBConn()
	if DB == nil || err != nil {
		log.Println("g.InitDB,get db fail")
	}
	dbConnMap = make(map[string]*sql.DB)
	log.Println("g.InitDB ok")
}
func makeDBConn() (conn *sql.DB, err error) {
	conn, err = sql.Open("mysql", Config().DB.Dsn)
	if err != nil {
		return nil, err
	}
	conn.SetMaxIdleConns(Config().DB.MaxIdle)
	err = conn.Ping()
	return conn, err
}

func closeDBConn(conn *sql.DB) {
	if conn != nil {
		conn.Close()
	}
}

//GetDBConn 获取DB链接
func GetDBConn(connName string) (c *sql.DB, e error) {
	dbLock.Lock()
	defer dbLock.Unlock()
	var err error
	var dbConn *sql.DB
	dbConn = dbConnMap[connName]
	if dbConn == nil {
		dbConn, err = makeDBConn()
		if err != nil || dbConn == nil {
			closeDBConn(dbConn)
			return nil, err
		}
		dbConnMap[connName] = dbConn
	}
	err = dbConn.Ping()
	if err != nil {
		closeDBConn(dbConn)
		delete(dbConnMap, connName)
		return nil, err
	}
	return dbConn, err
}
