# 文件上传渠道 · 管理端详设（manager）

> 文档定位：超管后台「数据管理」的 UI / API / 权限 / 探测。  
> 配套：[use.md](./use.md)（后续文件上传需开放的接口）· [file_upload_channel.sql](../../db-design/file_upload_channel.sql) · [FIleStorage.md](../../FIleStorage.md)  
> 本目录两份文档合计为 **FileUploadChannel 第一份详设**，不另建总览文件。

---

## 0. 已拍板结论

| # | 结论 |
|---|------|
| 权限 | 与运维其它页一致：**仅超管**（`RootAuth` / `ROLE.SUPER_ADMIN`） |
| 菜单 | 系统设置 → 运维，插在 **日志维护** 与 **性能** 之间，名「数据管理」 |
| UI | **`web/default` + `web/classic` 均做** |
| `type=0` 本地存储 | **本期不开放配置**（码值预留，UI 不出现或禁用） |
| 删除 | **有引用禁止删除**，仅可停用（`status=0`） |
| 探测 | **已保存渠道** + **未保存表单**均可测；通过标准见 §6 |
| 字段/码值 | 以 `db-design/file_upload_channel.sql` 为准（含 `config_proflle` 拼写） |

---

## 1. 目标与范围

### 1.1 目标

1. 超管可对文件上传渠道做增删改查、启停、设默认。  
2. 切换 `type` 时表单字段自动切换（1 Webdav / 2 ImgBed / 3 S3）。  
3. 风格、i18n、多主题与现有系统设置页一致。  
4. 可检测渠道实际连通性（不依赖先保存）。  

### 1.2 非目标（本期）

- 用户侧上传 / 秒传 / `file`·`user_file` 管理 UI（见后续详设 + [use.md](./use.md) 接口预留）  
- `type=0` 本地存储配置与探测 UI  
- Classic/Default 之外的第三套 UI  

---

## 2. 信息架构与菜单位置

### 2.1 Default（`web/default`）

| 项 | 值 |
|----|-----|
| 分组 | 系统管理 → **运维**（`Operations`） |
| section `id` | `data-management` |
| i18n titleKey | `Data Management`（zh：`数据管理`） |
| 路由 | `/system-settings/operations/data-management` |
| 插入顺序 | `logs` → **`data-management`** → `performance` → `update-checker` |

改动点：

- `features/system-settings/operations/section-registry.tsx`：在 `logs` 与 `performance` 之间注册 section  
- 新建 section 组件，例如 `features/system-settings/maintenance/data-management-section.tsx`（或 `storage/` 子目录）  
- `i18n/locales/zh.json`（及 en 等）：补齐文案；key 用英文句子，与现网一致  

系统设置门禁保持现有：`ROLE.SUPER_ADMIN`（`system-settings/route.tsx`）。

### 2.2 Classic（`web/classic`）

Classic 无独立「运维」树，日志在「运营设置」、性能为顶层 Tab。本期对齐方式：

| 项 | 值 |
|----|-----|
| 入口 | Setting 页新增顶层 Tab：**数据管理**（`itemKey: 'data-management'`） |
| 建议顺序 | 插在 **运营设置（operation）** 与 **性能设置（performance）** 之间（最接近「日志维护与性能中间」的语义） |
| 组件 | 如 `pages/Setting/Storage/SettingsDataManagement.jsx` |
| 权限 | 与系统设置页一致，仅根用户/超管可进（沿用 Setting 页现有门禁） |

若 Classic 门禁与 Default 不完全一致，以实现时 Setting 页已有校验为准，**不得**对普通 Admin 开放渠道密钥配置。

---

## 3. 页面交互（管理端）

### 3.1 布局（对齐现有 Settings 风格）

推荐结构（Default）：

```text
SettingsSection「Data Management」
  ├─ 说明文案（muted）：配置文件上传渠道；密钥仅服务端保存
  ├─ 工具条：【新建渠道】
  └─ 表格/卡片列表
        列：名称 | 类型 | 状态 | 默认 | 分片阈值 | 更新时间 | 操作
        操作：编辑 | 设为默认 | 检测 | 启用/停用 | 删除
```

Classic：Semi Design / 现有 Setting 卡片 + Table，字段与 Default 同一套 API，不另造数据模型。

### 3.2 新建 / 编辑抽屉或弹窗

**公共字段**（所有开放 type）：

| 字段 | 控件 | 校验 |
|------|------|------|
| `name` | Input | 必填，≤255 |
| `type` | Select：`1`/`2`/`3`（本期不出现 `0`） | 必填；**编辑时建议禁止改 type**（避免 config 形状错乱）；若允许改 type，切换后清空 `config_proflle` 并换表单 |
| `status` | Switch → `"1"`/`"0"` | 默认启用 |
| `is_default` | Switch → `"1"`/`"0"` | 勾选保存时服务端取消其它默认 |
| `chunk_threshold` | InputNumber（字节或 MiB 展示、存字节） | ≥0；默认 16777216 |
| `chunk_size` | InputNumber | ≥0；S3 建议 ≥5MiB；默认 8388608 |
| `max_size` | InputNumber | ≥0；`0`=不限制 |

**按 `type` 切换的 `config_proflle` 子表单**：

#### type=`1` Webdav

| JSON key | 控件 | 必填 |
|----------|------|------|
| `base_url` | Input | 是 |
| `username` | Input | 是* |
| `password` | Password（编辑时空=不修改） | 新建必填；编辑可选 |
| `auth_type` | Select：`basic` / `bearer` | 默认 `basic` |
| `path_prefix` | Input | 否 |
| `public_base_url` | Input | 否 |
| `timeout_ms` | InputNumber | 否，默认 600000 |

#### type=`2` Cloudflare-ImageBed

| JSON key | 控件 | 必填 |
|----------|------|------|
| `base_url` | Input | 是 |
| `api_token` | Password（编辑时空=不修改） | 新建必填；编辑可选 |
| `upload_channel` | Input/Select | 否，默认如 `cfr2` |
| `upload_folder` | Input | 否 |
| `return_format` | Select：`default` / `full` | 否 |

#### type=`3` S3

| JSON key | 控件 | 必填 |
|----------|------|------|
| `endpoint` | Input | 视厂商，R2/MinIO 建议必填 |
| `region` | Input | 是（R2 常用 `auto`） |
| `bucket` | Input | 是 |
| `access_key_id` | Input | 是 |
| `secret_access_key` | Password（编辑时空=不修改） | 新建必填；编辑可选 |
| `force_path_style` | Switch | 否 |
| `public_base_url` | Input | 否 |
| `key_prefix` | Input | 否 |

前端：`type` 变更 → 卸载旧子表单、挂载新 schema（zod/手动校验），**不把上一类型的密钥字段残留进提交 JSON**。

### 3.3 列表展示脱敏

读接口对敏感字段脱敏，例如：

- `password` / `secret_access_key` / `api_token` → 不回明文；可用 `***` 或 `has_password: true` + 掩码前缀  
- 前端编辑态：密码框 placeholder「不修改请留空」

### 3.4 设为默认

- 列表「设为默认」→ `PUT .../default` 或更新接口带 `is_default=1`  
- 服务端事务：目标行 `is_default=1`，其它行 `is_default=0`  
- 仅 `status=1` 的渠道允许设为默认（停用渠道设默认应拒绝或自动启用——**推荐拒绝并提示先启用**）

### 3.5 删除 / 停用

| 操作 | 规则 |
|------|------|
| 停用 | `status=0`；若当前为默认，须先转移默认或同时指定新默认（**推荐：禁止停用唯一默认，提示先改默认**） |
| 删除 | 若存在 `file.file_channel_id = id` **或** `user_file.file_channel_id = id` → **403/业务错误，禁止删除**；无引用才物理删行 |

### 3.6 可用性检测（UI）

| 入口 | 行为 |
|------|------|
| 列表「检测」 | `POST /api/file-upload-channel/:id/probe`（已保存） |
| 表单「检测连接」 | `POST /api/file-upload-channel/probe`，body 为当前表单（含 type + config；密码留空时对已保存 id 可带 `id` 让服务端用库中密钥） |

结果展示：成功 Toast + 可选延迟/桶名；失败展示 `message`（勿回显完整密钥）。

### 3.7 i18n 与主题

- 所有用户可见文案走 `t('...')`，英文 key；中文写入 `zh.json`（classic 用其现有 i18n 管线）。  
- 颜色/边框/背景只用语义 token（如 `text-muted-foreground`、`border-border`、Semi `tertiary`），避免写死亮色 hex，以适配亮/暗主题。  
- 类型展示名 i18n 示例：`WebDAV` / `Cloudflare ImageBed` / `S3 Compatible Storage`（码值仍存 `1/2/3`）。

---

## 4. 后端 API（管理端 · RootAuth）

统一信封：`{ "success": true|false, "message": "", "data": ... }`（HTTP 200 为主，与 new-api 一致）。  
路由组建议：

```text
/api/file-upload-channel
  middleware: RootAuth()
```

审计：走现有 Root/Admin 写操作审计链路（`RootAuth` 已含审计兜底则无需另挂）。

### 4.1 接口一览

| 方法 | 路径 | 说明 |
|------|------|------|
| `GET` | `/api/file-upload-channel/` | 列表（脱敏）；可选 query：`type`、`status` |
| `GET` | `/api/file-upload-channel/:id` | 详情（脱敏） |
| `POST` | `/api/file-upload-channel/` | 创建 |
| `PUT` | `/api/file-upload-channel/:id` | 更新（空密码字段=保持原密钥） |
| `PUT` | `/api/file-upload-channel/:id/default` | 设为默认 |
| `PUT` | `/api/file-upload-channel/:id/status` | 启停：`{ "status": "0"|"1" }` |
| `DELETE` | `/api/file-upload-channel/:id` | 删除（有引用则失败） |
| `POST` | `/api/file-upload-channel/:id/probe` | 探测已保存渠道 |
| `POST` | `/api/file-upload-channel/probe` | 探测未保存/表单配置 |

> 路径用 `file-upload-channel` 与表名语义对齐；实现时勿与 AI `channels` 路由混淆。

### 4.2 创建 Body 示例（type=3）

```json
{
  "name": "R2 Primary",
  "type": "3",
  "status": "1",
  "is_default": "1",
  "chunk_threshold": 16777216,
  "chunk_size": 8388608,
  "max_size": 0,
  "config_proflle": {
    "endpoint": "https://xxx.r2.cloudflarestorage.com",
    "region": "auto",
    "bucket": "playground",
    "access_key_id": "...",
    "secret_access_key": "...",
    "force_path_style": true,
    "public_base_url": "https://cdn.example.com",
    "key_prefix": "files/"
  }
}
```

服务端：

- `create_user_id` / `update_user_id` ← `c.GetInt("id")`  
- `config_proflle` 存 **text JSON 字符串**  
- `type` 校验 ∈ {`1`,`2`,`3`}（本期拒绝 `0`）  
- `is_default=1` 时取消其它默认  

### 4.3 更新密钥约定

- 请求中敏感字段为 `""` / `null` / 省略 → **不覆盖**库中原值。  
- 非空 → 更新。  
- 响应永不返回明文密钥。

### 4.4 删除引用检查

```sql
SELECT 1 FROM file WHERE file_channel_id = ? LIMIT 1
-- 或
SELECT 1 FROM user_file WHERE file_channel_id = ? LIMIT 1
```

任一命中 → `success=false`，message 如：`Channel is in use and cannot be deleted`。

### 4.5 Model / 落点建议

| 层 | 路径建议 |
|----|----------|
| Model | `model/file_upload_channel.go`，`TableName() = file_upload_channel`，挂 `AutoMigrate` |
| Controller | `controller/file_upload_channel.go` |
| Router | `router/api-router.go` 注册 RootAuth 组 |
| Provider probe | `service/storage/provider/...` 的 `Ping/Probe` |

---

## 5. 数据与默认值

与 SQL 一致，管理端写入时默认：

| 列 | 默认 |
|----|------|
| `status` | `"1"` |
| `is_default` | `"0"`（除非创建时显式设默认） |
| `chunk_threshold` | `16777216` |
| `chunk_size` | `8388608` |
| `max_size` | `0` |

全局允许 0 条默认（未配置完时）；**上传链路**无默认时由 [use.md](./use.md) 的 prepare 返回明确错误（管理端列表可黄条提示「未设置默认渠道」）。

---

## 6. 可用性探测标准

统一响应 `data`：

```json
{
  "ok": true,
  "type": "3",
  "latency_ms": 128,
  "detail": "HeadBucket ok"
}
```

失败：`ok=false`，`message`/`detail` 说明原因（无密钥）。

| type | 探测动作 | 通过条件 |
|------|----------|----------|
| `3` S3 | `HeadBucket` 或 `ListObjectsV2` MaxKeys=1 | 无鉴权/网络错误 |
| `2` ImgBed | `GET {base_url}/api/manage/list?count=1`（或等价只读），Bearer Token | HTTP 成功且非 Token 无效 |
| `1` WebDAV | `PROPFIND` Depth:0 对 `base_url` | `207`/`200` 等成功态 |
| `0` 本地 | （本期不做 UI；若内部调用：检查 `root_path` 存在且可写） | — |

表单探测：body 与创建同结构；若 `id>0` 且密码留空，服务端合并库中密钥再测。

超时：建议 10–15s；WebDAV 可用 `config_proflle.timeout_ms` 封顶到探测上限。

---

## 7. 其它建议开放的管理能力（本期纳入）

| 能力 | 说明 |
|------|------|
| 列表按 type/status 筛选 | 减少渠道变多后的噪音 |
| 「复制为新渠道」 | 可选；复制非密钥字段，密钥需重填（降低配错成本） |
| 默认渠道提示条 | 无默认或默认已停用时 Alert |
| 引用计数展示（可选） | 详情显示关联 `file`/`user_file` 数量，解释为何不能删 |
| 探测结果不入库 | 仅实时返回；不做历史探活表（避免范围膨胀） |

本期**不做**：渠道维度流量统计、自动故障转移多默认、密钥托管到外部 KMS。

---

## 8. 前端实现清单

### 8.1 Default

- [ ] `section-registry` 注册 `data-management`（插在 logs 与 performance 之间）  
- [ ] `DataManagementSection`：列表 + Dialog/Sheet 表单 + type 切换子表单  
- [ ] API client：`features/system-settings/...` 或 `features/storage/`  
- [ ] i18n：`Data Management` 及表单/错误文案  
- [ ] 脱敏与空密不覆盖  

### 8.2 Classic

- [ ] `Setting/index.jsx` 增加 Tab `data-management`  
- [ ] 同等功能页，复用同一套 `/api/file-upload-channel*`  
- [ ] i18n 与 Semi 主题变量  

### 8.3 后端

- [ ] Model + AutoMigrate  
- [ ] RootAuth CRUD + default + status + probe×2  
- [ ] 删除引用检查；默认互斥事务  
- [ ] Provider Probe 实现（1/2/3）  

---

## 9. 验收清单（管理端）

- [ ] 超管可见「数据管理」；非超管不可访问 API 与菜单  
- [ ] 菜单位于日志维护与性能之间（Default）；Classic Tab 位置符合 §2.2  
- [ ] 切换 type=1/2/3 表单字段正确切换，无串配置  
- [ ] 可设唯一默认；停用/删除规则符合 §3.5  
- [ ] 已保存探测 + 表单探测均可用；密钥不回显  
- [ ] 亮/暗主题与现有设置页观感一致；中英文切换正常  
- [ ] `type=0` 无法通过 UI/API（管理创建）写入（API 显式拒绝）  

---

## 10. 关联文档

- [use.md](./use.md) — 上传侧需提前开放的接口契约  
- [../../db-design/file_upload_channel.sql](../../db-design/file_upload_channel.sql)  
- [../../providers/S3-Provider.md](../../providers/S3-Provider.md)（type=`3`）  
- [../../providers/Cloudflare-ImageBed-Provider.md](../../providers/Cloudflare-ImageBed-Provider.md)（type=`2`）  
- [../../providers/WebDAV-Provider.md](../../providers/WebDAV-Provider.md)（type=`1`）
