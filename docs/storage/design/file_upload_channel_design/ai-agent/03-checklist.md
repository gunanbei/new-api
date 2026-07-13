# FileUploadChannel P0 · 检查清单（实现 + 测试共用）

> 用法：实现 Agent 自测勾选；测试 Agent 复验并在 `test-report.md` 引用条目 ID。  
> 详设：`../manager.md` · `../use.md` · `../../db-design/file_upload_channel.sql`

**图例**：`[ ]` 未做 · `[x]` 通过 · `[~]` 跳过（写原因）· `[!]` 失败（写缺陷）

---

## 1. 文档与约束对齐

| ID | 项 | 状态 | 证据/备注 |
|----|----|------|-----------|
| DOC01 | 使用列名 `config_proflle`（未擅自改名） | [ ] | |
| DOC02 | type 码值 0/1/2/3 与 SQL 注释一致 | [ ] | |
| DOC03 | 管理 API 均为 RootAuth | [ ] | |
| DOC04 | `GET /default` 为 UserAuth 且无密钥 | [ ] | |
| DOC05 | 本迭代未实现 prepare/complete 上传主链路（或仅文档占位） | [ ] | |

---

## 2. 后端 · Model / 迁移

| ID | 项 | 状态 | 证据/备注 |
|----|----|------|-----------|
| BE01 | `model/file_upload_channel.go` 存在且 TableName 正确 | [ ] | |
| BE02 | 已挂入 AutoMigrate | [ ] | |
| BE03 | 启动后表结构与 SQL 关键列一致 | [ ] | |

---

## 3. 后端 · 管理 API

| ID | 项 | 状态 | 证据/备注 |
|----|----|------|-----------|
| API01 | `GET /api/file-upload-channel/` 列表 | [ ] | |
| API02 | `GET /api/file-upload-channel/:id` 详情 | [ ] | |
| API03 | `POST /api/file-upload-channel/` 创建 | [ ] | |
| API04 | `PUT /api/file-upload-channel/:id` 更新 | [ ] | |
| API05 | `PUT /:id/default` 默认互斥 | [ ] | |
| API06 | `PUT /:id/status` 启停 | [ ] | |
| API07 | `DELETE /:id` 无引用可删 | [ ] | |
| API08 | `DELETE` 有引用拒绝 | [ ] | |
| API09 | `POST /:id/probe` | [ ] | |
| API10 | `POST /probe`（路由未被 :id 吞掉） | [ ] | |
| API11 | 创建 `type=0` 拒绝 | [ ] | |
| API12 | 列表/详情敏感字段脱敏 | [ ] | |
| API13 | 更新时空密码不覆盖 | [ ] | |
| API14 | 停用唯一默认被拒绝（或等价安全策略已测） | [ ] | |
| API15 | 停用渠道不能设为默认 | [ ] | |

---

## 4. 后端 · 上传侧 P0

| ID | 项 | 状态 | 证据/备注 |
|----|----|------|-----------|
| USE01 | `GET /api/file-upload-channel/default` 用户可访问 | [ ] | |
| USE02 | 无默认时 `data=null`（或约定错误）行为符合 use.md | [ ] | |
| USE03 | 响应无 `config_proflle` / 密钥 | [ ] | |
| USE04 | （可选）`GET /resolve?channel_type=` | [ ] | |

---

## 5. 后端 · 探测语义

| ID | 项 | 状态 | 证据/备注 |
|----|----|------|-----------|
| PRB01 | type=3 错误配置返回失败态且无密钥泄漏 | [ ] | |
| PRB02 | type=2 错误配置失败态 | [ ] | |
| PRB03 | type=1 错误配置失败态 | [ ] | |
| PRB04 | （可选）真实凭据 ok=true | [ ] | |
| PRB05 | 表单 probe + id + 空密码合并库密钥 | [ ] | |

---

## 6. 权限矩阵

| ID | 项 | 状态 | 证据/备注 |
|----|----|------|-----------|
| AUTH01 | 未登录禁管理 API | [ ] | |
| AUTH02 | 普通用户禁管理 API | [ ] | |
| AUTH03 | 非 Root 管理员禁管理 API | [ ] | |
| AUTH04 | Root 可管理 API | [ ] | |
| AUTH05 | 普通用户可读 `/default` | [ ] | |

---

## 7. 前端 · Default

| ID | 项 | 状态 | 证据/备注 |
|----|----|------|-----------|
| FD01 | section id=`data-management` | [ ] | |
| FD02 | 菜单位于 logs 与 performance **之间** | [ ] | |
| FD03 | 标题 i18n：`Data Management` / 中文「数据管理」 | [ ] | |
| FD04 | 列表展示渠道 | [ ] | |
| FD05 | 新建/编辑表单 | [ ] | |
| FD06 | 切换 type=1/2/3 字段自动切换且不串配置 | [ ] | |
| FD07 | 设默认 / 启停 / 删除 / 探测按钮可用 | [ ] | |
| FD08 | 暗色主题无明显样式破坏 | [ ] | |
| FD09 | type=0 不出现在可选类型中 | [ ] | |

---

## 8. 前端 · Classic

| ID | 项 | 状态 | 证据/备注 |
|----|----|------|-----------|
| FC01 | Setting Tab `data-management` 存在 | [ ] | |
| FC02 | Tab 位于 operation 与 performance 之间（或文档约定位置） | [ ] | |
| FC03 | 功能与 Default 对等（CRUD/默认/探测） | [ ] | |
| FC04 | i18n 中文可读 | [ ] | |

---

## 9. 质量与安全

| ID | 项 | 状态 | 证据/备注 |
|----|----|------|-----------|
| QS01 | `go build` / 相关 `go test` 通过 | [ ] | |
| QS02 | 仓库无真实密钥、无 `.env` 误提交 | [ ] | |
| QS03 | 错误信息不回显完整 Secret | [ ] | |
| QS04 | 与 AI `channels` 路由无冲突 | [ ] | |

---

## 10. 签收

| 角色 | 姓名/Agent | 日期 | 结论 |
|------|------------|------|------|
| 实现 | | | PASS / FAIL |
| 测试 | | | PASS / FAIL |
| 人工抽检（可选） | | | |

**阻塞缺陷列表**（ID → 简述 → 复现）：

1.  
2.  

**已知限制 / 跳过原因**：

1.  
2.  
