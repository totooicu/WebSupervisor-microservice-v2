# redis_cache-service

缓存服务：提供带命名空间（`app`）的 Redis 键值缓存，核心能力是 **`compare_and_save` 变更检测** —— 新旧数据不同才写入并上报 `changed=true`，是 `web_supervisor-manager` 判断"网页内容是否更新"的依据。

## 架构

```
调用方 ──compare_and_save/get/set/delete/get_and_set──► dev:redis_cache-stream
                                                            │
                                                    redis_cache-service
                                                            │
                                             存储Redis（custom.redis，可与通信Redis不同）
```

- 监听 Stream：`dev:redis_cache-stream`
- 消费者组：`redis_cache-group`
- 存储端：由 `custom.redis` 指定的 Redis（默认可与通信 Redis 同实例）
- 实际存储 key：`<app>:<key>`；数据用 JSON 序列化保存；**TTL=0 永不过期**

## config.json

### 通用字段（redis / stream）

见[总体 README](../../README.md#通用字段解释)。

| 字段 | 当前值 | 说明 |
|---|---|---|
| `stream.consumer_stream` | `dev:redis_cache-stream` | 本服务监听的 Stream |
| `stream.consumer_group` | `redis_cache-group` | 消费者组 |
| `stream.goroutine_num` | `5` | 并发上限 |
| `stream.max_message_bytes` | `204800` | 消息分片阈值 |
| `stream.cache_key_prefix` | `redis_cache:` | stream 库辅助缓存前缀 |

### custom 字段

| 字段 | 类型 | 说明 |
|---|---|---|
| `redis` | `object` | **存储用的 Redis 连接**：`{ "addr": "localhost:6379", "password": "", "db": 0 }`。缓存数据的读写都走这个实例，可与顶层 `redis`（通信 Redis）分开部署 |

## 服务接口

所有服务共用请求参数（`models.CacheParameter`）：

| 字段 | 类型 | 说明 |
|---|---|---|
| `app` | `string` | 命名空间，最终 key 为 `app:key` |
| `key` | `string` | 数据键 |
| `data` | `any` | 数据值（任意可 JSON 序列化内容），`get`/`delete` 不需要 |

### `compare_and_save` — 比较并保存（变更检测）

**请求**：`{ "app": "web_supervisor", "key": "https://a.com:[...]", "data": ["新内容"] }`

**响应 playload（成功）**

```json
{ "changed": true }
```

**行为**：
- key 不存在（首次）：先 `get` 报错 → 服务返回**错误响应**（`err_code=1`），调用方通常随后调用 `set` 写入
- key 存在且数据与 `data` 不同：写入新数据，返回 `{"changed": true}`
- key 存在且数据相同：不写入，返回 `{"changed": false}`

比较方式：新旧数据 JSON 序列化后字符串比对。

### `get` — 读取

**请求**：`{ "app": "app", "key": "k" }`

**响应 playload**（无论是否读到，均走成功响应）：

```json
{ "key": "app:k", "data": ["缓存内容"], "error": false }
```

`error=true` 表示 key 不存在或读取失败，此时 `data` 为 null。

### `set` — 写入

**请求**：`{ "app": "app", "key": "k", "data": {"a": 1} }`

**响应**：`{ "key": "app:k", "error": null }`

### `delete` — 删除

**请求**：`{ "app": "app", "key": "k" }`

**响应**：`{ "key": "app:k", "error": null }`

### `get_and_set` — 读取旧值并写入新值

**请求**：同 `set`。

**响应 playload**：

```json
{ "key": "app:k", "old_data": {"a": 1}, "changed": true, "error": null }
```

key 不存在时返回错误响应（与 `compare_and_save` 首次行为一致）。

## 调用示例

```go
// 变更检测
resp, err := stream.Send("dev:redis_cache-stream", "compare_and_save",
    map[string]interface{}{"app": "web_supervisor", "key": "page1", "data": newContent}, 10000)
if err != nil {
    // key 不存在 → 首次写入
    stream.Send("dev:redis_cache-stream", "set",
        map[string]interface{}{"app": "web_supervisor", "key": "page1", "data": newContent}, 10000)
} else if resp.Playload["changed"].(bool) {
    // 数据发生了变化
}
```
