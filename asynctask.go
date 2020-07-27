// This package aim to simplify goroutine management
// you can run goroutine and get result from it without make [channel](https://golang.org/doc/effective_go.html#channels)
// or use [waitgroup](https://golang.org/pkg/sync/#WaitGroup). Just define the function and let this package
// handle the rest for you
package asynctask

import (
	"context"
	"fmt"
	"runtime/debug"
	"sync"
)

type (
	result struct {
		resp interface{}
		err  error
	}

	// BaseAsyncTask hold the base context for asynctask
	BaseAsyncTask struct {
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
		b        *BaseAsyncTask
		id       string
		multiple bool
		f        func(param interface{}) (interface{}, error)
		param    interface{}
	}
)

// NewAsyncTask create new asynctask runner instance
func NewAsyncTask(ctx context.Context) *BaseAsyncTask {
	ctxNew, cancel := context.WithCancel(ctx)
	return &BaseAsyncTask{
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

func (b *BaseAsyncTask) cancelContext() {
	if b.cancelOnError {
		b.cancel()
	}
}

// SetRunnerPoolSize to set max goroutine can run at the same time
// goroutine will run as soon as the pool worker ready
func (b *BaseAsyncTask) SetRunnerPoolSize(size int) *BaseAsyncTask {
	b.runnerPoolSize = size
	return b
}

// CancelOnError is to flag if an error happen, immediately return or not
func (b *BaseAsyncTask) CancelOnError(flag bool) *BaseAsyncTask {
	b.cancelOnError = flag
	return b
}

// StartAndWait start the asynctask and wait for all task finish
func (b *BaseAsyncTask) StartAndWait() error {
	sem := make(chan int, b.runnerPoolSize)
	mapID := make(map[string]bool)
	for _, runner := range b.runners {
		// check runner ID, if runner multiple != true and the ID is exist before,
		// return error and cancel context
		if mapID[runner.id] && !runner.multiple {
			b.cancelContext()
			b.err = fmt.Errorf("ID %s have been used before without `SetMultiple()`", runner.id)
			break
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
func (b *BaseAsyncTask) GetResult(id string) interface{} {
	return b.mapResult[id]
}

// NewRunner create new asynctask runner
func (b *BaseAsyncTask) NewRunner() *Runner {
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
	chRes := make(chan result)
	defer close(chRes)

	go func() {
		defer r.recovery()
		resp, err := r.f(r.param)

		select {
		case <-chRes:
		default:
			chRes <- result{
				resp: resp,
				err:  err,
			}
		}
	}()

	select {
	case res := <-chRes:
		if res.err != nil {
			r.b.cancelContext()
			r.b.mutex.Lock()
			defer r.b.mutex.Unlock()
			r.b.err = res.err
			return
		}
		r.processResp(r.id, res.resp)
	case <-r.b.ctx.Done():
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

// Register is to register runner to asynctask
func (r *Runner) Register(id string) {
	r.id = id
	r.b.runners = append(r.b.runners, r)
}
