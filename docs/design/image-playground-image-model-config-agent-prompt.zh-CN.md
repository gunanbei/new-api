# 新对话 Agent 提示词：在线生图图片模型配置改造

你正在 `new-api` 仓库中继续实现“在线生图图片模型配置改造”需求。请严格按下面要求工作，不要自行扩展范围。

## 一、项目背景

当前 `new-api` 集成了 `gpt-image-playground`，在线生图页面通过：

- `GET /api/image-playground/bootstrap`

聚合返回用户可用的 Key、分组和模型。

当前代码存在两个问题：

1. 图片模型判定使用后端硬编码函数 `common.IsImageGenerationModel()`，管理员不可配置。
2. `gpt-image-playground` 的 Key 选择器使用三层结构：
   - Key
   - Group
   - Model

但在本业务里，一个 API Key 只会对应一个 `token.group`，所以三层结构多了一层无意义交互。

## 二、已经确认的需求

这些需求已经和用户确认过，不要再改方向：

1. **不改 `models` 表，不新增数据库字段。**
2. **不要把图片模型配置混进现有模型元数据 CRUD。**
3. **在 system settings / models 区域新增一个独立的「图片模型配置」入口。**
4. **图片模型配置使用 option JSON 持久化，不做纯 Redis 存储。**
5. **图片模型配置只影响 `/api/image-playground/bootstrap`。**
6. **一个 API Key 在在线生图场景下只对应一个分组，因此最终 UI 改为两层结构：**
   - 第 1 层：API Key
   - 第 2 层：图片模型列表
7. **group 不再单独作为一层展开，而是作为 Key 的元信息展示。**
8. **bootstrap 的模型来源必须是：该 `token.group` 下所有启用渠道模型的并集。**
9. **模型还要继续受这些约束：**
   - 用户可用分组
   - token.group
   - token `model_limits`
   - enabled ability / channel
10. **图片模型最终是否进入 bootstrap，由“图片模型配置”决定，不再依赖硬编码图片模型判断。**
11. **模型名按忽略大小写去重。**
12. **倍率来源使用现有全局倍率配置，不从渠道侧取。**
13. **底层类型字段按协议 / 适配器命名，但展示时按厂家 / 产品名显示。**
14. **后台录入 `model_name` 时，要支持“下拉选择现有模型名 + 允许手输”。**
15. **本次不实现多协议真实发图，只把 `adapter_type` 作为 bootstrap 元信息返回，供后续扩展。**

## 三、必须遵守的边界

不要做这些事：

1. 不修改 `models` 表 schema
2. 不新增 model 字段
3. 不改 `/api/user/models`
4. 不改通用模型管理页 CRUD 结构
5. 不改计费逻辑
6. 不改路由分发逻辑
7. 不把这套配置影响到 bootstrap 以外的其他模型分类接口
8. 不实现“一个 key 多 group”的三层结构预留

## 四、推荐实现方案

### 1. 新增 option key

新增一个独立 option key：

- `image_playground.model_registry`

值是 JSON，例如：

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

### 2. system settings 新增 section

文件重点：

- `new-api/web/default/src/features/system-settings/models/section-registry.tsx`

在 `Global Model Configuration` 和 `Routing Reliability` 之间新增：

- `Image Model Configuration`

新建一个 section 组件，负责：

1. 读取 `image_playground.model_registry`
2. 展示图片模型配置列表
3. 支持新增 / 编辑 / 删除 / 启用开关
4. 保存时调用现有 `useUpdateOption()`

### 3. model_name 录入体验

`model_name` 必须满足：

1. 可从现有模型名候选中选择
2. 允许手动输入新模型名

候选模型名优先考虑来自：

- `/api/channel/models_enabled`

因为 bootstrap 本身依赖实际已启用渠道模型。

### 4. adapter_type 枚举

建议内置这些值：

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

显示文案示例：

- `openai_images` -> `OpenAI Images`
- `gemini_imagen` -> `Google Imagen`
- `flux` -> `FLUX`

### 5. bootstrap 改造

重点文件：

- `new-api/controller/user.go`

当前 `imagePlaygroundModelsForToken()` 里依赖：

- `model.GetGroupEnabledModels(group)`
- `common.IsImageGenerationModel(modelName)`

需要改成：

1. 继续使用 `model.GetGroupEnabledModels(group)` 作为“分组下所有启用渠道模型并集”
2. 保留 `token.ModelLimitsEnabled` 过滤
3. 新增读取图片模型配置 registry
4. 只保留 registry 中存在且 `enabled=true` 的模型
5. 使用忽略大小写的 key 去重
6. 返回模型时额外补上：
   - `adapter_type`
   - `display_vendor`

### 6. bootstrap 返回结构

把当前三层语义：

- `token.groups[0].models`

改成两层语义：

- `token.models`

并在 token 节点直接返回：

- `group`
- `group_display_name`
- `group_ratio`

### 7. gpt-image-playground 改造

重点文件：

- `gpt-image-playground/src/lib/newApiKeySync.ts`
- `gpt-image-playground/src/components/NewApiKeySelector.tsx`

当前依赖三层：

- `groups[]`
- `expandedGroup`

需要改成两层：

- Key 展开后直接展示 `models[]`
- group 信息作为 Key 副标题展示

同时：

1. 删除 `NewApiGroupOption` 或停止使用
2. `NewApiKeyTreeOption` 改成直接包含：
   - `groupDisplayName`
   - `groupRatio`
   - `models`
3. `selectModel(item, group, model)` 改成 `selectModel(item, model)`
4. `group` 直接用 `item.group`

### 8. active profile 覆盖逻辑

不要重写主流程，继续沿用：

- `apiKey`
- `group`
- `model`
- `apiMode = images`
- `baseUrl = /v1`

即：

- `gpt-image-playground/src/lib/newApiKeySync.ts`
- `applyNewApiSelectionToActiveProfile()`

只需要让它适配两层选择器输入。

## 五、实现风格要求

1. 走最小改动路径
2. 优先复用已有 system settings option 读写模式
3. 不引入新依赖
4. 不为未来多 group 提前做复杂抽象
5. 不为了“协议扩展”提前重写发图主链路
6. 只做当前确认范围内的改造

## 六、验收标准

实现完成后必须满足：

1. system settings 中出现独立的 `Image Model Configuration`
2. 管理员可维护图片模型配置
3. `model_name` 支持下拉 + 手输
4. `adapter_type` 是下拉
5. bootstrap 只返回图片模型配置里启用的模型
6. 同名模型按忽略大小写去重
7. `gpt-image-playground` 选择器从三层改为两层
8. group 作为 Key 元信息显示，不再单独展开
9. 选中模型后仍能正确覆盖 active profile

## 七、禁止偏航

如果你在实现中发现“也许改 models 表更统一”，不要改。
如果你发现“未来可能一个 key 多 group”，不要提前做三层兼容抽象。
如果你发现“adapter_type 可以顺便驱动真实多协议发图”，这次也不要做。

本次目标就是：

**用最小代价，把图片模型定义权从硬编码迁到独立配置，并把选择器从三层收敛成两层。**

