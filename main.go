package main

import (
	"fmt"
	"test/timer"
	"time"
)

func main() {
	timer.PushTrigger(time.Now().Add(20*time.Second).Format("2006-01-02 15:04:05"), timer.Trigger{
		Fun: func(now int64, a interface{}) {
			tt := time.Unix(now, 0)
			fmt.Printf("now: %s, param: %v", tt.Format("2006-01-02 15:04:05"), a)
		},
		Param: "程序已启动20秒",
	})
	timer.PushTrigger(time.Now().Add(30*time.Second).Format("2006-01-02 15:04:05"), timer.Trigger{
		Fun: func(now int64, a interface{}) {
			tt := time.Unix(now, 0)
			fmt.Printf("now: %s, param: %v", tt.Format("2006-01-02 15:04:05"), a)
		},
		Param: "程序已启动30秒",
	})
	timer.Loop()
}
