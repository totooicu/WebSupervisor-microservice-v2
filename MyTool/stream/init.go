package stream

// 在 Init 中已初始化，这里保留空文件说明结构
import (
	"fmt"
	"time"
	"sync"
)
type HandlerFunc func(msg *StreamMessage)
var (
	HANDELS           = make(map[string]HandlerFunc)
	RESPONSES         = make(map[string]chan *StreamMessage)
	SHARDING_RESPONSES = make(map[string]*ShardingAssembler)
	mutex             sync.RWMutex
	semaphore         chan struct{}
	consumerStream      string
	activeHandlers    int32 // 当前正在执行的业务 handler 数量

	reassembledChan = make(chan *StreamMessage, 100)// 分片重组完成的消息通道
	shardManager = NewShardManager(reassembledChan)// 分片管理器

)
func RegisterService(name string, handler HandlerFunc) {
	mutex.Lock()
	defer mutex.Unlock()
	HANDELS[name] = handler
}


// Init 初始化库并启动消费者
func Init() error {
	if err := InitClient(); err != nil {
		return err
	}
	consumerStream = fmt.Sprintf("%s-%d", ServiceName, time.Now().UnixNano())
	semaphore = make(chan struct{}, cfg.Stream.GoroutineNum)

	// 注册内置响应处理
	RegisterService("response", handleResponse)
	RegisterService("ping", handlePing)
	// RegisterService("shard", handelShard)

	// 确保消费者组存在（如果 Stream 不存在则自动创建）
	if err := ensureConsumerGroup(); err != nil {
		return err
	}

	go consumeLoop()
	return nil
}