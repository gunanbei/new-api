# 创作台阶段二测试报告

> 状态：本地自动化验证通过。

## 已执行

- `go test ./...`
- `git diff --check`
- `web/default`: `bun run i18n:sync`、`bun run typecheck`、`bun run build`

## 自动化 UI

按 `local-start.md` 启动后端 `http://localhost:3000` 与前端 `http://localhost:5174`。浏览器自动化以本地测试用户登录并访问 `/imgen`，确认：

- 侧边栏“创作台”为站内 `/imgen` 链接，未携带 native 外跳属性。
- 图片工作区、模型和分组控件、创建按钮、作品历史空态均可访问。
- 没有 bootstrap 可见能力时，创建按钮处于禁用状态。

## 覆盖范围

- 任务迁移、状态 CAS、所有者隔离、导入重试、模型改名绑定失效与启动恢复。
- 图片数量上限、`stream`/受保护字段拒绝、Schema 参数合并与输入文件归属。
- OpenAI Images、Responses 图像、Claude SVG、Midjourney 既有轮询观察分别走既有 Relay/任务链路。
- S3、WebDAV、ImgBed 服务端导入；MIME、魔数、大小、SSRF、私网/保留地址与 SVG 恶意标记保护。
- 输出文件删除保护、普通重试独立任务与无额外计费的导入重试。
