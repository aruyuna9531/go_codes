package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"test/db"
	"test/timer"
	"time"
)

func main() {
	conf := db.MysqlConf{
		Username:   "root",
		Password:   "123456",
		RemoteIp:   "localhost",
		RemotePort: 3306,
		DbName:     "test",
	}
	db.GetDbPool().InitMysqlPool(conf)
	defer db.GetDbPool().ReleaseMysqlPool()
	go db.GetDbPool().Loop()

	go db.GetDbPool().AddQuery(&db.SqlQuery{
		FcId: 1,
		Stmt: "select * from test_table where id = ?;",
		Args: []any{1},
		CbFunc: func(data []*db.DBData, err error) {
			if err != nil {
				log.Println(err.Error())
				return
			}
			log.Printf("%v", data[0].Data)
		},
	})
	Loop()
}

func Loop() {
	timer.TimerTestCode()
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM, syscall.SIGUSR1)
	//signal.Notify(c)
	// SIGINT: kill -2 ctrl+C属于此列。
	// SIGKILL: kill -9 没有遗言的强杀（捕捉不到的信号，所以在这里写也没什么用）。不要乱用。Goland的停止按钮疑似SIGKILL（debug没抓到）
	// SIGTERM: kill -15 有遗言的退出
	tk := time.NewTicker(1 * time.Second)
	defer tk.Stop()
	looping := true
	for looping {
		select {
		case sig, ok := <-c:
			if !ok {
				log.Println("error by receiving channel signal")
				continue
			}
			log.Printf("receive signal %v, exit program", sig.String())
			looping = false
			close(c)
		case t, ok := <-tk.C:
			if !ok {
				continue
			}
			fmt.Printf("now: %s\n", t.Format("2006-01-02 15:04:05.000"))
			timer.GetInst().Trigger(t.Format("2006-01-02 15:04:05"))
		}
	}
}
