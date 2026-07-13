# 文件操作 · 用户侧详设（user）

> 定位：登录用户的文件 **增（含分片直传）/ 删 / 查**。  
> 配套：[admin.md](./admin.md) · 表 [file.sql](../../db-design/file.sql) / [user_file.sql](../../db-design/user_file.sql) · 渠道 [file_upload_channel.sql](../../db-design/file_upload_channel.sql) · 总览 [FIleStorage.md](../../FIleStorage.md) · 渠道契约 [../file_upload_channel_design/use.md](../file_upload_channel_design/use.md)  
> 本目录两份文档为 **File Operate 详设**（API only，无管理/用户前端页）。

---

## 0. 已拍板结论（本域）

| # | 结论 |
|---|------|
| 鉴权 | `UserAuth`；用户 id 仅 `c.GetInt("id")` |
| 操作对象 | 逻辑文件 `user_file`（列表/详情 join `file` 取 url/size/identifier 等） |
| 增 | 浏览器直传：`prepare` →（分片则 `presign-part`）→ 直传存储方 → `complete`；含跨用户秒传 |
| 删 | 仅删自己的 `user_file`；`file.ref_count--`，归零后删物理对象 |
| 查 | 全部 / 按 `file_suffix`；默认**不过滤 status**（前端自滤）；动态分页 |
| 字段/码值 | 以 `db-design` SQL 为准 |

---

## 1. 目标与非目标

### 1.1 目标

1. 用户可上传文件（自动 single / multipart，按渠道 `chunk_threshold`）。  
2. 用户可分页查询自己的文件（全量或按后缀）。  
3. 用户可删除自己的逻辑文件，并正确维护秒传引用与物理删除。  
4. 接口自洽，可与渠道管理、管理员文件接口并存。

### 1.2 非目标

- 管理端 UI / 用户文件柜 UI（本期 API only）  
- 管理员查删（见 [admin.md](./admin.md)）  
- `type=0` 本地存储上传（渠道侧未开放则 prepare 拒绝）

---

## 2. 领域模型与数据流

```text
user_file (逻辑)  N ──► 1  file (物理, identifier+channel_type 秒传)
                              │
                              ▼
                     file_upload_channel (配置/密钥仅服务端)
```

| 表 | 用户侧职责 |
|----|------------|
| `user_file` | 归属、展示名（`file_name`+`file_suffix`）、`source`、列表筛选 |
| `file` | 内容实体、`object_key`/`file_url`/`ref_count`/`status` |
| `file_upload_channel` | 解析默认渠道、阈值、Provider 配置（不下发密钥） |

### 2.1 状态

| 表 | status | 含义 |
|----|--------|------|
| `user_file` | `0` | 上传中（prepare 未 complete） |
| `user_file` | `1` | 可用 |
| `file` | `0` | 上传中 |
| `file` | `1` | 可用 |
| `file` | `2` | 待删除（引用归零清理中/失败重试） |

列表默认返回所有 status；可用 query `status` 过滤。

### 2.2 渠道解析（上传）

与渠道 use.md 一致，**仅认启用且默认**（避免静默挑渠）：

1. `file_channel_id` > 0 → 该渠（须 `status=1`）  
2. 否则 `channel_type` ∈ {`1`,`2`,`3`} → 该 type 下 `is_default=1` 且启用  
3. 否则 → 全局 `is_default=1` 且启用  
4. 否则 → 错误 `No default file upload channel`  
5. `type=0` → `Local storage channel is not supported`

### 2.3 分片决策

```text
if max_size > 0 && file_size > max_size → reject
if file_size > chunk_threshold → strategy = multipart
else → strategy = single
```

| channel type | multipart 实现要点 |
|--------------|-------------------|
| `3` S3 | CreateMultipartUpload + 每 part Presign PUT + Complete；`presign-part` 按需续签 |
| `2` ImgBed | 官方分块三步或 HF 直传（见 Provider）；plan 携带 uploadId / 分块参数，**不下发长期 Token** |
| `1` WebDAV | **无标准 multipart**：超阈值且无法安全直传时 prepare 失败或降级为短时中继分块（若实现中继）；否则拒绝超 `max_size` |

`chunk_size` 取渠道表列；S3 须 ≥ 5MiB（最后一块除外）。

---

## 3. 通用约定

### 3.1 响应信封

```json
{ "success": true, "message": "", "data": {} }
```

失败：`success=false`，`message` 为可 i18n 的英文句（与项目习惯一致）。

### 3.2 分页（动态）

对齐 `common.GetPageQuery`：

| Query | 说明 | 默认 / 上限 |
|-------|------|-------------|
| `p` | 页码，从 1 | 1 |
| `page_size` / `ps` / `size` | 每页条数 | 默认 `ItemsPerPage`(10)，上限 100 |

`data` 形态：

```json
{
  "page": 1,
  "page_size": 20,
  "total": 100,
  "items": [ /* UserFileView */ ]
}
```

### 3.3 列表项 `UserFileView`

```json
{
  "id": 99,
  "file_id": 12,
  "file_channel_id": 1,
  "channel_type": "3",
  "file_name": "out",
  "file_suffix": "png",
  "user_id": 7,
  "source": "playground",
  "status": "1",
  "file_size": 1048576,
  "mime_type": "image/png",
  "identifier": "d41d8cd98f00b204e9800998ecf8427e",
  "file_url": "https://cdn.example.com/files/...",
  "object_key": "files/...",
  "create_time": "2026-07-12T10:00:00Z",
  "update_time": "2026-07-12T10:00:00Z"
}
```

- `file_url`：优先 `file.file_url`；空则用渠道 `public_base_url`+`object_key`；私有读可返回短时签名 URL（实现可选，字段名仍 `file_url`）。  
- **禁止**返回渠道 `config_proflle`。

### 3.4 建议索引（实现时）

`user_file` 增加（若尚未有）：

```sql
KEY `idx_user_suffix` (`user_id`, `file_suffix`),
KEY `idx_user_status` (`user_id`, `status`)
```

---

## 4. 接口文档 · 增（上传）

前缀：`/api/storage`  
鉴权：`UserAuth`

### 4.1 `POST /api/storage/prepare`

秒传检测 + 签发上传计划（含分片计划）。

#### Request

```http
POST /api/storage/prepare
Authorization: ...
New-Api-User: {user_id}
Content-Type: application/json
```

```json
{
  "channel_type": "3",
  "file_channel_id": 0,
  "file_name": "out",
  "file_suffix": "png",
  "mime_type": "image/png",
  "file_size": 52428800,
  "identifier": "d41d8cd98f00b204e9800998ecf8427e",
  "source": "playground"
}
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `channel_type` | string | 否 | `"1"`\|`"2"`\|`"3"` |
| `file_channel_id` | int64 | 否 | `0`=未指定 |
| `file_name` | string | 是 | 不含后缀 |
| `file_suffix` | string | 是 | 不含点，小写归一化建议服务端做 |
| `mime_type` | string | 是 | |
| `file_size` | int64 | 是 | 字节，>0 |
| `identifier` | string | 是 | 文件 MD5（hex） |
| `source` | string | 否 | 默认 `""` |

#### Response · 秒传命中 `hit=true`

不签发上传 URL；确保当前用户有 `user_file`（已有则直接返回），`ref_count` 仅在新建引用时 +1。

```json
{
  "success": true,
  "data": {
    "hit": true,
    "file": { /* UserFileView */ }
  }
}
```

#### Response · 需上传 `hit=false` · single

```json
{
  "success": true,
  "data": {
    "hit": false,
    "ticket_id": "st_xxx",
    "upload_token": "hmac...",
    "expire_at": 1710000900,
    "file_channel_id": 1,
    "channel_type": "3",
    "chunk_threshold": 16777216,
    "chunk_size": 8388608,
    "object_key": "files/2026/07/uuid_out.png",
    "plan": {
      "strategy": "single",
      "method": "PUT",
      "upload_url": "https://presigned.example/...",
      "headers": {
        "Content-Type": "image/png"
      }
    }
  }
}
```

#### Response · 需上传 · multipart（S3 示例）

```json
{
  "success": true,
  "data": {
    "hit": false,
    "ticket_id": "st_xxx",
    "upload_token": "hmac...",
    "expire_at": 1710000900,
    "file_channel_id": 1,
    "channel_type": "3",
    "chunk_threshold": 16777216,
    "chunk_size": 8388608,
    "object_key": "files/2026/07/uuid_big.bin",
    "plan": {
      "strategy": "multipart",
      "upload_id": "s3-multipart-upload-id",
      "part_size": 8388608,
      "part_count": 7,
      "parts": [
        {
          "part_number": 1,
          "upload_url": "https://presigned-part-1...",
          "headers": {},
          "expires_at": 1710000900
        }
      ],
      "presign_mode": "lazy"
    }
  }
}
```

| `presign_mode` | 含义 |
|----------------|------|
| `eager` | `parts` 含全部分片 URL（分片少时） |
| `lazy` | 首包可只含元数据或首片；其余用 §4.2 续签 |

**ImgBed multipart** 时 `plan` 形状跟 Provider 分块协议对齐（含 `upload_id`、chunk 上传地址或会话字段），字段放在 `plan` 内扩展，前端按 `channel_type` 分支。

服务端副作用（建议）：

- Redis 票据：绑定 user_id、channel、identifier、size、object_key、strategy、upload_id、nonce、expire  
- 可选：预写 `file.status=0`、`user_file.status=0`（complete 再置 1）；或仅 complete 时写入  

#### Errors（示例 message）

| 场景 | message |
|------|---------|
| 无默认渠 | `No default file upload channel` |
| 渠停用 | `File upload channel is disabled` |
| type=0 | `Local storage channel is not supported` |
| 超 max_size | `File exceeds channel max size` |
| WebDAV 无法分片且超限 | `Multipart not supported for this channel` |
| 限流 | `Too many upload prepares` |

---

### 4.2 `POST /api/storage/presign-part`

为 multipart 某一 part 签发/续签上传 URL（`presign_mode=lazy` 或过期续签）。

#### Request

```json
{
  "ticket_id": "st_xxx",
  "upload_token": "hmac...",
  "part_number": 2
}
```

| 字段 | 必填 | 说明 |
|------|------|------|
| `ticket_id` | 是 | |
| `upload_token` | 是 | 与 prepare 一致 |
| `part_number` | 是 | 从 1 开始，≤ part_count |

#### Response

```json
{
  "success": true,
  "data": {
    "part_number": 2,
    "upload_url": "https://...",
    "headers": {},
    "expires_at": 1710000900
  }
}
```

校验：票据未过期、user 匹配、strategy=multipart、part_number 合法。

---

### 4.3 `POST /api/storage/complete`

确认直传完成，落库，核销票据。

#### Request

```json
{
  "ticket_id": "st_xxx",
  "upload_token": "hmac...",
  "proof": {
    "etag": "\"abc\"",
    "parts": [
      { "part_number": 1, "etag": "\"p1\"" },
      { "part_number": 2, "etag": "\"p2\"" }
    ],
    "src": "/file/..."
  }
}
```

| proof（按渠道） | 说明 |
|-----------------|------|
| S3 single | `etag` |
| S3 multipart | `parts[]`（CompleteMultipartUpload） |
| ImgBed | `src` / `publicUrl` 等 |
| WebDAV | 可选；服务端 HEAD 校验 size |

服务端：

1. 验票 + nonce 一次性  
2. Provider Complete / Head 校验 `file_size`（能校 MD5 则校 `identifier`）  
3. upsert `file`（`status=1`），upsert `user_file`（`status=1`），`ref_count`  
4. 失败：不核销或标记失败；multipart 可 Abort（S3）

#### Response

```json
{
  "success": true,
  "data": {
    "file": { /* UserFileView */ }
  }
}
```

---

### 4.4 `POST /api/storage/abort`（必要）

取消未完成上传，释放 multipart / 票据。

#### Request

```json
{
  "ticket_id": "st_xxx",
  "upload_token": "hmac..."
}
```

#### Response

```json
{ "success": true, "data": { "aborted": true } }
```

行为：AbortMultipartUpload（若有）；删 status=0 的预写行（若有）；DEL 票据。

---

## 5. 接口文档 · 查

### 5.1 `GET /api/storage/self` — 我的文件（全部 / 条件）

#### Query

| 参数 | 说明 |
|------|------|
| `p` / `page_size` | 分页 |
| `file_suffix` | 后缀，不含点；多值可用逗号 `png,jpg`（OR） |
| `status` | `0`\|`1`；不传=全部 |
| `source` | 精确匹配 |
| `keyword` | 匹配 `file_name`（模糊） |
| `channel_type` | 可选，码值过滤 |
| `order` | `create_time_desc`（默认）\|`create_time_asc`\|`file_size_desc` |

强制：`user_id = 当前用户`。

#### Response

`data` 为 PageInfo，`items` 为 `UserFileView[]`。

---

### 5.2 `GET /api/storage/self/suffixes` — 我的后缀聚合（必要）

便于前端「按类型」筛选器。

#### Response

```json
{
  "success": true,
  "data": {
    "items": [
      { "file_suffix": "png", "count": 12 },
      { "file_suffix": "mp4", "count": 3 }
    ]
  }
}
```

仅统计当前用户；可加 `status` query。

---

### 5.3 `GET /api/storage/self/:id` — 详情

- `:id` = `user_file.id`  
- 必须属于当前用户，否则失败  
- 返回单个 `UserFileView`（`data` 直接为对象，或 `{ "file": ... }`，**推荐** `{ "file": UserFileView }`）

---

## 6. 接口文档 · 删

### 6.1 `DELETE /api/storage/self/:id`

删当前用户的逻辑文件。

#### 行为

1. 查 `user_file` 且 `user_id` 匹配  
2. 删（或硬删行）`user_file`  
3. `file.ref_count = ref_count - 1`  
4. 若 `ref_count <= 0`：Provider `Delete(object_key)`，`file.status=2` 后删行或保留审计（**推荐物理删行**）  
5. Provider 删失败：`file.status=2`，留异步重试（可记日志）；接口仍可对用户返回成功（逻辑已不可见）或返回部分失败——**推荐**：逻辑删除成功即 `success=true`，物理失败打日志 + status=2

#### Response

```json
{ "success": true, "data": { "deleted": true, "id": 99 } }
```

### 6.2 `DELETE /api/storage/self` — 批量删（必要）

#### Request

```json
{ "ids": [1, 2, 3] }
```

- 最多 100 个/次  
- 仅处理属于自己的 id；非法 id 记入 `failed`  
- 每条按 §6.1 语义

#### Response

```json
{
  "success": true,
  "data": {
    "deleted": [1, 2],
    "failed": [{ "id": 3, "message": "not found" }]
  }
}
```

---

## 7. 防滥用（增）

| 机制 | 要求 |
|------|------|
| HMAC 票据 | 绑定 user、channel、identifier、size、object_key、expire、nonce |
| 一次性 nonce | complete/abort 消费 |
| 短 TTL | 建议 10–15 分钟 |
| Prepare 限流 | 按 user_id（及 IP） |
| 密钥 | 永不下发 `config_proflle` |

---

## 8. 前端 SDK 行为（无 UI 详设，仅契约）

```ts
async function upload(file: File, opt?: { channelType?: '1'|'2'|'3'; source?: string }) {
  const identifier = await md5(file)
  const { file_name, file_suffix } = splitName(file.name)
  const prep = await api.prepare({ ... })
  if (prep.hit) return prep.file
  try {
    if (prep.plan.strategy === 'single') await put(prep.plan, file)
    else await putMultipart(prep, file) // 内含 presign-part
    return (await api.complete({ ticket_id, upload_token, proof })).file
  } catch (e) {
    await api.abort({ ticket_id, upload_token }).catch(() => {})
    throw e
  }
}
```

---

## 9. 错误码表（用户侧）

| message | 场景 |
|---------|------|
| `No default file upload channel` | 解析失败 |
| `File upload channel is disabled` | 指定渠停用 |
| `Local storage channel is not supported` | type=0 |
| `File exceeds channel max size` | 超限 |
| `Multipart not supported for this channel` | WebDAV 等 |
| `Upload ticket invalid or expired` | 票无效 |
| `File not found` | 查/删不存在或不属于自己 |
| `Too many upload prepares` | 限流 |

---

## 10. 验收清单（用户 API）

- [ ] prepare 秒传命中不上传  
- [ ] 小文件 single、大文件 multipart（S3）闭环  
- [ ] `presign-part` / `abort` 可用  
- [ ] `GET /self` 分页；`file_suffix` 过滤；默认含各 status  
- [ ] `GET /self/suffixes` 聚合正确  
- [ ] 删除仅本人；引用归零删物理  
- [ ] 批量删部分失败不影响其它  
- [ ] 响应无渠道密钥  

---

## 11. 关联

- [admin.md](./admin.md)  
- Providers：`../../providers/`  
- 渠道管理：`../file_upload_channel_design/`
