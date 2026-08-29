package response

import (
	"errors"
	"sync"
	"time"

	mysync "github.com/totooicu/go-mytool/sync"
)

// ResponseManager 响应管理器
type ResponseManager struct {
	responseMap  map[int]*mysync.Messages[interface{}]
	mu           sync.RWMutex
	maxResponses int
}

var (
	globalResponseManager *ResponseManager
	once                  sync.Once

	ErrResponseChannelNotFound = errors.New("response channel not found")
	ErrResponseTimeout         = errors.New("response timeout")
)

// InitResponseManager 初始化响应管理器
func InitResponseManager(maxResponses int) {
	once.Do(func() {
		globalResponseManager = &ResponseManager{
			responseMap:  make(map[int]*mysync.Messages[interface{}]),
			maxResponses: maxResponses,
		}
	})
}

// GetResponseManager 获取响应管理器实例
func GetResponseManager() *ResponseManager {
	if globalResponseManager == nil {
		InitResponseManager(1000)
	}
	return globalResponseManager
}

// CreateResponseChannel 创建响应通道
func (rm *ResponseManager) CreateResponseChannel(msgID int) {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	if len(rm.responseMap) >= rm.maxResponses {
		panic("response channel limit exceeded")
	}

	if channel, exists := rm.responseMap[msgID]; exists {
		channel.Clear()
	}

	rm.responseMap[msgID] = mysync.NewMessages[interface{}]()
}

// GetResponseChannel 获取响应通道
func (rm *ResponseManager) GetResponseChannel(msgID int) (*mysync.Messages[interface{}], bool) {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	channel, exists := rm.responseMap[msgID]
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

	rm.mu.Lock()
	delete(rm.responseMap, msgID)
	rm.mu.Unlock()

	return response
}

// GetResponseWithTimeout 带超时的响应获取
func (rm *ResponseManager) GetResponseWithTimeout(msgID int, timeout time.Duration) (interface{}, error) {
	channel, exists := rm.GetResponseChannel(msgID)
	if !exists {
		return nil, ErrResponseChannelNotFound
	}

	resultChan := make(chan interface{})
	go func() {
		resultChan <- channel.Get()
	}()

	select {
	case response := <-resultChan:
		rm.mu.Lock()
		delete(rm.responseMap, msgID)
		rm.mu.Unlock()
		return response, nil
	case <-time.After(timeout):
		rm.mu.Lock()
		delete(rm.responseMap, msgID)
		rm.mu.Unlock()
		return nil, ErrResponseTimeout
	}
}

// DeleteResponseChannel 删除响应通道
func (rm *ResponseManager) DeleteResponseChannel(msgID int) {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	delete(rm.responseMap, msgID)
}

// CleanExpiredChannels 清理过期通道
func (rm *ResponseManager) CleanExpiredChannels() {
	go func() {
		for {
			time.Sleep(5 * time.Minute)
			rm.mu.Lock()
			for msgID := range rm.responseMap {
				// 简单的清理逻辑，可以根据实际需求扩展
				if rm.responseMap[msgID].Size() == 0 {
					delete(rm.responseMap, msgID)
				}
			}
			rm.mu.Unlock()
		}
	}()
}
