package tracer

import "sync"

type config struct {
	errorFunc func(err error) error
	panicFunc func(p any)
}

var (
	cfg  config
	once sync.Once
)

func SetErrorFunc(fn func(error) error) {
	once.Do(func() {
		cfg.errorFunc = fn
	})
}

func SetPanicFunc(fn func(any)) {
	once.Do(func() {
		cfg.panicFunc = fn
	})
}
