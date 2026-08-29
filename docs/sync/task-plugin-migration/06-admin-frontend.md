# 阶段 06：管理后台、沙箱调试与国际化

## 顶部移植提示词

请在后端契约稳定后移植任务插件管理后台，并严格使用当前 `web/default/` 目录。上游 `web/src/...` 文件必须映射到 `web/default/src/...`，禁止创建第二套前端。页面只允许根管理员访问，覆盖插件列表、详情、源码/版本、启用停用、上传、版本回滚、冲突错误、Sandbox dry-run、市场来源、usage schema、产物信息和运行时状态。所有用户可见文本用 `useTranslation()`/`t()`，同步受支持 locale；不要把价格写入插件描述或示例标签。测试交互行为而非实现细节，并在完成前运行 Bun 的类型检查、lint、格式化和构建。

## 范围

1. `/task-plugins` 管理路由、导航入口和权限保护。
2. 插件卡片、详情页、源码 diff、版本激活/回滚、上传对话框。
3. Sandbox hook 选择、args 输入、dry-run 结果和错误展示。
4. Marketplace/source picker、author/icon/description、usage schema 表格。
5. usage logs/task details 中的插件来源、usage facts、artifacts 和能力 URL。
6. 状态页：节点 generation、数据库 revision、重建结果和错误。
7. i18n locale、静态 key、可访问性和移动布局。

## 交付物

- `web/default/src/features/task-plugins/` 完整功能模块。
- `web/default/src/routes/_authenticated/task-plugins/` 路由。
- 用户、日志、系统设置中与插件相关的最小字段扩展。
- Vitest/RTL 行为测试和 i18n 同步报告。

## 前端约束

- 使用 Bun，不直接修改锁文件以外的依赖解析结果；新增依赖要有理由。
- 组件文件遵循 `web/default/AGENTS.md`，避免嵌套三元和 `any`。
- source diff、JavaScript viewer 和 dry-run 输出必须默认转义文本。
- 不展示私有 provider URL、密钥、请求头或插件环境变量。

## 底部移植完成度验证

- [ ] 根管理员权限、未授权跳转和 API 错误展示通过测试。
- [ ] 上传、版本激活/回滚、停用、冲突提示和 dry-run 交互通过测试。
- [ ] 版本 diff、author/icon/description、usage schema 和 artifacts 展示通过测试。
- [ ] 所有新增文本均有 i18n key，`bun run i18n:sync` 通过。
- [ ] `bunx oxfmt --check <touched files>` 通过。
- [ ] `bun run typecheck`、`bun run lint`、`bun run build` 的结果已记录；既有阻塞必须逐项说明。
- [ ] 移动端、键盘操作、文本转义和敏感字段隐藏有回归验证。

完成度：`通过条目数 / 7 × 100%`。若功能可用但存在未记录的类型/构建错误，本阶段最高只能记为 80%。
