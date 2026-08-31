# email-service

SMTP 邮件服务：以"配置发送"或"自带配置发送"两种方式发出纯文本邮件。是 `web_supervisor-manager` 的通知出口。

## 架构

```
调用方 ──email_by_config / email_by_custom──► dev:email-stream ──► email-service ──SMTP──► 邮件服务器
```

- 监听 Stream：`dev:email-stream`
- 消费者组：`email-group`
- SMTP 连接：端口 `465` 走 SSL 直连；其它端口（如 `587`）先明文连接再协商 `STARTTLS`
- 邮件格式：`text/plain; charset=UTF-8`，发件人固定为 `email_config.username`

## config.json

### 通用字段（redis / stream）

见[总体 README](../../README.md#通用字段解释)。

| 字段 | 当前值 | 说明 |
|---|---|---|
| `stream.consumer_stream` | `dev:email-stream` | 本服务监听的 Stream |
| `stream.consumer_group` | `email-group` | 消费者组 |
| `stream.goroutine_num` | `5` | 并发上限（受限于邮件服务商限流，勿调太大） |
| `stream.max_message_bytes` | `204800` | 消息分片阈值 |
| `stream.cache_key_prefix` | `email:` | stream 库辅助缓存前缀 |

### custom 字段

| 字段 | 类型 | 说明 |
|---|---|---|
| `default_email` | `string` | 默认使用的邮箱配置名，对应 `emails` 中的键（如 `"qq_email"`）。`email_by_config` 未指定或指定不存在的 `email_choose` 时回退到它 |
| `emails` | `map[string]EmailConfig` | 邮箱账号配置表，键为配置名。支持在配置文件顶层任意位置使用 `${ENV}` 引用环境变量（账号/密码示例：`${QQ_MAIL_ACCOUNT_1134}`） |
| `default_wait_time_ms` | `int` | 预留字段，**当前版本代码未读取**，不起作用 |

`emails.*`（EmailConfig）结构：

| 字段 | 类型 | 说明 |
|---|---|---|
| `host` | `string` | SMTP 服务器，如 `smtp.qq.com` |
| `port` | `int` | 端口；`465` 走 SSL，其它（如 `587`）走 STARTTLS |
| `username` | `string` | SMTP 账号（同时作为发件人） |
| `password` | `string` | SMTP 授权码（QQ 邮箱为授权码而非登录密码） |
| `wait_time_ms` | `int` | 每次发送前的延迟毫秒数，用于规避服务商限流；0 不延迟 |

示例：

```json
"custom": {
  "default_email": "qq_email",
  "default_wait_time_ms": 5000,
  "emails": {
    "qq_email": {
      "host": "smtp.qq.com",
      "port": 587,
      "username": "${QQ_MAIL_ACCOUNT_1134}",
      "password": "${QQ_MAIL_PASSWORD_1134}",
      "wait_time_ms": 5000
    }
  }
}
```

## 服务接口

### `email_by_config` — 使用服务端配置发送

**请求 playload**（`models.EmailRequestByConfig`）

| 字段 | 类型 | 说明 |
|---|---|---|
| `email_choose` | `string` | `emails` 配置名；留空或不存在时使用 `default_email` |
| `email_content` | `object` | 邮件内容，结构见下 |

`email_content`（EmailContent）：

| 字段 | 类型 | 说明 |
|---|---|---|
| `tos` | `[]string` | 收件人列表（多个收件人逐个 RCPT） |
| `subject` | `string` | 主题 |
| `body` | `string` | 正文（纯文本） |

```json
{
  "email_choose": "qq_email",
  "email_content": {
    "tos": ["someone@example.com"],
    "subject": "监控通知",
    "body": "网页 xxx 发生变化"
  }
}
```

**响应 playload（成功）**：`null`（空），以 `err_code=0` 表示发送成功。
**错误**：配置不存在 / SMTP 连接、认证、发送失败 → `err_code=1`，`err_msg` 为具体错误。

### `email_by_custom` — 使用请求自带配置发送

**请求 playload**（`models.EmailRequestByCustom`）

| 字段 | 类型 | 说明 |
|---|---|---|
| `email_config` | `object` | 与 `emails.*` 相同的 EmailConfig 结构 |
| `email_content` | `object` | 同上 |

```json
{
  "email_config": {
    "host": "smtp.qq.com", "port": 465,
    "username": "xxx@qq.com", "password": "授权码",
    "wait_time_ms": 10000
  },
  "email_content": { "tos": ["a@b.com"], "subject": "hi", "body": "hello" }
}
```

**响应**：同 `email_by_config`。

## 调用示例

```go
resp, err := stream.Send("dev:email-stream", "email_by_config", map[string]interface{}{
    "email_choose": "", // 用 default_email
    "email_content": map[string]interface{}{
        "tos":     []string{"me@example.com"},
        "subject": "hello",
        "body":    "hello world",
    },
}, 0)
```
