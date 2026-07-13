# File Operate 详设索引

文件操作（增删查）API 详设，**不做 UI**。

| 文档 | 内容 |
|------|------|
| [user.md](./user.md) | 用户：`prepare` / 分片 `presign-part` / `complete` / `abort`；查（含按后缀）；删 |
| [admin.md](./admin.md) | 管理员：全站查（按 `channel_type`/后缀等）；按渠道统计；删任意用户文件 |
| [generate-code.md](./generate-code.md) | **给实现 Agent 的完整提示词** |
| [check_list.md](./check_list.md) | **测试要点检查清单** |

## 拍板摘要

| 项 | 结论 |
|----|------|
| 管理员鉴权 | `AdminAuth` |
| 增 | 浏览器直传 + 分片（对齐渠道阈值） |
| 管理列表「按渠道」 | `channel_type` |
| 统计 | `user_file_count` + `file_count`（+ `total_size`）按 `file_channel_id` |
| 删 | 逻辑文件 + 引用归零删物理 |
| 列表 status | 默认全部，可筛 |
| 文档 | `user.md` + `admin.md` |

表：`../../db-design/file.sql` · `user_file.sql` · `file_upload_channel.sql`

## Agent 执行顺序

1. 实现：喂 `generate-code.md`，先读 `user.md` / `admin.md` / SQL。  
2. 测试：对照 `check_list.md` 逐项勾选并留证据。
