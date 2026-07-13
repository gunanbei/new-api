# 创作台第一阶段详设：配置域

## 1. 文档状态

**状态：可实施设计。**

本阶段对应《[创作台设计初稿](../creative-studio-draft.zh-CN.md)》的“阶段 A：基础域”，但按需建模：只交付下一阶段图片创作真正需要的目录、发布、渠道绑定和存储配置域。任务、作品和上游适配器在其首次被使用的第二阶段实现，避免空表和无调用代码长期成为兼容负担。

## 2. 目标与完成定义

### 2.1 目标

1. 建立创作台模型目录，统一描述图片、视频两类能力；SVG 是 image + vector，不是第三个分类。
2. 让管理员在“运维 → 创作台管理”配置模型、能力、用户分组发布、候选渠道和默认作品保存渠道。
3. 创作台目录由管理员独立配置；旧 image_playground.model_registry 与旧生图入口保持独立，不进行导入或回写。
4. 不影响现有在线生图的实际调用、渠道路由、额度、计费、文件管理和用户 API Key 行为。

### 2.2 完成定义

- SQLite、MySQL、PostgreSQL 均可自动迁移四张创作台配置表。
- 管理员可完成“模型 → 能力 → 分组发布 → 渠道绑定 → 存储渠道”的配置闭环，且看不到任何渠道密钥。
- 管理员从现有启用模型中选择目录模型，并使用内置供应商、协议及能力枚举完成配置。
- 当前 /api/image-playground/bootstrap 与现有图片生成调用路径不改动；创作台尚不向普通用户发起生成请求。

## 3. 本阶段边界

| 纳入 | 不纳入 |
| --- | --- |
| 模型目录、能力、分组发布、渠道绑定的 GORM Model、迁移、管理 API 和管理页面 | 图片、SVG、视频的提交、轮询、取消、重试和任务恢复 |
| 旧图片注册表的安全导入与状态展示 | creative_task、creative_task_asset、作品历史和“我的作品” |
| 默认作品保存渠道的选择、校验和停用/删除保护 | 上游结果下载、SVG 净化、服务端文件导入和存储策略 |
| 侧边栏文字改为“创作台” | 将 /imgen 从当前可用入口切换为未完成的默认前端工作区 |
| 运维设置中的“创作台管理”页 | 聊天模型、音频、3D、公开作品或协作功能 |

**路由策略：** 本阶段仅把侧边栏文案由“在线生图”改为“创作台”，继续保留 /imgen 的既有 native 跳转及其生成能力。阶段二的图片工作区通过完整验收后，才将该入口切换到 web/default 认证路由。这样不会在生成协议尚未接通时让用户失去现有可用功能。

## 4. 现状与原则

### 4.1 现有事实

- abilities 的主键为 group + model + channel_id，它是渠道选路索引，不是模型能力目录。
- 旧 image_playground.model_registry 只包含 model_name、adapter_type、display_vendor、enabled。当前 Bootstrap 还会以用户 Token 的分组和 abilities 过滤模型。
- 文件管理已有 file_upload_channel、file、user_file；创作台只引用其渠道 ID，不复制存储密钥或 Provider 配置。
- /api/option 使用 Root 权限，文件上传渠道与渠道管理使用 Admin 权限。创作台管理遵从后者，使用 AdminAuth。

### 4.2 原则

1. **目录不等于路由。** 目录记录产品能力，abilities 仍是现有渠道可用性的事实来源。
2. **先配置、后启用。** 本阶段渠道绑定只能处于“待协议校验”状态，不能让普通用户通过新 API 调用它。
3. **旧数据只导入、不回写。** 自动导入不得覆盖管理员在新管理页做出的修改。
4. **全部 JSON 为 TEXT。** 使用 common.Marshal / common.Unmarshal，不使用数据库专属 JSON 类型或 MySQL DDL。
5. **历史按需建表。** 任务与资产只有在阶段二开始写入时再迁移；数据库结构不预埋未使用的业务域。

## 5. 数据模型

### 5.1 实体关系

~~~mermaid
erDiagram
    CREATIVE_MODEL ||--o{ CREATIVE_MODEL_CAPABILITY : owns
    CREATIVE_MODEL_CAPABILITY ||--o{ CREATIVE_MODEL_PUBLICATION : published_as
    CREATIVE_MODEL_PUBLICATION ||--o{ CREATIVE_CHANNEL_BINDING : candidates
    FILE_UPLOAD_CHANNEL ||--o{ CREATIVE_STUDIO_SETTINGS : selected_by
    CHANNEL ||--o{ CREATIVE_CHANNEL_BINDING : selected_by
~~~

creative_studio_settings 不建表，使用单个 Option 值 creative_studio.settings；上图仅表示其对文件上传渠道的引用关系。

### 5.2 creative_model

全局目录中的一个模型，而不是现有 channels 或 abilities 的替代品。

| 字段 | 类型/约束 | 说明 |
| --- | --- | --- |
| id | GORM 主键 uint64 | 内部稳定 ID。 |
| model_name | varchar(255) | 保留展示和实际路由使用的规范模型名。 |
| model_key | varchar(255)，唯一索引 | strings.ToLower(strings.TrimSpace(model_name))，解决三种数据库中大小写唯一性差异。 |
| display_name | varchar(255) | 为空时服务层规范为 model_name。 |
| vendor | varchar(255) | 管理端与后续用户端展示的提供方名称。 |
| description | text | 可选描述。 |
| status | varchar(16)，索引 | enabled 或 disabled；由服务层显式赋值。 |
| sort_order | int | 同级排序，默认由服务层写入 0。 |
| created_by / updated_by | int64 | 最近操作管理员 ID；导入时为 0。 |
| created_at / updated_at | GORM 时间字段 | 审计时间。 |

### 5.3 creative_model_capability

一条记录表示模型的一种可配置创作能力；同一模型可以同时具备图片和视频能力，也可以有多个协议实现。

| 字段 | 类型/约束 | 说明 |
| --- | --- | --- |
| id | GORM 主键 uint64 | 内部稳定 ID。 |
| model_id | uint64，索引 | 逻辑关联 creative_model.id，不建物理外键。 |
| category | varchar(16)，索引 | 仅 image、video。 |
| operation | varchar(32) | 首期预定义：图片 generate / edit，视频 generate / remix。 |
| asset_kind | varchar(16) | 图片仅 raster / vector；视频仅 video。Claude SVG 为 image + generate + vector。 |
| protocol | varchar(64) | 协议标识，例如旧注册表的 openai_images；阶段一只保存，不执行。 |
| execution_mode | varchar(32) | sync、async 或 sync_artifact，供后续适配器选择。 |
| input_schema | text | JSON Schema 子集；本阶段允许 {}，阶段二才渲染及执行校验。 |
| default_params | text | JSON 默认参数；空值规范为 {}。 |
| enabled | bool，索引 | 由服务层显式赋值。 |
| sort_order | int | 能力排序。 |
| 审计字段 | 同模型表 | 创建、更新时间及操作人。 |

唯一索引为 model_id + category + operation + protocol。把 protocol 纳入唯一性，避免同一个模型通过不同上游协议提供同一操作时被错误合并。

### 5.4 creative_model_publication

发布是“某一能力对某一用户分组可见”的配置，计费仍完全复用既有模型倍率、分组倍率和 Relay 预扣费链路。

| 字段 | 类型/约束 | 说明 |
| --- | --- | --- |
| id | GORM 主键 uint64 | 内部稳定 ID。 |
| capability_id | uint64，索引 | 逻辑关联能力。 |
| group_name | varchar(64)，索引 | 不使用 SQL 保留词 group。 |
| enabled | bool，索引 | 后续用户端可见性的必要条件。 |
| sort_order | int | 当前分组下的显示排序。 |
| group_default_params | text | JSON；后续与能力默认值合并，不能承载价格配置。 |
| 审计字段 | 同模型表 | 创建、更新时间及操作人。 |

唯一索引为 capability_id + group_name。

### 5.5 creative_channel_binding

绑定将一个分组发布项与现有渠道关联；它避免后续仅按同名模型随机选择到不支持目标协议的渠道。

| 字段 | 类型/约束 | 说明 |
| --- | --- | --- |
| id | GORM 主键 uint64 | 内部稳定 ID。 |
| publication_id | uint64，索引 | 逻辑关联分组发布。 |
| channel_id | int，索引 | 现有 channels.id。 |
| request_model | varchar(255) | 实际发给该渠道的模型名；为空时后续使用目录模型名。 |
| priority | int | 后续适配器内的候选顺序，默认 0。不复制渠道全局权重。 |
| enabled | bool，索引 | 阶段一新建记录固定为 false。 |
| validation_status | varchar(16)，索引 | pending、invalid、verified。阶段一仅产生 pending / invalid。 |
| validation_message | text | 最近一次基础校验结果；不记录密钥或上游响应正文。 |
| 审计字段 | 同模型表 | 创建、更新时间及操作人。 |

唯一索引为 publication_id + channel_id。一个渠道可服务多个分组发布项，但同一发布项不会重复绑定到相同渠道。

### 5.6 全局设置 Option

creative_studio.settings 的值为以下 JSON，存入现有 Option 表并通过现有 Option 缓存刷新流程生效：

~~~json
{
  "version": 1,
  "default_file_channel_id": 0
}
~~~

- default_file_channel_id = 0 表示尚未配置，阶段二不得提交生成任务。
- 只能选中启用的非本地文件上传渠道；只返回 ID、名称、类型、状态，绝不返回 config_profile。
- 禁用或删除被该设置引用的文件上传渠道时，现有文件渠道控制器先检查此 Option；引用存在则拒绝操作，要求管理员先切换创作台默认渠道。

## 6. 数据初始化

主库 migrateDB 和 migrateDBFast 只创建四张创作台配置表，不导入、删除或改写 image_playground.model_registry。旧生图配置和创作台目录是两个独立配置源。

## 7. 管理 API 与校验

所有 API 位于 /api/creative/admin，使用 middleware.AdminAuth()；响应遵循现有 success / message / data 包装。

| 方法 | 路径 | 行为 |
| --- | --- | --- |
| GET | /bootstrap | 返回目录树、可选分组、无密钥渠道摘要、文件渠道摘要和设置。 |
| GET/POST | /models | 列表、新建模型。 |
| PUT/DELETE | /models/:id | 更新、停用或删除未被能力引用的模型。 |
| GET/POST | /models/:id/capabilities | 列表、新建能力。 |
| PUT/DELETE | /capabilities/:id | 更新、停用或删除未发布能力。 |
| GET/POST | /capabilities/:id/publications | 列表、新建分组发布。 |
| PUT/DELETE | /publications/:id | 更新、停用或删除无绑定发布。 |
| GET/POST | /publications/:id/bindings | 列表、新建候选渠道绑定并执行基础校验。 |
| PUT/DELETE | /bindings/:id | 更新、删除绑定；阶段一拒绝把 pending 绑定设为启用。 |
| GET/PUT | /settings | 读取或更新默认作品保存渠道。 |

### 7.1 写入校验

| 对象 | 必须校验 |
| --- | --- |
| 模型 | 名称 trim 后非空且不超过 255；model_key 全局唯一；status 合法。 |
| 能力 | category、operation、asset_kind、execution_mode 合法；vector 只能用于 image，video 只能用于 video；两个 JSON 字段必须是对象。 |
| 发布 | 分组必须存在；同一能力和分组只能一条发布；启用前提示没有可验证绑定，但阶段一不产生用户侧调用。 |
| 绑定 | 渠道存在且启用；该渠道的 abilities 必须包含发布分组及目录模型名；不匹配时保存为 invalid 并拒绝启用。协议级路径和轮询能力在阶段二适配器清单中校验。 |
| 设置 | 文件渠道存在、已启用且类型不是 local；禁止返回或接受渠道密钥配置。 |

删除遵循最小保护：存在下游记录时返回冲突，不做级联删除。管理员应先停用，再删除从属配置。

## 8. 管理端与导航

### 8.1 页面位置

- 新增运维子页：/system-settings/operations/creative-studio。
- 页面复用 OperationsSettings 的 section registry、现有 SettingsPage、表格、Dialog、Form、Select 与主题变量。
- 系统设置路由继续仅限超级管理员；后端 API 再用 AdminAuth 防止脱离前端的越权请求。

### 8.2 管理页内容

~~~text
创作台管理
├── 作品存储：默认文件上传渠道
├── 模型目录：名称、展示名、厂商、状态、排序
└── 能力与发布
    ├── 图片 / 视频能力、输出类型、协议、执行方式
    ├── 分组发布、状态、排序、默认参数
    └── 候选渠道、基础校验状态、实际模型名、优先级
~~~

页面使用现有亮色、暗色和系统主题语义变量，不新增独立色板。所有用户可见文案在实现时通过 i18n 提供六种前端语言翻译。

### 8.3 用户入口

- 现有侧边栏项目的翻译键改为 Creative Studio / “创作台”。
- 地址保持 /imgen，且本阶段继续 native: true，因此旧链接、书签和现有图片生成功能不变。
- 不增加空壳普通用户页面，也不把“尚未接通”的目录暴露为可提交表单。阶段二在图片适配器、持久化任务和文件导入均可用后，才移除 native 并接入默认主题工作区。

## 9. 实施顺序

1. 新增四个 GORM Model、仓储查询和 AutoMigrate 双路径；为 JSON 读写使用 common.* 包装。
2. 实现模型、能力、发布、绑定、设置的管理服务和 /api/creative/admin/* 路由。
4. 在文件上传渠道的停用/删除流程增加默认创作台存储渠道引用检查。
5. 在运维设置中加入“创作台管理”分区；旧图片模型设置继续独立服务旧在线生图。
6. 将侧边栏文案改为“创作台”，不变更 /imgen 的实际跳转方式。

## 10. 验收与测试

### 10.1 后端必测

1. SQLite 内存库下 migrateDB、migrateDBFast 都创建四张表和唯一索引；MySQL、PostgreSQL 集成环境运行相同迁移用例。
2. 管理 API 覆盖现有模型下拉、固定供应商和协议预设，以及渠道绑定校验。
3. 管理 API 覆盖：未登录、非管理员、合法 CRUD、无效 JSON、跨类型 asset_kind、重复分组发布、无能力/禁用渠道绑定。
4. 默认文件渠道覆盖：无渠道、停用渠道、local 渠道、被设为默认后的停用和删除拒绝。
5. 回归：GetImagePlaygroundBootstrap 的既有测试继续通过，证明旧 Option 与按分组能力过滤不受影响。

### 10.2 前端必测

1. 亮、暗、系统主题下管理页的表单、表格、状态标签和禁用态可读。
2. 旧图片模型配置与创作台目录保持独立，互不导入或回写。
3. 侧边栏展示“创作台”，点击仍访问原 /imgen 行为。
4. 在 web/default/ 使用 Bun 执行类型检查、i18n 同步和生产构建。

## 11. 阶段一后的交接条件

进入阶段二前，必须同时满足：

1. 管理员已为至少一个图片能力配置启用分组发布、待校验渠道绑定和默认作品保存渠道。
2. 适配器清单已逐项确认该协议的提交路径、结果格式、计费链路和可否异步。
3. 服务端生成资产导入、远程 URL 安全下载、SVG 白名单净化的安全设计及测试已经评审。

满足后，阶段二新增 creative_task 与 creative_task_asset、用户 Creative API、图片/SVG 适配器和默认主题的图片工作区；现有 /imgen 再切换到真正的创作台页面。
