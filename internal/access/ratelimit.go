package access

import (
	"sync"
	"time"
)

const windowSpan = time.Minute

// anonBucket 匿名桶全局单例: 全部匿名调用共享 (与术语表一致), 阈值管理面可调.
var anonBucket = newWindow(300)

// InitAnonLimit 设置匿名桶阈值 (立即生效).
func InitAnonLimit(n int) { anonBucket.setLimit(n) }

// AnonLimit 返回匿名桶当前阈值.
func AnonLimit() int { return anonBucket.currentLimit() }

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

// setLimit 更新窗口阈值; 运行时可调.
func (w *window) setLimit(n int) {
	if n < 1 {
		n = 1
	}
	w.mu.Lock()
	w.limit = n
	w.mu.Unlock()
}

// currentLimit 读窗口当前阈值.
func (w *window) currentLimit() int {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.limit
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
