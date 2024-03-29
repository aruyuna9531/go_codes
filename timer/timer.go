package timer

import (
	"fmt"
	"time"
)

type Trigger struct {
	Fun   func(int64, interface{})
	Param interface{}
	Now   int64
}

type Timer struct {
	triggers map[int64][]Trigger //TODO ←这里实际上用的是有序列表，有时间再手撸
}

func (t *Timer) PushTimerTrigger(at string, trigger Trigger) { // TODO at应为时间戳 跟上面的todo一起做
	if t.triggers == nil {
		t.triggers = make(map[int64][]Trigger)
	}
	tt, err := time.ParseInLocation("2006-01-02 15:04:05", at, time.Local)
	if err != nil {
		panic(err)
	}
	trigger.Now = tt.Unix()
	t.triggers[tt.Unix()] = append(t.triggers[tt.Unix()], trigger)
}

func (t *Timer) Trigger(now string) {
	tt, err := time.ParseInLocation("2006-01-02 15:04:05", now, time.Local)
	if err != nil {
		panic(err)
	}
	ts, ok := t.triggers[tt.Unix()]
	if !ok {
		return
	}
	for _, trigger := range ts {
		trigger.Fun(trigger.Now, trigger.Param)
	}
	delete(t.triggers, tt.Unix())
}

var tm = &Timer{
	triggers: map[int64][]Trigger{},
}

func GetInst() *Timer {
	return tm
}

func PushTrigger(at string, trigger Trigger) {
	tm.PushTimerTrigger(at, trigger)
}

func TimerTestCode() {
	PushTrigger(time.Now().Add(20*time.Second).Format("2006-01-02 15:04:05"), Trigger{
		Fun: func(now int64, a interface{}) {
			tt := time.Unix(now, 0)
			fmt.Printf("now: %s, param: %v", tt.Format("2006-01-02 15:04:05"), a)
		},
		Param: "程序已启动20秒",
	})
	PushTrigger(time.Now().Add(30*time.Second).Format("2006-01-02 15:04:05"), Trigger{
		Fun: func(now int64, a interface{}) {
			tt := time.Unix(now, 0)
			fmt.Printf("now: %s, param: %v", tt.Format("2006-01-02 15:04:05"), a)
		},
		Param: "程序已启动30秒",
	})
}
