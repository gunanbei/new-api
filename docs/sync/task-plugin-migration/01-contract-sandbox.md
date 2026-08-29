# 阶段 01：插件契约与 JS 沙箱运行时

## 顶部移植提示词

请只移植任务插件系统的基础契约与沙箱运行时。以 `docs/plugin-api/v1.md`、`v1.d.ts`、`v1.schema.json` 为规范来源，在当前 Go 分层中实现 `pkg/jsplugin/` 和必要的 `relay/plugin_protocol.go`、`dto/plugin_protocol.go` 类型。沙箱必须同步执行、无 `fetch`/文件系统/环境变量/`require`/导入/异步函数能力，并限制执行时长、并发、输入大小和输出形状。先不要改任务路由、数据库模型、内置适配器或前端。为每个导出 hook 建立确定性 fixture 和错误分支测试；所有 JSON 编解码遵守 `common.*` 包装器规则。

## 范围

1. 定义 `meta`、`native`、`protocols`、驱动 hook、`DecodedBody`、`RequestDescriptor`、`NormalizedTaskResult`、`usageSchema` 和 `TaskArtifact` 的宿主类型。
2. 实现插件源码编译、静态约束、运行时隔离、超时和并发限制。
3. 实现 fixture 执行能力：可调用 hook，但不得发起真实上游请求。
4. 实现 CLI 的 `plugin lint`、`plugin test` 基础命令，或提供等价的内部 API。
5. 统一错误清洗、日志前缀和 request-id/SYSTEM 标识，不打印密钥、请求体和私有 URL。

## 关键设计约束

- JavaScript 只做请求/响应转换和用量事实提取；价格、配额、预扣费和结算由 Go 控制。
- 文件只能以不透明 `FileReference` 或占位符进入插件，不能暴露字节、临时路径或 reader。
- `meta.author.name`、版本、路由和协议声明在加载时验证；错误信息要能定位字段。
- 任何执行失败都不能改变数据库、任务状态或扣费结果。
- 不要为了通过测试放宽沙箱；测试应证明违规代码被拒绝。

## 交付物

- `pkg/jsplugin/engine.go`、`request.go`、`utils.go` 及对应测试。
- `pkg/jsplugin/fixture.go`、CLI 或等价测试入口。
- `relay/plugin_protocol.go`、`dto/plugin_protocol.go` 中的宿主协议类型（如目标结构需要）。
- 契约版本、限制参数和调试说明文档。

## 底部移植完成度验证

- [ ] 合法插件可编译并调用全部必需 hook。
- [ ] 缺失导出、错误类型、非法元数据、非法路由和协议声明会确定性失败。
- [ ] `fetch`、`fs`、`process.env`、`require`、动态 import、Promise/async 等能力均被拒绝。
- [ ] 超时、并发、输入/输出大小限制有测试，且失败不会泄漏内部堆栈或凭据。
- [ ] fixture 覆盖成功、错误、usage、artifact 和 content 请求主路径。
- [ ] `go test ./pkg/jsplugin ./relay/...`（或实际受影响包）通过。
- [ ] `git diff --check`、`gofmt` 通过。

完成度：`通过条目数 / 7 × 100%`。任一安全边界测试失败，本阶段最高只能记为 50%，不得进入阶段 02。
