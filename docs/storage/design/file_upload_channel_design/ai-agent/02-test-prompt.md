# FileUploadChannel · 自动化测试提示词（给测试 Agent）

> 将本文**整段复制**给测试 Agent。  
> 前置：实现 Agent 已按 `01-codegen-prompt.md` 完成 P0；详设见同级 `../manager.md`、`../use.md`。  
> 检查清单对照：`03-checklist.md`（逐项勾选并写证据）。

---

## 角色与目标

你是 new-api 的 QA / 自动化测试 Agent。请在不破坏用户数据的前提下，对 **File Upload Channel P0** 做自动化验证（优先 API + 编译/单测；有条件再补 UI 冒烟）。

目标：证明管理端 Root 权限、CRUD、默认渠道互斥、脱敏、探测接口形态、UserAuth 默认渠道只读接口、以及关键负面用例如期工作。

---

## 环境假设

- 工作目录：`new-api/`  
- 可用：Go 测试、`curl`/httpie、可选 Playwright/浏览器（若环境允许）  
- 需要：**超管账号** session 或 Root 可登录态；另备 **普通用户** 与（若有）**普通管理员** 账号做权限对照  
- 探测类用例：允许用无效 endpoint 断言「失败形态」；真实成功探测仅在提供了测试用 S3/WebDAV/ImgBed 凭据时执行（凭据从环境变量读，禁止写进仓库）

环境变量建议（可选）：

```bash
TEST_ROOT_BASE_URL=http://127.0.0.1:3000
TEST_ROOT_AUTH=...          # 超管 Authorization / Cookie 方案按项目实际
TEST_USER_AUTH=...
TEST_ADMIN_AUTH=...         # 非 Root 管理员
# 可选真实探测：
TEST_S3_ENDPOINT=...
TEST_S3_REGION=...
TEST_S3_BUCKET=...
TEST_S3_AK=...
TEST_S3_SK=...
```

若项目已有测试工具函数（登录拿 token、API helper），**优先复用**，不要另起一套鉴权协议。

---

## 测试策略

### A. 静态 / 构建

1. `go test` 覆盖新增 model/controller 包（至少表名、脱敏、默认互斥、type=0 拒绝的单测；probe 可 httptest）。  
2. 前端：default/classic 相关包能通过现有 lint/tsc（若仓库有脚本则跑）。  
3. Grep 断言：  
   - `data-management` 在 `section-registry` 中且位于 `logs` 与 `performance` 之间  
   - classic `Setting/index.jsx` 含 `data-management`  
   - 管理路由使用 `RootAuth`  
   - 代码含 `config_proflle`（未「修正」拼写）

### B. API 自动化（核心）

统一断言响应含 `success`；成功时按需查 `data`。

对每条用例记录：请求、状态/body 摘要、通过/失败。

#### B1 权限

| ID | 步骤 | 期望 |
|----|------|------|
| P01 | 未登录 `GET /api/file-upload-channel/` | 失败（401/未登录 message） |
| P02 | 普通用户 `GET /api/file-upload-channel/` | 失败（权限不足） |
| P03 | 普通管理员（非 Root）`GET /api/file-upload-channel/` | 失败 |
| P04 | 超管 `GET /api/file-upload-channel/` | success |
| P05 | 普通用户 `GET /api/file-upload-channel/default` | success（data 可为 null） |
| P06 | 超管 `POST /api/file-upload-channel/` type=`0` | success=false |

#### B2 CRUD 与脱敏

| ID | 步骤 | 期望 |
|----|------|------|
| C01 | 超管创建 type=`3` 渠道（假 endpoint/密钥即可） | success，返回 id；body **无**明文 secret |
| C02 | `GET /:id` | 无明文 `secret_access_key`/`api_token`/`password` |
| C03 | `PUT /:id` 改 name，密钥字段传 `""` | name 变，密钥仍可用于后续（若曾真实探测）或至少未被写成空导致配置损坏——可用「再 GET 仍显示已配置掩码」间接验证 |
| C04 | `PUT /:id` 换 name + 有效新密钥（可选） | success |
| C05 | 创建第二条 type=`1` | success |
| C06 | 列表 `?type=3` | 仅 type3 |
| C07 | 删除无引用渠道 | success |
| C08 | 若能插入/模拟 `file` 或 `user_file` 引用 | 删除失败；去掉引用后可删 |

#### B3 默认渠道

| ID | 步骤 | 期望 |
|----|------|------|
| D01 | 渠道 A `PUT .../default` | A.is_default=1 |
| D02 | 渠道 B `PUT .../default` | 仅 B 为 1，A 为 0 |
| D03 | `GET /api/file-upload-channel/default`（用户） | data.id=B，且无 config_proflle |
| D04 | 停用当前默认（唯一默认） | 应失败或文档规定行为；按 manager：**拒绝停用唯一默认**则 assert 失败 |
| D05 | 停用非默认渠道 | success，status=0；resolve/default 不可再选到它 |

#### B4 探测接口形态

| ID | 步骤 | 期望 |
|----|------|------|
| T01 | `POST /probe` body type=3 + 明显错误 endpoint | success 业务层 ok=false 或 success=false；**响应无密钥**；含 detail/message |
| T02 | `POST /:id/probe` 对已存假配置 | 同上，接口 200 信封正常 |
| T03 | 路由：`POST /api/file-upload-channel/probe` 不被当成 id=probe | 必须命中表单探测 handler |
| T04 | （可选真实）有效 S3 凭据 probe | ok=true，latency_ms≥0 |

WebDAV/ImgBed 同理各至少 1 条「失败形态」即可。

#### B5 负面与边界

| ID | 步骤 | 期望 |
|----|------|------|
| N01 | chunk_threshold 负数 | 校验失败 |
| N02 | 缺必填 bucket（type3） | 失败 |
| N03 | 将停用渠道设默认 | 失败 |

### C. UI 冒烟（有浏览器时）

1. 超管打开 `/system-settings/operations/data-management`（default）：标题为数据管理/Data Management。  
2. 侧栏顺序：日志维护 → **数据管理** → 性能。  
3. 新建弹窗切换 type 1/2/3，字段集变化。  
4. Classic：Setting Tab「数据管理」可打开并列表加载。  
5. 亮/暗主题切换无裸眼明显坏布局（截图可选）。

无浏览器则在报告中标注 **UI SKIPPED**，不以失败论（但 checklist 对应项标跳过）。

---

## 产出要求

1. 在仓库生成测试代码优先位置（任选其现有风格）：  
   - `model/file_upload_channel_test.go`  
   - `controller/file_upload_channel_test.go`（或 `_test` 包）  
   - 可选脚本：`docs/storage/design/file_upload_channel_design/ai-agent/scripts/smoke-api.sh`  
2. 运行测试并保存摘要到：  
   `docs/storage/design/file_upload_channel_design/ai-agent/test-report.md`  
   内容包含：环境、通过/失败表（对应该 checklist ID）、失败日志摘录、未测项原因。  
3. 同步勾选 `03-checklist.md`（可复制一份 `03-checklist-result.md` 填写，勿弄丢原清单模板）。

---

## 禁止事项

- 不要把真实 AK/SK/Token 写入报告或 fixture 文件  
- 不要 `git push`、不要改 git config  
- 不要为了让测试通过而削弱 RootAuth 或去掉脱敏  
- 不要测试并实现 P1 上传 prepare/complete（超出 P0）

---

## 完成定义

- 权限矩阵 P01–P06 有自动化或明确手工证据  
- CRUD + 默认互斥 + probe 路由至少有一条自动化路径跑通  
- `test-report.md` 已生成且与 checklist 可互相索引  
- 失败项有复现步骤，而非仅 “failed”
