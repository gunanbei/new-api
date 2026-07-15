# 创作台第二阶段实施提示词

将以下内容完整交给负责实现和自检的 Agent。

---

你正在实现创作台第二阶段“图片与 SVG 工作区”。以 [Stage-Two.md](./Stage-Two.md) 为唯一业务范围，以 [Stage-One-Final-20260713.md](./Stage-One-Final-20260713.md) 为既有配置基线，以仓库根目录 `AGENTS.md` 为工程约束。先完整阅读这三份文件和涉及目录的 `AGENTS.md`，再开始任何改动。

## 目标

完成已登录用户的持久图片创作：

1. 实现图片/SVG 任务与任务资产的跨数据库数据模型、状态机、用户 API、作品历史和重试。
2. 复用现有图片、Responses、Claude、Midjourney Relay 及其选路、预扣费、结算、使用日志、渠道适配器和重试；覆盖当前图片能力，不复制供应商 HTTP 客户端或渠道密钥处理。
3. 对同步图片、OpenAI Responses 图片和 Claude SVG 导入既有文件系统；桥接既有 Midjourney 任务和轮询结果，仅导入图片成品。
4. 在 `/imgen` 提供默认前端图片工作区，移除该菜单项的 native 外跳语义。
5. 加固 SVG、上游产物下载、输入文件读取、参数模式和用户资源隔离。

## 不允许实现

- 不实现视频 UI、视频任务、视频导入、视频 remix、视频取消或视频轮询；阶段三再做。
- 不新建通用工作流/任务框架、供应商专用复制代码、用户 API Key 流程、浏览器直连上游或本机 HTTP 回调。
- 不读取、写入、迁移或兼容 `image_playground.model_registry`，不修改旧 image playground bootstrap 的行为。
- 不复制既有渠道选择、渠道密钥、倍率、预扣费、结算、使用日志、Midjourney 轮询器或文件渠道配置。
- 不新增依赖；不为阶段三预建接口、工厂、表或配置。

## 工作方式

1. 先执行 `git status`；工作区已有用户改动，只改本阶段直接涉及的文件，绝不重置、覆盖、格式化无关文件。
2. 如存在 `.codegraph`，定位/理解前先使用 CodeGraph；再阅读真实关联的 Relay、计费、任务、Midjourney、文件存储、前端路由与用户文件代码。
3. 按 `Router → Controller → Service → Model` 分层。服务层不可用 HTTP 调本机 Relay；从既有实现抽取最小的可复用内部执行入口。
4. 所有 JSON 编解码使用 `common.Marshal`、`common.Unmarshal`、`common.UnmarshalJsonStr` 或 `common.DecodeJson`。GORM 模型和迁移必须同时支持 SQLite、MySQL 5.7.8+、PostgreSQL 9.6+。
5. 前端开始前阅读 `web/default/AGENTS.md`；新增可见文案必须使用 i18n-translate skill，并在 `web/default` 使用 Bun。
6. 任何用户可控的计费数量在进入 Relay 前复用既有图片边界（特别是 `dto.MaxImageN`）；不以裸 `int` 转换、未界定 JSON 字段或 passthrough 绕过校验。

## 必须实现的后端行为

### 数据、设置与状态

- 新增并注册 `creative_task` 与 `creative_task_asset` 到 `migrateDB`、`migrateDBFast`。字段、索引、公开 `task_key`、JSON TEXT、逻辑关联、资产角色、任务快照及 CAS 状态转换符合 Stage-Two.md 第 5、6 节。
- 任务公开 API 只使用 `task_key`，按当前用户过滤。不得向用户返回私有结果清单、上游 URL、密钥、渠道配置或其他用户资产。
- 升级 `creative_studio.settings` 到版本 2，保留默认文件渠道并加入图片 MIME/大小策略；旧版本读到安全默认，写回规范版本 2。
- 模型名称变更时事务性更新下属绑定 `request_model`，将它们重置为 `pending` 且禁用；新增绑定重新验证/启用逻辑，只有 `valid` 才能启用。

### 目录可见性与请求校验

- `GET /api/creative/bootstrap`、任务创建/查询/重试/导入重试均经 `UserAuth`。只能看到当前用户可用分组内，模型/能力/发布均启用、至少一个 binding 已 `valid` 且启用、渠道与默认保存渠道仍可用的图片能力。
- 参数严格按服务端 `input_schema` 白名单校验并按能力默认→分组默认→用户允许字段合并。禁止客户端决定模型、渠道、价格、状态、系统提示词、保存渠道、输出 URL 或 `stream=true`。
- 文件参数只接受当前用户可访问的 `user_file_id`；禁止任意 URL、对象键、未受限 base64 和其他用户文件。输入资产关系写入任务。
- 固定支持 `openai_image`、`openai_responses_image`、`advanced_custom_image`、`midjourney_image`、`claude_svg`，且执行方式/能力组合/渠道兼容性必须依 Stage-Two.md 的协议表验证。不得允许管理员填写请求路径、脚本或密钥。

### 执行、计费、导入

- 复用既有图片 Relay 的模型映射、渠道选路、权重、健康状态、预扣/结算、使用日志与重试。创作台仅限制候选为已验证绑定，按绑定优先级排序；不要复制现有编排。
- 同步图片、Responses 图片和 SVG 必须先创建任务，再取得完整规范化产物，最后导入；所有输出导入成功后才 `succeeded`。
- `claude_svg` 强制非流式、固定系统指令、只提取单个 SVG、白名单净化并以 `image/svg+xml` 保存；不允许 `dangerouslySetInnerHTML` 或未经净化 SVG。
- Midjourney 仅提交图片动作，保留并观察既有任务 ID 和 `midjourney` 表/轮询器的终态；不复制/迁移/改写其轮询和状态，不处理视频 URL。
- 在 `service/storage` 增加仅服务端用的受控读取、`ImportGeneratedAsset`（或同等最小能力）。它要支持现有可用 S3/WebDAV/ImgBed 渠道的流式写入，复用 `file`/`user_file` 与渠道配置，不改变浏览器直传接口。
- 上游 URL 下载必须实施 SSRF 防护：协议、DNS/IP、重定向、超时、大小、MIME、魔数和流式读取限制。不得信任 URL 出自上游，也不得把它长期暴露或保存为作品地址。
- `asset_import_failed` 可在临时清单过期前执行仅导入重试且不再次计费；普通 retry 必须创建新任务、新请求和独立账务记录。

### API、前端与文件保护

- 实现 `GET /bootstrap`、`POST /tasks`、`GET /tasks`、`GET /tasks/:task_key`、`POST /tasks/:task_key/retry`、`POST /tasks/:task_key/import-retry`；阶段二没有取消 API。
- 作品输出文件被 `creative_task_asset` 引用时，沿既有删除路径拒绝删除；输入文件终态后仍可按原规则删除。
- `/imgen` 成为默认前端创作页；仅显示图片工作区和我的作品。复用现有表单、Tabs、主题变量、文件选择和 React Query；非终态任务轮询退避，终态停止。
- 所有可见文案走 i18n，并完成仓库支持语言翻译。SVG 只经安全文件 URL 显示。

## 验证要求

至少执行并报告：

1. 覆盖任务状态、可见性、跨用户隔离、参数边界、绑定重验证、模型改名、Midjourney 观察、导入重试、文件保护的有价值 Go 测试；使用 testify `require/assert`。
2. 覆盖 URL/b64/SVG 导入的 MIME、大小、SSRF 重定向与 SVG 恶意样本；确保失败不产生成功任务或孤儿资产。
3. 覆盖创作请求的现有计费/使用日志关联、普通重试独立计费和仅导入重试不再计费。
4. 既有 `GetImagePlaygroundBootstrap`、图片 Relay、Midjourney 与文件存储的相关回归测试。
5. 对修改的 Go 文件运行 `gofmt`，执行 `git diff --check`。
6. 在 `web/default` 使用 Bun 运行 i18n 同步和项目已有的最小类型检查/构建。

若命令因无关的现有问题失败，不修复无关问题；报告命令、失败原因与本阶段的关系。

## 完成标准

只有 Stage-Two.md 的范围、状态机、安全约束、用户 API、作品导入和验收要求都满足才结束。最终回复只包含：

- 修改的核心文件与行为；
- 已通过的验证；
- 未完成项或明确阻塞（如无则写“无”）。
