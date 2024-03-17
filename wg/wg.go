package wg

import (
	"log"
	"net/http"
	"sync"
	"sync/atomic"
)

type Mgr struct {
	fcId atomic.Uint32
	w    sync.WaitGroup
}

func (m *Mgr) Add(f func(fcId uint32)) {
	m.w.Add(1)
	go func(ff func(fcId uint32)) {
		ff(m.fcId.Add(1))
		m.w.Done()
	}(f)
}

func (m *Mgr) Wait() {
	m.w.Wait()
}

var mgr = &Mgr{}

func WgTest() {
	gr := func(fcId uint32) {
		resp, err := http.Get("https://httpbin.org/")
		if err != nil {
			panic(err)
		}
		defer resp.Body.Close()
		b := make([]byte, 10000)
		n, err := resp.Body.Read(b)
		if err != nil {
			panic(err)
		}
		log.Println(string(b[:n]))
		log.Printf("wg fcid %d done\n", fcId)
	}
	for i := 0; i < 10; i++ {
		mgr.Add(gr)
	}
	mgr.Wait()
}
