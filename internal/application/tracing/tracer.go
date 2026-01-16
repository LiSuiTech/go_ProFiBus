package tracing

import (
	"context"
	"go_ProFiBus/pkg/interfaces"
	"sync"
	"time"

	"github.com/google/uuid"
)

// TracerImpl Tracer 实现
type TracerImpl struct {
	repository  interfaces.TraceRepository
	subscribers []chan interfaces.TraceEvent
	mu          sync.RWMutex
	bufferSize  int
	buffer      []*interfaces.TraceEvent
	bufferMu    sync.Mutex
	flushTicker *time.Ticker
	stopChan    chan struct{}
	wg          sync.WaitGroup
}

// NewTracer 创建追踪器
func NewTracer(repo interfaces.TraceRepository, bufferSize int) *TracerImpl {
	if bufferSize <= 0 {
		bufferSize = 100
	}

	tracer := &TracerImpl{
		repository:  repo,
		subscribers: make([]chan interfaces.TraceEvent, 0),
		bufferSize:  bufferSize,
		buffer:      make([]*interfaces.TraceEvent, 0, bufferSize),
		flushTicker: time.NewTicker(1 * time.Second),
		stopChan:    make(chan struct{}),
	}

	// 启动定期刷新
	tracer.wg.Add(1)
	go tracer.flushLoop()

	return tracer
}

// TraceDataFlow 实现 Tracer 接口
func (t *TracerImpl) TraceDataFlow(ctx context.Context, event *interfaces.TraceEvent) error {
	// 生成事件ID
	if event.ID == "" {
		event.ID = uuid.New().String()
	}

	// 设置时间戳
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}

	// 添加到缓冲区
	t.bufferMu.Lock()
	t.buffer = append(t.buffer, event)
	shouldFlush := len(t.buffer) >= t.bufferSize
	t.bufferMu.Unlock()

	// 缓冲区满时立即刷新
	if shouldFlush {
		go t.flush()
	}

	// 广播给订阅者
	t.broadcastEvent(*event)

	return nil
}

// GetTraces 实现 Tracer 接口
func (t *TracerImpl) GetTraces(ctx context.Context, filter interfaces.TraceFilter) ([]interfaces.TraceEvent, error) {
	// 先刷新缓冲区
	t.flush()

	return t.repository.QueryTraceEvents(ctx, filter)
}

// Subscribe 实现 Tracer 接口
func (t *TracerImpl) Subscribe() <-chan interfaces.TraceEvent {
	t.mu.Lock()
	defer t.mu.Unlock()

	ch := make(chan interfaces.TraceEvent, 256)
	t.subscribers = append(t.subscribers, ch)

	return ch
}

// Unsubscribe 实现 Tracer 接口
func (t *TracerImpl) Unsubscribe(ch <-chan interfaces.TraceEvent) {
	t.mu.Lock()
	defer t.mu.Unlock()

	for i, sub := range t.subscribers {
		if sub == ch {
			close(sub)
			t.subscribers = append(t.subscribers[:i], t.subscribers[i+1:]...)
			break
		}
	}
}

// Close 实现 Tracer 接口
func (t *TracerImpl) Close() error {
	close(t.stopChan)
	t.flushTicker.Stop()

	// 等待刷新完成
	t.wg.Wait()

	// 最后刷新一次
	t.flush()

	// 关闭所有订阅者
	t.mu.Lock()
	for _, ch := range t.subscribers {
		close(ch)
	}
	t.subscribers = nil
	t.mu.Unlock()

	return nil
}

// broadcastEvent 广播事件给所有订阅者
func (t *TracerImpl) broadcastEvent(event interfaces.TraceEvent) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	for _, ch := range t.subscribers {
		select {
		case ch <- event:
		default:
			// 通道满时跳过
		}
	}
}

// flushLoop 定期刷新循环
func (t *TracerImpl) flushLoop() {
	defer t.wg.Done()

	for {
		select {
		case <-t.stopChan:
			return
		case <-t.flushTicker.C:
			t.flush()
		}
	}
}

// flush 刷新缓冲区到数据库
func (t *TracerImpl) flush() {
	t.bufferMu.Lock()
	if len(t.buffer) == 0 {
		t.bufferMu.Unlock()
		return
	}

	// 交换缓冲区
	toFlush := t.buffer
	t.buffer = make([]*interfaces.TraceEvent, 0, t.bufferSize)
	t.bufferMu.Unlock()

	// 批量写入数据库
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := t.repository.SaveTraceEventsBatch(ctx, toFlush); err != nil {
		// 记录错误但不阻塞
		// TODO: 添加日志
	}
}

// GetSubscriberCount 获取订阅者数量（用于调试）
func (t *TracerImpl) GetSubscriberCount() int {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return len(t.subscribers)
}

// GetBufferSize 获取当前缓冲区大小（用于调试）
func (t *TracerImpl) GetBufferSize() int {
	t.bufferMu.Lock()
	defer t.bufferMu.Unlock()
	return len(t.buffer)
}

// 确保实现了接口
var _ interfaces.Tracer = (*TracerImpl)(nil)
