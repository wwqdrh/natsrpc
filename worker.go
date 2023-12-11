package natsrpc

import (
	"io"
	"runtime/debug"
	"sync/atomic"
	"time"

	"github.com/wwqdrh/gokit/logger"
	"go.uber.org/zap"
)

type Worker interface {
	Post(f func())
	Run()
	Close()
	AfterPost(duration time.Duration, f func()) *time.Timer
	NewTicker(d time.Duration, f func()) io.Closer
	Len() int
}

type workTicker struct {
	done     chan struct{}
	worker   Worker
	duration time.Duration
	f        func()
}

func newWorkTicker(worker Worker, d time.Duration, f func()) *workTicker {
	return &workTicker{
		done:     make(chan struct{}, 1),
		worker:   worker,
		duration: d,
		f:        f,
	}
}

func (p *workTicker) run() {
	go func() {
		ticker := time.NewTicker(p.duration)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				p.worker.Post(p.f)
			case <-p.done:
				return
			}
		}
	}()
}

func (p *workTicker) Close() error {
	close(p.done)
	return nil
}

type Work struct {
	funChan chan func()

	closed int32
}

func NewWorker() Worker {
	p := new(Work)
	p.funChan = make(chan func(), 10240)
	return p
}

func (p *Work) Post(f func()) {
	if atomic.LoadInt32(&p.closed) == 0 {
		p.funChan <- f
	}
}

func (p *Work) TryPost(f func(), maxLen int) {
	if maxLen != 0 && len(p.funChan) > maxLen {
		logger.DefaultLogger.Warn("tryPost over maxLen", zap.Int("maxLen", maxLen), zap.Int("workerLen", len(p.funChan)))
		return
	}

	select {
	case p.funChan <- f:
	default:
		logger.DefaultLogger.Warn("worker tryPost,discard", zap.Int("workerLen", len(p.funChan)))
	}
}

func (p *Work) Run() {
	go func() {
		for f := range p.funChan {
			p.protectedFun(f)
		}
	}()
}

func (p *Work) Close() {
	if atomic.CompareAndSwapInt32(&p.closed, 0, 1) {
		close(p.funChan)
	}
}

func (p *Work) Len() int {
	return len(p.funChan)
}
func (p *Work) protectedFun(callback func()) {
	//TODO 每个函数都包装了defer，性能怎样？
	defer func() {
		if err := recover(); err != nil {
			debug.PrintStack()
		}
	}()
	callback()
}

func (p *Work) AfterPost(d time.Duration, f func()) *time.Timer {
	return time.AfterFunc(d, func() {
		p.Post(f)
	})
}

func (p *Work) NewTicker(d time.Duration, f func()) io.Closer {
	t := newWorkTicker(p, d, f)
	t.run()
	return t
}

// worker长度超过maxLen就丢弃f
func (p *Work) NewTryTicker(d time.Duration, maxLen int, f func()) *time.Ticker {
	ticker := time.NewTicker(d)
	go func() {
		for range ticker.C {
			p.TryPost(f, maxLen)
		}
	}()
	return ticker
}
