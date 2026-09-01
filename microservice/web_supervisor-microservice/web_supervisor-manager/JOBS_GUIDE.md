# jobs.json 使用说明书

`web_supervisor-manager` 通过 `config.json` 中 `custom.jobs_path`（默认 `./jobs.json`）加载监控任务。每个任务 = **抓取一个网页 → 提取若干片段 → 与上次比较 → 有变化发邮件**。

本文档的目标：让你（或 AI）不看源码就能写出正确的 `jobs.json`。

---

## 1. 运行流程（理解字段作用的前提）

每轮监控（间隔 = `config.json` 的 `custom.interval_second`）对每个任务执行：

```
1. 抓取   crawler-service   : 按 url/method/header/body 发 HTTP 请求
                            status != 200 时本任务本轮跳过
2. 解析   parser-service    : 按 htmlKeys/jsonKeys/xPathKeys 从 content 提取片段
                            得到 parsed_data = { "规则串": [片段1, 片段2, ...] }
3. 比对   redis_cache-service: 每个片段以 key = <url>:[规则串] 与上次比较
4. 通知   email-service     : 有变化的片段 → 发邮件给 custom.email_tos
```

> **首次运行**：所有片段第一次写入缓存都会被记为"变化"，首轮会收到一封汇总全部片段的邮件；第二轮起只报真正变化的内容。

---

## 2. 顶层结构

```json
{
  "projectName": "test",
  "header": {},
  "intervalSecond": 120,
  "urls": [ ... ]
}
```


| 字段             | 类型     | 必填   | 说明                                                            |
| ------------------ | ---------- | -------- | ----------------------------------------------------------------- |
| `urls`           | `array`  | **是** | 任务数组，**唯一被程序读取的字段**                              |
| `projectName`    | `string` | 否     | 当前版本不读取，仅为可读性保留                                  |
| `header`         | `object` | 否     | 当前版本不读取（每个任务用自己的`header`）                      |
| `intervalSecond` | `number` | 否     | 当前版本不读取！间隔由 config.json`custom.interval_second` 决定 |

---

## 3. urls[] 任务条目

每个条目描述"抓取哪个页面 + 提取什么"：

```json
{
  "url": "https://cmt.zstu.edu.cn/zsjy/yjszs.htm",
  "method": "GET",
  "header": {
    "User-Agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64)",
    "Accept": "*/*"
  },
  "body": {},
  "str_payload": "",
  "htmlKeys": [
    { "left": "<li id=\"line_u9_\\d+\">", "right": "</li>", "key": [".*"] }
  ],
  "jsonKeys": [],
  "xPathKeys": []
}
```

### 字段总表


| 字段                       | 类型     | 必填           | 说明                                          |
| ---------------------------- | ---------- | ---------------- | ----------------------------------------------- |
| `url`                      | `string` | ✅             | 目标页面地址                                  |
| `method`                   | `string` | ✅             | `"GET"` 或 `"POST"`（其它值任务会被静默跳过） |
| `header`                   | `object` | 建议           | 请求头，键值对，**每个值必须是字符串**        |
| `body`                     | `object` | POST 时        | JSON 请求体（键值任意）                       |
| `str_payload`              | `string` | POST 时        | 原始字符串请求体（与`body` 二选一）           |
| `htmlKeys`                 | `array`  | 三者至少配一个 | 左右标记提取规则（见 §4）                    |
| `jsonKeys`                 | `array`  |                | JSON 路径提取规则（见 §5）                   |
| `xPathKeys`                | `array`  |                | XPath 提取规则（见 §6，当前版本有已知问题）  |
| `output` / `test` / `type` | -        | 否             | **旧版遗留字段，当前版本不读取**，可删可留    |

> `htmlKeys`/`jsonKeys`/`xPathKeys` 不用的留空数组即可；三个都为空时程序会默认整页提取（`left=<html>`、`right=</html>`），一般不是你想要的。

---

## 4. htmlKeys —— 左右标记提取（HTML 页面首选，最常用）

从 `content` 中找出所有 `left … right` 之间的内容。适合提取"公告列表"等重复结构。


| 字段    | 类型       | 说明                                                                 |
| --------- | ------------ | ---------------------------------------------------------------------- |
| `left`  | `string`   | 左边界标记，**支持正则**                                             |
| `right` | `string`   | 右边界标记，支持正则                                                 |
| `key`   | `[]string` | 正则数组：片段命中**任意一条**即保留（OR 过滤）；`[".*"]` = 全部保留 |

### 正则转义（重要）

JSON 字符串里反斜杠要写两个：想在正则里写 `\d+`，JSON 必须写 `"\\d+"`；引号同理要写 `\"`。

```json
{ "left": "<li id=\"line_u9_\\d+\">", "right": "</li>", "key": [".*"] }
```

### 关键规则与已知限制

1. **结果 key 格式**为 `"[left,right,[keys]]"`（规则原样字符串化），并拼接进缓存 key：`<url>:[left,right,[keys]]`。
   ⇒ **修改 left/right/key 会换缓存 key**，导致重新全量缓存并再收到一封"首次"邮件。
2. **过滤正则只取 `htmlKeys[0]`（第一条规则）的 `key`**，并对所有规则的片段统一过滤。
   ⇒ 建议每条规则都写 `"key": [".*"]`；如需精确过滤，把正则放在第一条规则上，且一个任务内各规则的 key 保持一致。
3. `left`/`right` 是逐字符定位+正则查找，**不需要转义 HTML 特殊字符本身**，直接照网页源代码原样粘贴边界片段即可（如 `"<div class=\"main-right-text\">"`）。

### 选边界的技巧

- 打开目标页面 → 查看源代码 → 找到目标列表紧邻的、**稳定且唯一**的结构片段（如 `<ul class="list">`、`<li id="line_u9_数字">`、`</li>`、`<script>`）作为边界；
- 边界越贴近内容、越不含会变化的文字越好；
- 每条公告一个片段：用重复的行级标签（`<li …>`/`</li>`、`<dd …>`/`</dd>`）做边界。

---

## 5. jsonKeys —— JSON 路径提取（接口/JSON 页面）


| 字段   | 类型       | 说明                                                 |
| -------- | ------------ | ------------------------------------------------------ |
| `path` | `[]mixed`  | 路径数组：字符串 = 对象键，数字 = 数组下标，逐层取值 |
| `key`  | `[]string` | 当前版本**未使用**，占位即可                         |

```json
{
  "url": "https://example.com/api/list",
  "method": "GET",
  "jsonKeys": [
    { "path": ["data", "list", 0, "title"], "key": [] }
  ]
}
```

- 结果 key 为点分路径：`"data.list.0.title"`，value 为取到的值组成的数组；
- 取不到的路径会被静默跳过（不出现在结果里）。

---

## 6. xPathKeys —— XPath 提取（当前版本勿用）


| 字段       | 类型     | 说明                                       |
| ------------ | ---------- | -------------------------------------------- |
| `xpath`    | `string` | XPath 表达式                               |
| `attrName` | `string` | 为空取节点文本；非空取该属性值（如`href`） |

```json
{ "xpath": "//ul[@class='list']/li", "attrName": "" }
```

> ⚠️ **已知问题**：`web_supervisor-manager` 发送的字段名为 `xpathKeys`，而 parser-service 期望 `xPaths`，两者不匹配，**当前版本 XPath 提取实际收不到参数、结果为空**。请改用 `htmlKeys` 或 `jsonKeys`。此条保留是为了兼容旧文件格式。

---

## 7. 解析方式的选择（一个任务多种规则时）

`web_supervisor-manager` 按以下顺序互相覆盖，**最终只有一个生效**：

```
htmlKeys → xPathKeys → jsonKeys      （jsonKeys 优先级最高）
```

所以：**一个任务只配一种规则最稳妥**。想监控同一页面的 HTML 片段和 JSON 接口，请拆成两个任务条目。

---

## 8. 完整示例

```json
{
  "projectName": "my-monitor",
  "urls": [
    {
      "url": "https://cmt.zstu.edu.cn/zsjy/yjszs.htm",
      "method": "GET",
      "header": {
        "User-Agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64)",
        "Accept": "*/*"
      },
      "body": {},
      "str_payload": "",
      "htmlKeys": [
        { "left": "<li id=\"line_u9_\\d+\">", "right": "</li>", "key": [".*"] }
      ],
      "jsonKeys": [],
      "xPathKeys": []
    },
    {
      "url": "https://example.com/api/notice",
      "method": "GET",
      "header": { "User-Agent": "Mozilla/5.0" },
      "body": {},
      "str_payload": "",
      "htmlKeys": [],
      "jsonKeys": [
        { "path": ["data", "list", 0, "title"], "key": [] }
      ],
      "xPathKeys": []
    }
  ]
}
```

---

## 9. 让 AI 生成 jobs.json（提示词模板）

把下面的模板连同目标页面信息发给 AI（附上网页源代码片段效果最好）：

```
请帮我生成一份 WebSupervisor 的 jobs.json 监控任务文件，并严格遵守以下规范：

【顶层结构】只关注 "urls" 数组（projectName/header/intervalSecond 不生效）。
【任务条目字段】
- url: 目标地址
- method: 只能是 "GET" 或 "POST"
- header: 键值对，每个值必须是字符串
- body: POST 的 JSON 请求体（GET 时留 {}）
- str_payload: POST 原始字符串体（可留 ""）
- htmlKeys / jsonKeys / xPathKeys: 三选一配置，其余留空数组；不要使用 xPathKeys（当前版本有 bug）
【htmlKeys 规则】元素为 {"left": 正则, "right": 正则, "key": [".*"]}；
  left/right 支持正则，JSON 内反斜杠必须写成 \\（如 <li id="line_u9_\d+"> 要写成
  "<li id=\\"line_u9_\\\\d+\\">"），引号写成 \"；每条规则 key 固定写 [".*"]。
【jsonKeys 规则】元素为 {"path": ["层1","层2",0,"字段"], "key": []}，数字表示数组下标。
【输出要求】只输出合法的 JSON，不要多余解释。

目标页面信息如下：
URL: https://xxx.edu.cn/zsjy/yjszs.htm
抓取方式: GET
想监控的内容: 研究生招生的公告列表（每条公告一个片段）
网页源代码（目标区域）:
<!-- 在此粘贴网页源代码片段 -->
```

生成后按 §10 排查一遍再投入使用。

---

## 10. 常见问题排查


| 现象                         | 可能原因 / 处理                                                                                                                                                     |
| ------------------------------ | --------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 结果一直为空                 | ①`left`/`right` 与网页源代码不一致（注意动态渲染的页面要抓"源代码"而非 F12 的 DOM）；② 正则转义错误（`\d` 必须写成 `\\d`）；③ 该页实际是 JSON，应改用 `jsonKeys` |
| 任务被跳过                   | crawler 返回`status != 200`：检查 `header`（尤其 `User-Agent`、`Host`），有些站点需要去掉多余的 Host 头                                                             |
| 每轮都收到通知               | 提取片段里包含时间戳/随机数等天然变化内容 → 收紧`left`/`right`，把易变部分排除在片段外                                                                             |
| 改了规则后收到"全部变化"通知 | 正常：改 left/right/key 会换缓存 key，首轮全部视为新内容                                                                                                            |
| 邮件内容被截断               | 预期行为：单片段最多 1000 字符、整封正文最多 100000 字符                                                                                                            |
| POST 请求无响应              | `method` 必须大写 `GET`/`POST`；`body` 值不必都是字符串（只有 `header` 的值必须是字符串）                                                                           |
| 想改监控频率                 | 改`config.json` 的 `custom.interval_second`（jobs.json 的 `intervalSecond` 不生效）                                                                                 |
