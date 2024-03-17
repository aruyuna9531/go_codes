// Copyright 2011 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package sync

import (
	"internal/race"
	"sync/atomic"
	"unsafe"
)

// A WaitGroup waits for a collection of goroutines to finish.
// The main goroutine calls Add to set the number of
// goroutines to wait for. Then each of the goroutines
// runs and calls Done when finished. At the same time,
// Wait can be used to block until all goroutines have finished.
//
// A WaitGroup must not be copied after first use.
//
// In the terminology of the Go memory model, a call to Done
// “synchronizes before” the return of any Wait call that it unblocks.
type WaitGroup struct {
	noCopy noCopy // 定义为noCopy，即首次被使用后不得被复制

	// 64-bit value: high 32 bits are counter, low 32 bits are waiter count.
	// 64-bit atomic operations require 64-bit alignment, but 32-bit
	// compilers only guarantee that 64-bit fields are 32-bit aligned.
	// For this reason on 32 bit architectures we need to check in state()
	// if state1 is aligned or not, and dynamically "swap" the field order if
	// needed.
	state1 uint64 // 32位正在跑的goroutine计数+32位正在等待结束的goroutine计数
	state2 uint32
}

// state 返回指向 state 和 sema 的指针
func (wg *WaitGroup) state() (statep *uint64, semap *uint32) {
	if unsafe.Alignof(wg.state1) == 8 || uintptr(unsafe.Pointer(&wg.state1))%8 == 0 {
		// state1 以64位方式对齐的，直接返回两个state的指针（没太明白为什么这样能判定64位）
		return &wg.state1, &wg.state2
	} else {
		// state1 是32位对齐的而不是64位对齐的，(&state1)+4 是64位对齐的
		state := (*[3]uint32)(unsafe.Pointer(&wg.state1))
		return (*uint64)(unsafe.Pointer(&state[1])), &state[0]
	}
}

// Add 增加计数（可以是负数）到 WaitGroup 计数器.
// 计数器转至0的时候， 被 Wait 阻塞的所有goroutine被释放（继续执行）
// 如果计数器转到了负数会报panic
//
// 加一个正数差值这件事必须要在 Wait 之前执行，
// 而且这意味着 Add 必须要在创建goroutine（或者其他等待事件）之前被执行。
// 如果 WaitGroup 要重用到其他独立事件，新的 Add 必须要在所有 Wait 返回后执行。
func (wg *WaitGroup) Add(delta int) {
	statep, semap := wg.state()
	if race.Enabled {
		_ = *statep // trigger nil deref early
		if delta < 0 {
			// Synchronize decrements with Wait.
			race.ReleaseMerge(unsafe.Pointer(wg))
		}
		race.Disable() // 开启race的时候Add要临时关闭
		defer race.Enable()
	}
	state := atomic.AddUint64(statep, uint64(delta)<<32) //给正在跑的goroutine计数+1（参考state1定义）
	v := int32(state >> 32)                              // v=正在跑的协程计数
	w := uint32(state)                                   // w=目前在Wait的协程计数
	if race.Enabled && delta > 0 && v == int32(delta) {
		// wg首次使用，跟 Wait 必须是同步的（因为是从0开始计数）做一下保护措施
		race.Read(unsafe.Pointer(semap))
	}
	if v < 0 {
		panic("sync: negative WaitGroup counter")
	} // v不能为0
	if w != 0 && delta > 0 && v == int32(delta) {
		panic("sync: WaitGroup misuse: Add called concurrently with Wait")
	} // 在有Wait协程的场合，正在跑的协程从0开始增加是不正常的
	if v > 0 || w == 0 {
		return
	} // 如果当期正在跑的协程数为正或者没有协程在Wait，不需要后续的释放信号操作（该阻塞的还在阻塞，没有能阻塞的协程也无需多余操作）
	// 走到这里的时候正在跑的计数已经变为0，而存在Wait的协程
	// Now there can't be concurrent mutations of state:
	// - Adds must not happen concurrently with Wait,
	// - Wait 在看到counter == 0的时候不能再增加Wait的协程
	if *statep != state {
		panic("sync: WaitGroup misuse: Add called concurrently with Wait")
	} // 这边还是做一下靠谱性检测防止wg误用
	// Reset waiters count to 0.
	*statep = 0
	for ; w != 0; w-- {
		// Semrelease 原子性地增加semap的值，然后提醒一个等待中的被SemAcquire阻塞的协程
		// （该函数是内部库函数，不应被外部使用）
		runtime_Semrelease(semap, false, 0)
	}
}

// Done 简单粗暴，计数-1
func (wg *WaitGroup) Done() {
	wg.Add(-1)
}

// Wait 阻塞调用的goroutine直到计数器转到0
func (wg *WaitGroup) Wait() {
	statep, semap := wg.state()
	if race.Enabled {
		_ = *statep // trigger nil deref early
		race.Disable()
	}
	for {
		state := atomic.LoadUint64(statep)
		v := int32(state >> 32)
		w := uint32(state) // v,w 同Add
		if v == 0 {
			// 没有正在跑的协程，不用等
			if race.Enabled {
				race.Enable()
				race.Acquire(unsafe.Pointer(wg))
			}
			return
		}
		// 增加等待协程计数（外圈套for循环，保证这个Wait被计算上——这里没有用锁）
		if atomic.CompareAndSwapUint64(statep, state, state+1) {
			if race.Enabled && w == 0 {
				// Wait 必须和第一次Add操作是同步的，第一次Wait做一下保护措施，否则异步的多个Wait可能会相互race
				race.Write(unsafe.Pointer(semap))
			}
			// Semacquire 会一直阻塞到s（输入参数）的值>0，阻塞完成后原子操作减少它的值。
			runtime_Semacquire(semap)
			// 阻塞完成后的一些操作，然后协程退出Wait继续往下执行
			if *statep != 0 {
				panic("sync: WaitGroup is reused before previous Wait has returned")
			} // 第90行，Add把协程数减到0的时候将statep先置为了0，在这里检查如果不是0会报异常（不能在这一轮wg完成之前重用这个wg）
			if race.Enabled {
				race.Enable()
				race.Acquire(unsafe.Pointer(wg))
			}
			return
		}
	}
}

// wg的工作流程：
// （1）先Add，使协程计数>0（计数=0的时候跑Wait实际上等于没跑，会在第120行直接退出），开启这一期WaitGroup工作
// （2）执行要参与计数的等待任务，一般就是新开goroutine跑任务，也可以是其他形式的需要等待的任务（如：向其他服务器发了请求，要等待响应的场合）——这个执行操作本身不会对waitgroup产生什么影响，但任务结束的地方要显式执行Done()
// （3）在需要等待所有计数任务执行完毕的地方跑Wait，Wait计数+1，并在runtime_Semacquire的地方阻塞住（等待到semap>0才能解除）
// （4）1个任务执行完毕，执行done，协程计数-1，并且之后当场判定状态值。如果协程计数已经为0并且存在Wait，就会给semap指针添加Wait数量等量的信号量（runtime_Semrelease），这一期WaitGroup名义上已经完成（形式上还没有完成，但在形式上完成之前不能有任何操作再改动那个计数，否则释放wait时会报panic）。否则啥都不干，大家继续Wait
// （5）在runtime_Semacquire阻塞住的wait接收到semap信号量增加后，停止阻塞，并原子性地扣掉一个semap信号量，然后退出Wait，在调用Wait的地方继续往下执行
// （6）所有wait都成功退出后，这期wg在形式上完成，wg可以被重新使用
