package response

import (
	mysync "github.com/totooicu/go-mytool/sync"
	"sync"
)

// ResponseManager 响应管理器
type ResponseManager struct {
	channels    map[int]*mysync.Messages[interface{}]
	mu          sync.RWMutex
	maxChannels int
}

var (
	// GlobalResponseManager 全局响应管理器实例
	GlobalResponseManager *ResponseManager
	once                 sync.Once
)

// InitResponseManager 初始化响应管理器
func InitResponseManager(maxChannels int) {
	once.Do(func() {
		GlobalResponseManager = &ResponseManager{
			channels:    make(map[int]*mysync.Messages[interface{}]),
			maxChannels: maxChannels,
		}
	})
}

// GetResponseManager 获取响应管理器实例
func GetResponseManager() *ResponseManager {
	if GlobalResponseManager == nil {
		InitResponseManager(1000) // 默认最大通道数
	}
	return GlobalResponseManager
}

// CreateResponseChannel 创建响应通道
func (rm *ResponseManager) CreateResponseChannel(msgID int) {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	// 检查通道数量限制
	if len(rm.channels) >= rm.maxChannels {
		panic("response channel limit exceeded")
	}

	// 如果通道已存在，先关闭并删除
	if channel, exists := rm.channels[msgID]; exists {
		channel.Clear()
	}

	// 创建新通道
	rm.channels[msgID] = mysync.NewMessages[interface{}]()
}

// GetResponseChannel 获取响应通道
func (rm *ResponseManager) GetResponseChannel(msgID int) (*mysync.Messages[interface{}], bool) {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	channel, exists := rm.channels[msgID]
	return channel, exists
}

// SendResponse 发送响应到通道
func (rm *ResponseManager) SendResponse(msgID int, payload interface{}) bool {
	channel, exists := rm.GetResponseChannel(msgID)
	if !exists {
		return false
	}

	channel.Put(payload)
	return true
}

// GetResponse 阻塞等待响应
func (rm *ResponseManager) GetResponse(msgID int) interface{} {
	channel, exists := rm.GetResponseChannel(msgID)
	if !exists {
		return nil
	}

	response := channel.Get()
	
	// 获取响应后删除通道，避免内存泄漏
	rm.mu.Lock()
	delete(rm.channels, msgID)
	rm.mu.Unlock()
	
	return response
}

// DeleteResponseChannel 删除响应通道
func (rm *ResponseManager) DeleteResponseChannel(msgID int) {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	delete(rm.channels, msgID)
}

// GetActiveChannelsCount 获取活跃通道数量
func (rm *ResponseManager) GetActiveChannelsCount() int {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	return len(rm.channels)
}

// ClearAllChannels 清空所有通道（谨慎使用）
func (rm *ResponseManager) ClearAllChannels() {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	for msgID := range rm.channels {
		if channel, exists := rm.channels[msgID]; exists {
			channel.Clear()
			delete(rm.channels, msgID)
		}
	}
}