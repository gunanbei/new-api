# 在线生图图片模型配置改造方案

## 1. 背景

当前在线生图的 `/api/image-playground/bootstrap` 使用硬编码函数 `common.IsImageGenerationModel()` 判断什么模型算图片模型，并且前端 `gpt-image-playground` 的 Key 选择器按三层结构渲染：

- 第 1 层：API Key
- 第 2 层：分组
- 第 3 层：图片模型

现在需求已经收敛为：

1. 不修改 `models` 表，不新增数据库字段。
2. 在系统设置的 models 区域新增独立的「图片模型配置」入口。
3. 图片模型配置使用 `option` JSON 持久化，不使用模型表字段，不做纯 Redis 临时存储。
4. 图片模型配置只影响 `/api/image-playground/bootstrap`，不影响其他模型接口、路由、计费和通用模型元数据。
5. 一个 API Key 在在线生图场景下只对应一个 `token.group`，因此选择器应改为两层结构：
   - 第 1 层：API Key
   - 第 2 层：该 Key 下的图片模型列表
6. `group` 作为 Key 的元信息展示，不再占用独立一层交互。

## 2. 目标

### 2.1 功能目标

为管理员提供一份独立的图片模型注册表，用于声明：

- 某个模型是否可作为在线生图图片模型
- 该模型对应的协议 / 适配器类型
- 该模型在后台和 bootstrap 中展示的厂家 / 产品名

### 2.2 行为目标

`/api/image-playground/bootstrap` 的模型来源改为：

1. 对每个 token，只取唯一的 `token.group`
2. 查询该 group 下所有启用渠道对应的模型
3. 叠加用户权限、token `model_limits`、ability/channel 启用状态
4. 用图片模型配置过滤出启用的图片模型
5. 按模型名忽略大小写去重
6. 返回两层结构所需的数据

## 3. 非目标

本次改造不做以下事情：

1. 不修改通用 `models` 表结构
2. 不替换现有模型管理页 CRUD
3. 不修改 `/api/user/models`
4. 不修改计费、倍率、配额逻辑
5. 不让 `adapter_type` 立即驱动 `gpt-image-playground` 的多协议发图
6. 不实现多分组 Key 选择器

## 4. 最终交互

## 4.1 系统设置入口

在 `new-api/web/default/src/features/system-settings/models/section-registry.tsx` 中新增一个 section：

- 位置：`Global Model Configuration` 和 `Routing Reliability` 之间
- 标题：`Image Model Configuration`

## 4.2 在线生图选择器

从当前三层结构：

- Key
- Group
- Model

改为两层结构：

- Key
- Model

其中 Key 行展示以下元信息：

- `group`
- `group_display_name`
- `group_ratio`
- 掩码 key
- 图片模型数量
- `model_limits` 摘要

模型列表直接挂在 Key 下展开。

## 5. 配置存储设计

## 5.1 option key

新增独立 option key：

- `image_playground.model_registry`

值为 JSON 字符串。

## 5.2 JSON 结构

建议结构如下：

```json
{
  "version": 1,
  "items": [
    {
      "model_name": "gpt-image-1",
      "enabled": true,
      "adapter_type": "openai_images",
      "display_vendor": "OpenAI Images"
    }
  ]
}
```

## 5.3 字段定义

每个 item 包含：

- `model_name`: 模型名，字符串
- `enabled`: 是否启用，布尔值
- `adapter_type`: 协议 / 适配器类型，字符串枚举
- `display_vendor`: 展示用厂家 / 产品名，字符串

## 5.4 adapter_type 枚举

第一版内置以下值：

- `openai_images`
- `openai_responses_image_generation`
- `gemini_imagen`
- `fal`
- `replicate`
- `stability`
- `midjourney`
- `ideogram`
- `flux`
- `custom`

后台显示时使用对应文案，例如：

- `openai_images` -> `OpenAI Images`
- `gemini_imagen` -> `Google Imagen`
- `flux` -> `FLUX`

## 6. 后端改造方案

## 6.1 配置读取

在后端新增一组轻量 helper，用于：

1. 从 `common.OptionMap["image_playground.model_registry"]` 读取 JSON
2. 解析为结构体
3. 对模型名构建忽略大小写索引

建议新增文件：

- `new-api/service/image_playground_model_registry.go`

该文件职责仅限：

- 定义 registry 结构体
- 解析 option JSON
- 提供按模型名查找启用图片模型配置的函数

不需要新表，不需要新缓存层。

## 6.2 bootstrap 模型来源改造

当前入口：

- `new-api/controller/user.go`
- `GetImagePlaygroundBootstrap()`
- `imagePlaygroundModelsForToken()`

当前问题：

1. 只调用 `model.GetGroupEnabledModels(group)`
2. 通过 `common.IsImageGenerationModel()` 过滤图片模型
3. 返回单元素 `groups[]`

需要改为：

### A. 模型来源

仍然从该 group 下的 enabled ability 获取模型名集合。现有 `model.GetGroupEnabledModels(group)` 已经能满足“分组下所有启用渠道模型并集”的需求，因为它本质上是从 `abilities` 表对 `model` 做 `distinct`。

因此这里不需要自己再逐渠道遍历一遍。现有函数已经是更短的正确路径。

### B. 过滤逻辑

把 `common.IsImageGenerationModel(modelName)` 替换为：

1. 先 trim
2. 再按忽略大小写查图片模型 registry
3. 只有 registry 中存在且 `enabled=true` 的模型才能进入 bootstrap

### C. 去重逻辑

当前 `seen[modelName]` 是大小写敏感。

需要改成忽略大小写去重，例如：

- key 使用 `strings.ToLower(strings.TrimSpace(modelName))`

### D. token model limits

保留当前 `token.ModelLimitsEnabled` 过滤逻辑。

### E. 返回字段

`imagePlaygroundBootstrapModel` 结构体需要扩展字段：

- `adapter_type`
- `display_vendor`

用于未来前端扩协议和显示厂家文案。

## 6.3 bootstrap 返回结构调整

当前返回：

- `tokens[].groups[0].models[]`

新结构建议改为两层语义：

```json
{
  "tokens": [
    {
      "id": 1,
      "name": "Hiyo",
      "group": "hiyo",
      "group_display_name": "hiyo",
      "group_ratio": 0.25,
      "model_limits_enabled": true,
      "model_limits": "gpt-image-1,imagen-3.0-generate-002",
      "masked_key": "sk-xxxx...yyyy",
      "disabled": false,
      "models": [
        {
          "id": "gpt-image-1",
          "name": "gpt-image-1",
          "type": "image",
          "ratio": 2,
          "adapter_type": "openai_images",
          "display_vendor": "OpenAI Images",
          "disabled": false
        }
      ]
    }
  ]
}
```

这样更贴合最终 UI。

### 兼容建议

如果希望后端改动更平滑，也可以短期保留旧字段，同时新增新字段：

- 保留 `groups`
- 新增 `group_display_name`
- 新增 `group_ratio`
- 新增 `models`

然后在 `gpt-image-playground` 切到新字段后，再删旧字段。

如果追求最短路径，直接一次切到新字段也可以。

## 7. 前端系统设置改造方案

## 7.1 新 section

目标文件：

- `new-api/web/default/src/features/system-settings/models/section-registry.tsx`

新增 section：

- id: `image-model-config`
- titleKey: `Image Model Configuration`

位置放在：

- `global`
- `image-model-config`
- `routing-reliability`

## 7.2 新 section 组件

建议新增文件：

- `new-api/web/default/src/features/system-settings/models/image-model-config-section.tsx`

功能尽量轻量：

1. 从 `useSystemOptions()` 读取 `image_playground.model_registry`
2. 解析 JSON
3. 表格显示 items
4. 支持新增、编辑、删除、启用开关
5. 保存时调用 `useUpdateOption()`

## 7.3 录入方式

`model_name` 使用：

- 下拉选择现有模型名
- 同时允许手输

也就是组合输入，不做纯自由文本，也不做纯限制下拉。

模型名候选来源建议复用现有接口之一：

- `/api/channel/models_enabled`
- 或 `/api/models/`

如果以“真实线上可用模型”为准，优先 `channel/models_enabled`。
如果以“模型元数据目录”为准，优先 `/api/models/`。

本需求更偏 bootstrap 实际可用性，建议优先使用：

- `/api/channel/models_enabled`

## 7.4 adapter type 选择

使用下拉框。

字段：

- value: `adapter_type`
- label: 厂家 / 产品文案

## 7.5 display_vendor

默认值由 `adapter_type` 自动带出。第一版建议允许手工覆盖，便于运营定制展示文案。

## 8. gpt-image-playground 改造方案

## 8.1 改造目标

`gpt-image-playground` 需要从三层选择器改成两层选择器。

受影响文件：

- `gpt-image-playground/src/lib/newApiKeySync.ts`
- `gpt-image-playground/src/components/NewApiKeySelector.tsx`

## 8.2 newApiKeySync 数据结构调整

当前：

- `NewApiGroupOption`
- `NewApiKeyTreeOption.groups`

新结构建议：

```ts
export type NewApiModelOption = {
  id: string
  name: string
  type: 'image'
  ratio: number
  adapterType?: string
  displayVendor?: string
  disabled: boolean
}

export type NewApiKeyTreeOption = NewApiKeyOption & {
  disabled: boolean
  groupDisplayName: string
  groupRatio: number
  models: NewApiModelOption[]
}
```

不再需要：

- `NewApiGroupOption`
- `expandedGroup`
- `groups[]`

## 8.3 normalizeBootstrapTokens 调整

当前它把后端结果规范成：

- `token -> groups -> models`

需要改成：

- `token -> models`

并把 group 信息直接挂在 token 节点上：

- `group`
- `groupDisplayName`
- `groupRatio`

## 8.4 NewApiKeySelector 调整

当前组件依赖：

- `expandedGroup`
- `item.groups.map(...)`
- `group.models.map(...)`

需要改为：

1. 保留 `expandedKeyId`
2. 删除 `expandedGroup`
3. Key 行点击后直接展开 `item.models`
4. Key 副标题显示：
   - `groupDisplayName`
   - `maskKey(item.key)`
   - `summarizeModelLimits(item.modelLimits)`
   - `X{item.groupRatio}`

## 8.5 选中逻辑调整

当前：

- `selectModel(item, group, model)`
- `buildNewApiSelection(item, group, model)`

新逻辑：

- `selectModel(item, model)`
- group 直接使用 `item.group`

`applyNewApiSelectionToActiveProfile()` 不需要重写主流程，只需保证仍然写入：

- `apiKey`
- `group`
- `model`
- `apiMode = images`
- `baseUrl = /v1`

## 9. 兼容策略

## 9.1 短期兼容

短期建议：

1. 后端 bootstrap 同时返回旧字段和新字段
2. 前端先切到新字段
3. 稳定后删除旧字段

这样可以降低前后端联调风险。

## 9.2 空配置行为

当 `image_playground.model_registry` 为空或非法时：

1. bootstrap 不再回退到硬编码图片模型判定
2. 直接视为没有配置图片模型
3. token 对应的图片模型列表为空

这样管理员会明确感知“需要配置图片模型”，而不是被旧规则偷偷兜底。

如果希望更温和，也可以在第一版加一个临时兜底：

- registry 为空时退回 `common.IsImageGenerationModel()`

但按当前需求，“定义权交给管理员配置”更符合目标，因此建议不兜底。

## 10. 实施顺序

建议顺序如下：

1. 新增后端图片模型 registry 解析 helper
2. 调整 bootstrap 模型过滤逻辑
3. 扩展 bootstrap 返回结构
4. 新增 system settings 的 `Image Model Configuration` section
5. 完成前端图片模型配置录入 UI
6. 调整 `gpt-image-playground` 的 `newApiKeySync`
7. 调整 `NewApiKeySelector` 为两层结构
8. 联调和验证

## 11. 验收标准

1. 管理员可在系统设置里维护图片模型配置
2. `model_name` 支持下拉选择现有模型名且允许手输
3. `adapter_type` 使用下拉框
4. bootstrap 只返回图片模型配置里启用的模型
5. 同名模型按忽略大小写去重
6. token 仍受用户权限、group、`model_limits`、channel enabled 约束
7. `gpt-image-playground` 选择器变成两层结构
8. group 不再单独展开，而是作为 Key 元信息展示
9. 选中模型后仍能正确覆盖 active profile 并正常发图

## 12. 风险与取舍

### 12.1 主要风险

1. registry 配置为空时，在线生图会看不到模型
2. 旧前端若仍依赖 `groups[]`，会与新 bootstrap 结构不兼容
3. 同名模型若被管理员配置了冲突类型，只能信任 registry 中唯一配置

### 12.2 当前取舍

1. 不碰 models 表，减少数据迁移和全局副作用
2. 不做多分组，按一个 key 一个 group 收敛交互
3. 不让 `adapter_type` 立即改发图协议，先只做 bootstrap 元信息

