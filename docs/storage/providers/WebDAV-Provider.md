# WebDAV 协议存储渠道接入文档

> 协议参考：[RFC 4918 · HTTP Extensions for Web Distributed Authoring and Versioning (WebDAV)](https://datatracker.ietf.org/doc/html/rfc4918) · [RFC 2616/9110 · HTTP](https://httpwg.org/specs/rfc9110.html)  
> 常见实现：Nextcloud / ownCloud、坚果云、群晖 DSM、Apache `mod_dav`、Nginx + 第三方模块、Cloudflare ImgBed [`/dav/`](https://cfbed.sanyue.de/api/webdav.html)、各类网盘「WebDAV 兼容」网关（以厂商差异为准）

本文档面向本项目将 **WebDAV 兼容端点** 作为「上传渠道文件存储方之一」的接入评估与实现参考，覆盖：

- **新增（上传）**
- **删除**
- **展示（列出 / 访问）**
- **大文件上传能力与限制**

下文约定：

| 占位符 | 含义 | 示例 |
|--------|------|------|
| `{BASE}` | WebDAV 根 URL（须以 `/` 结尾更稳妥） | `https://dav.example.com/remote.php/dav/files/alice/` |
| `{PATH}` | 相对路径（对象键） | `playground/user_1/abc.png` |
| `{URL}` | 完整资源 URL | `{BASE}{PATH}` → `https://.../playground/user_1/abc.png` |

鉴权常见为 **HTTP Basic**（`Authorization: Basic base64(user:pass)`）或 **Bearer Token**（部分网关）；Digest 较少见。长效账号密码**只留在服务端**（管理员配置）；浏览器直传需走 Facade 短期票据策略（见抽象层文档），**不得**把 WebDAV 密码下发给前端。

---

## 1. 渠道定位与能力摘要

| 维度 | 说明 |
|------|------|
| 角色 | 基于 HTTP 的远程文件系统协议；以 **集合（目录）+ 资源（文件）** 寻址，路径即「对象键」 |
| 鉴权 | 多为 Basic / 应用密码；部分实现支持 Token；无 S3 式标准化 Presigned |
| 核心能力 | `PUT` 上传、`DELETE` 删除、`PROPFIND` 列举、`GET`/`HEAD` 读取、`MKCOL` 建目录、`MOVE`/`COPY`（可选） |
| 适合场景 | 已有网盘/NAS/自建 WebDAV；运维熟悉「挂盘」心智；中小文件、目录语义清晰 |
| 不适合场景 | 需要标准 Multipart、短时预签名、高并发对象存储计费模型；纯浏览器无 CORS/无代理时的直传 |

### 1.1 建议配置项（Provider）

| 配置 | 必需 | 说明 |
|------|------|------|
| `base_url` | 是 | WebDAV 根路径，如 `https://host/dav/` 或 Nextcloud `.../remote.php/dav/files/{user}/` |
| `username` | 是* | Basic 用户名；*部分 Token 方案可仅填 Token |
| `password` | 是* | 密码或应用专用密码；仅服务端保存，读出脱敏 |
| `auth_type` | 否 | `basic`（默认）/ `bearer` |
| `path_prefix` | 否 | 业务前缀，如 `playground/`，做环境隔离 |
| `public_base_url` | 否 | 若文件另有 HTTP 公开域名（CDN/直链），用于拼接展示 URL；否则走鉴权 `GET` 或 Facade 代理 |
| `timeout_ms` | 否 | 单次请求超时（大文件 PUT 需加大） |
| `verify_tls` | 否 | 自签证书场景可关（仅内网） |
| `chunk_threshold` | 否 | 超过则尝试分块策略（见 §5；依赖服务端能力） |
| `chunk_size` | 否 | 分块大小（若使用扩展分块协议） |

与本项目 Facade 对齐时：`base_url` / `username` / `password` 由**管理员**写入 `file_upload_channel.config_proflle`（`type='1'`，同 ImgBed Token 模式；字段名以 SQL 为准）。

### 1.2 方法能力矩阵（本渠道用到的子集）

| 方法 | 功能 | 本项目用途 |
|------|------|------------|
| `OPTIONS` | 探测允许的方法 | 联调 / 健康检查 |
| `MKCOL` | 创建集合（目录） | 上传前确保父路径存在 |
| `PUT` | 写入/覆盖文件 | **新增上传**（主路径） |
| `GET` | 下载文件 | 服务端校验、私有读 |
| `HEAD` | 元数据（长度、类型等） | `exists` / complete 校验 |
| `DELETE` | 删除文件或集合 | **删除** |
| `PROPFIND` | 列目录、取属性 | **展示 / 列表** |
| `MOVE` / `COPY` | 移动 / 复制 | 可选；改名可用 `MOVE` |
| `LOCK` / `UNLOCK` | 锁 | 一般不做；忽略即可 |

---

## 2. 新增（上传）

### 2.1 普通上传 · PUT（推荐小文件默认路径）

| 项 | 值 |
|----|-----|
| 方法 | `PUT` |
| URL | `{BASE}{PATH}` |
| 认证 | Basic / Bearer |
| Body | 文件原始字节 |
| 常见成功码 | `201 Created`（新建）或 `204 No Content` / `200 OK`（覆盖，视实现） |

#### 常用请求头

| Header | 必需 | 说明 |
|--------|------|------|
| `Content-Type` | 建议 | 如 `image/png`；部分服务器会忽略并按扩展名推断 |
| `Content-Length` | 是* | 体长度；Chunked Transfer 时行为因服而异，优先明确 Length |
| `Overwrite` | 否 | 部分实现支持；`T`/`F` 控制是否覆盖（非所有服务器实现） |
| `If-None-Match: *` | 否 | 仅当不存在时创建（支持条件请求的服务器） |

#### 请求示例

```http
PUT /remote.php/dav/files/alice/playground/user_1/abc.png HTTP/1.1
Host: dav.example.com
Authorization: Basic YWxpY2U6c2VjcmV0
Content-Type: image/png
Content-Length: 102400

<binary>
```

```bash
curl -X PUT \
  -u 'alice:app-password' \
  -H 'Content-Type: image/png' \
  --data-binary @./abc.png \
  'https://dav.example.com/remote.php/dav/files/alice/playground/user_1/abc.png'
```

#### 父目录不存在

多数实现要求父集合已存在，否则 `PUT` 返回 `409 Conflict`。上传前应对路径逐级 `MKCOL`（已存在通常 `405`/`409`，可忽略）：

```http
MKCOL /remote.php/dav/files/alice/playground/ HTTP/1.1
Authorization: Basic ...

MKCOL /remote.php/dav/files/alice/playground/user_1/ HTTP/1.1
Authorization: Basic ...
```

#### 接入约定（建议）

1. `PATH` 规范：`{path_prefix}{tenantOrUser}/{yyyy}/{mm}/{uuid}_{filename}`。  
2. Complete 时用 `HEAD` 核对 `Content-Length` 与票据中的 `size`。  
3. **浏览器直传**：WebDAV 少有短时预签名；推荐由 Facade 签发短期计划后，由**受控环境**上传，或服务端用管理员配置的账号代签/薄代理（见 §5.3）。默认不要把 `password` 给浏览器。

---

### 2.2 分块 / 断点类上传（非 RFC 标配）

标准 WebDAV **没有** 类似 S3 Multipart 的统一三步协议。大文件能力完全依赖服务端扩展：

| 方式 | 说明 | 支持度 |
|------|------|--------|
| 单次 `PUT` 大 Body | 最通用；受反向代理 `client_max_body_size`、网关超时限制 | ✅ 普遍 |
| `Transfer-Encoding: chunked` | HTTP 分块传编码，**不是**业务分片存储；中间件常改写/禁止 | ⚠️ |
| `Content-Range` + 多次 PUT | 部分服务器支持追加/分片写；**非标准，兼容性差** | ⚠️ 少 |
| Nextcloud 分块上传 API | 专有（如 `/remote.php/dav/uploads` 相关流程） | ⚠️ 仅 Nextcloud |
| TUS / 厂商分片 | 超出 WebDAV 本身 | 另议 |

**对本项目的含义**：WebDAV 渠道的 `chunk_threshold` 若触发「分片」，只能：

1. **优先**：仍单次 `PUT`（仅提高超时），或  
2. **按厂商适配**：实现 Nextcloud 等扩展分块 Adapter 分支，或  
3. **降级**：超过 `max_size` 直接拒绝，引导用户改用 `s3` 渠道。

不建议假设「所有 WebDAV 都能像 S3 一样 multipart」。

---

## 3. 删除

### 3.1 删除文件 · DELETE

| 项 | 值 |
|----|-----|
| 方法 | `DELETE` |
| URL | `{BASE}{PATH}` |
| 成功 | 常见 `204 No Content` / `200 OK` |
| 不存在 | 常见 `404`（部分实现仍 `204`，需按厂商实测） |

```http
DELETE /remote.php/dav/files/alice/playground/user_1/abc.png HTTP/1.1
Host: dav.example.com
Authorization: Basic ...
```

```bash
curl -X DELETE -u 'alice:app-password' \
  'https://dav.example.com/remote.php/dav/files/alice/playground/user_1/abc.png'
```

### 3.2 删除目录

对集合 URL（目录）发 `DELETE`，通常**递归删除**其下所有成员（RFC 允许；实际深度与权限因服而异）。业务删「前缀」时：

- 若确定是业务专属目录且可整目录回收 → `DELETE` 目录；  
- 否则先 `PROPFIND Depth:1/infinity` 列出再逐文件 `DELETE`（更安全）。

### 3.3 接入约定（建议）

- Provider `delete(path)` → 单资源 `DELETE`。  
- 与秒传 `ref_count` 配合：仅引用归零后删物理路径。  
- 删除目录前确认 `path` 不会误伤共享前缀。

---

## 4. 展示（列出 / 访问）

### 4.1 列表 · PROPFIND

| 项 | 值 |
|----|-----|
| 方法 | `PROPFIND` |
| URL | 目录 `{BASE}{DIR}/` 或探测单文件 |
| Header | `Depth: 0`（仅自身）/ `1`（直接子项）/ `infinity`（递归，**慎用**，部分服务器禁用） |
| Body | XML，声明需要的属性 |

#### 最小请求示例

```http
PROPFIND /remote.php/dav/files/alice/playground/user_1/ HTTP/1.1
Host: dav.example.com
Authorization: Basic ...
Depth: 1
Content-Type: application/xml; charset=utf-8

<?xml version="1.0" encoding="utf-8" ?>
<d:propfind xmlns:d="DAV:">
  <d:prop>
    <d:displayname/>
    <d:getcontentlength/>
    <d:getcontenttype/>
    <d:getlastmodified/>
    <d:resourcetype/>
    <d:getetag/>
  </d:prop>
</d:propfind>
```

```bash
curl -X PROPFIND -u 'alice:app-password' \
  -H 'Depth: 1' \
  -H 'Content-Type: application/xml' \
  --data '<?xml version="1.0"?><d:propfind xmlns:d="DAV:"><d:prop><d:getcontentlength/><d:resourcetype/><d:getlastmodified/></d:prop></d:propfind>' \
  'https://dav.example.com/remote.php/dav/files/alice/playground/user_1/'
```

#### 响应要点

- 状态常见 `207 Multi-Status`。  
- Body 为 `multistatus`：每个 `response` 含 `href` + `propstat`。  
- `resourcetype` 含 `collection` → 目录；否则为文件。  
- `getcontentlength` → 文件大小；目录可能无此属性。

#### 分页

标准 WebDAV **无** `continuation-token`。大目录需：

- 使用 `Depth: 1` 分层浏览，或  
- 依赖本项目 DB 元数据列表（`user_file`），WebDAV 仅作物理存储，**列表以库为准**（推荐）。

### 4.2 元数据 / 存在性 · HEAD

```http
HEAD /remote.php/dav/files/alice/playground/user_1/abc.png HTTP/1.1
Authorization: Basic ...
```

| 结果 | 说明 |
|------|------|
| `200` | 存在；可读 `Content-Length` / `Content-Type` / `ETag` |
| `404` | 不存在 |

适合 `complete` 校验与 `exists`。

### 4.3 下载 · GET

```http
GET /remote.php/dav/files/alice/playground/user_1/abc.png HTTP/1.1
Authorization: Basic ...
```

支持 `Range` 的服务器可分段下载（非上传分片）。

### 4.4 公开展示 URL

| 模式 | 做法 | 适用 |
|------|------|------|
| 独立公网直链 | 配置 `public_base_url`，展示 `{public_base_url}/{PATH}` | 网盘另开了公开分享/CDN |
| 仅 WebDAV 私有 | Facade `getUrl` 返回短时**下载票**或由 new-api 鉴权后流式 `GET` 反代 | 默认更安全 |
| ImgBed `/dav/` | 物理在 ImgBed 时，也可改走其 `/file/{id}` 公网路径（若有） | 混合部署 |

**CORS**：浏览器直接对 WebDAV 主机 `PUT`/`PROPFIND` 常被拦；生产若坚持浏览器触达存储方，必须在 WebDAV 反代上配置 CORS，并解决 Basic Auth 暴露问题——**难度高于 S3 Presigned**，选型时需知情。

---

## 5. 大文件上传支持程度评估

### 5.1 结论一览

| 能力 | 支持程度 | 说明 |
|------|----------|------|
| 小文件 `PUT` | ✅ 完整 | 实现最简单，跨厂商最一致 |
| 标准 Multipart（S3 级） | ❌ | RFC WebDAV 无对等协议 |
| 单次大 Body `PUT` | ⚠️ 受限 | 取决于反向代理 body 上限与超时；常 100MB～数 GB 不等 |
| 断点续传 | ⚠️ 厂商扩展 | Nextcloud 等有专有流程；不能当作通用能力 |
| 浏览器无密钥直传 | ⚠️ 弱 | 无标准化 Presigned；需 Facade 票 + 代理或短期专用账号 |
| 目录语义 / `MKCOL` | ✅ 完整 | 强于纯对象存储前缀模拟 |
| `PROPFIND` 海量列举 | ⚠️ | 无标准分页；大目录应依赖业务库 |

### 5.2 常见后端差异

| 后端 | 注意点 |
|------|--------|
| **Nextcloud / ownCloud** | 路径含用户名；推荐「应用密码」；大文件有专有分块，可单独适配 |
| **坚果云等网盘 WebDAV** | 有流量/大小策略；限速；应用密码；兼容性需实测 `Depth` |
| **群晖 / NAS** | 内网友好；公网需 HTTPS 与防火墙；注意证书 |
| **Apache mod_dav / 自建** | 行为接近标准；注意 `LimitXMLRequestBody`、目录权限 |
| **Cloudflare ImgBed `/dav/`** | 图床附带 WebDAV；与 REST `/upload` 是另一条入口；大文件仍受其底层渠道限制，详见 [Cloudflare-ImageBed-Provider.md](./Cloudflare-ImageBed-Provider.md) |

### 5.3 对本项目 Storage Provider 的实现建议

1. **默认路径**：一律 `MKCOL`（按需）+ `PUT`；`chunk_threshold` 默认可设较高或与 `max_size` 相同，避免假装通用分片。  
2. **大文件**：`max_size` 按目标 WebDAV 实测（如 100MB/512MB）配置；超限提示换 `s3`。若只对接 Nextcloud，再开 `chunk_mode=nextcloud` 扩展。  
3. **浏览器直传与防滥用**：  
   - 长效 `username/password` 仅管理员配置、仅服务端使用；  
   - Prepare 返回的「上传计划」优先指向 **new-api 短期上传中继**（流式反代到 WebDAV，不落盘），或短期专用凭据；  
   - 中继 URL 必须带 HMAC 票据绑定 `path/md5/size/expire`（与抽象层 §5 一致）。  
4. **列表**：用户文件库以 DB 为准；WebDAV `PROPFIND` 用于运维对账 / complete 校验，而非主列表。  
5. **配置暴露**：写入 `file_upload_channel`（`type='1'`）——表列 `chunk_threshold`/`chunk_size`/`max_size`/`timeout` 类超时放 `config_proflle.timeout_ms`；账号在 `config_proflle`（`base_url`、`username`、`password`、`auth_type`、`path_prefix`、`public_base_url`）。

### 5.4 与 S3 / ImgBed 对比（选型参考）

| 维度 | WebDAV | S3 Provider | Cloudflare ImgBed REST |
|------|--------|-------------|-------------------------|
| 协议 | 文件系统语义 HTTP | 对象存储 | 图床 HTTP API |
| 大文件 | 弱（缺统一分片） | 强（Multipart） | 中（自有分块/直传） |
| 浏览器直传 | 难（CORS + 密码） | 易（Presigned） | 中（勿下发长期 Token） |
| 目录 | 原生 `MKCOL` | 前缀模拟 | 有 `uploadFolder` |
| 可移植性 | 换网盘/NAS 需测兼容 | 换 R2/MinIO 成本低 | 绑定 ImgBed 实例 |
| 典型定位 | 存量 WebDAV / NAS 复用 | **生产主渠道推荐** | 图床生态 / 多后端网关 |

---

## 6. Provider 接口映射（便于后续编码）

| 本项目能力 | WebDAV | 备注 |
|------------|--------|------|
| `put` / `upload` | `MKCOL`* + `PUT` | *父路径按需创建 |
| `delete` | `DELETE` | 目录删除可能递归 |
| `list` | `PROPFIND` Depth:1 | 业务列表优先走 DB |
| `head` / `exists` | `HEAD` | 404 → 不存在 |
| `get` | `GET` | 可 Range |
| `getUrl` | `public_base_url+path` 或鉴权代理 URL | 无私有预签名标准 |
| `abortUpload` | 无标准 | 中断即停；失败可 `DELETE` 不完整文件（若已创建） |

本渠道在库表中的类型码值：`file_upload_channel.type` / `file.channel_type` = **`1`**（Webdav）。业务列表以 `user_file` 为准。

---

## 7. 最小联调清单

- [ ] `OPTIONS` / 账号可访问 `{BASE}`  
- [ ] `MKCOL` 创建业务前缀目录  
- [ ] `PUT` 上传小文件，`HEAD` 大小一致，`GET` 可下载  
- [ ] `PROPFIND Depth:1` 能看到该文件  
- [ ] `DELETE` 后 `HEAD` 为 404  
- [ ] 测清反向代理 **最大 Body** 与 **超时**，写入渠道 `max_size` / `timeout_ms`  
- [ ] （可选）浏览器 CORS + 直传可行性；不可行则确认走服务端票据中继  
- [ ] （可选）Nextcloud 应用密码与大文件扩展分块  

---

## 8. 管理员配置 JSON 示例

写入 `file_upload_channel`（`type='1'`；分片阈值用表列 `chunk_threshold`/`max_size`；密钥在 `config_proflle`，读出脱敏）：

```json
{
  "base_url": "https://dav.example.com/remote.php/dav/files/alice/",
  "username": "alice",
  "password": "app-password-or-secret",
  "auth_type": "basic",
  "path_prefix": "playground/",
  "public_base_url": "",
  "timeout_ms": 600000
}
```

> `max_size` / `chunk_threshold` 以表列为准，不必重复写进 JSON（若 JSON 也有同名 key，实现时以表列优先）。

---

## 9. 参考链接

- [RFC 4918 · WebDAV](https://datatracker.ietf.org/doc/html/rfc4918)  
- [RFC 6578 · Collection Synchronization for WebDAV](https://datatracker.ietf.org/doc/html/rfc6578)（同步场景，可选）  
- [Nextcloud · WebDAV](https://docs.nextcloud.com/server/latest/user_manual/en/files/access_webdav.html)  
- [Cloudflare ImgBed · WebDAV](https://cfbed.sanyue.de/api/webdav.html)  
- 同目录：[S3-Provider.md](./S3-Provider.md) · [Cloudflare-ImageBed-Provider.md](./Cloudflare-ImageBed-Provider.md)
