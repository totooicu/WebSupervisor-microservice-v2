package mysync

import (
	"sync"
)

// Messages 泛型线程安全消息队列
type Messages[T any] struct {
	messages []T
	mu       sync.RWMutex
	cond     *sync.Cond
}

// NewMessages 创建新的泛型消息队列
func NewMessages[T any]() *Messages[T] {
	m := &Messages[T]{
		messages: make([]T, 0),
	}
	m.cond = sync.NewCond(&m.mu)
	return m
}

// Put 向队列追加消息
func (m *Messages[T]) Put(msg T) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.messages = append(m.messages, msg)

	// 通知等待的消费者有新消息
	m.cond.Signal()
}

// Get 从队列取出消息，队列为空时阻塞
func (m *Messages[T]) Get() T {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 如果队列为空，等待直到有消息
	for len(m.messages) == 0 {
		m.cond.Wait()
	}

	// 取出第一个消息
	msg := m.messages[0]
	m.messages = m.messages[1:]

	return msg
}

// Size 返回队列中的消息数量
func (m *Messages[T]) Size() int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return len(m.messages)
}

// Peek 查看但不移除第一个消息
func (m *Messages[T]) Peek() (T, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if len(m.messages) == 0 {
		var zero T
		return zero, false
	}
	return m.messages[0], true
}

// Clear 清空队列
func (m *Messages[T]) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.messages = make([]T, 0)
}

// GetAll 获取并清空队列中的所有消息（非必需，但很有用）
func (m *Messages[T]) GetAll() []T {
	m.mu.Lock()
	defer m.mu.Unlock()

	messages := m.messages
	m.messages = make([]T, 0)
	return messages
}

// PutBatch 批量添加消息
func (m *Messages[T]) PutBatch(msgs ...T) {
	if len(msgs) == 0 {
		return
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	m.messages = append(m.messages, msgs...)

	// 通知所有等待的消费者
	if len(msgs) > 1 {
		m.cond.Broadcast()
	} else {
		m.cond.Signal()
	}
}
