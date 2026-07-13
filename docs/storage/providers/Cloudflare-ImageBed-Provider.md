# Cloudflare ImgBed 存储渠道接入文档

> 官方文档：[API 基本介绍](https://cfbed.sanyue.de/api/) · [上传](https://cfbed.sanyue.de/api/upload.html) · [删除](https://cfbed.sanyue.de/api/delete.html) · [列出](https://cfbed.sanyue.de/api/list.html) · [随机图](https://cfbed.sanyue.de/api/random.html)  
> 项目：[MarSeventh/CloudFlare-ImgBed](https://github.com/MarSeventh/CloudFlare-ImgBed)

本文档面向本项目将 **Cloudflare ImgBed** 作为「上传渠道文件存储方之一」的接入评估与实现参考，覆盖：

- **新增（上传）**
- **删除**
- **展示（列出 / 访问）**
- **大文件上传能力与限制**

下文中 `{BASE_URL}` 表示图床实例根地址，例如 `https://your.imgbed.domain`。

---

## 1. 渠道定位与能力摘要

| 维度 | 说明 |
|------|------|
| 角色 | 自托管图床 / 网盘网关，上层暴露统一 REST + WebDAV，底层可对接 Telegram / R2 / S3 / Discord / HuggingFace / WebDAV 等 |
| 鉴权 | API Token（推荐）或上传认证码 `authCode` |
| 核心能力 | 上传、删除、列表检索、文件公开访问、随机图、WebDAV |
| 适合场景 | 需要统一文件生命周期（增删查）的图片/多媒体存储；可与多上传渠道并存，作为可选 Storage Provider |
| 不适合场景 | 需要强一致事务、对象级 ACL、原生 multipart 语义与 S3 完全一致的场景（需自行适配本 API） |

### 1.1 建议权限划分

在图床管理端：`系统设置 → 安全设置 → API Token 管理` 创建 Token（仅显示一次）。

| 权限 | 本项目用途 |
|------|------------|
| `upload` | 写入 / 新增文件 |
| `delete` | 删除文件或目录 |
| `list` | 列表展示、检索、统计 |
| `manage` | Token 管理（一般不必给业务侧） |

请求头：

```http
Authorization: Bearer YOUR_API_TOKEN
# 或
Authorization: YOUR_API_TOKEN
```

---

## 2. 新增（上传）

### 2.1 基本信息

| 项 | 值 |
|----|-----|
| 端点 | `POST {BASE_URL}/upload` |
| Content-Type | `multipart/form-data` |
| 认证 | 上传认证码 **或** 具备 `upload` 权限的 API Token |

### 2.2 普通上传（推荐小文件默认路径）

适用于常规图片、小视频等单次可完整提交的文件。

#### Query 参数

| 参数 | 类型 | 必需 | 默认 | 说明 |
|------|------|------|------|------|
| `authCode` | string | 否 | - | 上传认证码（未用 Token 时使用） |
| `uploadChannel` | string | 否 | `telegram` | 底层渠道：`telegram` / `cfr2` / `s3` / `discord` / `huggingface` / `webdav` |
| `channelName` | string | 否 | - | 多渠道时指定名称；可用 `{BASE_URL}/api/channels` 查询 |
| `serverCompress` | boolean | 否 | `true` | 服务端压缩（仅 Telegram 渠道图片） |
| `autoRetry` | boolean | 否 | `true` | 失败时自动切换渠道重试 |
| `uploadNameType` | string | 否 | `default` | `default`（前缀_原名）/ `index` / `origin` / `short` |
| `returnFormat` | string | 否 | `default` | `default` → `/file/{id}`；`full` → 当前站点完整 URL |
| `uploadFolder` | string | 否 | - | 相对目录，如 `img/playground` |

#### Body（FormData）

| 参数 | 类型 | 必需 | 说明 |
|------|------|------|------|
| `file` | File | 是 | 待上传文件 |

#### 请求示例

```bash
curl -X POST "{BASE_URL}/upload?returnFormat=full&uploadFolder=playground&uploadChannel=cfr2" \
  -H "Authorization: Bearer YOUR_API_TOKEN" \
  -F "file=@./result.png"
```

#### 成功响应

响应为**数组**（便于批量扩展；单文件时取 `[0]`）：

```json
[
  {
    "src": "/file/abc123_result.png",
    "publicUrl": "https://cdn.example.com/abc123_result.png"
  }
]
```

| 字段 | 说明 |
|------|------|
| `src` | 站内访问路径；`returnFormat=full` 时可为完整链接 |
| `publicUrl` | 可选。若管理端配置了「默认 URL 前缀」，则返回前缀 + 文件 ID 的公开链接 |

#### 接入约定（建议）

对本项目 Storage Provider：

1. 优先使用 `Authorization: Bearer` + `upload` Token。  
2. 上传时带 `returnFormat=full`，或配置默认 URL 前缀后优先落库 `publicUrl`。  
3. 业务侧保存：`src`（图床内 ID/路径）+ `url`（最终可访问地址）+ 可选 `uploadFolder` 前缀，便于后续删除。

---

### 2.3 分块上传（大文件 · Telegram / R2 / S3 / Discord）

当文件超过单次请求或渠道上限时，使用客户端分块：

**初始化 → 逐块上传 → 合并**（可选：取消时 Cleanup）。

> **注意**：HuggingFace 渠道**不支持**本分块流程，大文件见 [2.4](#24-huggingface-大文件直传)。

#### 推荐分块大小

| 底层渠道 | 推荐分块 | 说明 |
|----------|----------|------|
| `telegram` | 16MB | Bot getFile 约 20MB 上限，留余量 |
| `cfr2` / `s3` | 5MB–20MB | 对齐 S3 Multipart，最小 5MB |
| `discord` | 8MB | 免费约 10MB 上限 |

#### Step 1 · 初始化

`POST {BASE_URL}/upload?initChunked=true&uploadChannel={channel}`

FormData：

| 参数 | 必需 | 说明 |
|------|------|------|
| `originalFileName` | 是 | 原始文件名 |
| `originalFileType` | 是 | MIME，如 `image/png` |
| `totalChunks` | 是 | 分块总数 |

响应示例：

```json
{
  "success": true,
  "uploadId": "upload_1713500000000_abc123def",
  "sessionInfo": {
    "uploadId": "upload_1713500000000_abc123def",
    "originalFileName": "video.mp4",
    "totalChunks": 5,
    "uploadChannel": "telegram"
  }
}
```

#### Step 2 · 上传分块

`POST {BASE_URL}/upload?chunked=true&uploadChannel={channel}`

FormData：`file`（分块二进制）、`uploadId`、`chunkIndex`（从 0）、`totalChunks`、`originalFileName`、`originalFileType`。

- 请求同步：服务端落库完成后再返回。  
- 可并发上传；**R2/S3** 须先完成 `chunkIndex=0`（Multipart 初始化），其余块最多等待 60s。

#### Step 3 · 合并

`POST {BASE_URL}/upload?chunked=true&merge=true&uploadChannel={channel}`

可选 Query：`returnFormat`、`uploadFolder`。

FormData：`uploadId`、`totalChunks`、`originalFileName`、`originalFileType`。

成功响应格式与普通上传相同（数组，含 `src` / `publicUrl`）。合并时服务端最多对失败块重试 5 次。

#### Cleanup（取消）

`POST {BASE_URL}/upload?cleanup=true&uploadId={id}&totalChunks={n}`

释放临时会话数据。

---

### 2.4 HuggingFace 大文件直传

面向 `uploadChannel=huggingface`：

| 场景 | 方式 |
|------|------|
| &lt; 20MB | 普通 `POST /upload?uploadChannel=huggingface`（后端代理） |
| ≥ 20MB | 客户端直传 LFS/S3：签名 → PUT → commit（绕过 CF 100MB body / CPU 限制） |

直传三步（摘要）：

1. 客户端计算文件 SHA-256（hex）与前 512 字节 Base64（`fileSample`）。  
2. `POST /upload/huggingface/getUploadUrl`（JSON + Bearer）获取 `uploadAction`。  
3. 按 `uploadAction` PUT 到 HuggingFace；完成后 `POST /upload/huggingface/commitUpload` 登记元数据。

本项目若不以 HuggingFace 为 ImgBed 底层渠道，可暂不实现直传；若支持，需单独适配 Provider。

---

### 2.5 单次请求与渠道体积上限（硬限制）

| 限制来源 | 上限 | 备注 |
|----------|------|------|
| Cloudflare Pages 请求体 | 100MB | 普通上传单次上限 |
| Telegram | 单文件约 20MB | 更大需分块 |
| Discord | 10MB / Nitro 25MB | 更大需分块 |
| R2 / S3 | 视配置 | 大文件用分块 Multipart |
| HuggingFace 普通上传 | 受 Workers CPU/代理限制 | ≥20MB 建议直传 |

---

## 3. 删除

### 3.1 基本信息

| 项 | 值 |
|----|-----|
| 端点 | `GET {BASE_URL}/api/manage/delete/{path}` |
| 方法 | `GET`（官方约定） |
| 认证 | 需要 `delete` 权限的 API Token |
| Content-Type | `application/json`（响应） |

### 3.2 参数

#### Path

| 参数 | 类型 | 必需 | 说明 |
|------|------|------|------|
| `path` | string | 是 | 文件或目录相对路径，如 `playground/abc123_result.png` 或 `playground` |

> 路径分隔使用 `/`。业务侧应以上传返回的 `src` 解析出文件 ID/路径（去掉 `/file/` 前缀后作为 `path`）。

#### Query

| 参数 | 类型 | 必需 | 默认 | 说明 |
|------|------|------|------|------|
| `folder` | boolean | 否 | `false` | `true` 时递归删除整个目录及其内容 |

### 3.3 行为说明

| 模式 | 条件 | 行为 |
|------|------|------|
| 单文件 | `folder` 为 false / 省略 | 删除指定文件 |
| 目录 | `folder=true` | 递归删除目录下所有子项 |

### 3.4 请求示例

```bash
# 删除单个文件
curl -X GET "{BASE_URL}/api/manage/delete/playground/abc123_result.png" \
  -H "Authorization: Bearer YOUR_API_TOKEN"

# 删除整个目录
curl -X GET "{BASE_URL}/api/manage/delete/playground?folder=true" \
  -H "Authorization: Bearer YOUR_API_TOKEN"
```

### 3.5 响应

单文件成功：

```json
{
  "success": true,
  "fileId": "playground/abc123_result.png"
}
```

目录成功：

```json
{
  "success": true,
  "deleted": [
    "playground/a.png",
    "playground/b.png"
  ],
  "failed": []
}
```

失败：

```json
{
  "success": false,
  "error": "Delete file failed"
}
```

### 3.6 接入约定（建议）

- Provider `delete(objectKey)`：将内部 objectKey 映射为图床 `path`，调用本接口。  
- 批量删除：可循环单文件删除，或按业务目录使用 `folder=true`（谨慎，避免误删）。  
- 删除结果以 `success` 为准；目录删除需检查 `failed` 数组。

---

## 4. 展示（列出 / 访问）

「展示」包含两类能力：**元数据列表检索** 与 **文件内容访问**。

### 4.1 列表 API（管理侧展示 / 同步索引）

| 项 | 值 |
|----|-----|
| 端点 | `GET {BASE_URL}/api/manage/list` |
| 认证 | 需要 `list` 权限 |
| 响应 | `application/json` |

#### 常用 Query

| 参数 | 类型 | 默认 | 说明 |
|------|------|------|------|
| `start` | number | `0` | 分页起点 |
| `count` | number | `50` | 返回条数；`-1` 不限制 |
| `sum` | boolean | `false` | 与 `count=-1` 配合：只返回总数 |
| `recursive` | boolean | `false` | 是否递归子目录 |
| `dir` | string | `""` | 目录，如 `playground` |
| `search` | string | `""` | 文件名关键词 |
| `includeTags` / `excludeTags` | string | `""` | 标签筛选，逗号分隔 |
| `channel` | string | `""` | `TelegramNew` / `CloudflareR2` / `S3` / `Discord` / `HuggingFace` / `External` |
| `channelName` | string | `""` | 如 `TelegramNew:default` |
| `listType` | string | `""` | `White` / `Block` / `None` |
| `accessStatus` | string | `""` | `normal` / `blocked` |
| `label` | string | `""` | `normal` / `teen` / `adult` |
| `fileType` | string | `""` | `image` / `video` / `audio` / `other` |
| `action` | string | `""` | 特殊操作：`rebuild` / `info` 等 |

#### 请求示例

```bash
# 分页列出某目录下图片
curl -X GET "{BASE_URL}/api/manage/list?dir=playground&fileType=image&start=0&count=50" \
  -H "Authorization: Bearer YOUR_API_TOKEN"

# 仅统计数量
curl -X GET "{BASE_URL}/api/manage/list?dir=playground&count=-1&sum=true" \
  -H "Authorization: Bearer YOUR_API_TOKEN"
```

#### 列表响应示例

```json
{
  "files": [
    {
      "name": "playground/abc123_result.png",
      "metadata": {
        "Channel": "telegram",
        "TimeStamp": "1754020094217",
        "File-Mime": "image/jpeg",
        "File-Size": "1024000"
      }
    }
  ],
  "directories": ["playground/sub"],
  "totalCount": 100,
  "returnedCount": 50,
  "indexLastUpdated": "1754020094217",
  "isIndexedResponse": true
}
```

统计响应：

```json
{
  "sum": 100,
  "indexLastUpdated": "1754020094217"
}
```

错误：

```json
{
  "error": "Internal server error",
  "message": "详细错误信息"
}
```

#### 接入约定（建议）

- UI「文件库 / 历史结果」：用 `dir` + `fileType=image` + 分页。  
- 访问 URL：`{BASE_URL}/file/{name}`，或上传时保存的 `publicUrl` / `returnFormat=full` 结果。  
- 若列表滞后，可调用 `action=rebuild` 重建索引（异步，管理端操作）。

---

### 4.2 文件访问（公开展示）

| 方式 | URL 形态 | 说明 |
|------|----------|------|
| 默认路径 | `{BASE_URL}/file/{fileId}` | 与上传返回的 `src` 对应 |
| 完整链接 | `returnFormat=full` 或配置的 `publicUrl` | 推荐业务直接存此字段 |
| WebDAV | `{BASE_URL}/dav/...` | 客户端挂载场景，非 Web UI 主路径 |

浏览器 / `<img>` / CSS 可直接引用上述 URL（需图床侧访问策略允许）。

---

### 4.3 随机图（可选展示能力）

| 项 | 值 |
|----|-----|
| 端点 | `GET {BASE_URL}/random` |
| 前置 | 管理端开启随机图功能 |

| 参数 | 默认 | 说明 |
|------|------|------|
| `content` | `image` | `image` / `video`，逗号分隔 |
| `type` | `path` | `img` 直接返回图片流；`url` 返回完整 URL |
| `form` | `json` | `text` 时返回纯文本路径（`type=img` 时无效） |
| `dir` | - | 限定目录（含子目录） |
| `orientation` | - | `landscape` / `portrait` / `square` / `auto` |

示例：

```bash
curl "{BASE_URL}/random?type=img&dir=playground&orientation=landscape"
```

默认 JSON：

```json
{ "url": "/file/4fab4d423d039b4665a27.jpg" }
```

本项目若仅作结果存储，随机图非必需；可作为「素材池 / 壁纸」类功能复用。

---

## 5. 大文件上传支持程度评估

本节用于判断 ImgBed **是否适合作为大文件存储方**，以及本项目适配时应选哪条路径。

### 5.1 结论一览

| 能力 | 支持程度 | 说明 |
|------|----------|------|
| 小文件普通上传 | ✅ 完整 | `POST /upload` + multipart，实现成本最低 |
| 分块上传（TG / R2 / S3 / Discord） | ✅ 完整 | 官方三步协议；适合作为大文件主路径 |
| HuggingFace 直传 | ✅ 完整（专用协议） | 与分块 API 不兼容，需单独实现 |
| 单请求 > 100MB（无分块） | ❌ | 受 Cloudflare Pages/Workers 请求体限制 |
| 断点续传（跨会话） | ⚠️ 有限 | 依赖同一次 `uploadId` 会话；中断需 Cleanup 后重来或自行重试未完成块 |
| 与原生 S3 Presigned 完全一致 | ❌ | 需适配 ImgBed 自有 API，而非直接调 S3 SDK |

### 5.2 按底层渠道的大文件策略

| ImgBed 的 `uploadChannel` | 小文件 | 大文件推荐 | 本项目适配优先级建议 |
|---------------------------|--------|------------|----------------------|
| `cfr2` | 普通上传 | 分块（5–20MB） | **高**（容量与稳定性好） |
| `s3` | 普通上传 | 分块（5–20MB） | **高** |
| `telegram` | 普通上传（≤20MB） | 分块（16MB） | 中（免费但体积与速率受限） |
| `discord` | ≤10/25MB | 分块（8MB） | 低 |
| `huggingface` | &lt;20MB 普通上传 | ≥20MB 直传三步 | 中（实现成本高） |
| `webdav` | 普通上传 | 视远端 WebDAV | 视部署 |

### 5.3 对本项目 Storage Provider 的实现建议

1. **默认路径**：`upload`（普通）→ 保存 `src` + 最终 URL。  
2. **阈值阈值**（建议）：单文件 &gt; 15–20MB 或接近渠道上限时，自动走分块三步（`cfr2`/`s3`/`telegram`/`discord`）。  
3. **删除 / 列表**：统一走 `delete` + `list`，与底层渠道无关（由 ImgBed 管理索引）。  
4. **配置项建议**暴露给运营：  
   - `baseUrl`  
   - `apiToken`（至少 `upload`；完整生命周期需 `delete` + `list`）  
   - `uploadChannel` / `channelName`  
   - `uploadFolder`（按用户/任务隔离目录）  
   - `returnFormat` / 是否使用 `publicUrl`  
   - `chunkSize` / `largeFileThreshold`  
5. **错误与过期**：Token 过期时可能返回 `{ "valid": false, "error": "Token 已过期" }`，Provider 应映射为可重试的鉴权错误。

### 5.4 能力边界（接入前确认）

- ImgBed 是**网关**，最终容量、速率、费用取决于其配置的底层渠道（R2/S3/TG 等）。  
- 内容审查、黑白名单、访问屏蔽会影响「展示」可达性（`accessStatus` / `listType` / `label`）。  
- WebDAV（`/dav/`）可作为运维旁路，不建议作为 Web 业务主上传路径。

---

## 6. Provider 接口映射（便于后续编码）

将本渠道抽象为本项目统一 Storage 能力时的建议映射：

| 本项目能力 | ImgBed API | 备注 |
|------------|------------|------|
| `put` / `upload` | `POST /upload` 或分块三步 | 按体积分支 |
| `delete` | `GET /api/manage/delete/{path}` | path 来自上传 `src` |
| `list` | `GET /api/manage/list` | `dir` / 分页 / `fileType` |
| `getUrl` | `src` → `{BASE_URL}{src}` 或存 `publicUrl` | 勿重复拼接 |
| `exists` / 详情 | `list` + `search`/`dir` 或依赖本地索引 | 无独立 HEAD 文档时用 list |

本渠道在库表中的类型码值：`file_upload_channel.type` / `file.channel_type` = **`2`**（Cloudflare-ImageBed）。管理员将 `base_url` / `api_token` 等写入 `file_upload_channel.config_proflle`（不下发浏览器）；分片阈值用表列 `chunk_threshold` / `chunk_size`。

---

## 7. 最小联调清单

- [ ] Token 具备 `upload` / `delete` / `list`  
- [ ] 普通上传一张图，拿到 `src` / `publicUrl`，浏览器可打开  
- [ ] `list?dir=...` 能看到该文件  
- [ ] `delete` 后列表与访问均不可用  
- [ ] （可选）&gt; 阈值的文件走分块上传并成功合并  
- [ ] （可选）配置 `uploadFolder` 做租户/任务隔离  

---

## 8. 参考链接

- [API 基本介绍](https://cfbed.sanyue.de/api/)  
- [上传 API](https://cfbed.sanyue.de/api/upload.html)  
- [删除 API](https://cfbed.sanyue.de/api/delete.html)  
- [列出 API](https://cfbed.sanyue.de/api/list.html)  
- [随机图 API](https://cfbed.sanyue.de/api/random.html)  
- [Token 管理 API](https://cfbed.sanyue.de/api/token.html)  
- [WebDAV](https://cfbed.sanyue.de/api/webdav.html)  
- [GitHub · CloudFlare-ImgBed](https://github.com/MarSeventh/CloudFlare-ImgBed)
