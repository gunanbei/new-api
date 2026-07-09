# 在途日志「调试日志」功能设计文档

> **版本**: 1.0  
> **日期**: 2026-07-09  
> **状态**: 待实现  
> **目标读者**: 实现本功能的 Coding Agent

---

## 1. 背景与目标

### 1.1 背景

项目在途日志（Inflight Logs）功能已在 Redis 中记录请求的**元数据**（状态、时间线、重试链等），但不包含完整的 HTTP 请求/响应内容。调试时用户需要查看客户端与网关之间的原始交互。

### 1.2 目标

在在途日志列表的**右键菜单**和**行末操作菜单**中新增「**调试日志**」入口，弹出与「详情」风格类似的卡片，展示该请求的完整 HTTP 请求与响应（含 headers），便于调试。

### 1.3 非目标（Out of Scope）

- 不记录网关 ↔ 上游 channel 的请求/响应
- 不记录每次 retry 的独立 upstream 交互
- 不支持 WebSocket Realtime（`/v1/realtime`）的帧级 trace
- 不在「详情」弹窗内增加跳转链接
- 不支持管理员查看其他用户的 trace（仅请求所有者）

---

## 2. 已确认的产品决策

| # | 决策项 | 结论 |
|---|--------|------|
| 1 | 记录范围 | 仅 **客户端 ↔ 网关** |
| 2 | 完整度 | **尽量完整记录**（在管理员配置上限内） |
| 3 | 菜单与记录开关联动 | `记录关闭` → **菜单强制隐藏**（`TraceMenuVisible` 不可单独生效） |
| 4 | 权限 | **仅请求所有者**（沿用 `/inflight/self` 归属校验） |
| 5 | 体积限制 | 默认 `0` = 不额外限制（仍受全局 `MAX_REQUEST_BODY_MB` 约束）；管理员可配；支持跟在途日志一样清理 |
| 6 | 查看时机 | **仅 terminal 后**（`completed` / `failed`）可打开菜单和 API |
| 7 | 记录范围 | **全部请求**（成功/失败）均记录，带状态标记 |
| 8 | 管理入口 | **系统设置 → 运维 → 日志维护**（`LogSettingsSection`） |
| Q1 | 流式展示 | 默认 **原始 SSE 文本**，提供切换到「事件列表」 |
| Q2 | 二进制响应 | 仅 **metadata + 下载按钮**，页面内不预览 |
| Q3 | multipart 请求 | 记录 **原始 multipart 字节**（base64），展示为可下载的二进制 |
| Q4 | 截断策略 | 超上限时保留 **头部 N 字节** + 标记 `truncated` |
| Q5 | `Max*Bytes = 0` | `0` = 不额外限制 |
| Q6 | 历史 trace | 关闭记录开关后，**已存在的 trace 仍可查看**，直到被清理 |
| Q7 | 菜单文案 | **「调试日志」**（i18n key: `Debug Log`） |
| Q8 | 详情弹窗跳转 | **不要** |

---

## 3. 架构概览

```
┌──────────────┐     ┌─────────────────────────────────────────┐
│   Client     │────▶│  Gin Relay (controller/relay.go)        │
└──────────────┘     │  ├─ traceCapture (request 缓存)          │
                     │  ├─ traceResponseWriter (response tee)   │
                     │  └─ on terminal → persist trace (async)   │
                     └──────────────┬──────────────────────────┘
                                    │
                     ┌──────────────▼──────────────────────────┐
                     │  Redis                                     │
                     │  inflight:item:{request_id}  (现有元数据)  │
                     │  inflight:trace:{request_id} (新增 trace)  │
                     └──────────────┬──────────────────────────┘
                                    │
                     ┌──────────────▼──────────────────────────┐
                     │  API                                       │
                     │  GET /api/log/inflight/self (列表+meta)    │
                     │  GET /api/log/inflight/self/:id/trace      │
                     └──────────────┬──────────────────────────┘
                                    │
                     ┌──────────────▼──────────────────────────┐
                     │  Frontend (inflight-tasks-tab.tsx)       │
                     │  菜单「调试日志」→ InflightTaskTraceDialog │
                     └─────────────────────────────────────────┘
```

### 3.1 核心设计原则

1. **Trace 与元数据分离存储**：列表 `MGet` 不包含 body，避免性能退化
2. **Terminal 后一次性写入**：进行中不落 Redis trace key，降低复杂度
3. **异步持久化**：不阻塞 relay 热路径
4. **清理联动**：删除 `inflight:item` 时同步删除 `inflight:trace`
5. **默认全关**：双开关默认 `false`，完全兼容现有部署

---

## 4. 管理员配置

### 4.1 配置项

放在 `web/default/src/features/system-settings/maintenance/log-settings-section.tsx`，在现有在途清理配置下方新增分组 **「Inflight Debug Log」**（i18n 待翻译）。

| Option Key | 类型 | 默认值 | 说明 |
|------------|------|--------|------|
| `InflightTaskTraceEnabled` | bool | `false` | 是否记录 trace。关闭时：不写入、菜单隐藏、新请求不产生 trace |
| `InflightTaskTraceMenuVisible` | bool | `false` | 是否在菜单显示「调试日志」。仅当 `TraceEnabled=true` 时可编辑；否则 UI 灰显 |
| `InflightTaskTraceMaxRequestBytes` | int | `0` | 请求体记录上限（字节）。`0` = 不额外限制 |
| `InflightTaskTraceMaxResponseBytes` | int | `0` | 响应体记录上限（字节）。`0` = 不额外限制 |

### 4.2 需修改的文件（配置链路）

| 文件 | 改动 |
|------|------|
| `model/option.go` | 注册默认值；`UpdateOption` switch 分支同步到 `common` 变量 |
| `common/constants.go` 或 `service/inflight_trace.go` | 运行时读取的配置变量/函数 |
| `controller/option.go` | 无需特殊处理（走通用 Option 更新） |
| `web/default/src/features/system-settings/types.ts` | `OperationsSettings` 增加 4 个字段 |
| `web/default/src/features/system-settings/operations/index.tsx` | 默认值 |
| `web/default/src/features/system-settings/operations/section-registry.tsx` | 传给 `LogSettingsSection` |
| `web/default/src/features/system-settings/maintenance/log-settings-section.tsx` | UI 开关与数字输入 |
| `web/default/src/features/system-settings/hooks/use-update-option.ts` | 白名单增加新 key（若有限制） |

### 4.3 设置页文案要点

- 开启后会增加 Redis 占用
- 可能包含 API Key、用户 prompt 等敏感信息
- 请求/响应头中的密钥字段会自动脱敏，但 body 不做字段级脱敏
- 流式响应会拼接为完整文本后存储
- `TraceMenuVisible` 依赖 `TraceEnabled`

---

## 5. 数据模型

### 5.1 Redis Key

```
inflight:trace:{request_id}
```

- 与 `inflight:item:{request_id}` **独立**
- TTL 与 item 相同（`inflightTaskRetentionTTL`：terminal 为 0/持久，非 terminal 有过期）
- 由于仅 terminal 后写入，实际为持久化直到清理

### 5.2 Go 结构体

新建 `service/inflight_trace.go`：

```go
const inflightTaskTraceKeyPrefix = "inflight:trace:"

type InflightTaskTrace struct {
    RequestID  string `json:"request_id"`
    UserID     int    `json:"user_id"`
    Status     string `json:"status"`      // completed | failed
    Kind       string `json:"kind"`        // chat | image | audio
    ModelName  string `json:"model_name"`
    IsStream   bool   `json:"is_stream"`
    CreatedAt  int64  `json:"created_at"`
    UpdatedAt  int64  `json:"updated_at"`
    RecordedAt int64  `json:"recorded_at"`

    ClientRequest  *InflightTraceHTTPPart `json:"client_request,omitempty"`
    ClientResponse *InflightTraceHTTPPart `json:"client_response,omitempty"`

    Flags InflightTaskTraceFlags `json:"flags"`
}

type InflightTaskTraceFlags struct {
    RequestTruncated    bool `json:"request_truncated"`
    ResponseTruncated   bool `json:"response_truncated"`
    ResponseIncomplete  bool `json:"response_incomplete"`
    UnsupportedRealtime bool `json:"unsupported_realtime,omitempty"`
}

type InflightTraceHTTPPart struct {
    Method      string            `json:"method,omitempty"`
    Path        string            `json:"path,omitempty"`
    Query       string            `json:"query,omitempty"`
    Protocol    string            `json:"protocol,omitempty"`
    StatusCode  int               `json:"status_code,omitempty"`
    Headers     map[string]string `json:"headers,omitempty"`
    Body        string            `json:"body,omitempty"`
    BodyEncoding string           `json:"body_encoding,omitempty"` // text | base64 | empty
    BodyBytes   int64             `json:"body_bytes,omitempty"`
    ContentType string            `json:"content_type,omitempty"`
}
```

### 5.3 列表接口轻量扩展

`InflightTask` 结构增加（仅 API 响应，Redis item 不存）：

```go
HasTrace bool `json:"has_trace"`
```

或在 `ListUserInflightTasks` 返回前，对 terminal 任务批量 `EXISTS inflight:trace:{id}`。

列表响应增加 `meta`：

```json
{
  "success": true,
  "data": {
    "items": [...],
    "total": 100,
    "page": 1,
    "page_size": 100,
    "meta": {
      "trace_menu_visible": true
    }
  }
}
```

`trace_menu_visible = InflightTaskTraceEnabled() && InflightTaskTraceMenuVisible()`

---

## 6. 记录逻辑（后端核心）

### 6.1 启用条件

```go
func InflightTaskTraceEnabled() bool {
    // 读 OptionMap["InflightTaskTraceEnabled"]
}

func InflightTaskTraceMenuVisible() bool {
    if !InflightTaskTraceEnabled() {
        return false
    }
    // 读 OptionMap["InflightTaskTraceMenuVisible"]
}
```

### 6.2 请求捕获

**时机**：`controller/relay.go` 中 `GenRelayInfo` 成功后，且 `InflightTaskTraceEnabled()` 为 true。

**来源**：
- Method / Path / Query：`c.Request`
- Headers：`c.Request.Header`（脱敏后）
- Body：`common.GetBodyStorage(c)` → `Bytes()`

**Realtime 排除**：`relayFormat == types.RelayFormatOpenAIRealtime` 时不启动 trace capture（或标记 `unsupported_realtime`）。

**内存持有**：请求周期内将 capture state 挂在 `gin.Context`：

```go
const ContextKeyInflightTraceCapture = "inflight_trace_capture"
```

### 6.3 响应捕获

参考 `middleware/audit.go` 的 `auditResponseWriter`，新建 `traceResponseWriter`：

```go
type traceResponseWriter struct {
    gin.ResponseWriter
    buf           *bytes.Buffer
    maxBytes      int64  // 0 = unlimited
    totalWritten  int64
    truncated     bool
    statusCode    int
    headers       http.Header // 从 ResponseWriter 收集
}
```

- 在 `Write` / `WriteString` 中 tee 到 `buf`（受 `maxBytes` 约束，超出后 `truncated=true` 但继续透传）
- 记录 `Status()` / `WriteHeader` 的 status code
- 响应 headers 在 `WriteHeader` 时快照

**包装时机**：relay 入口启用 trace 时 `c.Writer = traceWriter`。

### 6.4 Terminal 后持久化

在 `UpdateInflightTaskStatusAsync` 检测到 terminal status（`completed` / `failed`）时，或在 `controller/relay.go` 的 defer 中：

```go
if isTerminal && capture != nil {
    gopool.Go(func() {
        persistInflightTaskTrace(ctx, capture, relayInfo, finalStatus)
    })
}
```

`persistInflightTaskTrace` 职责：
1. 组装 `InflightTaskTrace`
2. 应用请求/响应 body 上限（`InflightTaskTraceMaxRequestBytes` / `MaxResponseBytes`）
3. 设置 `flags`
4. `common.Marshal` → Redis `SET inflight:trace:{request_id}`
5. TTL 与 item 对齐（调用与 `persistInflightTask` 相同的 TTL 逻辑）

### 6.5 Body 编码规则

| 条件 | `body_encoding` | `body` |
|------|-----------------|--------|
| 空 body | `empty` | 省略或 `""` |
| 有效 UTF-8 文本 | `text` | 原文字符串 |
| 非 UTF-8 | `base64` | 标准 base64 |

### 6.6 敏感 Header 脱敏

记录前将以下 header（大小写不敏感）的值替换为 `***`：

```
Authorization
X-Api-Key
Api-Key
Cookie
Set-Cookie
X-Goog-Api-Key
Proxy-Authorization
```

请求体/响应体：**不做字段级脱敏**。

### 6.7 截断规则（Q4A）

当 `maxBytes > 0` 且 body 超过上限：
- 保留前 `maxBytes` 字节
- `flags.request_truncated` 或 `flags.response_truncated = true`
- `body_bytes` 记录原始总大小

当 `maxBytes == 0`：
- 不截断（请求侧仍受 `constant.MaxRequestBodyMB` 约束，超大请求可能根本无法进入 relay）

### 6.8 失败场景

| 场景 | 处理 |
|------|------|
| 响应未写出（连接断开） | 写 trace，`client_response` 为空或仅 status，`flags.response_incomplete=true` |
| Redis 不可用 | 静默失败，不影响 relay |
| BodyStorage 读取失败 | 记录 headers，`client_request.body` 为空，`flags` 可增 `request_unavailable`（可选） |

---

## 7. API 规格

### 7.1 列表（扩展）

```
GET /api/log/inflight/self
```

**现有行为不变**，扩展：

- 每条 item 增加 `has_trace: bool`（仅 terminal 任务检测 trace key 存在性）
- `data.meta.trace_menu_visible: bool`

**性能注意**：`has_trace` 检测应批量 `EXISTS` 或 pipeline，避免 N 次往返。仅对 `completed`/`failed` 的 item 检测。

### 7.2 Trace 详情（新增）

```
GET /api/log/inflight/self/:request_id/trace
```

| 项 | 说明 |
|----|------|
| Auth | `middleware.UserAuth()` |
| 归属 | trace 的 `user_id` 必须等于当前用户 |
| Terminal | 对应 inflight item 必须为 terminal；否则 `409` + message |
| 响应 | 完整 `InflightTaskTrace` JSON |

**错误码**：

| HTTP | 场景 |
|------|------|
| 404 | trace key 不存在 |
| 403 | 非所有者 |
| 409 | 任务仍在进行中 |
| 503 | Redis 不可用 |

**路由注册**（`router/api-router.go`）：

```go
logRoute.GET("/inflight/self/:request_id/trace", middleware.UserAuth(), controller.GetUserInflightTaskTrace)
```

### 7.3 Stats（扩展，建议）

`GET /api/log/inflight/stats` 扩展 `InflightTaskStats`：

```go
type InflightTaskStats struct {
    UserCount      int64 `json:"user_count"`
    ItemCount      int64 `json:"item_count"`
    TotalSize      int64 `json:"total_size"`
    TraceCount     int64 `json:"trace_count"`      // 新增
    TraceTotalSize int64 `json:"trace_total_size"` // 新增
}
```

`GetInflightTaskStats` 扫描 `inflight:trace:*` 前缀计数。

---

## 8. 清理逻辑

### 8.1 自动清理

复用 `InflightTaskCleanupRule` 与 cron 清理：在删除 `inflight:item:{id}` 时 **同步 `DEL inflight:trace:{id}`**。

修改 `DeleteTerminalInflightTasksBefore`（`service/inflight_task.go`）：

```go
removeKeys = append(removeKeys, keys[i])
removeKeys = append(removeKeys, inflightTaskTraceKey(requestID)) // 新增
```

自动 cron 清理路径同样走此函数，无需额外改动。

### 8.2 手动清理

现有「清理在途日志」系统任务（`POST /api/system_task/inflight-log-cleanup`）通过 `DeleteTerminalInflightTasksBefore` 执行，trace 自动联动删除。

### 8.3 Stats 一致性

清理后 `trace_count` / `trace_total_size` 应下降（下次查询 stats 时反映）。

---

## 9. 前端实现

### 9.1 文件规划

| 文件 | 职责 |
|------|------|
| `web/default/src/features/usage-logs/components/inflight-task-trace-dialog.tsx` | **新建**：调试日志弹窗 |
| `web/default/src/features/usage-logs/components/inflight-tasks-tab.tsx` | 菜单项、状态、打开弹窗 |
| `web/default/src/features/usage-logs/types.ts` | TS 类型（可选，或放 dialog 文件内） |
| `web/default/src/features/system-settings/maintenance/log-settings-section.tsx` | 管理开关 |
| `web/default/src/i18n/locales/*.json` | 6 语言翻译 |

**建议**：将 trace dialog 抽离为独立文件，避免 `inflight-tasks-tab.tsx`（已 1900+ 行）继续膨胀。

### 9.2 菜单入口

在以下两处添加「调试日志」（i18n: `Debug Log`）：

1. `ContextMenuContent` 内 `ContextMenuItem`
2. `DataTableRowActionMenu` 内 `DropdownMenuItem`

**显示条件**（同时满足）：

```ts
meta?.trace_menu_visible === true
&& row.original.has_trace === true
&& isInflightTaskTerminal(row.original)
```

**不添加**行末独立 icon 按钮。  
**不在** `InflightTaskDetailsDialog` 内添加跳转链接。

### 9.3 弹窗 `InflightTaskTraceDialog`

复用 `Dialog` 组件（与 `InflightTaskDetailsDialog` 一致）：

```
标题: t('Debug Log')
描述: {request_id} · {model_name} · {status} · {stream?}
contentClassName: 与详情弹窗相同
```

#### Tab 结构

使用 `Tabs`：`Overview` | `Request` | `Response`

**Tab: Overview（概览）**

| 字段 | 来源 |
|------|------|
| Request ID | mono，可复制 |
| Status | Badge（completed 绿 / failed 红） |
| Kind | chat / image / audio |
| Stream | Yes / No |
| Recorded At | `recorded_at` |
| Request Size | `client_request.body_bytes` 格式化 |
| Response Size | `client_response.body_bytes` 格式化 |
| 截断警告 | `flags.request_truncated` / `response_truncated` → Alert |
| 不完整警告 | `flags.response_incomplete` → Alert |

**Tab: Request（请求）**

1. **行信息**：`POST /v1/chat/completions`
2. **请求头表格**：两列 Header / Value；右上角「复制 Headers」
3. **请求体**：
   - 工具栏：`格式化 JSON` | `原始` | `复制` | `下载`
   - `application/json` → 默认格式化
   - `multipart/*` → 显示「Binary multipart (N bytes). Download to inspect.」+ 下载按钮（从 base64 解码）
   - 其他 text → `whitespace-pre-wrap`
   - 截断时顶部黄条：「Content truncated to first N bytes」

**Tab: Response（响应）**

1. **状态行**：`HTTP 200 OK`
2. **响应头表格**（同请求）
3. **响应体**：
   - `application/json` → 格式化 JSON
   - `text/event-stream` → **默认原始 SSE 视图**；提供子切换「Event List」
   - `image/*` / `audio/*` / 其他 binary → metadata + 下载按钮，**不内联预览**
   - 截断/不完整警告同上

#### SSE Event List 模式（Q1A）

解析 `data: ...\n\n` 为可折叠列表：

```
▼ [1] data: {"id":"chatcmpl-xxx",...}
▼ [2] data: {"choices":[{"delta":{"content":"Hi"}}]}
▶ [48] data: [DONE]
```

解析失败时回退到原始 SSE 文本视图。

#### 数据加载

```ts
useQuery({
  queryKey: ['inflight-trace', requestId],
  queryFn: () => api.get(`/api/log/inflight/self/${requestId}/trace`),
  enabled: open && !!requestId,
})
```

Terminal 任务无需轮询（trace 一次性写入）。

#### 复用组件

- `InflightDetailRow` / `InflightDetailSection`：从 `inflight-tasks-tab.tsx` 提取到共享文件，或在 trace dialog 内复制最小子集（优先提取到 `inflight-detail-primitives.tsx` 避免循环依赖）
- `CodeBlock`（`@/components/ai-elements/code-block`）用于 JSON/SSE 展示
- `useCopyToClipboard`（参考 `prompt-dialog.tsx`）
- 下载：`Blob` + `URL.createObjectURL`

### 9.4 TypeScript 类型

```ts
export type InflightTaskTraceFlags = {
  request_truncated: boolean
  response_truncated: boolean
  response_incomplete: boolean
  unsupported_realtime?: boolean
}

export type InflightTraceHTTPPart = {
  method?: string
  path?: string
  query?: string
  protocol?: string
  status_code?: number
  headers?: Record<string, string>
  body?: string
  body_encoding?: 'text' | 'base64' | 'empty'
  body_bytes?: number
  content_type?: string
}

export type InflightTaskTrace = {
  request_id: string
  user_id: number
  status: string
  kind: string
  model_name: string
  is_stream: boolean
  created_at: number
  updated_at: number
  recorded_at: number
  client_request?: InflightTraceHTTPPart
  client_response?: InflightTraceHTTPPart
  flags: InflightTaskTraceFlags
}
```

扩展 `InflightTask`：

```ts
has_trace?: boolean
```

扩展列表响应 `data.meta`：

```ts
meta?: { trace_menu_visible: boolean }
```

### 9.5 Mock 模式

`?mock=inflight` 时：
- 为 1-2 条 mock 任务提供 `has_trace: true`
- `fetchInflightTaskTrace` mock 返回示例 JSON + SSE 响应

---

## 10. i18n 新增 Key（英文源字符串）

实现时使用 `i18n-translate` skill 补全 6 语言。

| Key | 英文 |
|-----|------|
| `Debug Log` | Debug Log |
| `Record inflight debug logs` | Record inflight debug logs |
| `Show debug log menu item` | Show debug log menu item |
| `Max inflight debug log request size (bytes)` | Max inflight debug log request size (bytes) |
| `Max inflight debug log response size (bytes)` | Max inflight debug log response size (bytes) |
| `0 means no extra limit` | 0 means no extra limit |
| `Inflight debug logs may contain sensitive data and increase Redis usage.` | ... |
| `Debug log menu requires recording to be enabled.` | ... |
| `Overview` | Overview |
| `Request` | Request |
| `Response` | Response |
| `Request Headers` | Request Headers |
| `Response Headers` | Response Headers |
| `Request Body` | Request Body |
| `Response Body` | Response Body |
| `Raw` | Raw |
| `Formatted JSON` | Formatted JSON |
| `Event List` | Event List |
| `Raw SSE` | Raw SSE |
| `Copy Headers` | Copy Headers |
| `Copy All` | Copy All |
| `Download` | Download |
| `Content truncated to first {{size}}` | Content truncated to first {{size}} |
| `Response may be incomplete` | Response may be incomplete |
| `Binary content ({{size}}, {{type}}). Download to inspect.` | ... |
| `Binary multipart ({{size}}). Download to inspect.` | ... |
| `Debug log is not available for in-progress requests` | ... |
| `Debug log not found` | ... |
| `Inflight debug log entries` | Inflight debug log entries |
| `Total inflight debug log size` | Total inflight debug log size |

---

## 11. 安全与隐私

1. **默认关闭**：两个开关均 `false`
2. **仅所有者**：API 校验 `user_id`
3. **Header 脱敏**：见 6.6
4. **Body 不脱敏**：设置页明确告知管理员
5. **无 admin 跨用户查看**：本期不做
6. **关闭记录后历史可读**（Q6A）：直到清理删除

---

## 12. 测试计划

### 12.1 后端单元测试（`service/inflight_trace_test.go`）

| 测试 | 断言 |
|------|------|
| `TestRedactTraceHeaders` | 敏感 header 变为 `***` |
| `TestTruncateTraceBody` | 超上限保留头部，`truncated=true` |
| `TestTraceBodyEncoding` | 非 UTF-8 → base64 |
| `TestInflightTaskTraceMenuVisible` | 记录关 → 菜单不可见 |
| `TestMaxBytesZeroMeansUnlimited` | `0` 不截断 |

### 12.2 后端集成测试

| 测试 | 断言 |
|------|------|
| `TestDeleteTerminalInflightTasksAlsoDeletesTrace` | 清理时 trace key 同步删除 |
| `TestGetInflightTaskTraceForbidden` | 非所有者 403 |
| `TestGetInflightTaskTraceInProgress` | 进行中 409 |

### 12.3 前端

- 菜单仅在 `trace_menu_visible && has_trace && terminal` 时显示
- `TraceMenuVisible` 在 `TraceEnabled=false` 时灰显
- JSON 格式化 / SSE 事件列表切换
- 二进制下载按钮可用

### 12.4 测试风格

遵循 `AGENTS.md`：使用 `testify/require` + `testify/assert`，不测实现细节。

---

## 13. 实现清单（按顺序）

### Phase 1: 后端基础

- [ ] `service/inflight_trace.go`：结构体、配置读取、脱敏、截断、编码
- [ ] `traceResponseWriter` + `inflightTraceCapture` context 持有
- [ ] `controller/relay.go`：接入 capture（排除 Realtime）
- [ ] terminal 时 `persistInflightTaskTrace` 异步写入
- [ ] `controller/inflight_task.go`：`GetUserInflightTaskTrace`
- [ ] `router/api-router.go`：注册路由
- [ ] `ListUserInflightTasks`：`has_trace` + `meta.trace_menu_visible`
- [ ] `DeleteTerminalInflightTasksBefore`：联动删除 trace key
- [ ] `GetInflightTaskStats`：trace 统计
- [ ] `model/option.go`：4 个新 Option
- [ ] `service/inflight_trace_test.go`

### Phase 2: 前端

- [ ] `log-settings-section.tsx`：4 个配置 UI
- [ ] `types.ts` / `operations/index.tsx` / `section-registry.tsx`：类型与默认值
- [ ] `inflight-task-trace-dialog.tsx`：弹窗组件
- [ ] `inflight-tasks-tab.tsx`：菜单、类型、`has_trace` 处理
- [ ] i18n 6 语言
- [ ] mock 数据

### Phase 3: 验证

- [ ] `go test ./service/... -run InflightTrace`
- [ ] `bun run typecheck`（`web/default/`）
- [ ] `bun run lint`（涉及文件）

---

## 14. 关键代码位置参考

| 用途 | 文件 | 符号 |
|------|------|------|
| Relay 入口 | `controller/relay.go` | `Relay()`, `relayHandler` 循环 |
| 在途状态更新 | `service/inflight_task.go` | `UpdateInflightTaskStatusAsync` |
| 在途列表 API | `controller/inflight_task.go` | `GetUserInflightTasks` |
| 在途清理 | `service/inflight_task.go` | `DeleteTerminalInflightTasksBefore` |
| 请求体读取 | `common/gin.go` | `GetBodyStorage` |
| Response tee 参考 | `middleware/audit.go` | `auditResponseWriter` |
| 前端在途 Tab | `web/default/src/features/usage-logs/components/inflight-tasks-tab.tsx` | `InflightTasksTab` |
| 详情弹窗参考 | 同上 | `InflightTaskDetailsDialog` |
| 日志维护设置 | `web/default/src/features/system-settings/maintenance/log-settings-section.tsx` | `LogSettingsSection` |
| Option 默认值 | `model/option.go` | `InitOptionMap` |
| 全局请求体上限 | `common/init.go` | `MaxRequestBodyMB` |

---

## 15. 边界情况速查

| 场景 | 行为 |
|------|------|
| Redis 未启用 | 与在途日志一致，功能不可用，不报错给用户 |
| 记录关 + 菜单开 | 后端强制 `trace_menu_visible=false` |
| 进行中任务 | 无菜单、API 409 |
| Realtime WebSocket | 不记录，可选标记 unsupported |
| 流式大响应 | tee 全量到内存 buffer，受 `MaxResponseBytes` 约束 |
| 图片生成返回 binary | base64 存储，前端仅下载 |
| multipart image edit | base64 存储，前端仅下载 |
| 关闭记录后的旧 trace | 仍可查看直到清理 |
| 列表性能 | `has_trace` 批量 EXISTS，不把 body 放入列表 |

---

## 16. 注意事项（给 Coding Agent）

1. **JSON 必须使用 `common.Marshal` / `common.Unmarshal`**，不要直接 `encoding/json`
2. **数据库无关**：本功能仅 Redis，无 DB migration
3. **不要修改**受保护的项目品牌信息（见 `AGENTS.md`）
4. **前端 i18n**：英文 key 为源字符串，使用 `i18n-translate` skill
5. **懒实现**：trace 逻辑集中在 `inflight_trace.go`，relay 仅 2-3 个调用点
6. **`inflight-tasks-tab.tsx` 已很大**：trace dialog 必须拆文件
7. **提交**：用户未要求前不要 git commit

---

## 附录 A：确认项汇总（给用户核对）

- [x] 客户端 ↔ 网关 only
- [x] 尽量完整记录
- [x] 记录关 → 菜单隐藏
- [x] 仅所有者
- [x] 体积管理员可配，0=不限
- [x] 仅 terminal 后可看
- [x] 全部记录 + 状态 + headers
- [x] 设置入口：系统设置 → 运维 → 日志维护
- [x] Q1-Q6 按推荐默认
- [x] 菜单文案：调试日志
- [x] 详情弹窗无跳转
