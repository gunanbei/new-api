# 阶段 05：用量、计费与任务生命周期

## 顶部移植提示词

请在阶段 03/04 的任务和协议链路上接入插件用量与计费。开始前完整阅读 `pkg/billingexpr/expr.md` 和 `AGENTS.md` 的 Billing safety invariants。插件只能返回经过 schema 校验的 seconds/count/token/credit/boolean 事实，不能计算价格、配额或退款。所有倍率、时长、分辨率、批量数和 upstream deduction 都要在请求边界或 adaptor 边界限幅；所有浮点/decimal 到配额的转换使用 `common.QuotaFromFloat*`、`QuotaRound*`、`QuotaFromDecimal*`，并把 clamp 通过 `relayInfo.QuotaClamp`/任务结算链路写入管理员日志。完整追踪预扣费→上游完成→差额结算→失败退款→重复回调幂等路径。

## 范围

1. `usageSchema`、usage examples、submit/complete usage hook 和 facts 校验。
2. 任务价格矩阵、动态倍率、按平台/模型/动作的计费映射。
3. 预扣费容量检查、结算差额、失败退款、取消退款和 CAS 幂等。
4. 重试切换渠道/插件时的最终 group、最终 usage 和重复扣费保护。
5. token/second/count/credit 的跨单位转换和饱和审计。
6. 管理日志中的 usage facts、pricing breakdown、quota saturation 和安全脱敏。

## 关键不变量

- 任何用户可控数量都必须有上界；无效、NaN、Inf 或溢出不能产生负收费。
- 预扣费对饱和的大额请求应返回额度不足，而不是静默换算或溢出。
- 插件失败、网络重试和重复轮询不能多扣或多退。
- 账单表达式和动态价格必须与现有 `billingexpr` 版本规则兼容。
- 用量事实与价格计算分离，插件不得接触用户余额或数据库事务。

## 交付物

- `pkg/billingexpr/`、`service/task_billing.go`、`setting/task_pricing_setting/` 的插件用量集成。
- 用量 schema/事实校验、限幅和 quota clamp 审计实现。
- 预扣费、结算、退款、重试切换和重复回调测试。
- 至少一个 token 计费和一个时间/次数计费平台的端到端回归。

## 底部移植完成度验证

- [ ] usageSchema 字段、单位、examples 和未知字段校验通过。
- [ ] seconds/count/token/credit 的边界、NaN/Inf、超大整数和负值测试通过。
- [ ] 预扣费、结算、退款、取消和重复回调均保持不变量。
- [ ] 重试切换使用最终 group/usage，未发生重复扣费。
- [ ] quota clamp 被记录到管理员日志并输出 request-correlated warning。
- [ ] 动态计费测试覆盖目标平台，且未直接使用裸 `int(...)` 配额转换。
- [ ] 受影响 Go 包和计费回归测试通过。

完成度：`通过条目数 / 7 × 100%`。任何负收费、重复退款或未审计饱和都将本阶段判定为 0%。
