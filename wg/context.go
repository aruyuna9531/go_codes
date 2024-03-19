package wg

import (
	"context"
	"log"
	"time"
)

func Work(ctx context.Context) {
	c := make(chan struct{})
	go func() {
		time.Sleep(3 * time.Second)
		select {
		//case c <- struct{}{}:
		//	如果context的timeout大于上面sleep的时间，c被成功写入会打印这个
		//	FIX： 即使done了，由于select是随机执行因此这一条也有概率被执行但因为外层Done的时候把c关掉，往关掉的channel写入会panic，因此c<-不能和Done平级
		//	log.Println("Work goroutine finished successfully")
		case <-ctx.Done():
			// 这个在sleep3秒后打印，因为ctx已经done，所以会打印这个
			log.Println("Work goroutine failed, ctx done")
		default:
			// 正确写法应该在这，如果Done没被执行，才推入c
			c <- struct{}{}
			log.Println("Work goroutine finished successfully")
		}
	}()
	select {
	case <-ctx.Done():
		// ctx先done，所以不ok
		log.Printf("Work not ok, msg: %s\n", ctx.Err().Error())
		close(c)
	case <-c:
		log.Printf("Work ok")
	}
}

func ContextTest() {
	log.Println("ContextTest start")
	for i := 0; i < 50; i++ {
		go func() {
			ctx, cf := context.WithTimeout(context.Background(), 2*time.Second) // 限制2秒内跑完Work
			defer cf()
			Work(ctx)
		}()
	}
}
