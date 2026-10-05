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
	flushBatch    = 50
	flushEvery    = 10 * time.Second
	queueCap      = 1024
	retentionDays = 30 // 调用明细保留窗口; 累计口径由 daily_calls 承载, 不受影响.
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

// Start 启动异步落库与明细清理; 生命周期由组合根驱动.
func Start() {
	srv.queue = make(chan CallEntry, queueCap)
	srv.done = make(chan struct{})
	srv.wg.Add(2)
	go srv.loop()
	go srv.cleanup()
}

// Stop 停止并落盘队列剩余条目.
func Stop() {
	if srv.done != nil {
		close(srv.done)
		srv.wg.Wait()
	}
}

// cleanup 定期删除超出保留窗口的调用明细.
func (s *service) cleanup() {
	defer s.wg.Done()
	prune := func() {
		n, err := store.DeleteCallLogsBefore(time.Now().UTC().AddDate(0, 0, -retentionDays))
		if err != nil {
			log.Printf("[metrics] prune call logs failed: %v", err)
			return
		}
		if n > 0 {
			log.Printf("[metrics] pruned %d call logs older than %dd", n, retentionDays)
		}
	}
	prune()
	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			prune()
		case <-s.done:
			return
		}
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
		type aggKey struct {
			date string
			ch   uint
		}
		agg := make(map[aggKey]int64)
		for _, l := range buf {
			ch := uint(0)
			if l.ChannelID != nil {
				ch = *l.ChannelID
			}
			k := aggKey{l.CreatedAt.UTC().Format("2006-01-02"), ch}
			agg[k]++
		}
		deltas := make([]store.DailyCallDelta, 0, len(agg))
		for k, n := range agg {
			deltas = append(deltas, store.DailyCallDelta{Date: k.date, ChannelID: k.ch, N: n})
		}
		if err := store.FlushCallLogs(buf, deltas); err != nil {
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
