package async

import (
	"errors"
	"github.com/Jinnrry/pmail/utils/context"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cast"
	"runtime/debug"
	"sync"
)

type Callback func(params any)

type Async struct {
	wg        sync.WaitGroup
	errMu     sync.Mutex
	lastError error
	ctx       *context.Context
}

func New(ctx *context.Context) *Async {
	return &Async{
		ctx: ctx,
	}
}

func (as *Async) LastError() error {
	as.errMu.Lock()
	defer as.errMu.Unlock()
	return as.lastError
}

func (as *Async) WaitProcess(callback Callback, params any) {
	as.wg.Add(1)
	go func() {
		defer as.wg.Done()
		as.run(callback, params)
	}()
}

func (as *Async) Process(callback Callback, params any) {
	go as.run(callback, params)
}

func (as *Async) Wait() {
	as.wg.Wait()
}

func (as *Async) run(callback Callback, params any) {
	defer func() {
		if recovered := recover(); recovered != nil {
			err := as.HandleErrRecover(recovered)
			as.errMu.Lock()
			as.lastError = err
			as.errMu.Unlock()
		}
	}()
	callback(params)
}

// HandleErrRecover panic恢复处理
func (as *Async) HandleErrRecover(err interface{}) (returnErr error) {
	switch err.(type) {
	case error:
		returnErr = err.(error)
	default:
		returnErr = errors.New(cast.ToString(err))
	}

	log.WithContext(as.ctx).Errorf("goroutine panic:%s  \n %s", err, string(debug.Stack()))

	return
}
