package streamtool

import (
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"sync"
	"time"

	"github.com/totooicu/go-mytool/redis"
	"github.com/totooicu/go-mytool/streamtool/models"
	"github.com/totooicu/go-mytool/streamtool/response"
	mysync "github.com/totooicu/go-mytool/sync"
)

// StreamTool Stream开发工具包
type StreamTool struct {
	redisClient     *redis.Client
	responseManager *response.ResponseManager
	serviceMap      map[string]*mysync.Messages[interface{}]
	serviceNames    map[string]bool
	maxStreamLength int64 // 最大消息数量限制
	mu              sync.RWMutex
	debug           bool
}

var (
	globalStreamTool *StreamTool
	once             sync.Once
)

// InitStreamTool 初始化Stream工具
func InitStreamTool(config *models.StreamToolConfig) {
	once.Do(func() {
		redisClient := redis.NewClient(config.Redis.Host, config.Redis.Port, config.Redis.Password, config.Redis.DB)

		response.InitResponseManager(1000)
		responseManager := response.GetResponseManager()
		responseManager.CleanExpiredChannels()

		// 设置默认最大消息数量为1000
		maxStreamLength := int64(1000)
		if config.MaxStreamLength > 0 {
			maxStreamLength = int64(config.MaxStreamLength)
		}

		globalStreamTool = &StreamTool{
			redisClient:     redisClient,
			responseManager: responseManager,
			serviceMap:      make(map[string]*mysync.Messages[interface{}]),
			serviceNames:    make(map[string]bool),
			maxStreamLength: maxStreamLength,
			debug:           config.Debug,
		}

		for _, service := range config.Services {
			globalStreamTool.serviceNames[service.Name] = true
			globalStreamTool.serviceMap[service.Name] = mysync.NewMessages[interface{}]()
		}
	})
}

// GetStreamTool 获取Stream工具实例
func GetStreamTool() *StreamTool {
	if globalStreamTool == nil {
		panic("StreamTool not initialized")
	}
	return globalStreamTool
}

// GetMessageID 生成消息ID
func (st *StreamTool) GetMessageID() int {
	return int(time.Now().UnixNano())
}

// StreamPush 推送消息到stream
func (st *StreamTool) StreamPush(msg interface{}, stream string) bool {
	// 设置源信息
	if streamMsg, ok := msg.(*models.StreamMessage); ok {
		streamMsg.SourceStream = stream
		// 源服务名称可以从消息的ServiceName推断，或者由调用者设置
		if streamMsg.SourceService == "" {
			streamMsg.SourceService = streamMsg.ServiceName
		}
	}

	if err := st.redisClient.PublishMessageWithMaxLen(stream, msg, st.maxStreamLength); err != nil {
		if st.debug {
			log.Printf("Debug - Stream push failed: %v", err)
		}
		return false
	}
	return true
}

// StreamGet 从stream获取消息
func (st *StreamTool) StreamGet(stream, group, consumer string) (*models.StreamMessage, error) {
	messages, err := st.redisClient.ReadMessages(stream, group, consumer, 1, 1000)
	if err != nil {
		if err.Error() != "redis: nil" {
			return nil, fmt.Errorf("%w: %v", ErrStreamGetFailed, err)
		}
		return nil, nil
	}

	if len(messages) == 0 {
		return nil, nil
	}

	msg := messages[0]
	var streamMsg models.StreamMessage
	streamMsg.Playload = make(map[string]interface{})

	if messageValue, ok := msg.Values["message"]; ok {
		if messageStr, ok := messageValue.(string); ok {
			if err := json.Unmarshal([]byte(messageStr), &streamMsg); err != nil {
				if st.debug {
					log.Printf("Debug - Failed to unmarshal message: %v", err)
				}
			}
		}
	} else {
		if messageID, ok := msg.Values["message_id"].(string); ok {
			streamMsg.MessageID = messageID
		}
		if replyID, ok := msg.Values["reply_id"].(string); ok {
			streamMsg.ReplyID = replyID
		}
		if serviceName, ok := msg.Values["service_name"].(string); ok {
			streamMsg.ServiceName = serviceName
		}
		if callbackStream, ok := msg.Values["callback_stream"].(string); ok {
			streamMsg.CallbackStream = callbackStream
		}
		if playload, ok := msg.Values["playload"]; ok {
			switch v := playload.(type) {
			case string:
				// 尝试解析JSON字符串
				var playloadMap map[string]interface{}
				if err := json.Unmarshal([]byte(v), &playloadMap); err != nil {
					// 如果解析失败，直接作为字符串存储
					streamMsg.Playload["raw"] = v
				} else {
					streamMsg.Playload = playloadMap
				}
			case map[string]interface{}:
				streamMsg.Playload = v
			default:
				streamMsg.Playload["raw"] = v
			}
		}
	}

	if st.debug {
		log.Printf("Debug - Parsed stream message: %+v", streamMsg)
	}

	return &streamMsg, nil
}

// Send 发送消息并返回响应对象
func (st *StreamTool) Send(msg *models.StreamMessage, stream string) *Response {
	msgID := st.GetMessageID()
	msg.MessageID = strconv.Itoa(msgID)

	st.responseManager.CreateResponseChannel(msgID)

	if !st.StreamPush(msg, stream) {
		st.responseManager.DeleteResponseChannel(msgID)
		return nil
	}

	return &Response{
		msgID: msgID,
		rm:    st.responseManager,
	}
}

// Response 响应对象
type Response struct {
	msgID int
	rm    *response.ResponseManager
}

// Get 阻塞获取响应
func (r *Response) Get() interface{} {
	return r.rm.GetResponse(r.msgID)
}

// GetWithTimeout 带超时获取响应
func (r *Response) GetWithTimeout(timeout time.Duration) (interface{}, error) {
	return r.rm.GetResponseWithTimeout(r.msgID, timeout)
}

// RegisterService 注册服务
func (st *StreamTool) RegisterService(serviceName string) {
	st.mu.Lock()
	defer st.mu.Unlock()

	if _, exists := st.serviceMap[serviceName]; !exists {
		st.serviceMap[serviceName] = mysync.NewMessages[interface{}]()
		st.serviceNames[serviceName] = true
	}
}

// StartGateway 启动消息网关
func (st *StreamTool) StartGateway(inputStream, group, consumer string) {
	go func() {
		log.Println("Starting message gateway...")

		// 创建消费者组
		if err := st.redisClient.CreateConsumerGroup(inputStream, group); err != nil {
			log.Printf("Failed to create consumer group: %v", err)
		} else {
			log.Printf("Consumer group '%s' created for stream '%s'", group, inputStream)
		}

		for {
			msg, err := st.StreamGet(inputStream, group, consumer)
			if err != nil {
				log.Printf("Error getting message: %v", err)
				time.Sleep(1 * time.Second)
				continue
			}

			if msg == nil {
				continue
			}
			// log.Printf(">>>StartGateway Gateway received message: %+v", msg)
			//打印所有字段但是不打印playload
			pl := msg.Playload
			msg.Playload = nil
			log.Printf("Debug - Gateway received message: %+v   playload len: %d", msg, len(pl))
			//playload前后100字符
			pl_str := fmt.Sprintf("%v", pl)
			if len(pl_str) < 200 {
				log.Printf("Debug - playload: %s", pl_str)
			} else {
				log.Printf("Debug - playload: %s", pl_str[:100]+"<---------->"+pl_str[len(pl_str)-100:])
			}
			msg.Playload = pl

			if st.debug {
				log.Printf("Debug - Gateway received message: %+v", msg)
			}
			if msg.ServiceName == "response" {
				// 处理响应消息
				if replyID, err := strconv.Atoi(msg.ReplyID); err == nil {
					st.responseManager.SendResponse(replyID, msg.Playload)
				}
				continue
			}
			if _, exists := st.serviceNames[msg.ServiceName]; !exists {
				// 服务不存在，原路返回错误
				errorMsg := &models.StreamMessage{
					MessageID:      strconv.Itoa(st.GetMessageID()),
					ReplyID:        msg.MessageID,
					ServiceName:    "response",
					CallbackStream: msg.CallbackStream,
					Playload:       map[string]interface{}{"error": "service not found"},
				}
				st.StreamPush(errorMsg, msg.CallbackStream)
				continue
			}

			// 分发消息到对应服务
			if channel, exists := st.serviceMap[msg.ServiceName]; exists {
				channel.Put(msg)
			}
		}
	}()
}

// StartService 启动服务
func (st *StreamTool) StartService(serviceName string, handler func(msg *models.StreamMessage)) {
	if _, exists := st.serviceMap[serviceName]; !exists {
		st.RegisterService(serviceName)
	}

	go func() {
		log.Printf("Starting service: %s", serviceName)

		for {
			msg := st.serviceMap[serviceName].Get().(*models.StreamMessage)
			log.Printf(">>>StartService Service %s is running , msg: %+v", serviceName, msg)

			handler(msg)
		}
	}()
}

// StartAllServices 启动所有服务
func (st *StreamTool) StartAllServices(serviceHandlers map[string]func(msg *models.StreamMessage)) {
	for serviceName, handler := range serviceHandlers {
		st.StartService(serviceName, handler)
	}
}

// SendResponse 发送响应
func (st *StreamTool) SendResponse(replyID, callbackStream string, playload map[string]interface{}) {
	responseMsg := &models.StreamMessage{
		MessageID:      strconv.Itoa(st.GetMessageID()),
		ReplyID:        replyID,
		ServiceName:    "response",
		CallbackStream: callbackStream,
		Playload:       playload,
	}
	st.StreamPush(responseMsg, callbackStream)
}
