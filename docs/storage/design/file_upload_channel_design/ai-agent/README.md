# FileUploadChannel · Agent 提示词索引

本目录供其它 Agent **生成代码 / 自动化测试** 使用。详设正文在上一级目录。

| 文件 | 用途 | 给谁 |
|------|------|------|
| [01-codegen-prompt.md](./01-codegen-prompt.md) | **实现提示词**：P0 后端+Default+Classic | 代码生成 Agent |
| [02-test-prompt.md](./02-test-prompt.md) | **测试提示词**：API/单测/冒烟与报告 | 测试 Agent |
| [03-checklist.md](./03-checklist.md) | **检查清单**：实现自测 + 测试复验 | 双方 |

## 推荐执行顺序

1. 实现 Agent：只喂 `01-codegen-prompt.md`，并要求先读 `../manager.md`、`../use.md`（P0）、`../../db-design/file_upload_channel.sql`。  
2. 测试 Agent：喂 `02-test-prompt.md` + `03-checklist.md`，对照实现产出 `test-report.md`。  
3. 人工：抽检菜单位置、脱敏、Root 权限三条即可。

## 范围提醒（P0）

- **做**：渠道 CRUD、默认、启停、探测、双端 UI、`GET /default`  
- **不做**：`prepare`/`complete` 上传、秒传、`type=0` 本地配置 UI
