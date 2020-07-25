package asynctask

import (
	"context"
	"fmt"
	"runtime/debug"
	"sync"
)

type (
	baseAsyncTask struct {
		ctx     context.Context
		cancel  func()
		wg      *sync.WaitGroup
		mutex   *sync.Mutex
		mapID   map[string]bool
		mapResp map[string]interface{}
		mapErr  map[string]error
	}

	runner struct {
		b *baseAsyncTask

		multiple bool
		f        func(param interface{}) (interface{}, error)
		param    interface{}
	}

	result struct {
		Resp interface{}
		Err  error
	}
)

// NewAsyncTaskRunner create new asynctask runner instance
func NewAsyncTaskRunner(ctx context.Context) *baseAsyncTask {
	ctxNew, cancel := context.WithCancel(ctx)
	return &baseAsyncTask{
		ctx:     ctxNew,
		cancel:  cancel,
		wg:      new(sync.WaitGroup),
		mutex:   new(sync.Mutex),
		mapID:   make(map[string]bool),
		mapResp: make(map[string]interface{}),
		mapErr:  make(map[string]error),
	}
}

func (b *baseAsyncTask) recovery() {
	if r := recover(); r != nil {
		errorLogger.Println(fmt.Sprintf("panic recovered. message: %v. stacktrace: %s", r, string(debug.Stack())))
	}
}

func (b *baseAsyncTask) Wait() error {
	b.wg.Wait()

	for id, err := range b.mapErr {
		if err != nil {
			return fmt.Errorf("ID %s have an error. Err: %s", id, err.Error())
		}
	}
	return nil
}

func (b *baseAsyncTask) GetResult(id string) interface{} {
	return b.mapResp[id]
}

func (b *baseAsyncTask) SetFunc(f func(param interface{}) (interface{}, error)) *runner {
	return &runner{
		b: b,
		f: f,
	}
}

func (r *runner) recovery(chRes chan result) {
	rc := recover()
	if rc != nil {
		errorLogger.Println(fmt.Sprintf("panic recovered. message: %v. stacktrace: %s", rc, string(debug.Stack())))

		select {
		case <-chRes:
		case <-r.b.ctx.Done():
		default:
			chRes <- result{
				Resp: nil,
				Err:  fmt.Errorf("panic: %v", rc),
			}
		}
	}
}

func (r *runner) processResp(id string, resp interface{}) {
	if resp == nil {
		return
	}

	r.b.mutex.Lock()
	defer r.b.mutex.Unlock()

	if r.multiple {
		if r.b.mapResp[id] == nil {
			r.b.mapResp[id] = make([]interface{}, 0)
		}

		oldResp, ok := r.b.mapResp[id].([]interface{})
		if !ok {
			r.b.mapErr[id] = fmt.Errorf("cannot append result. looks like the ID is used before without calling `SetMultiple()`")
		}

		resp = append(oldResp, resp)
	}

	r.b.mapResp[id] = resp
}

func (r *runner) SetParam(param interface{}) *runner {
	r.param = param
	return r
}

func (r *runner) SetMultiple() *runner {
	r.multiple = true
	return r
}

func (r *runner) Do(id string) {
	var errID error
	if r.b.mapID[id] && !r.multiple {
		errID = fmt.Errorf("ID %s is already used", id)
	}

	r.b.mapID[id] = true

	r.b.wg.Add(1)
	go func() {
		defer r.b.recovery()
		defer r.b.wg.Done()

		if errID != nil {
			r.b.cancel()
			r.b.mutex.Lock()
			defer r.b.mutex.Unlock()
			r.b.mapErr[id] = errID
		}

		chRes := make(chan result)
		defer close(chRes)

		go func() {
			defer r.recovery(chRes)
			resp, err := r.f(r.param)

			select {
			case <-chRes:
			case <-r.b.ctx.Done():
			default:
				chRes <- result{
					Resp: resp,
					Err:  err,
				}
			}
		}()

		select {
		case res := <-chRes:
			if res.Err != nil {
				r.b.cancel()
				r.b.mutex.Lock()
				defer r.b.mutex.Unlock()
				r.b.mapErr[id] = res.Err
				return
			}
			r.processResp(id, res.Resp)
		case <-r.b.ctx.Done():
		}
	}()
}
