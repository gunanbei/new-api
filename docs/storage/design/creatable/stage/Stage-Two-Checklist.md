# 创作台第二阶段检查清单

本清单与 [Stage-Two.md](./Stage-Two.md) 配套使用。全部勾选后，阶段二才可交付。

## 1. 开始前

- [x] 已阅读根目录及涉及目录的 `AGENTS.md`、[Stage-One-Final-20260713.md](./Stage-One-Final-20260713.md)、[Stage-Two.md](./Stage-Two.md)。
- [x] 已检查 `git status`，未覆盖用户已有改动。
- [x] 已在需要定位或理解代码时优先使用 CodeGraph。
- [x] 已确认本阶段是全量图片与 SVG，不实现任何视频功能。
- [x] 已确认不会读取、迁移、回写或兼容旧 image playground 注册表和 bootstrap。

## 2. 数据与设置

- [x] `creative_task` 与 `creative_task_asset` 已加入 `migrateDB`。
- [x] `creative_task` 与 `creative_task_asset` 已加入 `migrateDBFast`。
- [x] 新表使用 GORM、逻辑关联、TEXT JSON；不使用物理外键、数据库 JSON 类型、AUTO_INCREMENT、SERIAL 或专属 UPSERT。
- [x] `task_key` 不可猜测、全局唯一，用户 API 不接受内部任务 ID。
- [x] 任务保存用户、能力、绑定、模型/分组/协议/执行方式、实际渠道、请求 ID、外部任务 ID、参数与错误快照。
- [x] 任务保存 `retry_of_id`，普通重试不会覆盖旧任务。
- [x] 任务状态更新使用 CAS，重复轮询/重试不会双重导入。
- [x] 任务资产只允许 `input`、`output`、`thumbnail` 角色并保持位置排序。
- [x] 输出资产引用的用户文件不能通过既有删除路径直接删除；终态输入文件仍可按既有规则删除。
- [x] `creative_studio.settings` 已升级为版本 2，并保留安全的旧版本读取默认值。
- [x] 设置包含默认保存渠道、允许图片 MIME、位图大小上限与 SVG 大小上限。
- [x] 设置响应和任务响应均不泄露渠道凭据、私有导入清单或上游临时 URL。

## 3. 管理配置与协议

- [x] 绑定重新验证会同时检查目录/发布状态、分组、`abilities`、渠道、协议、执行方式和保存渠道。
- [x] 只有 `validation_status=valid` 的绑定可以启用。
- [x] 模型改名事务性更新绑定 `request_model`，将受影响绑定禁用并标为 `pending`。
- [x] 管理端没有任意请求路径、脚本、URL 或密钥编辑字段。
- [x] `openai_image` 只允许图片 `generate|edit`、位图、同步结果。
- [x] `openai_responses_image` 只接受可解析的图片工具结果。
- [x] `advanced_custom_image` 仅允许具备匹配图片路由的渠道。
- [x] `midjourney_image` 只允许图片动作，拒绝视频 URL/动作。
- [x] `claude_svg` 只允许图片生成、SVG 矢量、`sync_artifact`。
- [x] 无可聚合终态、无法规范化结果或不兼容渠道的绑定无法启用。

## 4. 用户权限与参数

- [x] 所有 `/api/creative/*` 用户路由均经 `middleware.UserAuth()`。
- [x] bootstrap 仅返回当前用户可用分组内的已发布、启用、具有 valid 启用绑定且保存渠道可用的图片能力。
- [x] 任务列表、详情、重试、导入重试均先按当前用户过滤。
- [x] 无法猜测 `task_key`，且其他用户无法以它读取或操作任务。
- [x] 参数按服务端 schema 验证，未知字段、错误类型、越界、缺失必填字段被拒绝。
- [x] 默认参数按能力→分组→用户允许字段合并并保存实际快照。
- [x] 用户不能提交模型、渠道、价格、状态、保存渠道、系统提示词、输出 URL 或 `stream=true`。
- [x] 图片数量复用 `dto.MaxImageN` 等既有计费边界，所有计费乘数在计费前受限。
- [x] 文件参数只接受当前用户自己的可用 `user_file_id`。
- [x] 任意远程 URL、对象键、越权文件和无界 base64 均被拒绝。

## 5. 执行与计费

- [x] 创作台复用既有 Relay 的模型映射、渠道健康、权重、预扣费、结算、使用日志和重试，未复制该链路。
- [x] 创作台只把 valid 启用绑定作为候选渠道，按 binding priority 排序。
- [x] 任务先落库，再进入同步执行或 Midjourney processing。
- [x] 同步图片全部导入成功后才标记 `succeeded`。
- [x] OpenAI 图片 URL、`b64_json`、Responses 图片产物均被规范化并导入。
- [x] Midjourney 仅保存/观察既有任务 ID 与既有轮询终态；未创建第二个上游轮询器或改写其表。
- [x] Midjourney 终态图片只导入一次，服务重启后仍可恢复观察。
- [x] 阶段二没有取消 API、取消按钮或虚假的“已取消”状态。
- [x] `request_id` 可关联既有使用日志。
- [x] 普通 retry 创建新任务、新上游请求和独立账务记录。
- [x] `asset_import_failed` 在清单有效期内仅重试导入，不再上游请求或收费。
- [x] 同步派发中断、Midjourney processing、导入中断均按设计的恢复策略处理。

## 6. SVG 与文件安全

- [x] Claude SVG 使用固定非流式系统指令，用户不能覆盖它或请求工具调用。
- [x] 仅接受一个完整 SVG 根节点；文本、多个根、DOCTYPE/实体或截断 SVG 失败。
- [x] SVG 净化移除/拒绝脚本、事件属性、`foreignObject`、外部资源、危险 URL 和不允许的动画/滤镜。
- [x] SVG 以白名单重序列化为 `image/svg+xml`，前端不使用 `dangerouslySetInnerHTML`。
- [x] `ImportGeneratedAsset` 是服务端入口，不改变浏览器直传接口。
- [x] 导入复用既有 `file`、`user_file`、渠道配置与删除机制。
- [x] S3、WebDAV、ImgBed 的服务端导入实现均流式处理，不将整张文件读入内存。
- [x] 上游 URL 导入校验协议、user-info、DNS/IP、重定向、超时、MIME、魔数、声明和实际大小。
- [x] 环回、私网、链路本地、保留 IP 与重定向到这些地址均被拒绝。
- [x] 超限、非法 MIME、下载失败、SVG 净化失败不会产生成功任务、输出资产或持久化上游 URL。
- [x] 默认保存渠道被停用、删除或不满足策略时，绑定验证和任务提交会被阻止。

## 7. 前端与兼容性

- [x] 已阅读 `web/default/AGENTS.md` 并遵循 Bun 工作流。
- [x] `/imgen` 是认证的默认前端创作页，侧边栏不再 `native=true`。
- [x] 页面只显示本阶段图片工作区，不出现视频功能或占位实现。
- [x] 模型/分组选择只使用 bootstrap 返回的数据；切换会清理不符合新 schema 的字段。
- [x] 文件控件复用现有用户文件能力，只提交文件 ID。
- [x] 非终态任务使用退避轮询，终态停止；刷新后从持久历史恢复。
- [x] 作品卡使用服务端文件 URL；SVG 通过安全文件 URL 预览和下载。
- [x] 复用只回填安全参数和输入文件 ID，不复用渠道、价格、状态或临时 URL。
- [x] 保存失败优先显示无额外计费的导入重试；普通失败显示新任务重试。
- [x] 使用既有组件、主题语义变量、键盘操作与可访问性模式。
- [x] 所有新增可见文案使用 i18n，并完成所有支持语言翻译。
- [x] 旧 image playground 页面、`/api/image-playground/bootstrap` 与旧注册表回归行为未改变。

## 8. 测试与交付

- [x] 新增的 Go 测试使用 `testify/require` 与 `assert`。
- [x] 已覆盖 SQLite 迁移、唯一约束、状态 CAS 和跨数据库兼容建模。
- [x] 已覆盖未登录、越权用户、不可见能力、无 valid binding、无可用保存渠道和参数校验。
- [x] 已覆盖图片数量等计费边界、输入文件归属及未知字段/`stream` 拒绝。
- [x] 已覆盖 URL、b64、Responses、SVG、Midjourney 图片的成功与关键失败路径。
- [x] 已覆盖 MIME/大小/SSRF 重定向/SVG 恶意样本失败后不落成功资产。
- [x] 已覆盖绑定重验证、模型改名失效、普通重试与仅导入重试。
- [x] 已覆盖 `request_id` 使用日志关联和不重复计费。
- [x] 已运行相关旧图片 Relay、Midjourney、存储及 `GetImagePlaygroundBootstrap` 回归测试。
- [x] 已对所有修改 Go 文件运行 `gofmt`。
- [x] 已在 `web/default` 使用 Bun 运行 i18n 同步与最小类型检查/构建。
- [x] 已运行 `git diff --check`。
- [x] 未因本阶段修复/格式化无关问题；若有外部失败，交付说明已明确命令、原因和关联性。
- [x] 最终交付只报告实际变更、验证结果和阻塞项。
