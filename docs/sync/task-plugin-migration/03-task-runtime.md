# 阶段 03：任务路由、渠道绑定与轮询运行时

## 顶部移植提示词

请在阶段 02 已通过的基础上，移植通用任务运行时，但保留旧适配器可回退。为第三方平台提供 Task Plugin 渠道类型（官方为 type-59）和 `task_plugin_key` 绑定；实现按插件 generation 固定的渠道选择、提交、查询、批量查询、轮询、重试、取消和任务持久化。插件只返回请求描述和任务事实，Go 负责连接、状态、重试、日志、计费触发和结算。不得把客户端路径逻辑塞入 driver hook；native route、Responses、Video 都必须汇聚到统一的 submit/query intent。先完成一个平台端到端样例，再批量迁移其余内置平台。

## 范围

1. 渠道约束、插件 key 绑定、模型范围和 channel selection。
2. `buildSubmitRequest`、`parseSubmitResponse`、`buildQueryRequest`、`parseTaskResult` 及 batch 变体的调用链。
3. 任务创建耐久化、upstream task id、generation/plugin version 固定和后台轮询。
4. 重试、客户端断开、超时、取消、失败状态和幂等处理。
5. 旧 `relay/channel/task/*` 适配器的兼容层与明确回退规则。
6. 任务 API：创建、查询、取消、状态和错误 envelope。

## 关键不变量

- 任务提交成功前必须有可恢复的持久化记录。
- 轮询使用的插件版本必须能解析进行中任务的历史响应。
- 同一个外部任务不能重复创建、重复退款或重复完成。
- 失败/取消不能绕过已有的任务 CAS、退款和日志链路。
- 渠道选择失败要在插件 hook 运行前返回用户可理解的错误。

## 交付物

- `relay/channel/task/jsplugin/` 驱动适配器。
- `relay/relay_task.go`、`service/task_polling.go`、`model/task.go` 的最小集成。
- type-59 渠道 DTO、路由、控制器和测试。
- 至少一个视频/异步平台从创建到完成的集成 fixture 或 e2e 测试。

## 底部移植完成度验证

- [ ] Task Plugin 渠道可创建、绑定插件、选择模型并提交任务。
- [ ] 提交、查询、批量查询、轮询、取消和超时状态均有测试。
- [ ] 插件 generation/version 在任务生命周期内固定且可读取历史任务。
- [ ] 重试、客户端断开、CAS 完成和重复退款回归测试通过。
- [ ] 旧适配器仍可工作，回退条件和日志明确。
- [ ] 任务 API 错误格式与现有 OpenAI/Claude 客户端兼容。
- [ ] 受影响 Go 包测试通过，且 `git diff --check` 通过。

完成度：`通过条目数 / 7 × 100%`。若只能提交任务但无法可靠轮询/结算，本阶段不得记为完成。
