package db

// 链接/查询mysql基本代码 重点不在这里

import (
	"database/sql"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"log"
	"strconv"
	"strings"
	"sync"
)

type SqlQuery struct {
	FcId   int
	Stmt   string
	Args   []any
	CbFunc func([]*DBData, error)
}

type MysqlPool struct {
	Inited    bool
	Db        *sql.DB
	m         sync.Mutex
	queryList chan *SqlQuery
}

type MysqlConf struct {
	Username   string `xml:"user_name" json:"user_name"`
	Password   string `xml:"password" json:"password"`
	RemoteIp   string `xml:"remote_ip" json:"remote_ip"`
	RemotePort int    `xml:"remote_port" json:"remote_port"`
	DbName     string `xml:"db_name" json:"db_name"`
}

type DBData struct {
	Data map[string][]byte // key-库表的列名，value-这条数据的这一列的值（用[]byte表示，之后在上层转化为需要的类型如protobuf的Unmarshal）
}

func NewMysqlPool() *MysqlPool {
	return &MysqlPool{
		Inited:    false,
		Db:        nil,
		m:         sync.Mutex{},
		queryList: nil,
	}
}

func (mysql *MysqlPool) InitMysqlPool(conf *MysqlConf) {
	if mysql.Inited {
		fmt.Println("InitMysqlPool failed: Mysql Inited")
		return
	}

	mysql.m.Lock()
	defer mysql.m.Unlock()

	var err error
	mysql.Db, err = sql.Open("mysql", conf.Username+":"+conf.Password+"@tcp("+conf.RemoteIp+":"+strconv.Itoa(conf.RemotePort)+")/"+conf.DbName)
	if err != nil {
		fmt.Println("Init Mysql error: " + err.Error())
		return
	}
	mysql.Db.SetMaxOpenConns(1)
	mysql.Db.SetMaxIdleConns(1)
	err = mysql.Db.Ping()
	if err != nil {
		fmt.Println("Init Mysql error: " + err.Error())
		return
	}
	mysql.queryList = make(chan *SqlQuery, 10)
	mysql.Inited = true
	log.Printf("init mysql pool success")
}

func (mysql *MysqlPool) ReleaseMysqlPool() {
	if !mysql.Inited {
		fmt.Println("ReleaseMysqlPool failed: Mysql not inited")
		return
	}

	mysql.m.Lock()
	defer mysql.m.Unlock()

	mysql.Db.Close()
	close(mysql.queryList)
	mysql.Inited = false
	log.Printf("release mysql pool success")
}

func (mysql *MysqlPool) Loop() {
	for mysql.Inited {
		q := <-mysql.queryList
		if q == nil {
			log.Printf("Loop detected nil ptr")
			continue
		}
		log.Printf("query received, stmt = %s, args = %v", q.Stmt, q.Args)
		sqlType := strings.ToLower(strings.Split(q.Stmt, " ")[0])
		switch sqlType {
		case "select":
			result, err := mysql.Query(q.Stmt, q.Args...)
			q.CbFunc(result, err)
		case "insert":
			fallthrough
		case "update":
			fallthrough
		case "delete":
			fallthrough
		case "replace":
			err := mysql.Exec(q.Stmt, q.Args)
			q.CbFunc(nil, err)
		default:
			log.Printf("illegal mysql operation type %s", sqlType)
			continue
		}
	}
}

var db = NewMysqlPool()

func GetDbPool() *MysqlPool {
	return db
}

// Query 需要确保sql里是1条查询语句，如果有多条select，需要循环rows.NextResultSet遍历所有结果集（建议憋搞那么复杂）
func (mysql *MysqlPool) Query(sql string, args ...any) (result []*DBData, err error) {
	if !mysql.Inited {
		fmt.Println("Query failed: Mysql not inited")
		return
	}
	mysql.m.Lock()
	defer mysql.m.Unlock()
	rows, err := mysql.Db.Query(sql, args...)
	if err != nil {
		return
	}
	defer rows.Close()
	columns, _ := rows.Columns()

	for rows.Next() {
		b := &DBData{
			Data: make(map[string][]byte),
		}
		buff := make([]interface{}, len(columns))
		scanners := make([][]byte, len(columns))
		for i, _ := range buff {
			buff[i] = &scanners[i]
		}
		if err := rows.Scan(buff...); err != nil {
			log.Fatal(err)
		}
		for i, data := range scanners {
			b.Data[columns[i]] = data
		}
		result = append(result, b)
	}
	return
}

func (mysql *MysqlPool) Exec(sql string, args ...any) (err error) {
	if !mysql.Inited {
		fmt.Println("Query failed: Mysql not inited")
		return
	}
	mysql.m.Lock()
	defer mysql.m.Unlock()
	_, err = mysql.Db.Exec(sql, args...)
	return
}

func (mysql *MysqlPool) AddQuery(query *SqlQuery) {
	mysql.queryList <- query
}
