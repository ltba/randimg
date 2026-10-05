// Package metrics 调用统计: 公开面调用的异步计量, 批量落库与统计聚合.
// 模块定义见 .trellis/spec/arch/metrics.md.
package metrics

import (
	"log"
	"sync"
	"time"

	"randimg/internal/store"
)

const (
	flushBatch = 50
	flushEvery = 10 * time.Second
	queueCap   = 1024
)

// CallEntry 单次调用计量; ChannelID 为空表示匿名调用.
type CallEntry struct {
	ChannelID *uint
	Path      string
}

type service struct {
	queue   chan CallEntry
	dropped uint64
	mu      sync.Mutex
	done    chan struct{}
	wg      sync.WaitGroup
}

var srv = &service{}

// Record 非阻塞计量; 队列满时丢弃并计数, 不阻塞请求路径.
func Record(e CallEntry) {
	select {
	case srv.queue <- e:
	default:
		srv.mu.Lock()
		srv.dropped++
		srv.mu.Unlock()
	}
}

// Dropped 返回累计丢弃条目数.
func Dropped() uint64 {
	srv.mu.Lock()
	defer srv.mu.Unlock()
	return srv.dropped
}

// Start 启动异步落库; 生命周期由组合根驱动.
func Start() {
	srv.queue = make(chan CallEntry, queueCap)
	srv.done = make(chan struct{})
	srv.wg.Add(1)
	go srv.loop()
}

// Stop 停止并落盘队列剩余条目.
func Stop() {
	if srv.done != nil {
		close(srv.done)
		srv.wg.Wait()
	}
}

func (s *service) loop() {
	defer s.wg.Done()
	buf := make([]store.CallLog, 0, flushBatch)
	ticker := time.NewTicker(flushEvery)
	defer ticker.Stop()

	flush := func() {
		if len(buf) == 0 {
			return
		}
		if err := store.BatchInsertCallLogs(buf); err != nil {
			log.Printf("[metrics] flush failed: %v", err)
		}
		buf = buf[:0]
	}

	for {
		select {
		case e := <-s.queue:
			buf = append(buf, store.CallLog{ChannelID: e.ChannelID, Path: e.Path, CreatedAt: time.Now().UTC()})
			if len(buf) >= flushBatch {
				flush()
			}
		case <-ticker.C:
			flush()
		case <-s.done:
			for {
				select {
				case e := <-s.queue:
					buf = append(buf, store.CallLog{ChannelID: e.ChannelID, Path: e.Path, CreatedAt: time.Now().UTC()})
				default:
					flush()
					return
				}
			}
		}
	}
}
