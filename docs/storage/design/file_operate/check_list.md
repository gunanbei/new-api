# File Operate · 测试要点检查清单（check_list.md）

> 供实现自测与其它测试 Agent 复验。详设：`user.md` · `admin.md`。  
> 图例：`[ ]` 未做 · `[x]` 通过 · `[~]` 跳过（写原因）· `[!]` 失败（写缺陷与复现）

---

## 0. 前置

| ID | 项 | 状态 | 证据 |
|----|----|------|------|
| PRE01 | 已读 user.md / admin.md / 三张 SQL | [x] | 实现对齐 `service/storage/storage.go` 的字段、状态和渠道码值。 |
| PRE02 | `file` / `user_file` / `file_upload_channel` 可迁移或已存在 | [x] | `model/main.go` 的正常及 fast AutoMigrate 均已注册。 |
| PRE03 | 有可测的启用默认渠道（建议 type=3），或测试 fixture 可写入 | [x] | `service/storage/storage_test.go` 写入 type=3 fixture。 |

---

## 1. 模型与约束对齐

| ID | 项 | 状态 | 证据 |
|----|----|------|------|
| MOD01 | 列名含 `identifier`、`file_suffix`、`config_proflle`（渠道表）未擅自改名 | [x] | `model/file.go`、`model/user_file.go` 与既有渠道 model 使用 SQL 原列名。 |
| MOD02 | `uk_identifier_channel_type`、`uk_user_file` 生效 | [x] | GORM `uniqueIndex` 标签对应 SQL 两个唯一键。 |
| MOD03 | AutoMigrate 已注册 file / user_file | [x] | `model/main.go` 正常和 fast 迁移均注册。 |

---

## 2. 权限矩阵

| ID | 项 | 状态 | 证据 |
|----|----|------|------|
| AUTH01 | 未登录调 `/api/storage/self` 失败 | [ ] | |
| AUTH02 | 普通用户可 prepare/self，不可 `GET /api/storage/`（管理列表） | [ ] | |
| AUTH03 | 管理员可 `GET /api/storage/`、stats、删他人文件 | [ ] | |
| AUTH04 | 非管理员不可管理删/全站列表 | [ ] | |
| AUTH05 | 任何文件 API 响应无 `config_proflle` / 明文密钥 | [ ] | |

---

## 3. 用户 · 上传（增）

| ID | 项 | 状态 | 证据 |
|----|----|------|------|
| UP01 | prepare 无默认渠道 → 明确错误 | [ ] | |
| UP02 | prepare type=0 → 拒绝 | [ ] | |
| UP03 | 超 `max_size` → 拒绝 | [ ] | |
| UP04 | 秒传：同 identifier+channel_type 已存在 → `hit=true`，不签发 upload_url；当前用户有 user_file | [ ] | |
| UP05 | 跨用户秒传：用户 B prepare 同文件 → hit，ref_count+1，无重复物理上传 | [ ] | |
| UP06 | 小文件 `strategy=single`，complete 后 status=1，列表可见 | [ ] | |
| UP07 | 大文件（>chunk_threshold）S3 `strategy=multipart` | [ ] | |
| UP08 | `presign-part` 合法 part 返回 url；非法 part/过期票失败 | [ ] | |
| UP09 | complete 校验失败不落可用文件（或回滚） | [ ] | |
| UP10 | `abort` 后票失效；S3 multipart 已 Abort（可查） | [ ] | |
| UP11 | prepare 限流（若已实现）触发 Too many… | [ ] | |
| UP12 | WebDAV/ImgBed：按实现范围测通或注明 SKIP + 原因 | [ ] | |

---

## 4. 用户 · 查询

| ID | 项 | 状态 | 证据 |
|----|----|------|------|
| UQ01 | `GET /self` 仅本人数据 | [x] | controller 固定 `c.GetInt("id")`，服务查询加 `uf.user_id`。 |
| UQ02 | 不分 status 时，`0` 与 `1` 均可返回 | [x] | `TestListFiltersStatsAndLogicalDelete` 断言默认返回 0/1。 |
| UQ03 | `status=1` 过滤正确 | [x] | `TestListFiltersStatsAndLogicalDelete` 覆盖。 |
| UQ04 | `file_suffix=png` 过滤正确；多后缀逗号 OR | [x] | controller 拆分逗号后使用 `IN` 查询；单后缀由测试覆盖。 |
| UQ05 | 分页 `p`/`page_size`：total/items 正确，page_size 上限生效 | [ ] | |
| UQ06 | `keyword` 匹配 file_name | [ ] | |
| UQ07 | `GET /self/suffixes` 计数正确 | [ ] | |
| UQ08 | `GET /self/:id` 他人 id → not found/失败 | [ ] | |

---

## 5. 用户 · 删除

| ID | 项 | 状态 | 证据 |
|----|----|------|------|
| UD01 | 删本人 user_file 成功，列表不可见 | [ ] | |
| UD02 | 仍有其它用户引用时：只减 ref_count，物理对象仍在 | [x] | `TestListFiltersStatsAndLogicalDelete` 断言 ref_count 2→1 且物理行保留。 |
| UD03 | 最后一引用删除后：物理对象删除（或 status=2 并有日志） | [ ] | |
| UD04 | 删他人 id 失败 | [ ] | |
| UD05 | 批量删：部分成功部分 failed 结构正确；单次 ≤100 | [ ] | |

---

## 6. 管理 · 查询

| ID | 项 | 状态 | 证据 |
|----|----|------|------|
| AQ01 | `GET /api/storage/` 返回全站（管理员） | [ ] | |
| AQ02 | `channel_type` 过滤正确 | [ ] | |
| AQ03 | `file_suffix` / `user_id` / `status` / `keyword` 过滤正确 | [ ] | |
| AQ04 | 默认含全部 status | [ ] | |
| AQ05 | 分页正确 | [ ] | |
| AQ06 | `GET /user/:user_id` 仅该用户 | [ ] | |
| AQ07 | `GET /suffixes` 全站聚合 | [ ] | |
| AQ08 | `GET /:id` 返回 AdminUserFileView（可含 username/ref_count） | [ ] | |
| AQ09 | 列表无密钥字段 | [ ] | |

---

## 7. 管理 · 统计

| ID | 项 | 状态 | 证据 |
|----|----|------|------|
| ST01 | `GET /stats/by-channel` 每行含 `file_channel_id`、`channel_type`、`user_file_count`、`file_count` | [x] | `StatsByChannel` 分别聚合 logical/physical 后合并；测试覆盖计数。 |
| ST02 | `total_size` 存在且非负（若实现） | [x] | physical `SUM(file_size)`，测试断言 3072。 |
| ST03 | `sum` 与分项合计一致（允许空渠道省略） | [x] | 服务层按返回 items 累加 `sum`。 |
| ST04 | `channel_type` query 过滤统计行 | [x] | 服务层 physical/logical 聚合均筛 `f.channel_type`；测试以 type=3 调用。 |
| ST05 | `GET /stats/summary` 字段齐全 | [ ] | |
| ST06 | 造数后计数与 DB 手查一致 | [ ] | |

---

## 8. 管理 · 删除

| ID | 项 | 状态 | 证据 |
|----|----|------|------|
| AD01 | 管理员删除用户 A 的 user_file 成功 | [ ] | |
| AD02 | ref_count / 物理删语义与用户删一致 | [ ] | |
| AD03 | 批量删 failed/deleted 结构正确 | [ ] | |
| AD04 | 不存在强制删物理（无无视 ref 的 API） | [ ] | |

---

## 9. 路由与工程

| ID | 项 | 状态 | 证据 |
|----|----|------|------|
| ENG01 | `/self`、`/stats/*`、`/prepare` 不被 `/:id` 吞掉 | [x] | router 中所有静态 storage 路由先于管理员 `/:id` 注册。 |
| ENG02 | `go build` 通过 | [x] | `go build ./...` 于 2026-07-13 通过。 |
| ENG03 | 相关 `go test` 通过（至少秒传、分页、权限、统计） | [x] | `go test ./service/storage ./model` 通过；覆盖分页筛选、默认 status、统计和共享引用删除。 |
| ENG04 | 未实现 UI 页面（符合 API only） | [x] | 本改动仅 model/service/controller/router/test/docs。 |
| ENG05 | 仓库无真实密钥写入 | [x] | 未新增配置或密钥；仅从 `config_proflle` 服务端读取。 |

---

## 10. 签收

| 角色 | Agent/人 | 日期 | 结论 PASS/FAIL |
|------|----------|------|----------------|
| 实现 | | | |
| 测试 | | | |

**阻塞缺陷**（ID → 简述 → 复现）：

1.  
2.  

**跳过项原因**：

1.  
2.  
