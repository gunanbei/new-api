# 任务插件 JS 沙箱系统移植计划

## 顶部移植提示词

请作为移植负责人，在当前 `develop` 分支分阶段移植官方 `origin/main` 的任务插件 JS 沙箱系统。执行前完整阅读本 README、`docs/plugin-api/README.md`、`docs/plugin-api/v1.md`、`docs/plugin-api/v1.d.ts`、`docs/plugin-api/v1.schema.json`、`pkg/billingexpr/expr.md`、`AGENTS.md` 以及 `web/default/AGENTS.md`。注意：上述 `docs/plugin-api/*` 规范文件目前只存在于 `origin/main`，若当前分支缺失，必须用 `git show origin/main:<path>`（或从提交 `eb48396d5fe97d27772d0cd5e3ca8aa5caa4f3e9` 读取）核对，不能假定它们已经在工作树中；进入阶段 01 时再按交付物要求移植到目标分支。以官方任务插件提交 `eb48396d5fe97d27772d0cd5e3ca8aa5caa4f3e9` 及其后续修订为行为参考，并先审计当前代码与提交差异，再决定实现顺序。

严格按 01→02→03→04/05→06→07 执行：每阶段只改动本阶段范围，完成底部验证清单、相关测试和 `git diff --check` 后才进入下一阶段；任一安全、计费、数据库兼容或回滚门槛失败，必须停留在 `In progress`/`Blocked` 并记录证据，不得以“代码已合并”替代验收。不要整段 cherry-pick 官方提交或引入无关重构；保留旧 `relay/channel/task/*` 适配器作为可回滚兼容路径。

实现边界必须保持：插件是单文件、同步 ECMAScript 模块，只负责请求/响应转换、协议渲染和用量事实提取；Go 负责认证、HTTP、持久化、任务状态、轮询、重试、产物代理、权限、计费、预扣费、结算和退款。沙箱禁止网络、文件系统、环境变量、`require`/import、`async`/Promise，并限制执行时长、并发、输入和输出大小。注册表发布必须 generation-atomic，任务固定插件版本；Responses/Video/native routes、artifact 能力 URL、SSRF/重定向防护、usage schema 与 quota saturation 审计必须覆盖。前端只能落在 `web/default/`，所有用户文案接入 i18n；不得移除或改写 `new-api`、`QuantumNous` 相关受保护信息。

每阶段交付实现、迁移说明、fixture/回归测试和验证证据；同步更新本目录复选框及 `docs/sync/official-main-sync.md` 的源端 SHA、目标 HEAD、决策和 blocker。最终只有 7 个阶段、63 个行为矩阵单元全部通过，且无安全/计费阻塞并完成灰度回滚演练，才可将 Deferred 改为 Completed。

## 目标与基线

- 上游功能来源：`eb48396d5fe97d27772d0cd5e3ca8aa5caa4f3e9`（任务插件架构）及其后续修订。
- 目标分支：`develop`。
- 当前目标代码仍保留 `relay/channel/task/*` 内置适配器；最终目标是由 JS 插件驱动任务平台，同时保留可回滚的兼容路径。
- 前端映射：上游 `web/src/...` → 当前 `web/default/src/...`。
- 计费改动必须先阅读 `pkg/billingexpr/expr.md`，并遵守配额饱和、预扣费、结算和审计约束。

## 规范文件现状

当前 `develop` 工作树尚未包含 `docs/plugin-api/`。规范来源位于 `origin/main`，包括：

- `docs/plugin-api/README.md`
- `docs/plugin-api/v1.md`
- `docs/plugin-api/v1.d.ts`
- `docs/plugin-api/v1.schema.json`

阶段 01 负责将这些规范及其对应实现移植到目标分支；在此之前，所有阶段文档中的引用均指向上游规范来源，不表示本地文件已经存在。

## 阶段依赖

```text
01 契约与沙箱运行时
        ↓
02 插件模型、存储与注册表
        ↓
03 任务路由、渠道绑定与轮询运行时
        ├──────────────┐
        ↓              ↓
04 协议适配与产物安全   05 用量、计费与生命周期
        └──────┬───────┘
               ↓
06 管理后台、沙箱调试与文档
               ↓
07 兼容迁移、灰度切换与最终验收
```

## 文档清单

| 阶段 | 文档 | 结果 |
|---|---|---|
| 01 | [01-contract-sandbox.md](./01-contract-sandbox.md) | 可验证的 JS 插件 API、沙箱执行器和 fixture 工具 |
| 02 | [02-registry-storage.md](./02-registry-storage.md) | 插件版本、数据库模型、原子注册表和冲突检查 |
| 03 | [03-task-runtime.md](./03-task-runtime.md) | 通用任务渠道、提交/查询/轮询和旧适配器兼容层 |
| 04 | [04-protocol-artifacts.md](./04-protocol-artifacts.md) | Responses/Video 协议、原生路由和产物能力 URL |
| 05 | [05-billing-lifecycle.md](./05-billing-lifecycle.md) | 用量事实、计费、重试、预扣费、结算和退款 |
| 06 | [06-admin-frontend.md](./06-admin-frontend.md) | 管理后台、上传/调试/市场界面、国际化和运维状态 |
| 07 | [07-cutover-acceptance.md](./07-cutover-acceptance.md) | 内置平台插件化、灰度发布、回滚和最终验收 |

## 总体验收标准

只有同时满足以下条件，才可把本功能从 Deferred 改为 Completed：

- 所有内置任务平台均有等价插件或明确保留兼容驱动。
- 插件无法访问文件系统、环境变量、网络 API、异步执行和未授权导入。
- 请求、任务、协议、产物、计费、日志和权限链路均有确定性回归测试。
- SQLite、MySQL、PostgreSQL 的迁移和查询路径均经过检查。
- 前端在 `web/default/` 下完成，用户文案已接入 i18n。
- 可在单节点和多节点场景下执行插件版本回滚，且不会破坏进行中的任务。

## 底部移植完成度验证

- [x] 01 阶段验证通过
- [ ] 02 阶段验证通过
- [ ] 03 阶段验证通过
- [ ] 04 阶段验证通过
- [ ] 05 阶段验证通过
- [ ] 06 阶段验证通过
- [ ] 07 阶段验证通过
- [ ] `git diff --check` 通过，且同步台账已记录源端 SHA、目标 HEAD、决策和阻塞项

完成度：`通过阶段数 / 7 × 100%`。未通过阶段必须保留为 `In progress` 或 `Blocked`，不能以“代码已合并”代替行为验证。

阶段 01 当前状态：`Completed`。已切换到 Go 1.25.1 工具链并补齐 Sobek 间接依赖，阶段测试通过。
