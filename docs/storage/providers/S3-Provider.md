# S3 协议存储渠道接入文档

> 协议参考：[Amazon S3 API](https://docs.aws.amazon.com/AmazonS3/latest/API/Welcome.html) · [PutObject](https://docs.aws.amazon.com/AmazonS3/latest/API/API_PutObject.html) · [DeleteObject](https://docs.aws.amazon.com/AmazonS3/latest/API/API_DeleteObject.html) · [ListObjectsV2](https://docs.aws.amazon.com/AmazonS3/latest/API/API_ListObjectsV2.html) · [Multipart Upload](https://docs.aws.amazon.com/AmazonS3/latest/userguide/mpuoverview.html)  
> 兼容实现：AWS S3、Cloudflare R2、MinIO、阿里云 OSS（S3 兼容）、腾讯云 COS（S3 兼容）、Backblaze B2 等（以各厂商差异为准）

本文档面向本项目将 **S3 兼容对象存储** 作为「上传渠道文件存储方之一」的接入评估与实现参考，覆盖：

- **新增（上传）**
- **删除**
- **展示（列出 / 访问）**
- **大文件上传能力与限制**

下文约定：

| 占位符 | 含义 | 示例 |
|--------|------|------|
| `{ENDPOINT}` | S3 API 端点 | `https://s3.ap-northeast-1.amazonaws.com` / `https://xxx.r2.cloudflarestorage.com` |
| `{BUCKET}` | 桶名 | `playground-assets` |
| `{KEY}` | 对象键（路径） | `playground/user_1/abc.png` |
| `{REGION}` | 区域 | `ap-northeast-1` / `auto`（R2） |

鉴权默认使用 **AWS Signature Version 4**（`Authorization: AWS4-HMAC-SHA256 ...`），由官方 SDK（`@aws-sdk/client-s3`、`boto3`、`aws-sdk-go` 等）自动签署；下文接口以 REST 语义描述，实现时优先用 SDK，避免手写签名。

---

## 1. 渠道定位与能力摘要

| 维度 | 说明 |
|------|------|
| 角色 | 业界标准对象存储协议；对象以 `bucket + key` 寻址，无真正目录（`/` 仅为 key 前缀约定） |
| 鉴权 | Access Key ID + Secret Access Key（SigV4）；可选临时凭证（STS）、Presigned URL |
| 核心能力 | 上传、删除、列举、下载/Head、Multipart 大文件、预签名直传、生命周期与 CDN 对接 |
| 适合场景 | 生产级图片/多媒体持久化；多租户按前缀隔离；需要大文件、高并发、可替换后端（R2/MinIO/OSS） |
| 不适合场景 | 需要强 POSIX 目录语义、对象内事务、强一致跨对象改名（S3 无原子 rename，需 Copy+Delete） |

### 1.1 建议配置项（Provider）

| 配置 | 必需 | 说明 |
|------|------|------|
| `endpoint` | 视厂商 | AWS 可省略走默认；R2/MinIO/自定义必须填 |
| `region` | 是 | R2 常用 `auto` |
| `bucket` | 是 | 桶名 |
| `accessKeyId` / `secretAccessKey` | 是* | *也可用 IAM Role / STS，无静态密钥 |
| `forcePathStyle` | 否 | MinIO 等常需 `true`（`/{bucket}/{key}`）；AWS 虚拟托管风格默认 `false` |
| `publicBaseUrl` | 否 | CDN / 自定义域名，用于拼接展示 URL |
| `keyPrefix` | 否 | 全局前缀，如 `playground/`，做环境或产品隔离 |
| `defaultAcl` / Bucket Policy | 否 | 公有读或仅预签名访问 |

### 1.2 IAM / 权限最小集

业务侧建议仅授予当前桶的必要操作：

| 操作 | 本项目用途 |
|------|------------|
| `s3:PutObject` | 新增上传 |
| `s3:GetObject` | 服务端拉取 / 校验 |
| `s3:DeleteObject` / `s3:DeleteObjectVersion` | 删除 |
| `s3:ListBucket` | 列表展示（需 Resource 含 bucket，并限制 `prefix`） |
| `s3:AbortMultipartUpload` / `s3:ListMultipartUploadParts` | 大文件分片与清理 |
| `s3:CreateMultipartUpload` / `s3:UploadPart` / `s3:CompleteMultipartUpload` | Multipart |

---

## 2. 新增（上传）

### 2.1 普通上传 · PutObject（推荐小文件默认路径）

适用于常规图片、小对象等一次可完整提交的文件。

| 项 | 值 |
|----|-----|
| 方法 | `PUT` |
| 路径（虚拟托管） | `https://{BUCKET}.{host}/{KEY}` |
| 路径（Path-style） | `{ENDPOINT}/{BUCKET}/{KEY}` |
| 认证 | SigV4 |
| Body | 对象原始字节 |

#### 常用请求头

| Header | 必需 | 说明 |
|--------|------|------|
| `Content-Type` | 建议 | 如 `image/png`、`image/jpeg`、`video/mp4` |
| `Content-Length` | 是* | HTTP 体长度（SDK 通常自动设置） |
| `Content-MD5` | 否 | 完整性校验 |
| `x-amz-acl` | 否 | 部分厂商支持 `public-read`；R2 等可能忽略，改用 Bucket Policy |
| `x-amz-meta-*` | 否 | 用户元数据，如 `x-amz-meta-task-id` |
| `x-amz-storage-class` | 否 | `STANDARD` / `INTELLIGENT_TIERING` 等（视厂商） |
| `Cache-Control` | 否 | 如 `public, max-age=31536000, immutable` |

#### 请求示例（概念）

```http
PUT /playground/user_1/abc.png HTTP/1.1
Host: playground-assets.s3.ap-northeast-1.amazonaws.com
Authorization: AWS4-HMAC-SHA256 Credential=...
Content-Type: image/png
Content-Length: 102400

<binary>
```

#### SDK 示例（TypeScript / AWS SDK v3）

```typescript
import { PutObjectCommand, S3Client } from '@aws-sdk/client-s3'

const client = new S3Client({
  region: process.env.S3_REGION,
  endpoint: process.env.S3_ENDPOINT || undefined,
  forcePathStyle: process.env.S3_FORCE_PATH_STYLE === 'true',
  credentials: {
    accessKeyId: process.env.S3_ACCESS_KEY_ID!,
    secretAccessKey: process.env.S3_SECRET_ACCESS_KEY!,
  },
})

await client.send(
  new PutObjectCommand({
    Bucket: 'playground-assets',
    Key: 'playground/user_1/abc.png',
    Body: fileBuffer,
    ContentType: 'image/png',
    CacheControl: 'public, max-age=31536000, immutable',
  }),
)
```

#### 成功响应要点

| 项 | 说明 |
|----|------|
| HTTP | `200 OK` |
| `ETag` | 对象校验标识（单次 Put 多为内容 MD5 的引号形式） |
| `VersionId` | 开启版本控制时返回 |

#### 单次 Put 体积上限

| 限制 | 值 |
|------|-----|
| S3 单次 `PutObject` | **最大 5 GiB** |
| 实践建议 | 超过 **8–16 MiB**（或网关/反代限制更低时）改走 Multipart 或预签名直传，避免服务端内存与超时 |

---

### 2.2 预签名上传 · Presigned PUT / POST（推荐浏览器直传）

服务端不经手文件字节，只签发短时 URL，客户端直传对象存储。适合减轻 API 服务器带宽与超时压力。

#### Presigned PUT

1. 服务端用密钥生成预签名 URL（有效期如 5–15 分钟）。  
2. 客户端 `PUT` 该 URL，Body 为文件；`Content-Type` 须与签名时一致。  
3. 成功后业务侧登记 `{KEY}` 与公开/CDN URL。

```typescript
import { PutObjectCommand } from '@aws-sdk/client-s3'
import { getSignedUrl } from '@aws-sdk/s3-request-presigner'

const url = await getSignedUrl(
  client,
  new PutObjectCommand({
    Bucket: 'playground-assets',
    Key: 'playground/user_1/abc.png',
    ContentType: 'image/png',
  }),
  { expiresIn: 900 },
)
// 客户端: fetch(url, { method: 'PUT', headers: { 'Content-Type': 'image/png' }, body: file })
```

#### Presigned POST（可选）

表单字段 + `policy` 条件（限制前缀、长度、Content-Type），适合浏览器 `<form>` / 更细条件约束。部分兼容实现支持度不一，接入前需验证目标厂商。

#### 接入约定（建议）

1. Key 规范：`{keyPrefix}{tenantOrUser}/{yyyy}/{mm}/{uuid}_{filename}`，避免覆盖与枚举。  
2. 上传完成后以 `HeadObject` 或业务回调确认对象存在，再写库。  
3. 展示 URL：优先 `publicBaseUrl + '/' + key`；私有桶则用短时 Presigned GET。

---

### 2.3 分片上传 · Multipart Upload（大文件主路径）

官方 Multipart 流程，适用于大文件与不稳定网络；兼容实现普遍支持。

| 步骤 | API | 说明 |
|------|-----|------|
| 1. 初始化 | `CreateMultipartUpload` | 返回 `UploadId` |
| 2. 上传分片 | `UploadPart` | 每片独立上传，返回分片 `ETag` |
| 3. 完成 | `CompleteMultipartUpload` | 提交 `{ PartNumber, ETag }[]`，服务端合并 |
| 取消 | `AbortMultipartUpload` | 释放未完成分片，避免堆积计费 |

#### 分片约束（AWS S3 规范，兼容实现大体遵循）

| 规则 | 值 |
|------|-----|
| 单对象最终大小 | 最大 **5 TiB** |
| 分片数量 | 最多 **10,000** |
| 单分片大小 | **5 MiB – 5 GiB**（最后一片可 &lt; 5 MiB） |
| 推荐分片 | **8–16 MiB**（吞吐与请求次数折中）；高延迟链路可到 32–64 MiB |

#### 流程示意

```http
# 1) 初始化
POST /{KEY}?uploads
→ UploadId = EXAMPLEUPLOADID

# 2) 上传第 N 片（PartNumber 从 1 开始）
PUT /{KEY}?partNumber=1&uploadId=EXAMPLEUPLOADID
→ ETag = "etag-part-1"

# 3) 完成
POST /{KEY}?uploadId=EXAMPLEUPLOADID
Content-Type: application/xml

<CompleteMultipartUpload>
  <Part><PartNumber>1</PartNumber><ETag>"etag-part-1"</ETag></Part>
  <Part><PartNumber>2</PartNumber><ETag>"etag-part-2"</ETag></Part>
</CompleteMultipartUpload>
```

#### SDK 注意

- AWS SDK 高层 `Upload`（`@aws-sdk/lib-storage`）可自动在阈值以上切 Multipart。  
- 分片可并发；完成时 **PartNumber 必须升序提交**。  
- 失败或取消必须 `AbortMultipartUpload`，并可用生命周期规则清理过期未完成上传。

---

### 2.4 服务端中转 vs 客户端直传

| 模式 | 优点 | 缺点 | 建议 |
|------|------|------|------|
| 服务端 `PutObject` | 实现简单、易鉴权 | 占 API 带宽与内存 | &lt; 数 MB 的缩略图/结果图 |
| Presigned PUT | 不经业务服务器 | 需处理 CORS、过期、Content-Type 一致性 | 浏览器 / 客户端上传 |
| Multipart（服务端或客户端） | 大文件可靠 | 实现与状态管理复杂 | ≥ 阈值（建议 8–16 MiB） |

---

## 3. 删除

### 3.1 单对象删除 · DeleteObject

| 项 | 值 |
|----|-----|
| 方法 | `DELETE` |
| 路径 | `/{KEY}`（相对桶） |
| 认证 | SigV4；需 `s3:DeleteObject` |

```http
DELETE /playground/user_1/abc.png HTTP/1.1
Host: playground-assets.s3.ap-northeast-1.amazonaws.com
Authorization: AWS4-HMAC-SHA256 Credential=...
```

| 响应 | 说明 |
|------|------|
| `204 No Content` | 成功（**幂等**：对象不存在也常返回成功） |
| `VersionId` / `DeleteMarker` | 开启版本控制时的版本语义 |

```typescript
import { DeleteObjectCommand } from '@aws-sdk/client-s3'

await client.send(
  new DeleteObjectCommand({
    Bucket: 'playground-assets',
    Key: 'playground/user_1/abc.png',
  }),
)
```

### 3.2 批量删除 · DeleteObjects

单次最多删除 **1000** 个 key（XML body）。

```http
POST /?delete
Content-Type: application/xml

<Delete>
  <Quiet>true</Quiet>
  <Object><Key>playground/a.png</Key></Object>
  <Object><Key>playground/b.png</Key></Object>
</Delete>
```

响应含已删与错误项；`Quiet=true` 时仅返回错误。

### 3.3 「删除目录」

S3 **无目录实体**。删除「前缀」需：

1. `ListObjectsV2` + `Prefix` 分页列出所有 key；  
2. 分批 `DeleteObjects`（每批 ≤ 1000）。

### 3.4 接入约定（建议）

- Provider `delete(key)` → `DeleteObject`；批量任务用 `DeleteObjects`。  
- 业务库删除与对象删除顺序：先删对象再删库，或先标记软删再异步清对象，避免脏链。  
- 版本桶：明确是否删「当前版本」还是所有版本（`DeleteObject` vs 列出版本再删）。

---

## 4. 展示（列出 / 访问）

### 4.1 列表 · ListObjectsV2

| 项 | 值 |
|----|-----|
| 方法 | `GET` |
| Query | `list-type=2` |
| 认证 | `s3:ListBucket` |

#### 常用参数

| 参数 | 说明 |
|------|------|
| `prefix` | 前缀过滤，如 `playground/user_1/` |
| `delimiter` | 常用 `/`，实现「虚拟目录」；返回 `CommonPrefixes` |
| `max-keys` | 单次最多返回数（默认 1000，上限 1000） |
| `continuation-token` | 分页续传令牌 |
| `start-after` | 从某 key 之后开始（可选） |

#### 请求示例

```http
GET /?list-type=2&prefix=playground/user_1/&delimiter=/&max-keys=100 HTTP/1.1
Host: playground-assets.s3.ap-northeast-1.amazonaws.com
Authorization: AWS4-HMAC-SHA256 Credential=...
```

#### 响应要点（XML / SDK 结构化）

| 字段 | 说明 |
|------|------|
| `Contents[].Key` | 对象键 |
| `Contents[].Size` | 字节大小 |
| `Contents[].LastModified` | 最后修改时间 |
| `Contents[].ETag` | ETag |
| `CommonPrefixes[].Prefix` | 子「目录」前缀（使用 delimiter 时） |
| `IsTruncated` / `NextContinuationToken` | 是否还有下一页 |

```typescript
import { ListObjectsV2Command } from '@aws-sdk/client-s3'

const out = await client.send(
  new ListObjectsV2Command({
    Bucket: 'playground-assets',
    Prefix: 'playground/user_1/',
    Delimiter: '/',
    MaxKeys: 100,
  }),
)
```

#### 接入约定（建议）

- UI「文件库」：`Prefix = keyPrefix + userId + '/'`，`Delimiter = '/'`。  
- 分页：循环直至 `IsTruncated === false`。  
- S3 列表**不支持**服务端按 MIME 过滤；需靠 key 后缀约定或本地元数据库。

---

### 4.2 元数据探测 · HeadObject

| 项 | 值 |
|----|-----|
| 方法 | `HEAD` |
| 用途 | 判断是否存在、读 `Content-Type` / `Content-Length` / 用户元数据，无需下载 Body |

不存在时通常 `404`。适合上传后确认、展示前校验。

---

### 4.3 下载 · GetObject

| 项 | 值 |
|----|-----|
| 方法 | `GET` |
| 用途 | 服务端读取对象；支持 `Range` 字节范围 |

展示给终端用户时，更常见的是 **不经业务服务器**，直接给出可访问 URL（见下节）。

---

### 4.4 公开展示 URL

| 模式 | URL 形态 | 适用 |
|------|----------|------|
| 虚拟托管公开读 | `https://{BUCKET}.s3.{REGION}.amazonaws.com/{KEY}` | 桶策略允许公共 `GetObject` |
| Path-style | `{ENDPOINT}/{BUCKET}/{KEY}` | MinIO 等 |
| 自定义域名 / CDN | `{publicBaseUrl}/{KEY}` | **生产推荐**（CloudFront、R2 自定义域等） |
| 私有桶 | Presigned GET（短时） | 鉴权素材、付费内容 |

```typescript
import { GetObjectCommand } from '@aws-sdk/client-s3'
import { getSignedUrl } from '@aws-sdk/s3-request-presigner'

const viewUrl = await getSignedUrl(
  client,
  new GetObjectCommand({ Bucket: 'playground-assets', Key: key }),
  { expiresIn: 3600 },
)
```

#### CORS（浏览器直传 / 直读时必配）

桶 CORS 需允许业务前端 Origin，以及实际上传用的方法（`PUT`/`POST`/`GET`/`HEAD`）和必要 Header（如 `Content-Type`、`Authorization`、`ETag` 暴露）。

---

## 5. 大文件上传支持程度评估

### 5.1 结论一览

| 能力 | 支持程度 | 说明 |
|------|----------|------|
| 小文件 `PutObject` | ✅ 完整 | 实现最简单；单对象至 5 GiB（不建议撑满） |
| Multipart Upload | ✅ 完整（协议级） | 至 5 TiB、最多 1 万片；生产大文件标准路径 |
| Presigned 直传 | ✅ 完整 | 浏览器/客户端绕过业务机；大文件可结合客户端 Multipart + 预签名分片 |
| 断点续传 | ✅ 可实现 | 同一 `UploadId` 下列出已传分片（`ListParts`）后续传；需持久化 `UploadId` 与已完成 Part |
| 服务端自动 Upload | ✅ SDK 支持 | `@aws-sdk/lib-storage` 等按阈值自动分片 |
| 目录级语义 / 服务端 MIME 搜索 | ❌ / ⚠️ | 需前缀约定或外置索引 DB |
| 与「非 S3 图床 API」混用 | — | 本渠道直接走对象存储，不经 ImgBed 网关 |

### 5.2 与各兼容后端的差异提示

| 后端 | 大文件相关注意 |
|------|----------------|
| **AWS S3** | 规范最完整；注意区域、存储类、请求费用 |
| **Cloudflare R2** | S3 兼容；无 egress 到 CF 生态的卖点；`region=auto`；ACL 弱化，靠公开 URL / 签名 |
| **MinIO** | 常需 `forcePathStyle=true`；自托管容量即上限 |
| **阿里云 OSS / 腾讯云 COS** | 开 S3 兼容端点；个别 Header/ACL 行为有差异，联调验证 |
| **Backblaze B2** | S3 兼容网关；注意签名与区域端点 |

### 5.3 对本项目 Storage Provider 的实现建议

1. **默认路径**：&lt; 阈值 → `PutObject`（或小文件 Presigned PUT）；≥ 阈值 → Multipart（服务端 Upload 工具类或客户端分片）。  
2. **阈值建议**：`8 MiB` 或 `16 MiB`（可配置 `largeFileThreshold` / `partSize`）。  
3. **Key 与展示**：库表存 `bucket`、`key`、`contentType`、`size`、`etag`；对外 URL 用 `publicBaseUrl` 或临时签名。  
4. **删除 / 列表**：`DeleteObject` / `DeleteObjects` + `ListObjectsV2`；「按用户展示」严格依赖 key 前缀约定。  
5. **配置暴露**：写入 `file_upload_channel`——表列 `chunk_threshold`/`chunk_size`/`max_size`；差异项进 `config_proflle`（`endpoint`、`region`、`bucket`、`access_key_id`、`secret_access_key`、`force_path_style`、`public_base_url`、`key_prefix`）。  
6. **运维**：配置未完成 Multipart 的生命周期清理（如 7 天中止），防止碎片占容量。

### 5.4 与 Cloudflare ImgBed 渠道对比（选型参考）

| 维度 | S3 Provider | Cloudflare ImgBed |
|------|-------------|-------------------|
| 协议 | 原生对象存储 | HTTP 图床网关 |
| 大文件 | Multipart 至 TiB 级 | 受 CF 请求体与底层渠道限制；需跟其分块/HF 直传 |
| 列表过滤 | 前缀为主 | 标签/渠道/类型等更丰富 |
| 可移植性 | 换 R2/MinIO/OSS 成本低 | 绑定 ImgBed 实例 API |
| 实现成本 | SDK 成熟，分片需状态 | 上传 API 简单，分块协议需单独适配 |

---

## 6. Provider 接口映射（便于后续编码）

| 本项目能力 | S3 API | 备注 |
|------------|--------|------|
| `put` / `upload` | `PutObject` 或 Multipart / Presigned | 按体积分支 |
| `delete` | `DeleteObject` | 批量用 `DeleteObjects` |
| `list` | `ListObjectsV2` | `Prefix` + `Delimiter` + 分页 |
| `head` / `exists` | `HeadObject` | 404 → 不存在 |
| `get` | `GetObject` | 服务端读流 |
| `getUrl` | `publicBaseUrl/key` 或 Presigned GET | 私有桶必须签名 |
| `abortUpload` | `AbortMultipartUpload` | 大文件取消 |

本渠道在库表中的类型码值：`file_upload_channel.type` / `file.channel_type` = **`3`**（S3存储）。管理员配置写入 `file_upload_channel.config_proflle`；分片阈值用表列 `chunk_threshold` / `chunk_size` / `max_size`。物理路径落 `file.object_key`，MD5 落 `file.identifier`。

---

## 7. 最小联调清单

- [ ] 凭证可对目标桶执行 Put / Get / Delete / List  
- [ ] `PutObject` 上传小图，用公开 URL 或 Presigned GET 可访问  
- [ ] `ListObjectsV2` + 前缀能列出该对象  
- [ ] `DeleteObject` 后 Head/Get 为 404（或版本桶语义符合预期）  
- [ ] （可选）≥ 阈值文件 Multipart 上传成功，中断后 Abort 无残留  
- [ ] （可选）浏览器 Presigned PUT + CORS 打通  
- [ ] （可选）`forcePathStyle` / 自定义 `endpoint` 在 R2 或 MinIO 验证通过  

---

## 8. 参考链接

- [Amazon S3 API Reference](https://docs.aws.amazon.com/AmazonS3/latest/API/Welcome.html)  
- [PutObject](https://docs.aws.amazon.com/AmazonS3/latest/API/API_PutObject.html)  
- [DeleteObject](https://docs.aws.amazon.com/AmazonS3/latest/API/API_DeleteObject.html) / [DeleteObjects](https://docs.aws.amazon.com/AmazonS3/latest/API/API_DeleteObjects.html)  
- [ListObjectsV2](https://docs.aws.amazon.com/AmazonS3/latest/API/API_ListObjectsV2.html)  
- [Multipart upload overview](https://docs.aws.amazon.com/AmazonS3/latest/userguide/mpuoverview.html)  
- [CreateMultipartUpload](https://docs.aws.amazon.com/AmazonS3/latest/API/API_CreateMultipartUpload.html) · [UploadPart](https://docs.aws.amazon.com/AmazonS3/latest/API/API_UploadPart.html) · [CompleteMultipartUpload](https://docs.aws.amazon.com/AmazonS3/latest/API/API_CompleteMultipartUpload.html)  
- [Presigned URLs](https://docs.aws.amazon.com/AmazonS3/latest/userguide/PresignedUrlUploadObject.html)  
- [AWS SDK for JavaScript v3 · S3 Client](https://docs.aws.amazon.com/AWSJavaScriptSDK/v3/latest/client/s3/)  
- [Cloudflare R2 · S3 API compatibility](https://developers.cloudflare.com/r2/api/s3/api/)
