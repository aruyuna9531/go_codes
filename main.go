package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"test/db"
	"test/panic_recover"
	"test/timer"
	"time"
)

func main() {
	defer panic_recover.PanicRecoverTrace()
	conf := db.MysqlConf{
		Username:   "root",
		Password:   "mysql",
		RemoteIp:   "localhost",
		RemotePort: 3306,
		DbName:     "test_db",
	}
	db.GetDbPool().InitMysqlPool(conf)
	defer db.GetDbPool().ReleaseMysqlPool()
	Loop()
}

func Loop() {
	timer.TimerTestCode()
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGINT, syscall.SIGKILL, syscall.SIGTERM)
	//signal.Notify(c)
	// SIGINT: kill -2 ctrl+C属于此列。
	// SIGKILL: kill -9 没有遗言的强杀（捕捉不到的信号，可能进程没了还会遗留一些现象，比如打点计时器还在跑）。不要乱用。Goland的停止按钮疑似SIGKILL（debug没抓到）
	// SIGTERM: kill -15 有遗言的退出
	tk := time.NewTicker(1 * time.Second)
	looping := true
	for looping {
		select {
		case t := <-tk.C:
			fmt.Printf("now: %s\n", t.Format("2006-01-02 15:04:05"))
			timer.GetInst().Trigger(t.Format("2006-01-02 15:04:05"))
		case sig := <-c:
			if sig == syscall.SIGWINCH || sig == syscall.SIGURG {
				continue
			}
			log.Printf("receive signal %v, exit program", sig.String())
			looping = false
			close(c)
			tk.Stop()
			break
		}
	}
}
