# 阶段 02：插件模型、存储与原子注册表

## 顶部移植提示词

请在阶段 01 已通过的前提下，移植任务插件的持久化、版本管理和注册表。保持 Router → Controller → Service → Model 分层，在 `model/` 增加插件、版本、启用状态、备注和审计所需模型，在 `setting/` 增加运行参数；实现数据库迁移时同时考虑 SQLite、MySQL 5.7.8+ 和 PostgreSQL 9.6+。注册表发布必须 generation-atomic：请求固定一个插件 generation，后台轮询可使用更新版本，但进行中的任务仍能解析旧版本。上传时执行实时路由/协议冲突预检，并支持强制保存但不启用。先不要替换内置任务适配器。

## 范围

1. 插件 source/version/key/author/metadata/active/remark 的模型与索引。
2. 启用、停用、激活、回滚、同 key/version 不同源码拒绝。
3. 注册表快照、generation、数据库 revision、重建结果和插件级错误。
4. 上传预检：渠道类型、原生路由、协议模型范围冲突。
5. 根管理员状态接口（节点本地 generation、数据库 revision、最近一次重建结果）。
6. 审计事件：上传、启用、停用、回滚、强制保存和重建失败。

## 数据库与并发要求

- 使用 GORM 查询和 `lockForUpdate(tx)`，不要使用旧版 `gorm:query_option`。
- 不使用数据库专属 JSON 类型或只在某一数据库可用的 ALTER 语法。
- 唯一约束和 upsert 必须给出三种数据库的有效行为。
- 注册表替换不能让并发请求看到半成品；失败重建要保留上一代可用快照。

## 交付物

- `model/task_plugin.go` 及迁移注册。
- `setting/task_plugin.go` 与执行限制配置。
- `pkg/jsplugin/registry.go`、routing/conflict 检查和状态快照。
- 管理 API、权限资源和审计记录。
- 模型、迁移、并发发布和回滚测试。

## 底部移植完成度验证

- [ ] 三种数据库的 AutoMigrate/迁移路径均有明确兼容说明或测试。
- [ ] key/version/source 不变量、启用/停用/回滚行为通过测试。
- [ ] generation 发布具备原子性，重建失败保留旧 generation。
- [ ] 路由/协议/渠道冲突在启用前被拒绝，并指出冲突插件。
- [ ] 节点状态接口不因数据库临时不可用而丢失本地状态。
- [ ] 权限与审计覆盖上传、激活、停用、回滚和强制保存。
- [ ] `go test ./model ./pkg/jsplugin ./controller ./service ./router`（按实际包调整）通过。

完成度：`通过条目数 / 7 × 100%`。若存在跨数据库未解释行为，本阶段标记 `Blocked`。
