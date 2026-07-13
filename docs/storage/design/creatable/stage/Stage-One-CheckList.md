# 创作台第一阶段检查清单

本清单与 [Stage-One.md](./Stage-One.md) 配套使用。全部勾选后，第一阶段才可交付。

## 1. 开始前

- [ ] 已阅读根目录 AGENTS.md、[Stage-One.md](./Stage-One.md) 及涉及目录的 AGENTS.md。
- [ ] 已检查 git status，并确认未覆盖用户已有改动。
- [ ] 已在需要定位或理解代码时优先使用 CodeGraph。
- [ ] 已确认本阶段不实现任务、作品、协议适配器、文件导入或用户 Creative API。

## 2. 数据与初始化

- [ ] 新增 creative_model，含规范化 model_key 唯一索引。
- [ ] 新增 creative_model_capability，含 model_id + category + operation + protocol 唯一索引。
- [ ] 新增 creative_model_publication，含 capability_id + group_name 唯一索引。
- [ ] 新增 creative_channel_binding，含 publication_id + channel_id 唯一索引。
- [ ] 表字段不使用物理外键、数据库专属 JSON 类型、AUTO_INCREMENT 或 SERIAL。
- [ ] JSON 字段为 TEXT，读写通过 common.* JSON 包装。
- [ ] migrateDB 已注册四张表。
- [ ] migrateDBFast 已注册四张表。
- [ ] SQLite、MySQL、PostgreSQL 的字段长度、布尔默认值、索引策略均可兼容。
- [ ] 没有依赖 GORM default:true 作为业务默认值。

## 3. 独立配置

- [ ] 不读取、迁移或回写 image_playground.model_registry。
- [ ] 不创建 creative_studio.registry_migration_v1 或迁移重试接口。
- [ ] 模型名来自现有启用模型下拉。
- [ ] 供应商、协议、分类、操作、资产类型和执行方式使用固定下拉预设。

## 4. 管理 API 与权限

- [ ] 所有新路由位于 /api/creative/admin。
- [ ] 所有新路由均经过 middleware.AdminAuth()。
- [ ] GET /bootstrap 不泄露渠道 Key、文件渠道 config_profile 或其他凭据。
- [ ] 模型 CRUD 支持状态、排序和审计字段。
- [ ] 能力 CRUD 校验 category、operation、asset_kind、protocol、execution_mode 和 JSON 对象。
- [ ] vector 仅可用于 image，video 仅可用于 video。
- [ ] Claude SVG 能力被表达为 image + generate + vector + claude_svg。
- [ ] 发布 CRUD 校验 group_name 存在且不可重复。
- [ ] 绑定 CRUD 校验渠道存在、启用且 abilities 匹配分组和模型。
- [ ] 不匹配绑定标为 invalid，且不能启用。
- [ ] pending、invalid 绑定在阶段一不能设为 enabled=true。
- [ ] 删除存在下游引用的模型、能力或发布时返回冲突，不级联删除。
- [ ] GET/PUT /settings 仅接受启用、非 local 的文件上传渠道。

## 5. 文件渠道保护

- [ ] creative_studio.settings 使用现有 Option 体系，版本为 1。
- [ ] 默认保存渠道未设置时 default_file_channel_id 为 0。
- [ ] 设置响应只暴露文件渠道 ID、名称、类型、状态。
- [ ] 停用被默认设置引用的文件上传渠道会被拒绝。
- [ ] 删除被默认设置引用的文件上传渠道会被拒绝。
- [ ] 切换到另一个有效渠道后，原渠道可按原有规则停用或删除。

## 6. 前端与兼容性

- [ ] 已阅读 web/default/AGENTS.md 并遵循 Bun 工作流。
- [ ] 所有新增可见文案使用 useTranslation 和 i18n 键。
- [ ] 已按 i18n-translate skill 更新所有支持语言。
- [ ] 运维设置新增 /system-settings/operations/creative-studio。
- [ ] 管理页面包含存储渠道、模型、能力、发布、绑定。
- [ ] 页面复用现有 SettingsPage、表单、表格、Dialog 和主题变量。
- [ ] 亮色、暗色、系统主题均无固定颜色可读性问题。
- [ ] 旧“图片模型配置”页与创作台保持独立。
- [ ] 侧边栏显示“创作台”。
- [ ] 侧边栏仍使用 /imgen 且 native=true。
- [ ] /api/image-playground/bootstrap 和现有在线生图行为未改变。

## 7. 测试与交付

- [ ] 新增的 Go 测试使用 testify require/assert。
- [ ] 已覆盖 SQLite 迁移和唯一约束。
- [ ] 已覆盖模型唯一性、输入校验、重复发布、无效绑定和默认渠道保护。
- [ ] 已覆盖未登录、非管理员、输入校验、重复发布、无效绑定和默认渠道保护。
- [ ] 已通过既有 GetImagePlaygroundBootstrap 回归测试。
- [ ] 已对所有修改的 Go 文件运行 gofmt。
- [ ] 已在 web/default 使用 Bun 执行 i18n 同步和项目最小类型检查/构建。
- [ ] 已运行 git diff --check。
- [ ] 未因本阶段修复或格式化无关问题；如有外部失败，已在交付说明中明确列出。
- [ ] 最终交付只报告实际变更、验证结果和阻塞项。
