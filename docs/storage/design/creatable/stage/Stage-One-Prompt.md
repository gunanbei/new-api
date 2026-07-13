# 创作台第一阶段实施提示词

将以下内容完整交给负责实现和自检的 Agent。

---

你正在实现创作台第一阶段“配置域”。以 [Stage-One.md](./Stage-One.md) 为唯一业务范围，以仓库根目录 AGENTS.md 为工程约束。先阅读两份文件，再开始任何改动。

## 目标

完成创作台的配置域，不实现实际生成：

1. 建立模型目录、模型能力、分组发布、渠道绑定四个跨数据库数据模型。
2. 让管理员独立配置创作台目录，不读取、迁移或回写旧 image_playground.model_registry。
3. 提供 AdminAuth 保护的创作台管理 API。
4. 在“运维 → 创作台管理”提供管理页面，并配置默认作品保存渠道。
5. 将侧边栏文案改为“创作台”，但保留当前 /imgen 的 native 跳转和旧生图功能。
6. 为上述行为提供最小且有价值的测试和验证。

## 不允许实现

- 不新增 creative_task、creative_task_asset 或任何作品历史表。
- 不实现图片、SVG、视频的提交、轮询、取消、重试、文件导入或 SVG 净化。
- 不新增用户侧 /api/creative API，不让浏览器持有上游 Key。
- 不将 /imgen 切换为默认前端工作区，不修改或删除旧 image_playground.model_registry。
- 不改动既有 Relay、计费、额度、渠道选路和现有图片生成功能。
- 不添加新依赖、独立主题系统、通用工作流引擎或聊天模型支持。

## 工作方式

1. 先执行 git status，工作区可能已有用户改动；只改本阶段直接涉及的文件，绝不重置、覆盖或格式化无关文件。
2. 如仓库存在 .codegraph，定位和理解代码前先使用 CodeGraph；再阅读实际关联的 Model、迁移、路由、控制器、文件上传渠道和默认前端设置代码。
3. 后端遵守 Router → Controller → Service → Model 分层。所有 JSON 编解码使用 common.Marshal、common.Unmarshal、common.UnmarshalJsonStr 或 common.DecodeJson。
4. 所有数据库改动必须同时支持 SQLite、MySQL 5.7.8+、PostgreSQL 9.6+：使用 GORM，不写 AUTO_INCREMENT、SERIAL、数据库 JSON 类型、专属 UPSERT 或物理外键。
5. 前端改动前先阅读 web/default/AGENTS.md；新增或变更用户可见文案前必须使用 i18n-translate skill，并使用 Bun。
6. 不写可选过程性说明；在完成时仅报告实际变更、执行过的验证及未解决阻塞。

## 必须实现的后端行为

### 数据模型

新增并注册以下 GORM Model，使用逻辑关联，不建立物理外键：

| 表 | 关键字段及约束 |
| --- | --- |
| creative_model | model_name、全局唯一且规范化的 model_key、display_name、vendor、description、status、sort_order、创建/更新操作人和时间。 |
| creative_model_capability | model_id、category、operation、asset_kind、protocol、execution_mode、input_schema TEXT、default_params TEXT、enabled、sort_order、审计字段；唯一键 model_id + category + operation + protocol。 |
| creative_model_publication | capability_id、group_name、enabled、sort_order、group_default_params TEXT、审计字段；唯一键 capability_id + group_name。 |
| creative_channel_binding | publication_id、channel_id、request_model、priority、enabled、validation_status、validation_message、审计字段；唯一键 publication_id + channel_id。 |

枚举与校验：

- category 仅 image、video。
- 图片 asset_kind 仅 raster、vector；视频仅 video。
- Claude SVG 使用 image + generate + vector + claude_svg，不建第三种用户分类。
- execution_mode 仅 sync、async、sync_artifact。
- 所有 JSON 字段只能接受 JSON 对象；空值规范为 {}。
- 不依赖 GORM 的 default:true；由服务层/构造逻辑明确写入业务默认值。

将四张表同时加入 migrateDB 和 migrateDBFast 的 AutoMigrate 路径。

### 独立配置

创作台不包含旧注册表迁移、迁移标记或重试 API。旧 image_playground.model_registry 和 /api/image-playground/bootstrap 保持原有独立行为，创作台只从管理员写入的四张配置表读取数据。

### 管理 API

在 /api/creative/admin 下注册并使用 middleware.AdminAuth()：

- GET /bootstrap：目录树、分组、无密钥渠道摘要、文件渠道摘要和设置。
- 模型、能力、分组发布、渠道绑定的 CRUD API，路径和对象关系遵循 Stage-One.md。
- GET/PUT /settings：维护 creative_studio.settings。

控制器不得返回 channel key、config_profile 或其他存储/渠道凭据。写入必须验证：

- model_key 唯一，模型名称非空且不超过 255。
- group_name 存在；同一能力和分组不可重复发布。
- 绑定渠道存在、启用，且 abilities 含该 group_name 和模型；失败时为 invalid 且不可启用。
- 本阶段不得将 pending 或 invalid 绑定设为 enabled=true。
- default_file_channel_id 只能是启用的非 local 文件上传渠道。
- 模型、能力、发布存在下游引用时，返回冲突而非级联删除。

在文件上传渠道的停用和删除路径加入保护：若其被 creative_studio.settings 引用为默认保存渠道，拒绝操作，要求先切换设置。

### 前端

- 在 OperationsSettings section registry 中新增 creative-studio 区段，路由为 /system-settings/operations/creative-studio。
- 页面包含：默认作品保存渠道、模型目录、能力、分组发布、候选渠道绑定及基础校验状态。
- 模型名来自现有启用模型下拉；供应商、协议、分类、操作、资产类型和执行方式使用固定下拉预设。
- 复用现有 SettingsPage、表格、Dialog、Form、Select、toast 和主题语义变量；不创建独立 UI 基础设施。
- 原“图片模型配置”页继续独立维护旧在线生图配置，不与创作台互相导入或回写。
- 侧边栏显示“创作台”，保持 url=/imgen 与 native=true。
- 所有可见文本走 i18n，完成所有仓库支持语言的翻译。

## 验证要求

至少执行并报告：

1. 针对管理服务/API、文件渠道保护的 Go 测试；测试使用 testify 的 require/assert。
2. 现有 GetImagePlaygroundBootstrap 相关回归测试。
3. gofmt 检查所有修改的 Go 文件。
4. 在 web/default 使用 Bun 运行 i18n 同步、类型检查/构建中项目已有的最小可用命令。
5. git diff --check。

若某项命令因现有无关问题失败，不修复无关问题；说明命令、失败原因和与本阶段的关系。

## 完成标准

只有在 Stage-One.md 的“完成定义”“写入校验”“验收与测试”均满足后才结束。最终回复只包含：

- 修改的核心文件与行为；
- 已通过的验证；
- 未完成项或明确阻塞（如无则写“无”）。

---
