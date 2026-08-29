# 阶段 04：协议适配、原生路由与任务产物

## 顶部移植提示词

请在阶段 03 的通用任务运行时上移植宿主协议绑定和产物安全边界。实现 `openai_responses` 与 `openai_video` 的 create/retrieve/content 路径，严格按 `meta.protocols` 的 supports、models 和 hook 组合校验；stream、sync、background 模式在 hook 运行前拒绝不支持的请求。实现 `meta.routes` 的 native submit/query/dynamic 路由和 host allowlist。成功任务的产物只能通过宿主签名能力 URL 暴露，不能直接泄露 provider URL；内容代理必须校验归属、插件、渠道、允许域名、重定向和 Range/条件请求。协议 DTO 必须白名单化，覆盖未知字段、旧 task_id、大小写 URL 元数据和身份字段。

## 范围

1. Responses 请求解码、SSE framing、stream 事件和 terminal render。
2. Video JSON/multipart 创建、状态检索、内容 HEAD/GET。
3. Native routes 的 method/path/taskIdParam/models 冲突和模型预匹配。
4. `listArtifacts`、`buildContentRequest`、能力 URL 签发/验证与多节点密钥要求。
5. SSRF 防护：初始 URL、每次重定向、allowedHosts、credentialless GET/HEAD 限制。
6. 协议/原生错误 envelope、日志脱敏和前端展示所需任务详情字段。

## 安全与兼容要求

- 产物 URL 永久有效但依赖 `CRYPTO_SECRET`；密钥轮换必须使旧 URL 失效。
- 只有 SUCCESS 任务获得 artifact map；失败和非终态不得暴露产物。
- 能力 URL 验证后仍要重新加载任务、所有者和插件，不得只相信 token。
- 插件 renderer 失败只能影响该次观察结果，不能改任务、计费或退款状态。
- 原生 provider URL 只能在认证 native presenter 且通过 host 校验时返回。

## 交付物

- `relay/plugin_protocol.go`、协议路由和 SSE/DTO 转换。
- `middleware/task_artifact_access.go`、`service/task_artifact_*` 和相关模型/路由。
- native route 注册、冲突检查和协议 claims 验证。
- 产物权限、SSRF、重定向、HEAD/Range、非终态和 renderer 失败测试。

## 底部移植完成度验证

- [ ] Responses stream/sync/background/retrieve 行为与 supports 声明一致。
- [ ] Video create/retrieve/content 的 JSON、multipart、HEAD 和错误路径通过测试。
- [ ] native route 的 path/method/models 冲突和 channel host 校验通过。
- [ ] 产物仅经能力 URL 暴露，所有权、插件、任务状态和密钥轮换均验证。
- [ ] SSRF、重定向、allowedHosts、credentialless 和请求头白名单测试通过。
- [ ] 协议 DTO 白名单和敏感字段脱敏测试通过。
- [ ] 受影响 Go 包及协议集成测试通过。

完成度：`通过条目数 / 7 × 100%`。任一 SSRF 或越权产物测试失败，本阶段必须阻断后续前端发布。
