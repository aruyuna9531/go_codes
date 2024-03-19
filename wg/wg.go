package wg

import (
	"context"
	"log"
	"sync"
	"sync/atomic"
	"time"
)

type Mgr struct {
	fcId atomic.Uint32
	w    sync.WaitGroup
}

func (m *Mgr) Add(f func() <-chan struct{}, timeLimit time.Duration) {
	m.w.Add(1)
	go func() {
		defer m.w.Done()
		ctx, cf := context.WithTimeout(context.Background(), timeLimit)
		defer cf()
		realFcId := m.fcId.Add(1)
		select {
		case <-ctx.Done():
			log.Printf("fcId %d not ok, reason: %s\n", realFcId, ctx.Err().Error())
		case <-f(): // 这个写法有点问题 f阻塞5秒之后还能报ok
			log.Printf("fcId %d ok\n", realFcId)
		}
	}()
}

func (m *Mgr) Wait() {
	m.w.Wait()
}

var mgr = &Mgr{}

func TestFunc() {
	//resp, err := http.Get("https://httpbin.org/")
	//if err != nil {
	//	panic(err)
	//}
	//defer resp.Body.Close()
	//b := make([]byte, 10000)
	//n, err := resp.Body.Read(b)
	//if err != nil {
	//	panic(err)
	//}
	//log.Println(string(b[:n]))
	time.Sleep(5 * time.Second)
	log.Println("TestFunc finished in sleep 5 seconds")
}

func WgTest() {
	log.Println("WgTest start")
	gr := func() <-chan struct{} {
		done := make(chan struct{}, 1) // 这里返回的chan必须要有缓冲区 否则会阻塞在done <-那里
		TestFunc()
		done <- struct{}{}
		close(done)
		return done
	}
	for i := 0; i < 3; i++ {
		mgr.Add(gr, 3*time.Second)
	}
	mgr.Wait()
	log.Println("WgTest finished")
}
