# 文件存储抽象层设计

> 适用范围：`new-api` 作为控制面，浏览器直传各存储渠道；渠道细节见 [`providers/`](./providers/)；表结构见 [`db-design/`](./db-design/)。
> **字段命名与码值以 `db-design/*.sql` 为准。**  
> 对齐目标：可配置分片阈值、渠道可插拔、秒传、防滥用签发、用户/管理员文件查询。

本文档是 **Storage Facade** 设计说明，不绑定某一厂商 SDK。下游业务（如 Image Playground）只声明 `channel_type` 码值（及可选 `file_channel_id`），不感知分片、预签名、图床分块等内部机制。

---

## 0. 目标与非目标

### 0.1 目标

| # | 目标 | 设计落点 |
|---|------|----------|
| 1 | 各渠道可配置「超过多大自动分片」；**浏览器端**完成分片与上传，**文件字节不经 new-api 中转** | Prepare 返回策略 + 短期凭证；前端 Storage SDK 执行 |
| 2 | 接入方只指定渠道类型即可切换 | Facade API + Provider Adapter |
| 3 | **跨用户秒传**：同 MD5 + 同渠道类型，任意用户可复用已有物理对象 | `file.identifier`+`channel_type` 去重 + `ref_count`；新用户只建 `user_file` |
| 4 | 防上传链接被盗用/刷资源 | 短时票据 + 一次性 nonce + 绑定约束 + 限流（见 §5） |
| 5 | 贴合 new-api：鉴权、表结构、响应信封 | `UserAuth` / `AdminAuth`、GORM 模型、`{success,message,data}` |
| 6 | **默认渠道、ImgBed BaseUrl/Token 等均由管理员配置** | `file_upload_channel` + 管理端 CRUD；密钥仅服务端持有 |

### 0.2 非目标（本期不做）

- 服务端代传大文件（违背「浏览器承担上传压力」）
- 跨渠道自动迁移 / 统一物理去重到单一后端
- 完整 POSIX 目录、对象级 ACL 编辑器
- 用 SHA-256 替换 MD5 作为主指纹（可预留字段，默认仍用 MD5 以降低浏览器算力；安全场景可后续升级）

---

## 1. 总体架构

```text
┌─────────────┐   prepare / complete / list   ┌──────────────────────┐
│  Browser    │ ─────────────────────────────► │  new-api             │
│  Storage    │ ◄── ticket + upload plan ──── │  Storage Facade      │
│  SDK        │                               │  (controller/service)│
└──────┬──────┘                               └──────────┬───────────┘
       │ 直传（PUT / 分片 / ImgBed chunk）               │
       ▼                                                 ▼
┌─────────────┐                               ┌──────────────────────┐
│ S3 / R2 /   │                               │ DB: file_upload_channel / file / user_file        │
│ ImgBed ...  │◄── 仅服务端持有长效密钥 ───────│ Redis: ticket nonce  │
└─────────────┘                               └──────────────────────┘
```

**原则**：

1. **长效密钥只留在服务端**（S3 AK/SK、ImgBed API Token 等），永不下发浏览器。  
2. 浏览器只拿到 **短时、绑定用途的上传计划**（Presigned URL / 分片凭证 / ImgBed 会话参数）。  
3. 业务代码依赖 Facade，不 `import` 具体 Provider。

### 1.1 分层

| 层 | 职责 | 建议落点（new-api） |
|----|------|---------------------|
| HTTP Facade | 鉴权、参数校验、统一响应 | `controller/storage.go` + `router/api-router.go` |
| Domain Service | 秒传判定、票据签发/核销、元数据写入 | `service/storage/` |
| Provider Adapter | 按渠道生成上传计划、校验完成、删除物理对象 | `service/storage/provider/{local,webdav,imgbed,s3}.go` |
| Frontend SDK | 算 MD5、按 plan 单文件或分片直传、调 complete | `gpt-image-playground` 或 `web/.../lib/storage` |
| 渠道配置 | 阈值、endpoint、密钥、公开域名 | 表 `file_upload_channel`（或 Option JSON，见 §3） |

### 1.2 渠道类型枚举（码值以 SQL 为准）

| `type` / `channel_type` | 说明 | Provider 文档 |
|-------------------------|------|----------------|
| `0` | 本地存储 | （待补） |
| `1` | Webdav | [WebDAV-Provider.md](./providers/WebDAV-Provider.md) |
| `2` | Cloudflare-ImageBed | [Cloudflare-ImageBed-Provider.md](./providers/Cloudflare-ImageBed-Provider.md) |
| `3` | S3存储 | [S3-Provider.md](./providers/S3-Provider.md) |

新增渠道：分配新码值 + Adapter，**不改** Facade 路径与业务调用方式。

---

## 2. 核心流程

### 2.1 上传（含秒传、自动分片）

```text
1. 浏览器本地计算 identifier(MD5)、file_size、mime、file_name/file_suffix
2. POST /api/storage/prepare  { channel_type, identifier, file_size, ... }  // channel_type 为码值 "0"/"1"/"2"/"3"
3. 服务端：
   a. 鉴权 + 限流
   b. 解析渠道：指定 file_channel_id → 用该配置；否则按 type 码值 / 全局 is_default=1
   c. 查跨用户秒传：同 identifier + 同 channel_type 且 file.status=1
      → 为当前用户写 user_file（若尚无）、file.ref_count++ → 返回 hit=true（不签发上传 URL）
   d. 未命中：按该渠道 chunk_threshold 比较 file_size
      - size <= threshold → strategy=single，签发单次上传凭证
      - size >  threshold → strategy=multipart，签发分片计划
   e. 写入 pending 票据（Redis）；file.status=0 可选
4. 浏览器按 plan 直传对象存储（不经 new-api；长效密钥不下发）
5. POST /api/storage/complete  { ticket_id, ...provider_proof }
6. 服务端核销票据、Head/校验、写入/更新 file + user_file（status=1）
```

业务侧伪代码（无感知分片 / 默认渠道解析）：

```ts
// 不传 fileChannelId：走管理员配置的默认渠道；type 使用 SQL 码值
const file = await storageClient.upload(rawFile, { channelType: '3' }) // S3
// ImgBed base_url / api_token 在 file_upload_channel.config_proflle，前端不可见
const file2 = await storageClient.upload(rawFile, { channelType: '2' })
```

SDK 内部：`prepare` → 若 `hit` 直接返回 → 否则按 `strategy` 直传 → `complete`。

### 2.2 分片阈值（按渠道可配）

每个 `file_upload_channel` 行（或全局默认）包含：

| 字段 | 含义 | 建议默认 |
|------|------|----------|
| `chunk_threshold` | 超过则走分片（字节） | S3: `16777216`（16MiB）；ImgBed: `15728640`（约 15MiB，贴近 TG/CF 限制） |
| `chunk_size` | 分片大小（字节） | S3: `8388608`（8MiB，且 ≥5MiB）；ImgBed 按渠道文档（如 TG 16MiB） |
| `max_size` | 单文件上限 | 按业务与渠道能力 |

Prepare 响应中的 `strategy` **只由服务端根据配置计算**，前端不得自行决定是否分片（可做本地预检提示，但以服务端为准）。

### 2.3 删除

1. 用户/管理员调删除 API。  
2. 删除（或标记不可用）对应 `user_file` 行。  
3. `file.ref_count--`；当 `ref_count==0` 时将 `file.status` 置 `2` 并异步调 Provider `Delete`（按 `object_key`），避免误删仍被引用的物理对象。

### 2.4 展示 / 列表

- 用户：只查 `user_id = 当前用户`。  
- 管理员：可按 `user_id` / `identifier` / `channel_type`（码值）过滤全站。  
- 返回字段含可访问 `url`（公开域名或临时签名，由 Provider `GetURL` 决定）。

---

## 3. 数据模型（以 `db-design/*.sql` 为准）

> DDL 源文件：[file_upload_channel.sql](./db-design/file_upload_channel.sql) · [file.sql](./db-design/file.sql) · [user_file.sql](./db-design/user_file.sql)  
> GORM 映射时：JSON 用 **snake_case** 与列名一致；时间列按 SQL 为 `timestamp`（`create_time` / `update_time`）。

### 3.0 渠道类型码值（`type` / `channel_type`）

| 码值 | 含义 | Provider |
|------|------|----------|
| `0` | 本地存储 | （待补 Local Provider 文档） |
| `1` | Webdav | [WebDAV-Provider.md](./providers/WebDAV-Provider.md) |
| `2` | Cloudflare-ImageBed | [Cloudflare-ImageBed-Provider.md](./providers/Cloudflare-ImageBed-Provider.md) |
| `3` | S3存储 | [S3-Provider.md](./providers/S3-Provider.md) |

API / SDK 传渠道类型时使用上述**字符串码值**（`"0"`/`"1"`/`"2"`/`"3"`），与表字段 `varchar(1)` 一致。

### 3.1 `file_upload_channel` — 渠道配置（管理员可配）

**已拍板**：默认渠道、分片阈值、以及各 Provider 密钥（含 ImgBed `base_url` / `api_token`）由管理员配置；写入本表；**`config_proflle` 内密钥管理端读出脱敏，不下发浏览器**。

| 列 | 说明 |
|----|------|
| `id` | 主键 |
| `name` | 渠道名称 |
| `type` | 见 §3.0 |
| `status` | `0` 停用 / `1` 启用 |
| `is_default` | `0` 否 / `1` 是（全局同时仅一条为 `1`） |
| `chunk_threshold` / `chunk_size` / `max_size` | 分片阈值与上限（字节）；`max_size=0` 表示不在此限制 |
| `config_proflle` | JSON 配置（字段名以 SQL 为准，注意拼写） |
| `create_user_id` / `update_user_id` | 审计 |
| `create_time` / `update_time` | timestamp |

#### 默认渠道规则

| 规则 | 说明 |
|------|------|
| 设置方式 | 管理员将某条 `is_default` 置为 `1` |
| 互斥 | 全局同时仅一条 `is_default=1`（事务内取消旧默认） |
| 解析顺序 | 请求带 `file_channel_id` → 用该渠道；否则带 `type`/`channel_type` → 该 type 下默认且 `status=1`；再否则全局 `is_default=1`；仍无 → 错误「未配置默认存储渠道」 |
| 用户侧 | 可不传渠道 Id，只传 type 码值或完全依赖全局默认 |

#### `config_proflle` JSON（按 `type`）

**`type=3` S3**

```json
{
  "endpoint": "https://xxx.r2.cloudflarestorage.com",
  "region": "auto",
  "bucket": "playground",
  "access_key_id": "...",
  "secret_access_key": "...",
  "force_path_style": true,
  "public_base_url": "https://cdn.example.com",
  "key_prefix": "files/"
}
```

**`type=2` Cloudflare-ImageBed**

```json
{
  "base_url": "https://your.imgbed.domain",
  "api_token": "...",
  "upload_channel": "cfr2",
  "upload_folder": "playground",
  "return_format": "full"
}
```

**`type=1` Webdav**

```json
{
  "base_url": "https://dav.example.com/remote.php/dav/files/alice/",
  "username": "alice",
  "password": "...",
  "auth_type": "basic",
  "path_prefix": "playground/",
  "public_base_url": "",
  "timeout_ms": 600000
}
```

**`type=0` 本地存储**

```json
{
  "root_path": "/data/files",
  "public_base_url": ""
}
```

### 3.2 `file` — 物理对象（跨用户秒传）

按 **`identifier`（MD5）+ `channel_type`** 去重（`uk_identifier_channel_type`）。跨用户共享：`ref_count` 计量 `user_file` 引用。

| 列 | 说明 |
|----|------|
| `id` | 主键 |
| `file_channel_id` | 实际上传使用的 `file_upload_channel.id` |
| `channel_type` | 与渠道 `type` 码值一致 |
| `file_size` | 字节 |
| `object_key` | Provider 稳定路径（S3 key / ImgBed path / WebDAV path / 本地相对路径） |
| `file_url` | 访问 URL，可空（由 `public_base_url` + `object_key` 运行时拼接） |
| `identifier` | MD5 |
| `mime_type` / `etag` | 类型与校验 |
| `ref_count` | 引用计数；归零后可删物理对象 |
| `status` | `0` 上传中 / `1` 可用 / `2` 待删除 |
| `creater_user_id` | 首个上传者（秒传引用不改） |
| `create_time` / `update_time` | timestamp |

秒传命中：`identifier` + 目标 `channel_type` 且 `status='1'`。

### 3.3 `user_file` — 用户逻辑文件

| 列 | 说明 |
|----|------|
| `id` | 主键 |
| `file_id` | `file.id` |
| `file_channel_id` | `file_upload_channel.id` |
| `file_name` | 不含后缀的文件名 |
| `file_suffix` | 后缀（不含点，如 `png`） |
| `user_id` | 用户 Id |
| `source` | 来源，如 `playground` / `manual` |
| `status` | `0` 上传中 / `1` 可用 |
| `create_time` / `update_time` | timestamp |

约束：`uk_user_file (user_id, file_id)` —— 同用户秒传已挂载则直接返回该行。

### 3.4 票据（建议 Redis，不落库）

| Key | 示例 | TTL |
|-----|------|-----|
| `storage:ticket:{ticket_id}` | JSON：user_id, file_channel_id, identifier, file_size, mime_type, object_key, strategy, parts, nonce | 与签名有效期一致（如 10–15 分钟） |
| `storage:nonce:{nonce}` | `1` | 同 TTL；complete 时 DEL |

无 Redis 时可用内存缓存或临时表；优先 Redis。

---

## 4. Provider 统一接口（服务端）

```go
type UploadStrategy string // "single" | "multipart"

type PrepareInput struct {
	UserId    int
	Channel   *StorageChannel
	Filename  string
	MimeType  string
	Size      int64
	Md5       string
	ObjectKey string // 服务端生成
}

type UploadPlan struct {
	Strategy   UploadStrategy    `json:"strategy"`
	// single
	UploadURL  string            `json:"upload_url,omitempty"`
	Headers    map[string]string `json:"headers,omitempty"`
	Method     string            `json:"method,omitempty"` // PUT / POST
	// multipart
	UploadId   string            `json:"upload_id,omitempty"`
	PartSize   int64             `json:"part_size,omitempty"`
	Parts      []PartCredential  `json:"parts,omitempty"` // 可首包只返回规则，分片 URL 按需二次签发
	// 完成时回传所需字段提示
	CompleteHints map[string]string `json:"complete_hints,omitempty"`
}

type PartCredential struct {
	PartNumber int               `json:"part_number"`
	UploadURL  string            `json:"upload_url"`
	Headers    map[string]string `json:"headers,omitempty"`
	ExpiresAt  int64             `json:"expires_at"`
}

type CompleteInput struct {
	Channel   *StorageChannel
	ObjectKey string
	Md5       string
	Size      int64
	// Provider 相关证明：ETag、ImgBed src、UploadId+Parts 等
	Proof     map[string]any
}

type Provider interface {
	Type() string
	PrepareUpload(ctx context.Context, in PrepareInput) (*UploadPlan, error)
	CompleteUpload(ctx context.Context, in CompleteInput) (url string, etag string, err error)
	Delete(ctx context.Context, objectKey string) error
	GetURL(ctx context.Context, objectKey string, publicURL string) (string, error)
}
```

**分片凭证策略（防滥用 + 可控）**：

- **S3**：可为每个 part 单独 Presign（绑定 `partNumber` + `uploadId` + 过期）；或返回 `CreateMultipartUpload` 的 uploadId 后，提供 `POST /api/storage/presign-part`（需 ticket）按需签发，避免一次下发上百分片 URL。  
- **ImgBed**：服务端用 API Token 初始化 chunked 会话，只把 `uploadId` 与分块上传所需 **短期** 参数给浏览器；**不得**下发长期 API Token。若 ImgBed 无法做到无 Token 直传，则分块请求须带服务端签发的 **HMAC 代理票**（见 §5.3），或由极薄的 new-api 代理头转发（仍不接文件 body 到业务逻辑——仅透传流时需谨慎，默认仍坚持浏览器直连 ImgBed）。

> 默认立场：浏览器直连存储方；若某渠道协议强制长期 Token，则该渠道的「直传」改为「浏览器 → 存储方」，Token 仍只出现在服务端代签的短期票据校验路径中（ImgBed 若只能 Bearer Token，优先评估服务端签一次性 upload session，而不是把 Token 给前端）。

---

## 5. 防盗链 / 防滥用（简易有效）

上传链接一旦泄露可被刷流量或占存储，需多层约束。推荐组合（实现成本低、对当前项目够用）：

### 5.1 短时上传票据（必须）

`prepare` 成功后返回：

```json
{
  "ticket_id": "st_...",
  "expire_at": 1710000000,
  "upload_token": "<HMAC>",
  "plan": { }
}
```

HMAC 载荷建议包含：`ticket_id | user_id | file_channel_id | identifier | file_size | object_key | expire_at | nonce`。  
密钥用服务端 `SESSION_SECRET` 或独立 `STORAGE_TICKET_SECRET`。

`complete` 必须校验：签名正确、未过期、`user_id` 与登录用户一致、nonce 未使用。

### 5.2 绑定内容与大小（必须）

- Presigned / 计划中绑定 **Content-Type、Content-Length（或最大长度）**、目标 key。  
- `complete` 时 `HeadObject`（或 ImgBed 元数据）核对 **file_size**；能取 ETag/MD5 则与票据 `identifier` 比对（S3 单次 Put 的 ETag 常为 MD5；Multipart 的 ETag 不是 MD5，以服务端记录的客户端 identifier + file_size 为主，必要时抽查）。

攻击者拿到 URL 也无法上传任意大文件或改路径。

### 5.3 一次性 nonce（必须）

票据内 `nonce` 存 Redis；`complete` 成功或明确失败策略下删除。防止重放 complete 刷库。

### 5.4 签发限流（必须）

对 `prepare` 按 `user_id`（及 IP）限流，例如：

- 每分钟 N 次 prepare  
- 每小时累计申请上传字节上限  

对齐 new-api 现有限流中间件风格即可。

### 5.5 展示链防盗（可选，与上传分离）

| 模式 | 做法 |
|------|------|
| 公开读 + CDN | `public_base_url`；CDN 侧可配 Referer/Token（若有） |
| 私有读 | 列表/详情返回 **短时 Presigned GET**，不落长期公网直链 |

上传防滥用 ≠ 展示防盗链；展示按桶策略另配。

### 5.6 不做什么（避免过度设计）

- 不在浏览器存 AK/SK 或 ImgBed 长期 Token  
- 不签发超过 15–30 分钟的上传 URL  
- 不在无 file_size/identifier 绑定的情况下发「万能 PUT」

---

## 6. HTTP API（对齐 new-api 鉴权与信封）

统一响应：

```json
{ "success": true, "message": "", "data": {} }
```

鉴权：

- 用户接口：`middleware.UserAuth()`（session 或 AccessToken + `New-Api-User`）
- 管理接口：`middleware.AdminAuth()`
- 用户 id：`c.GetInt("id")`；**禁止**信任 body 里的 user_id

分页：`common.GetPageQuery` → `page` / `page_size` / `total` / `items`。

### 6.1 用户侧

| 方法 | 路径 | 说明 |
|------|------|------|
| `POST` | `/api/storage/prepare` | 秒传检测 + 签发上传计划 |
| `POST` | `/api/storage/presign-part` | （可选）持 ticket 为某一 part 续签 |
| `POST` | `/api/storage/complete` | 确认直传完成并入库 |
| `GET` | `/api/storage/self` | 当前用户文件列表 |
| `GET` | `/api/storage/self/:id` | 当前用户单文件（含可用 url） |
| `DELETE` | `/api/storage/self/:id` | 删除自己的逻辑文件 |

#### `POST /api/storage/prepare`

请求：

```json
{
  "channel_type": "3",
  "file_channel_id": 0,
  "file_name": "out",
  "file_suffix": "png",
  "mime_type": "image/png",
  "file_size": 1048576,
  "identifier": "d41d8cd98f00b204e9800998ecf8427e",
  "source": "playground"
}
```

- `file_channel_id` 省略或 `0`：按 §3.1 默认渠道规则解析（`is_default=1`）。  
- 下游只指定类型时传 `channel_type` 码值；亦可都不传，走全局默认。

响应（秒传命中）：

```json
{
  "success": true,
  "data": {
    "hit": true,
    "file": {
      "id": 12,
      "file_name": "out",
      "file_suffix": "png",
      "url": "https://cdn.example.com/files/...",
      "identifier": "...",
      "file_size": 1048576,
      "channel_type": "3"
    }
  }
}
```

响应（需要上传）：

```json
{
  "success": true,
  "data": {
    "hit": false,
    "ticket_id": "st_xxx",
    "upload_token": "hmac...",
    "expire_at": 1710000900,
    "chunk_threshold": 16777216,
    "plan": {
      "strategy": "multipart",
      "part_size": 8388608,
      "upload_id": "...",
      "parts": []
    }
  }
}
```

#### `POST /api/storage/complete`

```json
{
  "ticket_id": "st_xxx",
  "upload_token": "hmac...",
  "proof": {
    "etag": "...",
    "parts": [{ "part_number": 1, "etag": "..." }],
    "src": "/file/..."
  }
}
```

响应：`data.file` 同列表项。

#### `GET /api/storage/self`

Query：`p`、`page_size`、`channel_type`、`keyword`（匹配 file_name）、`file_suffix`。

实现要点：`WHERE user_id = ? AND deleted_at IS NULL`，对齐 `GetUserLogs` 模式。

### 6.2 管理侧

| 方法 | 路径 | 说明 |
|------|------|------|
| `GET` | `/api/storage/` | 全站文件；Query 可含 `user_id` |
| `GET` | `/api/storage/user/:user_id` | 指定用户文件 |
| `DELETE` | `/api/storage/:id` | 管理员删除逻辑文件 |
| `GET` | `/api/storage/channel` | 渠道列表（`api_token` / `secret_access_key` 脱敏） |
| `POST` | `/api/storage/channel` | 创建渠道（含 ImgBed `base_url`+`api_token`、S3 密钥、分片阈值等） |
| `PUT` | `/api/storage/channel/:id` | 更新渠道配置 |
| `PUT` | `/api/storage/channel/:id/default` | 设为全局默认渠道（取消原默认） |
| `DELETE` | `/api/storage/channel/:id` | 删除/停用渠道（有引用时建议仅停用） |

创建/更新渠道 Body 示例（`type=2` Cloudflare-ImageBed）：

```json
{
  "name": "生产图床",
  "type": "2",
  "status": "1",
  "is_default": "1",
  "chunk_threshold": 15728640,
  "chunk_size": 16777216,
  "config_proflle": {
    "base_url": "https://your.imgbed.domain",
    "api_token": "imgbed_xxx",
    "upload_channel": "cfr2",
    "upload_folder": "playground",
    "return_format": "full"
  }
}
```

路由注册风格（示意）：

```go
storageRoute := apiRouter.Group("/storage")
{
    storageRoute.POST("/prepare", middleware.UserAuth(), controller.StoragePrepare)
    storageRoute.POST("/complete", middleware.UserAuth(), controller.StorageComplete)
    storageRoute.GET("/self", middleware.UserAuth(), controller.GetSelfStorageFiles)
    storageRoute.GET("/self/:id", middleware.UserAuth(), controller.GetSelfStorageFile)
    storageRoute.DELETE("/self/:id", middleware.UserAuth(), controller.DeleteSelfStorageFile)

    storageRoute.GET("/", middleware.AdminAuth(), controller.GetAllStorageFiles)
    storageRoute.GET("/user/:user_id", middleware.AdminAuth(), controller.GetUserStorageFiles)
    storageRoute.DELETE("/:id", middleware.AdminAuth(), controller.AdminDeleteStorageFile)

    ch := storageRoute.Group("/channel", middleware.AdminAuth())
    {
        ch.GET("/", controller.GetStorageChannels)
        ch.POST("/", controller.CreateStorageChannel)
        ch.PUT("/:id", controller.UpdateStorageChannel)
        ch.PUT("/:id/default", controller.SetDefaultStorageChannel)
        ch.DELETE("/:id", controller.DeleteStorageChannel)
    }
}
```

---

## 7. 前端 SDK 职责（浏览器承担压力）

```ts
type UploadOptions = {
  channelType?: '0' | '1' | '2' | '3'  // 省略则走全局默认渠道
  fileChannelId?: number
  source?: string
}

async function upload(file: File, opt: UploadOptions): Promise<UserFile> {
  const identifier = await md5File(file)
  const name = file.name
  const dot = name.lastIndexOf('.')
  const file_name = dot > 0 ? name.slice(0, dot) : name
  const file_suffix = dot > 0 ? name.slice(dot + 1) : ''
  const prepared = await api.prepare({
    channel_type: opt.channelType,
    file_channel_id: opt.fileChannelId ?? 0,
    file_name,
    file_suffix,
    mime_type: file.type,
    file_size: file.size,
    identifier,
    source: opt.source,
  })
  if (prepared.hit) return prepared.file

  if (prepared.plan.strategy === 'single') {
    await putDirect(prepared.plan, file)
  } else {
    await putMultipart(prepared.plan, file, prepared) // 内部分片；可按需 presign-part
  }
  return api.complete({ ticket_id, upload_token, proof })
}
```

业务方只传 `channelType` 码值（或省略走默认）；阈值与是否分片由 Prepare 结果决定。

---

## 8. 秒传规则细则（跨用户 · 已拍板）

**范围**：同一 `channel_type`（码值）下，按 `identifier`（MD5）**全局（跨用户）** 复用 `file`；不同用户通过各自的 `user_file` 引用同一 `file.id`。

| 场景 | 行为 |
|------|------|
| 同用户、同 identifier、同 channel_type，已有 `user_file` | 直接返回已有记录（可更新 `file_name`/`file_suffix`） |
| **其他用户**已上传同 identifier、同 channel_type，当前用户无记录 | 新建 `user_file`，`ref_count++`，**不上传、不签发 URL** |
| 同 identifier、**不同** channel_type | 视为不同对象，需在新渠道上传 |
| prepare 命中后 | 不签发上传 URL，降低滥用面 |
| 用户删除自己的 `user_file` | `ref_count--`；仅当 `ref_count==0` 才删物理对象 |
| 客户端谎报 identifier | complete / Head 校验失败则拒收；可惩罚限流 |

> MD5 碰撞在恶意场景存在理论风险；本期接受该成本。若后续要加强，可增加 `sha256` 字段，prepare 双指纹。

---

## 9. 与渠道文档的分工

| 文档 | 内容 |
|------|------|
| **本文** | Facade、表结构、票据防滥用、秒传、用户/管理 API、SDK 行为 |
| [S3-Provider.md](./providers/S3-Provider.md) | type=`3`：PutObject / Multipart / Presign 等 |
| [Cloudflare-ImageBed-Provider.md](./providers/Cloudflare-ImageBed-Provider.md) | type=`2`：上传/分块/删除/列表 |
| [WebDAV-Provider.md](./providers/WebDAV-Provider.md) | type=`1`：PUT / DELETE / PROPFIND 等 |

Adapter 实现时对照对应 Provider 文档填 `UploadPlan` / `Complete` / `Delete`。

---

## 10. 落地顺序（建议）

1. 落库 DDL：`db-design/file_upload_channel.sql` / `file.sql` / `user_file.sql` + GORM AutoMigrate  
2. Facade：`prepare`（秒传 + type=3 S3 single Presign）+ `complete` + `self` 列表/删除  
3. 防滥用：HMAC 票据 + Redis nonce + prepare 限流  
4. S3 multipart + `presign-part`  
5. type=2 ImgBed / type=1 WebDAV Adapter  
6. 管理端渠道 CRUD（含 `config_proflle`、默认渠道）与全站文件查询  
7. 前端 Storage SDK 接入 Playground  

---

## 11. 验收清单

- [ ] 仅改 `channel_type` 码值（如 `"3"`↔`"2"`）即可切换 Provider，业务无改分片代码  
- [ ] 超过该渠道 `chunk_threshold` 时自动 multipart；小于则 single  
- [ ] 文件字节不经过 new-api 请求体（prepare/complete 仅 JSON）  
- [ ] **跨用户秒传**：同 `identifier` + 同 `channel_type` → 只加 `user_file` 引用  
- [ ] 管理员可配置默认渠道（`is_default=1`）；未指定 `file_channel_id` 时走默认  
- [ ] 管理员经 `config_proflle` 配置 ImgBed `base_url`+`api_token` / S3 / WebDAV 密钥；不下发浏览器  
- [ ] 泄露的 upload URL 过期后失效；改 size/type 无法借用；complete 不可重放  
- [ ] `GET /api/storage/self` 仅本人；管理员可查指定 `user_id`  
- [ ] 字段名与码值与 `db-design/*.sql` 一致  

---

## 12. 已拍板结论

| 议题 | 结论 |
|------|------|
| 秒传范围 | **跨用户**：同 `identifier` + 同 `channel_type` 共享 `file`，按 `ref_count` 管理物理删除 |
| 默认渠道 | **管理员配置**：`file_upload_channel.is_default='1'`；全局同时仅一条；用户可不传 `file_channel_id` |
| 渠道凭证 | **管理员配置**写入 `config_proflle`（ImgBed `base_url`/`api_token`、S3 密钥、WebDAV 账号等）；仅服务端使用，读出脱敏 |
| 类型码值 | `0` 本地 / `1` Webdav / `2` Cloudflare-ImageBed / `3` S3（以 SQL 为准） |

可选后续增强（非阻塞）：按 `source` 映射多套默认渠道；ImgBed 若协议限制导致无法纯浏览器直传时再评估薄代理。
