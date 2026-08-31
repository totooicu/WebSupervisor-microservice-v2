# calculator-service

计算服务，也是使用 [stream 库](../../README.md#如何用-stream-库构建一个微服务)构建微服务的**最小示例**。

## 架构

```
调用方 ──add/divide──► dev:calculator-stream ──► calculator-service ──response──► 调用方
```

- 监听 Stream：`dev:calculator-stream`（由 config.json `stream.consumer_stream` 决定）
- 消费者组：`calculator-group`
- 无状态、无外部依赖，适合作为压测样例（`microservice-manager` 中有用 100 万元素数组做串行/并发压测的示例）

## config.json

### 通用字段（redis / stream）

见[总体 README](../../README.md#通用字段解释)。本服务：

| 字段 | 当前值 | 说明 |
|---|---|---|
| `stream.consumer_stream` | `dev:calculator-stream` | 本服务监听的 Stream |
| `stream.consumer_group` | `calculator-group` | 消费者组 |
| `stream.goroutine_num` | `5` | handler 并发上限 |
| `stream.get_timeout_ms` | `5000` | 调用方默认超时参考值 |
| `stream.max_message_bytes` | `204800` | 消息分片阈值（200KB） |

### custom 字段

当前为空 `{}`，本服务无业务配置。

## 服务接口

### `add` — 数组求和

**请求 playload**

| 字段 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `nums` | `[]float64` | 是 | 参与求和的数字数组 |

```json
{ "nums": [1, 2, 3.5] }
```

**响应 playload（成功）**

| 字段 | 类型 | 说明 |
|---|---|---|
| `result` | `float64` | 求和结果 |

```json
{ "result": 6.5 }
```

**错误**：playload 无法解析时返回 `err_msg = "invalid payload"`。

### `divide` — 除法运算

对数组每个元素执行 `a / v`，返回同长度数组。

**请求 playload**

| 字段 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `a` | `float64` | 是 | 被除数 |
| `nums` | `[]float64` | 是 | 除数数组 |

```json
{ "a": 10, "nums": [2, 4, 5] }
```

**响应 playload（成功）**

```json
{ "result": [5, 2.5, 2] }
```

**错误**：数组中任一元素为 0 → `err_msg = "division by zero"`；解析失败 → `invalid payload`。

## 调用示例

```go
resp, err := stream.Send("dev:calculator-stream", "add",
    map[string]interface{}{"nums": []float64{1, 2, 3}}, 5000)
// resp.Playload -> {"result": 6}

resp, err = stream.Send("dev:calculator-stream", "divide",
    map[string]interface{}{"a": 10.0, "nums": []float64{2, 4}}, 5000)
// resp.Playload -> {"result": [5, 2.5]}
```
