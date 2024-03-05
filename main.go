package main

import (
	"encoding/xml"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"test/db"
	"test/timer"
	"test/tool_gen_code"
	"time"
)

type ServerConf struct {
	MysqlConf *db.MysqlConf `xml:"mysql" json:"mysql"`
}

func main() {
	if err := tool_gen_code.Gen(); err != nil {
		panic(err)
	}
	confFile, err := os.ReadFile("configs/main_conf.xml")
	if err != nil {
		panic(fmt.Sprintf("Server start failed in read config: %s", err.Error()))
	}
	conf := &ServerConf{}
	err = xml.Unmarshal(confFile, conf)
	if err != nil {
		panic(fmt.Sprintf("Server start failed in main_conf.xml unmarshal error: %s", err.Error()))
	}
	db.GetDbPool().InitMysqlPool(conf.MysqlConf)
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
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
	//signal.Notify(c)
	// SIGINT(interrupt): kill -2 非守护进程模式下敲ctrl+C属于此列。
	// SIGKILL(kill): kill -9 没有遗言的强杀（捕捉不到的信号，进程直接寄，下面receive signal日志都不会打印，所以在notify里注册也没什么用，可以不写）。不要乱用。Goland的停止按钮疑似SIGKILL（debug没抓到）
	// SIGTERM(terminate): kill -15 有遗言的退出。kill命令默认值，外部一般发这个指令杀进程（所以上面notify要指定SIGTERM）。
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
