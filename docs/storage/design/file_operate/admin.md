# 文件操作 · 管理侧详设（admin）

> 定位：管理员对全站逻辑文件的 **查 / 删 / 按渠道类型筛选 / 渠道用量统计**。  
> 配套：[user.md](./user.md)（用户增删查与分片上传）· 表结构 `../../db-design/`  
> **API only**，不做管理前端页。渠道密钥配置仍仅超管（Root），见渠道 manager 详设。

---

## 0. 已拍板结论（本域）

| # | 结论 |
|---|------|
| 鉴权 | **`AdminAuth`**（普通管理员可查删文件；不可读 `config_proflle`） |
| 列表筛选「渠道」 | 按 **`channel_type`**（码值 `0/1/2/3`），不是 `file_channel_id` |
| 统计 | 同时返回 **`user_file_count` + `file_count`**（及建议 `total_size`），按渠道实例聚合 |
| 删 | 可删任意用户的 `user_file`；引用归零再删物理；**不提供**无视 ref 的强制物理删 |
| 列表 status | 默认返回全部 status，可用参数过滤 |
| 分页 | 与用户侧相同的动态分页 |

---

## 1. 目标与非目标

### 1.1 目标

1. 管理员分页查询全站用户文件（全部 / 按后缀 / 按 `channel_type` / 按用户等）。  
2. 按上传渠道实例统计逻辑文件数与物理文件数。  
3. 删除任意用户的逻辑文件（与用户删同一套引用语义）。  
4. 提供详情、后缀聚合、按用户维度查询等运维必要接口。

### 1.2 非目标

- 渠道 CRUD / 探测 / 默认渠（Root，渠道详设）  
- 管理员代上传（增仍走用户 `prepare` 流程；管理员不另开上传口）  
- 管理端 UI  
- 强制删除物理对象（无视 `ref_count`）

---

## 2. 权限与数据范围

| 角色 | 能力 |
|------|------|
| 普通用户 | 仅 [user.md](./user.md) |
| 管理员 `AdminAuth` | 本文全部接口 |
| 超管 `RootAuth` | 包含管理员能力；另管渠道配置 |

所有写操作建议走现有 Admin 审计（`AdminAuth` 已带审计兜底则无需另挂）。

列表/详情 join：

- 主表：`user_file`  
- join：`file`（size、url、identifier、mime、channel_type、object_key）  
- 可选 join：`users` 取 `username`（列表展示）  
- **禁止** join 出 `file_upload_channel.config_proflle`

---

## 3. 通用约定

### 3.1 信封与分页

同 [user.md](./user.md) §3：`{ success, message, data }`；`p` + `page_size`（上限 100）；`PageInfo`。

### 3.2 列表项 `AdminUserFileView`

在 `UserFileView` 基础上增加：

```json
{
  "id": 99,
  "file_id": 12,
  "user_id": 7,
  "username": "alice",
  "file_channel_id": 1,
  "channel_type": "3",
  "channel_name": "R2 Primary",
  "file_name": "out",
  "file_suffix": "png",
  "source": "playground",
  "status": "1",
  "file_size": 1048576,
  "mime_type": "image/png",
  "identifier": "...",
  "file_url": "https://...",
  "object_key": "files/...",
  "ref_count": 3,
  "create_time": "...",
  "update_time": "..."
}
```

`channel_name`：由 `file_channel_id` 查渠道名（渠道已删则空字符串）。  
`ref_count`：来自 `file`，便于判断删逻辑文件后是否仍占用物理存储。

### 3.3 建议索引

```sql
-- user_file
KEY `idx_suffix` (`file_suffix`),
KEY `idx_channel_type_via_file` -- 若常按 channel_type 筛，优先用 file.channel_type + join，或冗余 channel_type 到 user_file（当前 SQL 未冗余，用 join file）

-- file
KEY `idx_channel_type` (`channel_type`),
KEY `idx_file_channel_id` (`file_channel_id`)  -- SQL 已有
```

按 `channel_type` 列表：`WHERE file.channel_type = ?`（join）。

---

## 4. 接口文档 · 查

前缀建议：`/api/storage`  
鉴权：`AdminAuth`（与用户 `/self` 并存；靠中间件与路径区分）

> 路由注册注意：先注册更具体的 `/self*`、`/stats*`、`/suffixes`，再注册 `/:id`，避免冲突。用户 prepare 等 POST 与管理员 GET 同前缀可行。

### 4.1 `GET /api/storage/` — 全站文件列表

#### Query

| 参数 | 必填 | 说明 |
|------|------|------|
| `p` / `page_size` | 否 | 分页 |
| `file_suffix` | 否 | 后缀；多值逗号 OR，如 `png,jpg` |
| `channel_type` | 否 | **按渠道类型**过滤：`0`\|`1`\|`2`\|`3` |
| `user_id` | 否 | 指定用户 |
| `username` | 否 | 用户名精确/前缀（实现选一种，推荐精确或 `like` 左匹配） |
| `status` | 否 | `user_file.status`；不传=全部 |
| `source` | 否 | |
| `keyword` | 否 | `file_name` 模糊 |
| `identifier` | 否 | 物理文件 MD5 |
| `file_channel_id` | 否 | **可选扩展**：实例 id 精确筛（拍板主筛为 channel_type；此参便于点进统计下钻，建议实现） |
| `order` | 否 | `create_time_desc`（默认）\|`create_time_asc`\|`file_size_desc` |

#### Response

```json
{
  "success": true,
  "data": {
    "page": 1,
    "page_size": 20,
    "total": 1000,
    "items": [ /* AdminUserFileView */ ]
  }
}
```

---

### 4.2 `GET /api/storage/user/:user_id` — 指定用户文件

与 §4.1 相同分页与 `file_suffix`/`status`/`source`/`keyword`/`channel_type`，路径固定用户。

等价于 `GET /api/storage/?user_id=:user_id`，保留独立路径以对齐 new-api「管理员查某用户资源」习惯（日志/任务同款）。

---

### 4.3 `GET /api/storage/suffixes` — 全站后缀聚合

#### Query

| 参数 | 说明 |
|------|------|
| `channel_type` | 可选 |
| `user_id` | 可选 |
| `status` | 可选 |

#### Response

```json
{
  "success": true,
  "data": {
    "items": [
      { "file_suffix": "png", "count": 1200 },
      { "file_suffix": "jpg", "count": 800 }
    ]
  }
}
```

---

### 4.4 `GET /api/storage/:id` — 详情

- `:id` = **`user_file.id`**（与用户侧一致，避免两套 id 语义）  
- 返回 `{ "file": AdminUserFileView }`  
- 不存在 → `File not found`

---

### 4.5 `GET /api/storage/stats/by-channel` — 按渠道统计数量

统计口径（拍板 **4.C**）：

| 指标 | 含义 |
|------|------|
| `user_file_count` | 该 `file_channel_id` 下 `user_file` 行数 |
| `file_count` | 该 `file_channel_id` 下物理 `file` 行数 |
| `total_size` | 建议：物理 `file.file_size` 求和（同一物理文件只计一次）；若实现成本高可改为 user_file join 累加并注明可能重复 |

聚合维度：**`file_channel_id`**（渠道实例）。每行带上 `channel_type`、`channel_name`，便于按类型展示。

#### Query

| 参数 | 说明 |
|------|------|
| `channel_type` | 可选；只统计该类型下的渠道实例 |
| `status` | 可选；作用于 `user_file_count` 时滤逻辑文件 status；`file_count` 默认可仅计 `file.status=1` 或全部——**推荐**：`user_file_count` 跟 `status`，`file_count`/`total_size` 仅 `file.status in ('1','0')` 排除已 `2` |

不分页（渠道数量通常有限）；若 >1000 再加分页（一般不需要）。

#### Response

```json
{
  "success": true,
  "data": {
    "items": [
      {
        "file_channel_id": 1,
        "channel_type": "3",
        "channel_name": "R2 Primary",
        "user_file_count": 1500,
        "file_count": 1200,
        "total_size": 9876543210
      },
      {
        "file_channel_id": 2,
        "channel_type": "2",
        "channel_name": "ImgBed",
        "user_file_count": 30,
        "file_count": 28,
        "total_size": 1234567
      }
    ],
    "sum": {
      "user_file_count": 1530,
      "file_count": 1228,
      "total_size": 9877777777
    }
  }
}
```

SQL 示意：

```sql
-- user_file_count per file_channel_id
SELECT file_channel_id, COUNT(*) AS user_file_count
FROM user_file
WHERE (? IS NULL OR status = ?)
GROUP BY file_channel_id;

-- file_count + total_size per file_channel_id
SELECT file_channel_id, channel_type, COUNT(*) AS file_count, COALESCE(SUM(file_size),0) AS total_size
FROM file
WHERE status IN ('0','1')
GROUP BY file_channel_id, channel_type;
```

应用层 merge，并补 `channel_name`；无数据的渠道可不出现或 count=0（**推荐只返回有数据的渠道**）。

---

### 4.6 `GET /api/storage/stats/summary` — 全局摘要（必要）

运维看板一行数字：

```json
{
  "success": true,
  "data": {
    "user_file_count": 1530,
    "file_count": 1228,
    "total_size": 9877777777,
    "uploading_user_file_count": 5
  }
}
```

---

## 5. 接口文档 · 删

### 5.1 `DELETE /api/storage/:id`

删任意用户的 `user_file`（id=`user_file.id`）。

语义同 [user.md](./user.md) §6.1（ref_count、物理删），**不校验** resource.user_id == 操作者。

#### Response

```json
{ "success": true, "data": { "deleted": true, "id": 99, "user_id": 7 } }
```

审计日志建议记录：操作者、被删 user_file id、file_id、原 user_id。

### 5.2 `DELETE /api/storage/` — 批量删

#### Request

```json
{
  "ids": [1, 2, 3]
}
```

- 最多 100 个/次  
- 返回 `deleted` / `failed[]`  

### 5.3 （不做）强制删物理文件

不提供 `DELETE /api/storage/file/:file_id` 无视引用接口。若未来需要，另开详设与 Root 权限。

---

## 6. 接口总表（管理侧）

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/storage/` | 全站列表（含 `channel_type` / `file_suffix` 等） |
| GET | `/api/storage/user/:user_id` | 指定用户列表 |
| GET | `/api/storage/suffixes` | 全站后缀聚合 |
| GET | `/api/storage/stats/by-channel` | 按渠道实例统计 user_file + file |
| GET | `/api/storage/stats/summary` | 全局摘要 |
| GET | `/api/storage/:id` | 详情 |
| DELETE | `/api/storage/:id` | 删逻辑文件 |
| DELETE | `/api/storage/` | 批量删 |

用户侧同前缀接口见 [user.md](./user.md)（`/self`、`prepare` 等）。

---

## 7. 与用户侧路由并存示意

```go
storage := apiRouter.Group("/storage")
{
    // UserAuth
    u := storage.Group("")
    u.Use(middleware.UserAuth())
    u.POST("/prepare", ...)
    u.POST("/presign-part", ...)
    u.POST("/complete", ...)
    u.POST("/abort", ...)
    u.GET("/self", ...)
    u.GET("/self/suffixes", ...)
    u.GET("/self/:id", ...)
    u.DELETE("/self/:id", ...)
    u.DELETE("/self", ...)

    // AdminAuth — 静态路径先于 /:id
    a := storage.Group("")
    a.Use(middleware.AdminAuth())
    a.GET("/stats/by-channel", ...)
    a.GET("/stats/summary", ...)
    a.GET("/suffixes", ...)
    a.GET("/user/:user_id", ...)
    a.GET("/", ...)
    a.DELETE("/", ...)
    a.GET("/:id", ...)
    a.DELETE("/:id", ...)
}
```

---

## 8. 错误约定（管理侧）

| message | 场景 |
|---------|------|
| `File not found` | id 不存在 |
| `Invalid channel_type` | 非 0–3 |
| `Too many ids` | 批量超过 100 |
| `Permission denied` / 鉴权中间件原文 | 非管理员 |

---

## 9. 验收清单（管理 API）

- [ ] Admin 可列表；非 Admin 403/失败  
- [ ] `channel_type`、`file_suffix`、分页、status 全量默认正确  
- [ ] `stats/by-channel` 同时给出 `user_file_count` 与 `file_count`（及 total_size）  
- [ ] `stats/summary` 可用  
- [ ] 删他人文件成功且 ref/物理语义与用户删一致  
- [ ] 批量删部分失败可回报  
- [ ] 响应无渠道密钥  
- [ ] 路由无与 `/self`、`/prepare` 冲突  

---

## 10. 关联

- [user.md](./user.md)  
- [../file_upload_channel_design/manager.md](../file_upload_channel_design/manager.md)  
- [../../FIleStorage.md](../../FIleStorage.md)
