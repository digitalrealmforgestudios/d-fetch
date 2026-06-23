package transport

import (
	"net/http"
	"sync"
)

type Overrider func(existing http.RoundTripper) http.RoundTripper

var global struct {
	sync.RWMutex
	fn Overrider
}

func SetGlobalOverrider(fn Overrider) {
	global.Lock()
	defer global.Unlock()
	global.fn = fn
}

func ApplyGlobalOverrider(existing http.RoundTripper) http.RoundTripper {
	global.RLock()
	fn := global.fn
	global.RUnlock()
	if fn == nil {
		return existing
	}
	return fn(existing)
}
