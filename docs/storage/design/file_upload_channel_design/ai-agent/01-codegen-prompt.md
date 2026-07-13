# FileUploadChannel · 代码生成提示词（给实现 Agent）

> 将本文**整段复制**给实现 Agent 作为任务说明。  
> 权威详设（必须先读再写）：  
> - `new-api/docs/storage/design/file_upload_channel_design/manager.md`  
> - `new-api/docs/storage/design/file_upload_channel_design/use.md`（仅 P0：`GET /default`）  
> - `new-api/docs/storage/db-design/file_upload_channel.sql`  
> Provider 细节（探测实现时对照）：`new-api/docs/storage/providers/{S3,Cloudflare-ImageBed,WebDAV}-Provider.md`

---

## 角色与目标

你是 new-api 仓库的实现工程师。请在仓库根目录 `new-api/` 内实现 **文件上传渠道管理（File Upload Channel）P0**，使超管可在后台配置 WebDAV / Cloudflare ImgBed / S3 渠道，支持默认渠道与连通性探测，并开放上传侧只读默认渠道接口。

**本迭代不做**：用户上传 prepare/complete、秒传、`file`/`user_file` 业务 UI、本地存储 `type=0` 的配置与探测 UI。

---

## 硬性约束（违反即不合格）

1. **字段名与码值以 SQL 为准**，包括拼写 `config_proflle`（不要「纠正」成 profile）。  
2. `type`：`0` 本地（API 创建拒绝）/ `1` Webdav / `2` Cloudflare-ImageBed / `3` S3。  
3. 管理 API：**仅 `middleware.RootAuth()`**（超管）。禁止用普通 `AdminAuth` 暴露密钥配置。  
4. 响应信封：`{ "success", "message", "data" }`，与现有 `common.ApiSuccess` / `ApiError` 一致。  
5. 密钥：列表/详情**脱敏**；更新时敏感字段空字符串/省略 = **不覆盖**原值；响应永不回明文。  
6. 删除：若 `file.file_channel_id` 或 `user_file.file_channel_id` 有引用 → **禁止删除**（表可能尚未建，用「表存在则查，不存在则跳过」或先确保 model 可查；若 `file`/`user_file` 未迁移，删除仅校验渠道表即可，但代码路径要预留引用检查）。  
7. 默认渠道：全局同时仅一条 `is_default='1'`（事务更新）。停用当前默认前须已有其它默认或拒绝。  
8. UI：`web/default` **与** `web/classic` 都要做；i18n 英文 key + zh；主题用语义色，勿写死亮色 hex。  
9. 少抽象、对齐现有代码风格（GORM model、controller、router、settings section）。不要引入无关依赖。  
10. 不要改 git config；不要提交密钥样例真值。

---

## 必读现有模式（对齐后再写）

| 用途 | 参考路径 |
|------|----------|
| 运维 section 注册 | `web/default/src/features/system-settings/operations/section-registry.tsx` |
| 设置页样式 | `.../maintenance/performance-section.tsx`、`log-settings-section.tsx` |
| Root 路由 | `router/api-router.go` 中 `/option`、`/performance` 的 `RootAuth` |
| 鉴权 | `middleware/auth.go` → `RootAuth` |
| AutoMigrate | `model/main.go` |
| Classic Setting Tab | `web/classic/src/pages/Setting/index.jsx` |
| API 客户端 | default 的 system-settings `api.ts`；classic 的 `API` 封装 |

---

## 后端交付清单

### 1. Model

- 文件：`model/file_upload_channel.go`  
- `TableName()` → `file_upload_channel`  
- 字段与 SQL 一一对应：`Id`, `Name`, `Type`, `Status`, `IsDefault`, `ChunkThreshold`, `ChunkSize`, `MaxSize`, `ConfigProflle`（json tag / column：`config_proflle`）, `CreateUserId`, `UpdateUserId`, `CreateTime`, `UpdateTime`  
- 时间：可用 `time.Time` 映射 timestamp，或项目习惯的方式；与 AutoMigrate 兼容即可  
- 注册到 `migrateDB` / `migrateDBFast`

### 2. Router（RootAuth）

前缀：`/api/file-upload-channel`

| Method | Path | 行为 |
|--------|------|------|
| GET | `/` | 列表，脱敏；query 可选 `type`,`status` |
| GET | `/:id` | 详情，脱敏 |
| POST | `/` | 创建；`type`∈{1,2,3}；`is_default=1` 时清其它默认 |
| PUT | `/:id` | 更新；空密钥不覆盖 |
| PUT | `/:id/default` | 设默认（渠道须 status=1） |
| PUT | `/:id/status` | body `{status:"0"|"1"}` |
| DELETE | `/:id` | 无引用才删 |
| POST | `/:id/probe` | 探测已保存 |
| POST | `/probe` | 探测表单/未保存配置（注意与 `/:id` 路由注册顺序，避免 `probe` 被当成 id） |

另（use.md P0，**UserAuth**）：

| Method | Path | 行为 |
|--------|------|------|
| GET | `/api/file-upload-channel/default` | 当前全局默认摘要，**无** `config_proflle`；无则 `data=null` |

可选：`GET /api/file-upload-channel/resolve?channel_type=3`（UserAuth）。

### 3. Probe 标准

| type | 动作 |
|------|------|
| 3 | S3 HeadBucket 或 ListObjectsV2 MaxKeys=1 |
| 2 | GET `{base_url}/api/manage/list?count=1` + Bearer token |
| 1 | PROPFIND Depth:0 对 base_url |
| 0 | 管理 API 拒绝；probe 若碰到返回不支持 |

返回：`{ ok, type, latency_ms, detail }`。超时约 10–15s。表单 probe：密码为空且带 `id` 时合并库中密钥。

### 4. Controller / Service

- `controller/file_upload_channel.go`  
- probe 逻辑可放 `service/storage/` 或同包私有函数；按 type 分支，可读 Provider 文档  
- 创建/更新校验 `config_proflle` JSON 必要 key（按 type）

---

## 前端交付清单

### Default（`web/default`）

1. 在 `section-registry.tsx` 于 **`logs` 与 `performance` 之间**插入：  
   - `id: 'data-management'`  
   - `titleKey: 'Data Management'`  
2. 实现 `DataManagementSection`（列表 + 新建/编辑 Dialog + type 切换子表单 + 探测 + 设默认 + 启停 + 删除）。  
3. type 选项仅 `1/2/3`；切换 type 清空旧 config 字段，避免串配置。  
4. i18n：`zh.json` 等增加「数据管理」及表单文案（英文 key）。  
5. 调用上述 Root API；注意 cookie/session + `New-Api-User` 头与现有设置页一致。

### Classic（`web/classic`）

1. `Setting/index.jsx` 增加 Tab `data-management`，插在 **operation 与 performance 之间**。  
2. 页面功能与 Default 等价，复用同一后端 API。  
3. 使用现有 Semi + i18n 模式。

---

## 实现顺序（建议）

1. SQL/Model + AutoMigrate  
2. Root CRUD + default + status + 脱敏  
3. Probe（可先 mock/skip 外网，但接口要通；有网络则真实探测）  
4. `GET /default` UserAuth  
5. Default UI  
6. Classic UI  
7. 自测：见同目录 `02-test-prompt.md` / `03-checklist.md`

---

## 完成定义（DoD）

- [ ] 超管可完成渠道 CRUD、设默认、启停、两种探测  
- [ ] 非超管调管理 API 失败  
- [ ] 用户可调 `GET .../default` 且无密钥字段  
- [ ] Default 菜单位于日志维护与性能之间，中文「数据管理」  
- [ ] Classic 有对应 Tab  
- [ ] `type=0` 无法创建  
- [ ] 有引用时删除失败（若相关表已存在）  
- [ ] 不提交真实密钥；代码可编译；相关 go test / 前端类型检查尽量通过  

完成后请输出：变更文件列表、如何本地启动验证、已知限制（如探测需外网）。
