# 任务插件 JS 沙箱系统移植计划

## 顶部移植提示词

请在当前 `develop` 分支按本目录的阶段顺序移植官方 `origin/main` 的任务插件 JS 沙箱系统。先阅读本文件、`docs/plugin-api/v1.md`、`docs/plugin-api/v1.d.ts` 和 `AGENTS.md`，再进入对应阶段。保持现有 `web/default/` 前端目录，不要复制官方 `web/` 树；不得移除或改写 `new-api`、`QuantumNous` 相关受保护信息。每个阶段只实现本阶段范围，先完成测试与验证，再进入下一阶段；不要通过整段 cherry-pick 引入无关重构。

## 目标与基线

- 上游功能来源：`eb48396d5fe97d27772d0cd5e3ca8aa5caa4f3e9`（任务插件架构）及其后续修订。
- 目标分支：`develop`。
- 当前目标代码仍保留 `relay/channel/task/*` 内置适配器；最终目标是由 JS 插件驱动任务平台，同时保留可回滚的兼容路径。
- 前端映射：上游 `web/src/...` → 当前 `web/default/src/...`。
- 计费改动必须先阅读 `pkg/billingexpr/expr.md`，并遵守配额饱和、预扣费、结算和审计约束。

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

- [ ] 01 阶段验证通过
- [ ] 02 阶段验证通过
- [ ] 03 阶段验证通过
- [ ] 04 阶段验证通过
- [ ] 05 阶段验证通过
- [ ] 06 阶段验证通过
- [ ] 07 阶段验证通过
- [ ] `git diff --check` 通过，且同步台账已记录源端 SHA、目标 HEAD、决策和阻塞项

完成度：`通过阶段数 / 7 × 100%`。未通过阶段必须保留为 `In progress` 或 `Blocked`，不能以“代码已合并”代替行为验证。
