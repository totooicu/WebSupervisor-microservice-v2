# microservice-manager

**客户端示例工程**：不对外提供业务服务，而是演示如何通过 stream 库调用本项目中的各个微服务。适合作为学习"如何写调用方"的起点。

## 架构

- 监听 Stream：`dev:client-stream`（仅用于接收响应，自身不注册业务 handler）
- 一次性运行程序：`main.go` 执行 `actions.Action()` 后睡眠 2 秒退出
- `actions/` 目录按被调服务分文件，每个文件是一种服务的调用范例：

| 文件 | 演示内容 | 目标服务 |
|---|---|---|
| `Basic.go` | `ping` 健康检查 | parser-service |
| `Calculator.go` | `add` 压测（串行 vs 并发 100 万元素数组） | calculator-service |
| `Crawler.go` | `http_request` 抓取网页 | crawler-service |
| `Parser.go` | `parse_html_by_xpath` 解析本地 HTML | parser-service |
| `RedisCache.go` | `set/get/compare_and_save/delete/get_and_set` | redis_cache-service |
| `Email.go` | `email_by_config` / `email_by_custom` | email-service |

在 `actions/main.go` 的 `Action()` 中取消注释即可运行对应测试。

## config.json

### 通用字段（redis / stream）

见[总体 README](../../README.md#通用字段解释)。

| 字段 | 当前值 | 说明 |
|---|---|---|
| `stream.consumer_stream` | `dev:client-stream` | 客户端接收响应的 Stream |
| `stream.consumer_group` | `client-group` | 消费者组 |
| `stream.get_timeout_ms` | `50000` | `Send` 默认超时 50 秒 |
| `stream.max_message_bytes` | `204800` | 消息分片阈值 |

### custom 字段

| 字段 | 类型 | 说明 |
|---|---|---|
| `calculator-stream` | `string` | calculator 服务的目标 Stream 名称。示例展示的"把下游服务 Stream 名放进 custom"是推荐做法：换环境时只改配置不改代码 |

其它服务的目标 Stream 在示例代码中直接硬编码（如 `"redis_cache-stream"`），接入真实项目时建议统一改为从 `custom` 读取（`web_supervisor-manager` 即是如此）。

## 使用方法

```bash
cd microservice/microservice-manager
go build -o microservice-manager.exe .
./microservice-manager.exe
```

前提：Redis 已运行，且要调用的目标服务已启动（可只启动被测服务并注释掉 `Action()` 中其它测试）。

## 典型调用片段

```go
// 目标 Stream 建议放配置里
targetStream := stream.Custom["calculator-stream"].(string)

// 发送请求：Send(目标Stream, 服务名, playload, 超时毫秒)
resp, err := stream.Send(targetStream, "add",
    map[string]interface{}{"nums": []float64{1, 2, 3}}, 0)
if err != nil {
    log.Fatal(err) // 超时或对端 ResponseErr
}
log.Println(resp.Playload) // {"result":6}
```
