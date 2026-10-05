package access

import (
	"sync"
	"time"
)

const windowSpan = time.Minute

// window 滑动窗口限流器.
type window struct {
	mu       sync.Mutex
	requests []time.Time
	limit    int
}

func newWindow(limit int) *window {
	if limit < 1 {
		limit = 1
	}
	return &window{limit: limit}
}

func (w *window) allow() bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	now := time.Now()
	cutoff := now.Add(-windowSpan)

	valid := w.requests[:0]
	for _, t := range w.requests {
		if t.After(cutoff) {
			valid = append(valid, t)
		}
	}
	w.requests = valid

	if len(w.requests) >= w.limit {
		return false
	}
	w.requests = append(w.requests, now)
	return true
}

func (w *window) remaining() int {
	w.mu.Lock()
	defer w.mu.Unlock()
	cutoff := time.Now().Add(-windowSpan)
	count := 0
	for _, t := range w.requests {
		if t.After(cutoff) {
			count++
		}
	}
	if r := w.limit - count; r > 0 {
		return r
	}
	return 0
}

// limiterRegistry Channel 窗口注册表.
type limiterRegistry struct {
	mu       sync.Mutex
	limiters map[string]*window
}

var (
	regOnce sync.Once
	reg     *limiterRegistry
)

func registry() *limiterRegistry {
	regOnce.Do(func() {
		reg = &limiterRegistry{limiters: make(map[string]*window)}
		go reg.cleanup()
	})
	return reg
}

func (r *limiterRegistry) get(key string, limit int) *window {
	r.mu.Lock()
	defer r.mu.Unlock()
	if w, ok := r.limiters[key]; ok {
		if limit >= 1 {
			w.mu.Lock()
			w.limit = limit
			w.mu.Unlock()
		}
		return w
	}
	w := newWindow(limit)
	r.limiters[key] = w
	return w
}

func (r *limiterRegistry) cleanup() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		r.mu.Lock()
		now := time.Now()
		for key, w := range r.limiters {
			w.mu.Lock()
			if len(w.requests) == 0 || now.Sub(w.requests[len(w.requests)-1]) > 10*time.Minute {
				delete(r.limiters, key)
			}
			w.mu.Unlock()
		}
		r.mu.Unlock()
	}
}
