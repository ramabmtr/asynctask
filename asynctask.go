// Package asynctask aim to simplify goroutine management
// you can run goroutine and get result from it without make [channel](https://golang.org/doc/effective_go.html#channels)
// or use [waitgroup](https://golang.org/pkg/sync/#WaitGroup). Just define the function and let this package
// handle the rest for you
package asynctask

import (
	"context"
	"fmt"
	"runtime/debug"
	"sync"
	"time"
)

type (
	result struct {
		resp interface{}
		err  error
	}

	// AsyncTask hold the base context for asynctask
	AsyncTask struct {
		ctx            context.Context
		cancel         func()
		wg             *sync.WaitGroup
		mutex          *sync.Mutex
		err            error
		cancelOnError  bool
		runnerPoolSize int
		runners        []*Runner
		mapResult      map[string]interface{}
	}

	// Runner hold the base context for asynctask runner
	Runner struct {
		b        *AsyncTask
		id       string
		multiple bool
		f        func(param interface{}) (interface{}, error)
		param    interface{}
		timeout  time.Duration
	}

	safeResultChan struct {
		chResult chan result
		chClose  chan bool
		wg       sync.WaitGroup
		mutex    sync.Mutex
	}
)

// newSafeResultChan return an instance of safeResultChan which provide safe write and close a channel
func newSafeResultChan() *safeResultChan {
	return &safeResultChan{
		chResult: make(chan result),
		chClose:  make(chan bool),
	}
}

// read from result channel
func (src *safeResultChan) read() <-chan result {
	return src.chResult
}

// write to safe write to a channel
func (src *safeResultChan) write(data result) {
	go func() {
		src.mutex.Lock()
		src.wg.Add(1)
		src.mutex.Unlock()
		defer src.wg.Done()

		select {
		case <-src.chClose:
			return
		default:
			src.chResult <- data
		}
	}()
}

// close is for safe close a channel, this func utilize waitgroup to close a channel
// every write will add 1 delta to waitgroup and when this func called, wait all the waitgroup
// before closing the channel
func (src *safeResultChan) close() {
	close(src.chClose)

	src.mutex.Lock()
	src.wg.Wait()
	src.mutex.Unlock()

	close(src.chResult)
}

// NewAsyncTask create new asynctask runner instance
func NewAsyncTask(ctx context.Context) *AsyncTask {
	ctxNew, cancel := context.WithCancel(ctx)
	return &AsyncTask{
		ctx:            ctxNew,
		cancel:         cancel,
		wg:             new(sync.WaitGroup),
		mutex:          new(sync.Mutex),
		cancelOnError:  true,
		runnerPoolSize: 0,
		runners:        make([]*Runner, 0),
		mapResult:      make(map[string]interface{}),
	}
}

func (b *AsyncTask) cancelContext() {
	if b.cancelOnError {
		b.cancel()
	}
}

// SetRunnerPoolSize to set max goroutine can run at the same time
// goroutine will run as soon as the pool worker ready
func (b *AsyncTask) SetRunnerPoolSize(size int) *AsyncTask {
	b.runnerPoolSize = size
	return b
}

// CancelOnError is to flag if an error happen, immediately return or not
func (b *AsyncTask) CancelOnError(flag bool) *AsyncTask {
	b.cancelOnError = flag
	return b
}

// StartAndWait start the asynctask and wait for all task finish
func (b *AsyncTask) StartAndWait() error {
	sem := make(chan int, b.runnerPoolSize)
	mapID := make(map[string]bool)
	for _, runner := range b.runners {
		// check runner ID, if runner multiple != true and the ID is exist before,
		// return error and cancel context
		if mapID[runner.id] && !runner.multiple {
			b.cancelContext()
			return fmt.Errorf("ID %s have been used before without `SetMultiple()`", runner.id)
		}

		mapID[runner.id] = true

		if b.runnerPoolSize > 0 {
			cont := true
			select {
			case <-b.ctx.Done():
				cont = false
			default:
				sem <- 1
			}
			if !cont {
				break
			}
		}

		b.wg.Add(1)
		go func(runner *Runner) {
			defer b.wg.Done()
			if b.runnerPoolSize > 0 {
				defer func() { <-sem }()
			}
			runner.do()
		}(runner)
	}
	b.wg.Wait()
	return b.err
}

// GetResult is to get result from asynctask by ID
func (b *AsyncTask) GetResult(id string) interface{} {
	return b.mapResult[id]
}

// NewRunner create new asynctask runner
func (b *AsyncTask) NewRunner() *Runner {
	return &Runner{
		b: b,
	}
}

func (r *Runner) recovery() {
	rc := recover()
	if rc != nil {
		r.b.mutex.Lock()
		r.b.err = fmt.Errorf("panic recovered. message: %v. stacktrace: %s", rc, string(debug.Stack()))
		r.b.mutex.Unlock()
		r.b.cancelContext()
	}
}

func (r *Runner) processErr(err error) {
	r.b.cancelContext()
	r.b.mutex.Lock()
	defer r.b.mutex.Unlock()
	r.b.err = err
	return
}

func (r *Runner) processResp(id string, resp interface{}) {
	if resp == nil {
		return
	}

	r.b.mutex.Lock()
	defer r.b.mutex.Unlock()

	if r.multiple {
		if r.b.mapResult[id] == nil {
			r.b.mapResult[id] = make([]interface{}, 0)
		}

		oldResp, ok := r.b.mapResult[id].([]interface{})
		if !ok {
			r.b.err = fmt.Errorf("cannot append result. looks like the ID is used before without calling `SetMultiple()`")
			return
		}

		resp = append(oldResp, resp)
	}

	r.b.mapResult[id] = resp
}

func (r *Runner) do() {
	ch := newSafeResultChan()
	defer ch.close()

	runnerCtx := context.Background()
	if int64(r.timeout) > 0 {
		ctx, cancel := context.WithTimeout(r.b.ctx, r.timeout)
		runnerCtx = ctx
		defer cancel()
	}

	go func() {
		defer r.recovery()
		resp, err := r.f(r.param)

		select {
		case <-ch.read():
		default:
			ch.write(result{
				resp: resp,
				err:  err,
			})
		}
	}()

	select {
	case res := <-ch.read():
		if res.err != nil {
			r.processErr(res.err)
			return
		}
		r.processResp(r.id, res.resp)
	case <-r.b.ctx.Done():
	case <-runnerCtx.Done():
		r.processErr(fmt.Errorf("runner with ID %s reached its time limit", r.id))
	}
}

// SetFunc is to set the function that will be executed
func (r *Runner) SetFunc(f func(param interface{}) (interface{}, error)) *Runner {
	r.f = f
	return r
}

// SetParam is to set param that will be thrown to executed function
func (r *Runner) SetParam(param interface{}) *Runner {
	r.param = param
	return r
}

// SetMultiple is to set asynctask runner can run multiple times on same ID
// if runner is set to multiple, the result will become slice of interface
func (r *Runner) SetMultiple() *Runner {
	r.multiple = true
	return r
}

// SetTimeout is to set asynctask runner wait time for func to return result
// if the func fail to give result before the time limit reached, it will thrown error
func (r *Runner) SetTimeout(x time.Duration) *Runner {
	r.timeout = x
	return r
}

// Register is to register runner to asynctask
func (r *Runner) Register(id string) {
	r.id = id
	r.b.runners = append(r.b.runners, r)
}
