# 创作台阶段一终版（2026-07-13）

> 本文是创作台阶段一的交付基线。若与 `Stage-One.md`、`Stage-One-Prompt.md` 或早期草案有冲突，以本文为准。

## 1. 目标与边界

阶段一交付“创作台管理目录”：管理员可以维护模型、能力、分组发布、渠道绑定及默认作品保存渠道，为后续用户创作工作区提供稳定配置源。

阶段一不交付以下内容：

- 用户创作请求、任务执行、轮询、取消、重试、作品导入与任务表。
- 图片、SVG、视频协议适配器和真实渠道分发。
- 旧 `image_playground.model_registry` 的读取、写入、迁移或兼容层。
- 旧在线生图流程的兼容；本次为破坏性变更，不以旧在线生图行为作为约束。

`abilities` 仍是既有渠道路由索引，不能作为创作模型目录。创作目录与渠道、分组和能力显式关联，避免仅按模型名误路由。

## 2. 管理结构

```text
模型目录
└── 能力
    └── 分组发布
        └── 渠道绑定
```

| 层级 | 作用 | 唯一性 | 排序 |
| --- | --- | --- | --- |
| 模型目录 | 定义一个可创作模型及其展示信息 | `model_key` 全局唯一 | `sort_order` 升序、ID 升序 |
| 能力 | 定义模型的一种图片或视频能力 | `model_id + category + operation + protocol` | `sort_order` 升序、ID 升序 |
| 分组发布 | 将能力发布给一个既有用户分组 | `capability_id + group_name` | `sort_order` 升序、ID 升序 |
| 渠道绑定 | 为发布项声明一个候选渠道 | `publication_id + channel_id` | `priority` 降序、ID 升序 |

前三层的“排序”只控制同级展示顺序。渠道绑定的“优先级”用于后续分发器选择候选渠道，数值越大优先级越高；它不修改既有渠道的全局优先级或权重。

## 3. 数据模型

所有新表通过 GORM 迁移，必须同时支持 SQLite、MySQL 和 PostgreSQL；JSON 配置使用 `TEXT` 保存，服务层通过 `common.Marshal` / `common.Unmarshal` 规范化。

### 3.1 `creative_model`

| 字段 | 规则 | 含义 |
| --- | --- | --- |
| `model_name` | 非空，最长 255 | 上游模型标识；必须与渠道能力中的模型名一致。 |
| `model_key` | `lower(trim(model_name))`，唯一 | 不可编辑的内部标准键，消除不同数据库的大小写差异。 |
| `display_name` | 为空时回退 `model_name` | 管理端及后续用户端展示名称，不参与路由。 |
| `vendor` | 固定预设选择 | 展示与分类信息；不决定协议或渠道。 |
| `description` | 可选 | 管理备注，不参与路由。 |
| `status` | `enabled` / `disabled` | 模型目录状态，不替代能力、发布或渠道状态。 |
| `sort_order` | 整数，默认 `0` | 模型目录展示顺序。 |

### 3.2 `creative_model_capability`

| 字段 | 规则 |
| --- | --- |
| `model_id` | 关联模型目录。 |
| `category` | 仅 `image`、`video`。 |
| `operation` | 图片：`generate`、`edit`；视频：`generate`、`remix`。 |
| `asset_kind` | 图片：`raster`、`vector`；视频：仅 `video`。 |
| `protocol` | 固定协议预设；纳入唯一性。 |
| `execution_mode` | `sync`、`async`、`sync_artifact`。 |
| `input_schema` | JSON 对象；阶段一允许 `{}`。 |
| `default_params` | JSON 对象；空值规范为 `{}`。 |
| `enabled` / `sort_order` | 能力可用状态与展示顺序。 |

SVG 不是独立顶级类别：它是 `image + generate + vector`，例如后续 `claude_svg` 适配能力。

### 3.3 `creative_model_publication`

| 字段 | 规则 |
| --- | --- |
| `capability_id` | 关联能力。 |
| `group_name` | 必须存在于既有可用用户分组。 |
| `enabled` / `sort_order` | 分组发布可用状态与展示顺序。 |
| `group_default_params` | JSON 对象；后续在能力默认参数之后合并。不得承载价格配置。 |

### 3.4 `creative_channel_binding`

| 字段 | 规则 |
| --- | --- |
| `publication_id` | 关联分组发布。 |
| `channel_id` | 现有 `channels.id`；同一发布项不可重复绑定同一渠道。 |
| `request_model` | 阶段一由一级模型目录继承，不在绑定弹框手工填写。 |
| `priority` | 后续候选渠道顺序，默认 `0`；不复制渠道全局优先级。 |
| `enabled` | 阶段一新建与更新均固定为 `false`，等待协议级校验与分发器落地。 |
| `validation_status` | 阶段一产生 `pending` 或 `invalid`。 |
| `validation_message` | 最近一次基础校验说明；不得记录密钥或上游响应正文。 |

## 4. 服务端规则

### 4.1 配置与存储

创作台设置保存于既有 `options` 表的 `creative_studio.settings`：

```json
{
  "version": 1,
  "default_file_channel_id": 0
}
```

默认作品保存渠道只能选择已启用且非本地类型的文件上传渠道；具体 S3、WebDAV 或其他渠道密钥仍只存在于既有文件上传渠道配置中。

### 4.2 校验与重复提示

新增和更新都先执行业务层重复校验，再由数据库唯一索引作为并发兜底。重复时返回具体对象：

- `模型目录：<模型名> 已存在`
- `能力：<分类>/<操作>｜<协议> 已存在`
- `分组：<分组名> 已存在`
- `渠道：<渠道名> 已存在`

能力校验类别、操作、媒体格式、执行方式与 JSON 对象；发布校验分组存在性；绑定检查渠道是否存在并启用，并检查该渠道的既有 `abilities` 是否含目录模型和发布分组。不匹配的绑定会保存为 `invalid`，但阶段一不会启用它。

父项存在下级记录时禁止删除：模型不能在存在能力时删除，能力不能在存在分组发布时删除，分组发布不能在存在渠道绑定时删除。

## 5. 管理 API

全部路由位于 `/api/creative/admin/*`，统一使用 `AdminAuth`。

| 资源 | 路由 |
| --- | --- |
| 聚合数据 | `GET /bootstrap` |
| 模型目录 | `GET/POST /models`、`PUT/DELETE /models/:id` |
| 能力 | `GET/POST /models/:id/capabilities`、`PUT/DELETE /capabilities/:id` |
| 分组发布 | `GET/POST /capabilities/:id/publications`、`PUT/DELETE /publications/:id` |
| 渠道绑定 | `GET/POST /publications/:id/bindings`、`PUT/DELETE /bindings/:id` |
| 创作台设置 | `GET/PUT /settings` |

`GET /bootstrap` 以一次读取返回模型、嵌套能力/发布/绑定、可用分组、候选渠道、文件上传渠道和创作台设置，供管理页构建分级视图。

## 6. 管理端交互基线

1. 模型目录、能力、分组、渠道绑定均采用逐级展开；外层字段完整展示。
2. 每个字段提供与弹框一致的说明；短说明紧邻标题，长说明以信息图标悬浮或点按查看，避免撑开卡片。
3. JSON 外层只展示可点击标签；悬浮提示可查看完整 JSON，点击后在弹框中查看。编辑时可维护 JSON 对象。
4. 编辑弹框限制在视口内，中间字段区可滚动，底部操作区固定可见。
5. 重复项在弹框内显示黄色警告并禁用保存；服务端仍强制校验。
6. 渠道选择器全宽显示，选项仅显示渠道名称；渠道 ID 仍在外层详情中展示。
7. 能力编辑可维护类别、操作、媒体格式、协议、执行方式、输入结构、默认参数、状态、排序；分组编辑可维护分组、状态、排序、分组默认参数；渠道绑定可维护渠道与优先级。

## 7. 验收清单

- [ ] 四张表已在主数据库迁移列表中注册，SQLite、MySQL、PostgreSQL 均可启动迁移。
- [ ] 管理 API 在未授权访问时被 `AdminAuth` 拦截。
- [ ] 四层新增和更新都能处理唯一性、引用存在性及 JSON 对象校验。
- [ ] 删除上层配置时不会级联删除下层配置。
- [ ] 管理页的字段、说明、中文枚举、JSON 查看、重复提示和弹框滚动符合第 6 节。
- [ ] 默认保存渠道不会接受本地或已禁用文件上传渠道。
- [ ] 旧注册表和旧在线生图流程没有迁移、回写或兼容代码。

## 8. 阶段二入口

阶段二开始前，必须为每个 `protocol` 定义真实请求路径、返回形式、异步查询、取消能力和结果导入契约；之后才实现用户创作 API、任务持久化、文件导入、真实渠道选择、计费结算和可观测性。

