# web_supervisor-manager

**网页监控编排器**：按固定周期执行 `jobs.json` 中定义的监控任务 —— 抓取网页 → 解析片段 → 与上次内容比较 → 有变化则发送邮件通知。它是本项目的顶层应用，串联了 crawler / parser / redis_cache / email 四个微服务。

> 只关心"怎么写 jobs.json"的用户请直接阅读 [JOBS_GUIDE.md](JOBS_GUIDE.md)。

## 架构

```
                     ┌────────────────────────────────────────────┐
                     │            web_supervisor-manager          │
                     │  循环: ping 自检 → 逐个执行 job → 等待间隔   │
                     └───────┬──────────┬──────────┬─────────┬────┘
              1 http_request │ 2 parse  │ 3 compare_and_save │ 4 email_by_config
                     ┌───────▼───┐ ┌────▼─────┐ ┌──────▼──────┐ ┌──▼─────────┐
                     │  crawler  │ │  parser  │ │ redis_cache │ │   email    │
                     │  -service │ │ -service │ │  -service   │ │  -service  │
                     └───────────┘ └──────────┘ └─────────────┘ └────────────┘
```

- 自身 Stream：`dev:web_supervisor-stream`（仅用于接收四个下游服务的响应；不注册对外业务服务）
- 主循环（`actions/Run.go`）：
  1. `PingServices()`：依次 ping 四个下游服务，任一失败则等 1 秒重试，直到全部健康
  2. 遍历 `jobs.json` 中的每个任务执行 `run_one`：
     - **抓取**：`http_request` → `{content, status}`；`status != 200` 跳过本任务
     - **解析**：按规则选择 parser 服务（优先级 `jsonKeys` > `xPathKeys` > `htmlKeys`；全空则默认整页提取），得到 `parsed_data`
     - **变更检测**：每个片段以 `key = <url>:[left,right,[keys]]`、`app = redis_app_name` 调用 `compare_and_save`；key 不存在时改用 `set` 写入
     - **通知**：存在变化片段时调用 `email_by_config` 发送邮件（收件人 = `custom.email_tos`），正文列出变化的 key 与新内容（单条截断 1000 字符，总量截断 100000 字符）
  3. 打印倒计时并等待 `interval_second` 秒，回到第 1 步

> **首次运行提醒**：所有片段第一次写入缓存都会被记为"变化"，因此首轮会收到一封汇总全部片段的邮件，从第二轮起才只报真正变化的内容。

## config.json

### 通用字段（redis / stream）

见[总体 README](../../README.md#通用字段解释)。

| 字段 | 当前值 | 说明 |
|---|---|---|
| `stream.consumer_stream` | `dev:web_supervisor-stream` | 自身 Stream（收响应用） |
| `stream.consumer_group` | `web_supervisor-group` | 消费者组 |
| `stream.get_timeout_ms` | `50000` | Send 默认超时 |
| `stream.max_message_bytes` | `2048000` | 分片阈值 2MB（页面内容较大时调高） |
| `stream.cache_key_prefix` | `web_supervisor:` | stream 库辅助缓存前缀 |

### custom 字段（本服务全部业务配置）

| 字段 | 类型 | 说明 |
|---|---|---|
| `crawler-service_consumer_stream` | `string` | crawler-service 的目标 Stream，如 `dev:crawler-stream` |
| `parser-service_consumer_stream` | `string` | parser-service 的目标 Stream |
| `redis_cache-service_consumer_stream` | `string` | redis_cache-service 的目标 Stream |
| `email-service_consumer_stream` | `string` | email-service 的目标 Stream |
| `jobs_path` | `string` | 任务文件路径，如 `./jobs.json`（结构见 [JOBS_GUIDE.md](JOBS_GUIDE.md)） |
| `redis_app_name` | `string` | 变更检测写入 redis_cache 时使用的 `app` 命名空间，如 `web_supervisor` |
| `email_tos` | `[]string` | 通知邮件收件人列表，支持 `${ENV}`（示例：`["${QQ_MAIL_ACCOUNT_2667}"]`） |
| `interval_second` | `number` | 每轮监控的间隔秒数（jobs.json 里的 `intervalSecond` 不生效，以此为准） |

> 注意 `email_tos` 只读取**一层的 `[]any`**，元素需为字符串；若写对象会被转为空串。

## jobs.json

任务定义文件的完整字段说明、解析规则写法与正则转义等注意事项，见 **[JOBS_GUIDE.md](JOBS_GUIDE.md)**。该文档同时提供"让 AI 帮你生成 jobs.json"的提示词模板。

## 运行

```bash
cd microservice/web_supervisor-manager
go build -o web_supervisor-manager.exe .
./web_supervisor-manager.exe
```

前提：Redis、crawler/parser/redis_cache/email 四个服务均已启动；`config.json` 与 `jobs.json` 位于工作目录。

## 已知限制

- `xPathKeys` 规则在 supervisor 与 parser 之间字段名不匹配（发送 `xpathKeys` / 接收 `xPaths`），当前版本 XPath 提取实际拿不到数据，建议使用 `htmlKeys` 或 `jsonKeys`（详见 JOBS_GUIDE.md）。
- 修改某个任务的 `left`/`right`/`key` 会改变缓存 key，导致该任务重新全量缓存并触发一次"首次"邮件通知。
