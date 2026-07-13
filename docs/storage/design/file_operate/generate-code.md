# File Operate · 代码生成提示词（generate-code.md）

> 将本文**整段复制**给实现 Agent。  
> **权威详设（必须先读再写）**：  
> - `new-api/docs/storage/design/file_operate/user.md`  
> - `new-api/docs/storage/design/file_operate/admin.md`  
> - `new-api/docs/storage/db-design/file.sql`  
> - `new-api/docs/storage/db-design/user_file.sql`  
> - `new-api/docs/storage/db-design/file_upload_channel.sql`  
> 依赖渠道能力时对照：`docs/storage/design/file_upload_channel_design/` 与 `docs/storage/providers/`  
> 验收对照：同目录 `check_list.md`

---

## 角色与目标

你是 new-api（仓库目录 `new-api/`）实现工程师。请实现 **文件操作 File Operate** 的后端 API（**不做 UI**）：

1. **用户侧**：浏览器直传上传（含 **分片 multipart**）、秒传、查（全量/按后缀）、删自己的逻辑文件。  
2. **管理侧**：`AdminAuth` 全站查（含按 `channel_type`、后缀）、渠道用量统计、删任意用户逻辑文件。

表：`file`（物理）+ `user_file`（逻辑）+ 已有 `file_upload_channel`（读配置/密钥，仅服务端）。

---

## 硬性约束（违反即不合格）

1. **字段名/码值以 SQL 为准**：`identifier`、`file_name`、`file_suffix`、`file_channel_id`、`config_proflle`（拼写勿改）、`channel_type` 码值 `0/1/2/3`。  
2. 响应信封：`{ "success", "message", "data" }`，用 `common.ApiSuccess` / `ApiError`；分页用 `common.GetPageQuery` → `PageInfo`（`p`/`page_size`，上限 100）。  
3. 用户 id **只**来自 `c.GetInt("id")`，禁止信任 body 里的 user_id。  
4. 用户 API：`UserAuth`；管理文件 API：`AdminAuth`。**禁止**在文件接口中返回 `config_proflle` 明文。  
5. 列表 **默认返回全部 status**；可用 query `status` 过滤（详设拍板）。  
6. 管理列表「按渠道」主筛：**`channel_type`**；统计按 **`file_channel_id`** 聚合，同时返回 `user_file_count` + `file_count`（建议含 `total_size`）。  
7. 删除：只删 `user_file`；`file.ref_count--`；归零后再 Provider 删物理对象。不做无视 ref 的强制物理删。  
8. 上传：浏览器直传，**文件字节不经业务 handler 落盘中转**；`file_size > chunk_threshold` 走 multipart；S3 必须跑通分片；ImgBed 按 Provider 分块；WebDAV 无标准分片则超限拒绝或按详设降级（须在注释/README 写明）。  
9. `type=0` 本地：prepare 拒绝。  
10. 少抽象、对齐现有 GORM/controller/router 风格；不引入无关依赖；不改 git config；不提交真实密钥。  
11. **不做** Default/Classic 管理页或用户文件柜 UI（API only）。

---

## 必读并对齐的现有代码

| 用途 | 参考 |
|------|------|
| 分页 | `common/page_info.go`、`GetUserLogs` 一类列表 |
| 鉴权 | `middleware/auth.go`：`UserAuth` / `AdminAuth` |
| 路由注册 | `router/api-router.go`（注意静态路径先于 `/:id`） |
| AutoMigrate | `model/main.go` |
| Root 渠道（若已实现） | `file_upload_channel` model/controller；上传时读渠道配置 |
| Provider | `docs/storage/providers/S3-Provider.md` 等 |

若渠道 CRUD 尚未合并，上传相关测试可用 fixture 写入 `file_upload_channel` 行；但代码路径必须从该表读配置。

---

## 后端交付清单

### 1. Model + 迁移

- `model/file.go`、`model/user_file.go`（若尚无）  
- `TableName`：`file`、`user_file`  
- 字段与 SQL 一致；挂 `AutoMigrate`  
- 建议索引：`user_file(user_id,file_suffix)`、`file(channel_type)`（详设已写）

### 2. 用户 API（`UserAuth`）— 路径以 `user.md` 为准

| Method | Path | 要点 |
|--------|------|------|
| POST | `/api/storage/prepare` | 秒传 or 签发 plan（single/multipart）+ HMAC 票据 |
| POST | `/api/storage/presign-part` | multipart 续签 part |
| POST | `/api/storage/complete` | 验票、校验、写 file/user_file、核销 |
| POST | `/api/storage/abort` | 取消 multipart/票据 |
| GET | `/api/storage/self` | 分页；`file_suffix`/`status`/`source`/`keyword`/`channel_type` |
| GET | `/api/storage/self/suffixes` | 当前用户后缀聚合 |
| GET | `/api/storage/self/:id` | 详情，必须本人 |
| DELETE | `/api/storage/self/:id` | 删本人逻辑文件 |
| DELETE | `/api/storage/self` | body `{ids:[]}` 批量，最多 100 |

渠道解析顺序、plan JSON、`UserFileView` 字段见 `user.md`（勿另造字段名）。

### 3. 管理 API（`AdminAuth`）— 路径以 `admin.md` 为准

| Method | Path | 要点 |
|--------|------|------|
| GET | `/api/storage/` | 全站列表；筛 `channel_type`、`file_suffix`、`user_id` 等 |
| GET | `/api/storage/user/:user_id` | 指定用户 |
| GET | `/api/storage/suffixes` | 全站后缀聚合 |
| GET | `/api/storage/stats/by-channel` | 按 `file_channel_id`：`user_file_count`+`file_count`(+`total_size`) |
| GET | `/api/storage/stats/summary` | 全局摘要 |
| GET | `/api/storage/:id` | 详情（`user_file.id`） |
| DELETE | `/api/storage/:id` | 删任意用户逻辑文件 |
| DELETE | `/api/storage/` | 批量删 |

路由注册顺序：先 `/self*`、`/stats*`、`/suffixes`、`/user/:id`、`/prepare` 等，再 `/:id`。

### 4. 服务层要点

- `service/storage/`：票据（Redis 优先）、秒传（`identifier+channel_type`）、ref_count、Provider 适配（S3 Put/Multipart/Delete；ImgBed；WebDAV）。  
- 长效密钥只在服务端读 `config_proflle`。  
- complete 失败可 Abort S3 multipart；物理删失败将 `file.status=2` 并打日志（逻辑删仍可成功，见 user.md）。

### 5. 分片（必须适配）

- 比较 `file_size` 与渠道 `chunk_threshold` / `chunk_size` / `max_size`。  
- S3：`CreateMultipartUpload` → part Presign（eager 或 lazy+`presign-part`）→ `CompleteMultipartUpload`。  
- 前端不在本迭代实现；但 API 契约与 plan 字段必须完整，便于后续 SDK。

---

## 建议实现顺序

1. Model + AutoMigrate + 基础 CRUD 查询/删除（用户 self + 管理列表）  
2. 统计接口  
3. prepare 秒传 + S3 single  
4. S3 multipart + presign-part + abort + complete  
5. ImgBed / WebDAV Adapter（按优先级；WebDAV 大文件策略按详设）  
6. 单测 + 对照 `check_list.md` 自检  

---

## 完成定义（DoD）

- [ ] 用户上传：秒传、single、S3 multipart 闭环（可用 minio/mock 或集成测试）  
- [ ] 用户查：分页、后缀、默认全 status  
- [ ] 用户/管理员删：ref_count 与物理删路径正确  
- [ ] 管理：按 `channel_type` 筛；`stats/by-channel` 双计数  
- [ ] 权限矩阵正确；无密钥泄漏  
- [ ] 无 UI 改动要求；`go build` / 相关 `go test` 通过  
- [ ] 输出变更文件列表与本地验证步骤  

完成后请对照 `check_list.md` 勾选并注明证据。
