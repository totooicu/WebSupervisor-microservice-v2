# parser-service

解析服务：从文本（HTML / JSON）中按规则提取数据。是 `web_supervisor-manager` 监控任务的解析环节。

## 架构

```
调用方 ──parse_*──► dev:parser-stream ──► parser-service ──► MyTool/http/parser
   ▲                                              │
   └──────────── response: {parsed_data} ◄───────┘
```

- 监听 Stream：`dev:parser-stream`
- 消费者组：`parser-group`
- 提取实现：`GetMid`（左右标记+正则过滤）、`htmlquery`（XPath）、`ParseJSON`（点分路径）

## config.json

### 通用字段（redis / stream）

见[总体 README](../../README.md#通用字段解释)。

| 字段 | 当前值 | 说明 |
|---|---|---|
| `stream.consumer_stream` | `dev:parser-stream` | 本服务监听的 Stream |
| `stream.consumer_group` | `parser-group` | 消费者组 |
| `stream.goroutine_num` | `5` | 并发上限 |
| `stream.max_message_bytes` | `204800` | 消息分片阈值 |
| `stream.cache_key_prefix` | `parser:` | stream 库辅助缓存前缀 |

### custom 字段

当前为空 `{}`，本服务无业务配置。

## 服务接口

三个服务共用一个参数骨架（`models.ParserParameter`），按服务取用其中对应字段：

```go
type ParserParameter struct {
    Content  string    `json:"content"`   // 待解析文本
    HTMLKeys []HTMLKey `json:"htmlKeys"`  // parse_html_by_get_mid 使用
    JSONKeys []JSONKey `json:"jsonKeys"`  // parse_json 使用
    XPaths   []XPathKey `json:"xPaths"`   // parse_html_by_xpath 使用
}
```

### `parse_html_by_get_mid` — 左右标记提取 HTML 片段

**请求 playload**

| 字段 | 类型 | 说明 |
|---|---|---|
| `content` | `string` | HTML 文本 |
| `htmlKeys` | `array` | 提取规则数组，元素结构见下 |

`htmlKeys` 元素：

| 字段 | 类型 | 说明 |
|---|---|---|
| `left` | `string` | 左边界标记，**支持正则**（如 `"<li id=\"line_u9_\\d+\">"`) |
| `right` | `string` | 右边界标记，支持正则 |
| `key` | `[]string` | 正则数组，片段命中**任意一条**即保留（OR 过滤）；`[".*"]` 表示全部保留 |

```json
{
  "content": "<ul><li id=\"line_u9_1\">公告A</li><li id=\"line_u9_2\">公告B</li></ul>",
  "htmlKeys": [
    { "left": "<li id=\"line_u9_\\d+\">", "right": "</li>", "key": [".*"] }
  ]
}
```

**响应 playload**

| 字段 | 类型 | 说明 |
|---|---|---|
| `parsed_data` | `map[string][]string` | key 为 `"[left,right,[keys]]"`（规则字符串化），value 为提取出的片段列表 |

```json
{
  "parsed_data": {
    "[<li id=\"line_u9_\\d+\">,</li>,[.*]]": ["公告A", "公告B"]
  }
}
```

**已知限制**：结果过滤时统一使用 `htmlKeys[0]`（第一个规则）的 `key` 正则数组对所有规则的片段做过滤。因此多个规则同时使用时，请保证过滤意图与第一个规则的 `key` 一致（通常各规则都写 `[".*"]`）。

### `parse_html_by_xpath` — XPath 提取

**请求 playload**

| 字段 | 类型 | 说明 |
|---|---|---|
| `content` | `string` | HTML 文本 |
| `xPaths` | `array` | 元素结构：`{ "xpath": "//ul/li", "attrName": "href" }` |

- `attrName` 为空字符串：提取匹配节点的 InnerText（文本）
- `attrName` 非空：提取节点对应属性值（如 `href`、`title`）

```json
{
  "content": "<html>...</html>",
  "xPaths": [ { "xpath": "/html/body/div[5]//ul/li", "attrName": "" } ]
}
```

**响应 playload**

```json
{
  "parsed_data": {
    "/html/body/div[5]//ul/li": ["公告A", "公告B"]
  }
}
```

key 即 xpath 表达式本身。

### `parse_json` — 点分路径提取 JSON 值

**请求 playload**

| 字段 | 类型 | 说明 |
|---|---|---|
| `content` | `string` | JSON 文本 |
| `jsonKeys` | `array` | 元素结构：`{ "path": ["data", "list", 0, "title"] }` |

- `path` 数组元素为字符串（对象键）或数字（数组下标），内部用 `.` 连接后逐层取值
- 元素中的 `key` 字段当前版本**未使用**

```json
{
  "content": "{\"data\":{\"list\":[{\"title\":\"公告A\"}]}}",
  "jsonKeys": [ { "path": ["data", "list", 0, "title"] } ]
}
```

**响应 playload**

```json
{
  "parsed_data": {
    "data.list.0.title": ["公告A"]
  }
}
```

key 为点分路径字符串，value 为长度 1 的数组。

## 调用示例

```go
resp, err := stream.Send("dev:parser-stream", "parse_html_by_get_mid",
    map[string]interface{}{
        "content": htmlText,
        "htmlKeys": []interface{}{
            map[string]interface{}{
                "left": "<li id=\"line_u9_\\d+\">", "right": "</li>",
                "key":  []string{".*"},
            },
        },
    }, 10000)
log.Println(resp.Playload["parsed_data"])
```
