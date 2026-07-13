# 文件上传渠道 · 上传侧接口详设（use）

> 文档定位：为**后续文件上传**（秒传、浏览器直传、分片）预留并约定需开放的接口；本期管理端只保证渠道可配（见 [manager.md](./manager.md)），上传 Facade 可分期实现，但契约以本文为准。  
> 字段/码值以 [db-design](../../db-design/) 与 [FIleStorage.md](../../FIleStorage.md) 为准。  
> 与 manager 合计为 **FileUploadChannel 第一份详设**。

---

## 0. 范围划分

| 文档 | 受众 | 内容 |
|------|------|------|
| [manager.md](./manager.md) | 超管后台 | 渠道 CRUD、默认、探测、UI |
| **本文 use.md** | 上传 SDK / Playground / 后续开发 | 用户侧与上传控制面 API、渠道解析、错误码约定 |

本期（渠道管理上线时）**建议同步实现**的最小集合：§2 只读解析接口（供上传前确认默认渠道）。  
§3–§5 为上传主链路，可随「文件上传」迭代交付，但路径与字段名**现在锁定**，避免管理端与上传端两套命名。

---

## 1. 渠道类型与解析（上传侧必知）

| 码值 | 含义 | 本期上传是否启用 |
|------|------|------------------|
| `0` | 本地存储 | 否（管理端不可配；上传若收到 `0` → 明确错误） |
| `1` | Webdav | 是（能力见 WebDAV Provider；大文件受限） |
| `2` | Cloudflare-ImageBed | 是 |
| `3` | S3存储 | 是（推荐主渠道） |

### 1.1 解析顺序（Prepare / 直传前）

1. 请求带 `file_channel_id` > 0 → 使用该渠道（须 `status=1`）。  
2. 否则带 `channel_type`（`"1"`|`"2"`|`"3"`）→ 该 type 下 `is_default=1` 且启用的渠道；若无，则任意一条该 type 且启用的渠道（**可选策略：仅认 default，无则报错**——**推荐仅认 default，避免静默挑错渠**）。  
3. 否则 → 全局 `is_default=1` 且 `status=1`。  
4. 仍无 → 错误：`No default file upload channel`。

> 与 [manager.md](./manager.md) 一致：停用渠道不可被解析选中。

### 1.2 密钥边界

- 长效密钥只在服务端读 `file_upload_channel.config_proflle`。  
- 浏览器只拿 Prepare 返回的**短时上传计划**（Presigned / 分片票 / ImgBed 会话参数）。  
- 用户侧 API **不得**返回 `config_proflle` 明文。

---

## 2. 本期建议同步开放的只读接口

供前端上传组件、健康检查、管理页「当前默认」展示复用。

### 2.1 获取当前默认渠道（脱敏摘要）

| 项 | 值 |
|----|-----|
| 方法 | `GET` |
| 路径 | `/api/file-upload-channel/default` |
| 鉴权 | `UserAuth`（登录用户即可）或 `RootAuth`（二选一；**推荐 UserAuth**，上传页可调） |
| 说明 | 返回当前全局默认渠道的**非敏感**摘要 |

响应示例：

```json
{
  "success": true,
  "data": {
    "id": 1,
    "name": "R2 Primary",
    "type": "3",
    "chunk_threshold": 16777216,
    "chunk_size": 8388608,
    "max_size": 0,
    "status": "1",
    "is_default": "1"
  }
}
```

无默认：`success=true, data=null` 或 `success=false` + message（推荐 **200 + data=null**，便于前端分支）。

### 2.2 按类型解析可用渠道摘要（可选）

| 项 | 值 |
|----|-----|
| 方法 | `GET` |
| 路径 | `/api/file-upload-channel/resolve?channel_type=3` |
| 鉴权 | `UserAuth` |
| 说明 | 按 §1.1 解析结果；无 `config_proflle` |

用于 SDK 在上传前展示「将使用 S3 / 阈值 xx」。

> 完整渠道列表与密钥配置仍仅 [manager.md](./manager.md) 的 RootAuth 接口。

---

## 3. 上传主链路接口（后续实现 · 契约锁定）

鉴权：一律 `UserAuth`（session 或 AccessToken + `New-Api-User`）。  
信封：`{ success, message, data }`。  
用户 id：仅 `c.GetInt("id")`。

### 3.1 `POST /api/storage/prepare`

秒传检测 + 签发上传计划（或命中秒传直接建 `user_file`）。

**请求：**

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

| 字段 | 说明 |
|------|------|
| `channel_type` | 码值字符串；可与 `file_channel_id` 二选一或都不传（走全局默认） |
| `file_channel_id` | `0`=未指定 |
| `file_name` / `file_suffix` | 与 `user_file` 表一致 |
| `identifier` | MD5，跨用户秒传键（配合 `channel_type`） |
| `file_size` | 字节；与渠道 `max_size` / `chunk_threshold` 比较 |
| `source` | 业务来源 |

**响应 · 秒传命中：**

```json
{
  "success": true,
  "data": {
    "hit": true,
    "file": {
      "id": 12,
      "user_file_id": 99,
      "file_name": "out",
      "file_suffix": "png",
      "file_url": "https://cdn.example.com/...",
      "identifier": "...",
      "file_size": 1048576,
      "channel_type": "3",
      "file_channel_id": 1
    }
  }
}
```

**响应 · 需上传：**

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
    "object_key": "files/...",
    "plan": {
      "strategy": "single",
      "method": "PUT",
      "upload_url": "https://...",
      "headers": {}
    }
  }
}
```

`plan.strategy`：`single` | `multipart`（由服务端按该渠道 `chunk_threshold` 与 `file_size` 决定）。  
`type=1` WebDAV 大文件策略见 Provider：可能强制 `single` 或拒绝超 `max_size`。

### 3.2 `POST /api/storage/presign-part`（可选）

持有效 `ticket_id` + `upload_token`，为 multipart 某一 part 续签。

```json
{
  "ticket_id": "st_xxx",
  "upload_token": "hmac...",
  "part_number": 2
}
```

### 3.3 `POST /api/storage/complete`

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

成功：写入/更新 `file`（`status=1`，`ref_count`）、`user_file`；核销票据。

### 3.4 用户文件查询 / 删除

| 方法 | 路径 | 说明 |
|------|------|------|
| `GET` | `/api/storage/self` | 当前用户 `user_file` 分页列表 |
| `GET` | `/api/storage/self/:id` | 详情（含可访问 URL） |
| `DELETE` | `/api/storage/self/:id` | 删逻辑文件；`file.ref_count--`，归零后删物理对象 |

列表 Query：`p`、`page_size`、`channel_type`、`keyword`（`file_name`）、`file_suffix`、`source`。

### 3.5 管理员文件只读（后续）

| 方法 | 路径 | 鉴权 |
|------|------|------|
| `GET` | `/api/storage/` | `AdminAuth` 或 `RootAuth`（实现时与项目文件管理权限统一；**渠道密钥配置仍仅 Root**） |
| `GET` | `/api/storage/user/:user_id` | 同上 |

与渠道管理权限分离：普通管理员可查用户文件 ≠ 可改 `config_proflle`。

---

## 4. 与渠道表字段的消费关系

上传实现读取 `file_upload_channel`：

| 列 | 上传侧用途 |
|----|------------|
| `type` | 选择 Provider Adapter |
| `status` | ≠`1` 不可用 |
| `is_default` | §1.1 解析 |
| `chunk_threshold` / `chunk_size` | 是否 multipart、分片大小 |
| `max_size` | 超限直接 prepare 失败 |
| `config_proflle` | 仅服务端：签 Presign / 调 ImgBed / WebDAV |

写入 `file` / `user_file`：

| 表 | 关键字段 |
|----|----------|
| `file` | `identifier`,`channel_type`,`file_channel_id`,`object_key`,`file_url`,`file_size`,`mime_type`,`etag`,`ref_count`,`status`,`creater_user_id` |
| `user_file` | `file_id`,`file_channel_id`,`file_name`,`file_suffix`,`user_id`,`source`,`status` |

秒传：`uk_identifier_channel_type`；命中则加 `user_file` + `ref_count++`。

---

## 5. 错误约定（上传侧）

| 场景 | message 示例（英文 key，前端再 t） |
|------|-------------------------------------|
| 无默认渠道 | `No default file upload channel` |
| 渠道停用 | `File upload channel is disabled` |
| type=0 | `Local storage channel is not supported` |
| 超 max_size | `File exceeds channel max size` |
| 票据无效/过期 | `Upload ticket invalid or expired` |
| 秒传库冲突 | 按唯一约束处理，返回已有 `user_file` |

---

## 6. 防滥用（Prepare 签发时必须遵守）

详见 [FIleStorage.md](../../FIleStorage.md) §5。摘要：

- 短时 HMAC 票据 + Redis nonce 一次性  
- 绑定 `identifier` / `file_size` / `object_key` / Content-Type  
- Prepare 按用户限流  
- 禁止下发 `config_proflle` 内长期密钥  

WebDAV（type=`1`）若无法 Presign：允许 **短时中继 URL**（new-api 流式反代到 WebDAV，不落盘），中继仍校验 ticket——在上传迭代中实现，管理端配置不因此延期。

---

## 7. 前端 SDK 期望（消费方）

```ts
// channelType 为 SQL 码值；省略则走默认渠道
await storageClient.upload(file, { channelType: '3', source: 'playground' })
```

内部：`GET default`（可选）→ 算 MD5 → `prepare` → 若 `hit` 结束 → 按 `plan` 直传 → `complete`。

Default / Classic / Playground **共用**本文 API，不各写一套字段名。

---

## 8. 分期交付建议

| 阶段 | 交付 |
|------|------|
| P0（与 manager 同发） | Root 渠道 CRUD + 探测；`GET .../default`（§2.1） |
| P1 | `prepare` / `complete` + S3 single + 秒传 |
| P2 | S3 multipart、`presign-part`；ImgBed；用户 `self` 列表删除 |
| P3 | WebDAV（含中继如需）；管理员查用户文件 |

---

## 9. 验收（上传契约）

- [ ] 文档字段与 SQL 一致：`identifier`、`file_name`、`file_suffix`、`file_channel_id`、`config_proflle`（仅服务端）  
- [ ] 码值仅 `1/2/3` 可上传；`0` 被拒绝  
- [ ] 解析顺序与默认渠道、停用规则与 manager 一致  
- [ ] 用户 API 不泄露密钥  
- [ ] P0 上线时至少 `GET /api/file-upload-channel/default` 可用  

---

## 10. 关联

- [manager.md](./manager.md)  
- [FIleStorage.md](../../FIleStorage.md)  
- [file.sql](../../db-design/file.sql) · [user_file.sql](../../db-design/user_file.sql) · [file_upload_channel.sql](../../db-design/file_upload_channel.sql)
