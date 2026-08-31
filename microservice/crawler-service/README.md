# crawler-service

HTTP 抓取服务：接收一个 HTTP 请求描述，代为执行并返回响应体与状态码。是 `web_supervisor-manager` 监控任务的抓取环节。

## 架构

```
调用方 ──http_request──► dev:crawler-stream ──► crawler-service ──► 目标网站
   ▲                                                        │
   └──────────── response: {content, status} ◄──────────────┘
```

- 监听 Stream：`dev:crawler-stream`
- 消费者组：`crawler-group`
- 依赖 `MyTool/http` 的 HttpClient 完成实际请求

## config.json

### 通用字段（redis / stream）

见[总体 README](../../README.md#通用字段解释)。

| 字段 | 当前值 | 说明 |
|---|---|---|
| `stream.consumer_stream` | `dev:crawler-stream` | 本服务监听的 Stream |
| `stream.consumer_group` | `crawler-group` | 消费者组 |
| `stream.goroutine_num` | `5` | 并发抓取上限（注意对目标站点的压力） |
| `stream.max_message_bytes` | `204800` | 消息分片阈值（200KB） |
| `stream.cache_key_prefix` | `crawler:` | stream 库辅助缓存前缀 |

### custom 字段

当前为空 `{}`，本服务无业务配置。

## 服务接口

### `http_request` — 执行 HTTP 请求

**请求 playload**（`models.CrawlerParameter`）

| 字段 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `url` | `string` | 是 | 目标地址；缺失时**不回复任何消息**（调用方超时） |
| `method` | `string` | 是 | `"GET"` 或 `"POST"`；其它值同样不回复 |
| `headers` | `map[string]string` | 否 | 请求头（字段名 `headers`，见下方已知差异说明） |
| `body` | `map[string]any` | POST 时 | JSON 请求体（POST 时与 `str_payload` 二选一） |
| `str_payload` | `string` | POST 时 | 原始字符串请求体 |

```json
{
  "url": "https://example.com/list.htm",
  "method": "GET",
  "headers": { "User-Agent": "Mozilla/5.0" },
  "body": {},
  "str_payload": ""
}
```

POST 示例：

```json
{
  "url": "https://example.com/api",
  "method": "POST",
  "headers": { "Content-Type": "application/json" },
  "body": { "page": 1, "size": 20 }
}
```

**响应 playload（成功）**

| 字段 | 类型 | 说明 |
|---|---|---|
| `content` | `string` | 响应体字符串 |
| `status` | `int` | HTTP 状态码（请求失败可能为 0） |

```json
{ "content": "<html>...</html>", "status": 200 }
```

**注意**：本服务只在 `method` 分支执行成功后回复消息；`url` 缺失、`method` 不支持、序列化失败等情况下**静默丢弃**，调用方将等到超时。

## 已知差异

- `models.CrawlerParameter` 的请求头字段 JSON tag 为 `headers`（复数）；而 `web_supervisor-manager` 内部模型使用 `header`（单数）并按 `header` 读取 jobs.json。调用本服务时请以 `headers` 为准。

## 调用示例

```go
resp, err := stream.Send("dev:crawler-stream", "http_request",
    map[string]interface{}{
        "url":    "https://example.com",
        "method": "GET",
        "headers": map[string]string{"User-Agent": "Mozilla/5.0"},
        "body":   map[string]interface{}{},
    }, 1000*60*2) // 抓取可能较慢，建议加大超时
if err == nil {
    log.Println(resp.Playload["status"], resp.Playload["content"])
}
```
