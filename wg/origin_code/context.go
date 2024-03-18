// Copyright 2014 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package context defines the Context type, which carries deadlines,
// cancellation signals, and other request-scoped values across API boundaries
// and between processes.
//
// Incoming requests to a server should create a Context, and outgoing
// calls to servers should accept a Context. The chain of function
// calls between them must propagate the Context, optionally replacing
// it with a derived Context created using WithCancel, WithDeadline,
// WithTimeout, or WithValue. When a Context is canceled, all
// Contexts derived from it are also canceled.
//
// The WithCancel, WithDeadline, and WithTimeout functions take a
// Context (the parent) and return a derived Context (the child) and a
// CancelFunc. Calling the CancelFunc cancels the child and its
// children, removes the parent's reference to the child, and stops
// any associated timers. Failing to call the CancelFunc leaks the
// child and its children until the parent is canceled or the timer
// fires. The go vet tool checks that CancelFuncs are used on all
// control-flow paths.
//
// Programs that use Contexts should follow these rules to keep interfaces
// consistent across packages and enable static analysis tools to check context
// propagation:
//
// Do not store Contexts inside a struct type; instead, pass a Context
// explicitly to each function that needs it. The Context should be the first
// parameter, typically named ctx:
//
//	func DoSomething(ctx context.Context, arg Arg) error {
//		// ... use ctx ...
//	}
//
// Do not pass a nil Context, even if a function permits it. Pass context.TODO
// if you are unsure about which Context to use.
//
// Use context Values only for request-scoped data that transits processes and
// APIs, not for passing optional parameters to functions.
//
// The same Context may be passed to functions running in different goroutines;
// Contexts are safe for simultaneous use by multiple goroutines.
//
// See https://blog.golang.org/context for example code for a server that uses
// Contexts.
package context

import (
	"errors"
	"internal/reflectlite"
	"sync"
	"sync/atomic"
	"time"
)

// A Context carries a deadline, a cancellation signal, and other values across
// API boundaries.
//
// Context's methods may be called by multiple goroutines simultaneously.
type Context interface {
	// Deadline 返回任务被取消的截止时间
	// 如果deadline没有被设置， Deadline 返回 ok==false
	Deadline() (deadline time.Time, ok bool)

	// Done 在任务取消后返回一个已关闭的channel.
	// Done 在context不会被取消的场合会return nil.
	// Done 返回的channel的关闭有可能会被异步在cancel函数返回后执行
	//
	// WithCancel 在cancel执行时会把 Done 关闭
	// WithDeadline 在deadline过去后会把 Done 关闭
	// WithTimeout 在一定超时时间后会把 Done 关闭
	//
	// Done 可以被用作select的项:
	//
	//  // Stream 通过 DoSomething 生成了一些值并把它们送到out，除非DoSomething 返回了err或者ctx.Done被关闭.
	//  func Stream(ctx context.Context, out chan<- Value) error {
	//  	for {
	//  		v, err := DoSomething(ctx)
	//  		if err != nil {
	//  			return err
	//  		}
	//  		select {
	//  		case <-ctx.Done():
	//  			return ctx.Err()
	//  		case out <- v:
	//  		}
	//  	}
	//  }
	Done() <-chan struct{}

	// Err
	// 如果 Done 还没被关闭, Err 返回nil
	// 如果 Done 已经关闭, Err 返回非nil值阐述理由:
	// 当context已经被取消，返回 Canceled
	// 如果deadline已过，返回 DeadlineExceeded
	Err() error

	// Value returns the value associated with this context for key, or nil
	// if no value is associated with key. Successive calls to Value with
	// the same key returns the same result.
	//
	// Use context values only for request-scoped data that transits
	// processes and API boundaries, not for passing optional parameters to
	// functions.
	//
	// A key identifies a specific value in a Context. Functions that wish
	// to store values in Context typically allocate a key in a global
	// variable then use that key as the argument to context.WithValue and
	// Context.Value. A key can be any type that supports equality;
	// packages should define keys as an unexported type to avoid
	// collisions.
	//
	// Packages that define a Context key should provide type-safe accessors
	// for the values stored using that key:
	//
	// 	// Package user defines a User type that's stored in Contexts.
	// 	package user
	//
	// 	import "context"
	//
	// 	// User is the type of value stored in the Contexts.
	// 	type User struct {...}
	//
	// 	// key is an unexported type for keys defined in this package.
	// 	// This prevents collisions with keys defined in other packages.
	// 	type key int
	//
	// 	// userKey is the key for user.User values in Contexts. It is
	// 	// unexported; clients use user.NewContext and user.FromContext
	// 	// instead of using this key directly.
	// 	var userKey key
	//
	// 	// NewContext returns a new Context that carries value u.
	// 	func NewContext(ctx context.Context, u *User) context.Context {
	// 		return context.WithValue(ctx, userKey, u)
	// 	}
	//
	// 	// FromContext returns the User value stored in ctx, if any.
	// 	func FromContext(ctx context.Context) (*User, bool) {
	// 		u, ok := ctx.Value(userKey).(*User)
	// 		return u, ok
	// 	}
	Value(key any) any
}

// Canceled 当context是因为被取消而Done的时候，Context.Err 会返回这个
var Canceled = errors.New("context canceled")

// DeadlineExceeded 当context是因为截止时间已过而Done的时候，Context.Err 会返回这个
var DeadlineExceeded error = deadlineExceededError{}

type deadlineExceededError struct{}

func (deadlineExceededError) Error() string   { return "context deadline exceeded" }
func (deadlineExceededError) Timeout() bool   { return true }
func (deadlineExceededError) Temporary() bool { return true }

// emptyCtx 不会被取消，没有value，没有deadline. 它不是struct{}类型因为用var定义这个的时候需要有唯一地址
type emptyCtx int

func (*emptyCtx) Deadline() (deadline time.Time, ok bool) {
	return
}

func (*emptyCtx) Done() <-chan struct{} {
	return nil
}

func (*emptyCtx) Err() error {
	return nil
}

func (*emptyCtx) Value(key any) any {
	return nil
}

func (e *emptyCtx) String() string {
	switch e {
	case background:
		return "context.Background"
	case todo:
		return "context.TODO"
	}
	return "unknown empty Context"
}

var (
	background = new(emptyCtx)
	todo       = new(emptyCtx)
)

// Background 返回一个非nil的空 Context
// 不会被取消，没有value，没有deadline
// 一般会用在main函数里用于初始化或者测试，作为最高层次的 Context 接受请求
func Background() Context {
	return background
}

// TODO 返回一个非nil的空 Context.
// context.TODO 可以用在不清楚该用什么Context的情况，或者context不可用的情况
// (因为外层函数还没有能接收一个Context参数）
func TODO() Context {
	return todo
}

// CancelFunc 告诉context放弃工作
// CancelFunc 不会等待工作结束
// CancelFunc 有可能同时被多个goroutine调用
// 在第一次被调用后，之后对它的调用不会有任何效果（即同个context的cancel只会执行1次）
type CancelFunc func()

// ---- cancelCtx 基础context类型之一

// WithCancel 返回parent的copy，但带上了一个新的Done channel。
// 当返回的cancel函数被调用，或者输入参数parent的Done channel被关闭的时候，返回的context的Done channel被关闭。
//
// cancel这个 context 会释放相关资源, 因此代码应该在这个context里跑的操作完成后立刻调用cancel
func WithCancel(parent Context) (ctx Context, cancel CancelFunc) {
	if parent == nil {
		panic("cannot create context from nil parent")
	}
	c := newCancelCtx(parent)
	propagateCancel(parent, &c)
	return &c, func() { c.cancel(true, Canceled) }
}

// newCancelCtx 返回一个初始化的 cancelCtx.
func newCancelCtx(parent Context) cancelCtx {
	return cancelCtx{Context: parent}
}

// goroutines 计算创建过的 goroutines 数量（测试用）
var goroutines int32

// propagateCancel 取消 child 用的
func propagateCancel(parent Context, child canceler) {
	done := parent.Done()
	if done == nil {
		return // parent is never canceled
	}

	select {
	case <-done:
		// parent 已经取消了
		child.cancel(false, parent.Err())
		return
	default:
	}

	if p, ok := parentCancelCtx(parent); ok {
		p.mu.Lock()
		if p.err != nil {
			// parent 已经取消了
			child.cancel(false, p.err)
		} else {
			if p.children == nil {
				p.children = make(map[canceler]struct{})
			}
			p.children[child] = struct{}{}
		}
		p.mu.Unlock()
	} else {
		atomic.AddInt32(&goroutines, +1)
		go func() {
			select {
			case <-parent.Done():
				child.cancel(false, parent.Err())
			case <-child.Done():
			}
		}()
	}
}

// &cancelCtxKey is the key that a cancelCtx returns itself for.
var cancelCtxKey int

// parentCancelCtx 返回parent祖先上的 *cancelCtx（因为它的父节点可能不是cancelCtx类型。比如valueCtx）
// 通过调用 parent.Value(&cancelCtxKey) 找到最深处的 *cancelCtx 然后检查 parent.Done() 是否和 *cancelCtx 匹配
// (若不匹配， *cancelCtx 会被包装到一个自定义的提供了不同done channel的实现, 而且还不能bypass它)
func parentCancelCtx(parent Context) (*cancelCtx, bool) {
	done := parent.Done()
	if done == closedchan || done == nil {
		return nil, false
	}
	p, ok := parent.Value(&cancelCtxKey).(*cancelCtx)
	if !ok {
		return nil, false
	}
	pdone, _ := p.done.Load().(chan struct{})
	if pdone != done {
		return nil, false
	}
	return p, true
}

// removeChild 从parent移除一个context.
func removeChild(parent Context, child canceler) {
	p, ok := parentCancelCtx(parent)
	if !ok {
		return
	}
	p.mu.Lock()
	if p.children != nil {
		delete(p.children, child)
	}
	p.mu.Unlock()
}

// canceler 是一个可以被直接取消的context类型. *cancelCtx 和 *timerCtx 实现了这个接口
type canceler interface {
	cancel(removeFromParent bool, err error)
	Done() <-chan struct{}
}

// closedchan 是可重复利用的已关闭 channel.
var closedchan = make(chan struct{})

func init() {
	close(closedchan)
}

// cancelCtx 可以被取消。 当它被取消时, 还会将实现了canceler的所有 children 一并取消
type cancelCtx struct {
	Context // 父节点

	mu       sync.Mutex            // 为下面变量加的锁
	done     atomic.Value          // 是 chan struct{} 类型，延迟创建，在第一次 cancel 调用时关闭
	children map[canceler]struct{} // 第一次cancel调用时，会被设置为nil
	err      error                 // 第一次cancel调用时，被设置为非nil
}

func (c *cancelCtx) Value(key any) any {
	if key == &cancelCtxKey {
		return c
	}
	return value(c.Context, key)
}

// Done 取的done的值，没有将自动创建
func (c *cancelCtx) Done() <-chan struct{} {
	d := c.done.Load()
	if d != nil {
		return d.(chan struct{})
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	d = c.done.Load()
	if d == nil {
		d = make(chan struct{})
		c.done.Store(d)
	}
	return d.(chan struct{})
}

func (c *cancelCtx) Err() error {
	c.mu.Lock()
	err := c.err
	c.mu.Unlock()
	return err
}

type stringer interface {
	String() string
}

func contextName(c Context) string {
	if s, ok := c.(stringer); ok {
		return s.String()
	}
	return reflectlite.TypeOf(c).String()
}

func (c *cancelCtx) String() string {
	return contextName(c.Context) + ".WithCancel"
}

// cancel 关闭 c.done, 取消所有的c下面的 children, 如果 removeFromParent=true, 将自身从parent的children列里移除.
func (c *cancelCtx) cancel(removeFromParent bool, err error) {
	if err == nil {
		panic("context: internal error: missing cancel error")
	} // err必须要有值（cancel肯定有原因）
	c.mu.Lock()
	if c.err != nil {
		c.mu.Unlock()
		return // 自身已经取消了
	}
	c.err = err
	d, _ := c.done.Load().(chan struct{}) // 这里保证c.done一定存了一个关闭的channel
	if d == nil {
		c.done.Store(closedchan)
	} else {
		close(d)
	}
	for child := range c.children {
		// NOTE: 这里会在持有parent锁的时候同时尝试给child加锁
		child.cancel(false, err)
	}
	c.children = nil
	c.mu.Unlock()

	if removeFromParent {
		removeChild(c.Context, c)
	}
}

// WithDeadline 返回了一个parent context的拷贝，deadline被调整为d.
// 如果输入的d已经晚于parent的deadline, WithDeadline(parent, d) 在语义上等于parent.
// 在deadline到达时，或者返回的cancel函数被调用时，或者parent的Done channel被关闭时，返回的context的Done channel会被关闭。
//
// 取消这个context会释放资源，因此……（参考上面 WithCancel ）
func WithDeadline(parent Context, d time.Time) (Context, CancelFunc) {
	if parent == nil {
		panic("cannot create context from nil parent")
	}
	if cur, ok := parent.Deadline(); ok && cur.Before(d) {
		// The current deadline is already sooner than the new one.
		return WithCancel(parent)
	} // 这里因为输入的d超过了parent的deadline，直接返回了parent自身
	c := &timerCtx{
		cancelCtx: newCancelCtx(parent),
		deadline:  d,
	}
	propagateCancel(parent, c)
	dur := time.Until(d)
	if dur <= 0 {
		c.cancel(true, DeadlineExceeded) // 执行的时候已经过了输入的d时间
		return c, func() { c.cancel(false, Canceled) }
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.err == nil {
		c.timer = time.AfterFunc(dur, func() {
			c.cancel(true, DeadlineExceeded)
		})
	}
	return c, func() { c.cancel(true, Canceled) }
}

// timerCtx 持有一个 timer 和一个 deadline
// 内嵌了一个 cancelCtx 以实现 Done 和 Err接口
// 实现了 cancel 用于停掉 cancelCtx.cancel 的timer
type timerCtx struct {
	cancelCtx
	timer *time.Timer // 会被 cancelCtx.mu 锁上

	deadline time.Time
}

func (c *timerCtx) Deadline() (deadline time.Time, ok bool) {
	return c.deadline, true
}

func (c *timerCtx) String() string {
	return contextName(c.cancelCtx.Context) + ".WithDeadline(" +
		c.deadline.String() + " [" +
		time.Until(c.deadline).String() + "])"
}

func (c *timerCtx) cancel(removeFromParent bool, err error) {
	c.cancelCtx.cancel(false, err)
	if removeFromParent {
		// 从parent（cancelCtx类）的children集里移除这个 timerCtx
		removeChild(c.cancelCtx.Context, c)
	}
	c.mu.Lock()
	if c.timer != nil {
		c.timer.Stop()
		c.timer = nil
	}
	c.mu.Unlock()
}

// WithTimeout 就是 WithDeadline的复用，timeout相对于当前系统时间。
func WithTimeout(parent Context, timeout time.Duration) (Context, CancelFunc) {
	return WithDeadline(parent, time.Now().Add(timeout))
}

// ---- valueCtx 类型，这个主要是保存context节点的一些信息。（但这不是一个可取消的context类型）

// WithValue 返回parent的一份拷贝， 但指定了一组kv。
//
// context的Values只用作在进程或api中转的链路追踪数据，而不是用来给函数传递可选参数。
//
// 给出的key必须是可比较的并且不应该是字符串类型或者其他内嵌类型，以避免使用context的包之间发生冲突。
// 使用 WithValue 的对象需要定义它们自己的key类型
// 为避免给interface{}类型赋值时申请内存，context的keys一般会使用固定类型 struct{}.
// 或者说，出口的context 的kv静态类型应该是指针或者interface
func WithValue(parent Context, key, val any) Context {
	if parent == nil {
		panic("cannot create context from nil parent")
	}
	if key == nil {
		panic("nil key")
	}
	if !reflectlite.TypeOf(key).Comparable() {
		panic("key is not comparable")
	}
	return &valueCtx{parent, key, val}
}

// valueCtx 有一组键值对
// It implements Value for that key and delegates all other calls to the embedded Context.
type valueCtx struct {
	Context  // 这个是父节点
	key, val any
}

// stringify tries a bit to stringify v, without using fmt, since we don't
// want context depending on the unicode tables. This is only used by
// *valueCtx.String().
func stringify(v any) string {
	switch s := v.(type) {
	case stringer:
		return s.String()
	case string:
		return s
	}
	return "<not Stringer>"
}

func (c *valueCtx) String() string {
	return contextName(c.Context) + ".WithValue(type " +
		reflectlite.TypeOf(c.key).String() +
		", val " + stringify(c.val) + ")"
}

func (c *valueCtx) Value(key any) any {
	if c.key == key {
		return c.val
	} // 输入值就是自身的key，返回自身的val
	return value(c.Context, key) // 搜索父节点，并返回父节点的key值
}

// value 从c开始找key对应的值，如果key类型不匹配，还会继续往父节点上找直到找到为止（或者找到了empty）
func value(c Context, key any) any {
	for {
		switch ctx := c.(type) {
		case *valueCtx:
			if key == ctx.key {
				return ctx.val
			}
			c = ctx.Context
		case *cancelCtx:
			if key == &cancelCtxKey {
				return c
			}
			c = ctx.Context
		case *timerCtx:
			if key == &cancelCtxKey {
				return &ctx.cancelCtx
			}
			c = ctx.Context
		case *emptyCtx:
			return nil
		default:
			return c.Value(key)
		}
	}
}
