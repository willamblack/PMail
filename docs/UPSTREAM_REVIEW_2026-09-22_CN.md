# 上游 v2.9.1 至最新源码的对应问题核验

## 版本与范围

- 检查日期：2026-09-22。
- 上游：[Jinnrry/PMail](https://github.com/Jinnrry/PMail)，默认分支 `master`。
- 当前最新发布：[v2.9.7](https://github.com/Jinnrry/PMail/releases/tag/v2.9.7)，2026-08-20 发布。
- 上游最新源码：[0ff28fbd67daa380278ef0aba1a05f8d032c5c14](https://github.com/Jinnrry/PMail/commit/0ff28fbd67daa380278ef0aba1a05f8d032c5c14)，提交时间 2026-08-30 06:45:59 UTC。
- 被审查分支：`willamblack/PMail:codex/admin-wildcard-catchall`。
- 修改前版本：[36313271b547039776acd691be3063637a4b66d3](https://github.com/willamblack/PMail/commit/36313271b547039776acd691be3063637a4b66d3)，即 `0.07` 源码。
- 为包含 **v2.9.1 自身修复**，取 `v2.9.0` 为差异基线；共核验 17 个上游提交、46 个变更文件，另对照后续定制差异与相关调用代码。
- 通过实时 GitHub API 和抓取后的 Git 历史确认：上游最新提交已经是该分支的祖先，没有遗漏需要直接 cherry-pick 的上游提交。

这次检查不是“合并上游”或“重新全仓安全审计”。目标是查明这些历史修复是否仍有效，只对可以确认的残余问题作局部修改。没有改前端、数据库结构、配置格式、单管理员 Catch-All、域名匹配和 Docker 挂载方式。

## 逐版本核验

| 上游版本/提交 | 上游修改 | 当前分支核验与处理 |
| --- | --- | --- |
| v2.9.1 / [b0913ee](https://github.com/Jinnrry/PMail/commit/b0913ee) / cd2c879 | `.txt`、`.html` 附件覆盖正文 | 显式 `attachment` 已正常；发现 mixed/related 正文后面的命名 inline 部件仍覆盖正文，补充修复及兼容测试。 |
| v2.9.1 前的 9908c53 / db20ed9 | CI 测试失败标签 | 已被定制分支完整测试、静态检查、race 的只读权限流水线取代，不恢复旧标签脚本。 |
| v2.9.2 / [68dc2f0](https://github.com/Jinnrry/PMail/commit/68dc2f0) | IMAP IDLE 能力、连接清理及消息数量 | 能力和计数逻辑保留；发现异步退出清理可能删除下一轮注册，修复生命周期。 |
| v2.9.2 / [6455635](https://github.com/Jinnrry/PMail/commit/6455635) | 规则移动到内置文件夹、列表/UI、空移动保护 | 对应修复存在，相关回归通过，不重复修改。 |
| v2.9.2 / 6455635 | STORE Deleted 立即删除 | 定制分支有意改为先标记、EXPUNGE 才删除以防误删；不恢复上游立即删除行为。 |
| v2.9.3 / [05bca5d](https://github.com/Jinnrry/PMail/commit/05bca5d) | 转发兼容：原文优先、重新发送回退、本地转发 | 原始转发、本地归属检查、自转发拒绝、From/To/Reply-To/Message-ID 逻辑及测试保留，不修改。 |
| v2.9.4 / [87d3ce3](https://github.com/Jinnrry/PMail/commit/87d3ce3) | Web 全选、反选 | 功能保留，并保留已实施的分页与搜索状态修复；本次不改前端。 |
| v2.9.5 / [da55756](https://github.com/Jinnrry/PMail/commit/da55756) | SpamBlock 设置越权 | 管理员校验及非管理员写入拒绝测试存在，不修改。 |
| v2.9.5 / [593c3fb](https://github.com/Jinnrry/PMail/commit/593c3fb) | 邮件详情、规则更新越权 | 查询/更新限定当前用户，定制分支另有加固，不回退。 |
| v2.9.5 / [430c9c6](https://github.com/Jinnrry/PMail/commit/430c9c6) | SPF/DKIM 认证结果传递 | 插件、数据库、恢复和列表路径保持一致；不将此标记等同 DMARC 对齐。无需修改。 |
| v2.9.6 / [cd8d549](https://github.com/Jinnrry/PMail/commit/cd8d549) | Date 重建保留时区 | RFC3339 保存/重建已有；输入仍只接受 RFC1123Z，补充合法邮件日期格式解析。 |
| v2.9.6 / [1da3e17](https://github.com/Jinnrry/PMail/commit/1da3e17) | 下游投递失败返回错误；DATA 已接受后不因 QUIT 失败重发 | 错误映射、最终 DATA 响应及 QUIT 回归存在，不直接修改这些函数；修正下述 MX 分组向其提供错误结果的边界。 |
| v2.9.7 / [a6049bf](https://github.com/Jinnrry/PMail/commit/a6049bf) | SQLite BUSY/LOCKED 重试、原子审计保存 | 初始邮件与归属关系、最终投递状态的事务及故障注入测试保留，无需修改。 |
| v2.9.7 之后 / [b62c4db](https://github.com/Jinnrry/PMail/commit/b62c4db) | 微信验证码推送 | 提取与普通邮件回退测试保留，不修改。 |
| v2.9.7 之后 / [d2a8269](https://github.com/Jinnrry/PMail/commit/d2a8269) | 按 MX 优先级故障转移 | 全部 MX 尝试逻辑存在；同域逐地址查 MX 可拆组并覆盖错误，改为每域一次查询/一个投递任务。 |
| 最新 / [0ff28fb](https://github.com/Jinnrry/PMail/commit/0ff28fb) | 已退出插件清理、HookList 并发保护 | map 锁与正确名称已存在；注册/退出仍有生命周期窗口，补充退出状态及实例匹配。 |

## Standards 轴

发现并修复 1 组 P2 生命周期问题（对应 #359）：

- 文件：`server/hooks/base.go`；入口 `Init`，新增内部 `hookRegistration`。
- 原问题：插件在获取名称期间退出，退出协程读到空名称并完成清理，启动线程随后仍注册已死插件。旧进程退出还可能删掉同名的新插件。
- 处理：同一生命周期锁串行化注册/退出；退出后禁止注册；移除时匹配具体 HookSender 实例。
- 证据：真实 `Init()` 流程的隔离子进程测试在旧代码上失败，修复后通过。另覆盖退出前后注册、同名替换与并发交错。
- 纯维护性提示（如未使用的辅助类型）不作为独立修复理由；不作无关格式或架构调整。

## Spec 轴

发现并修复 4 组 P2 残余问题：

1. **附件与正文分离**：`server/dto/parsemail/email.go: NewEmailFromReader / formatContent / getFileName`。解析 MIME disposition/filename 参数；mixed/related 已有正文后出现的命名文本部件作为附件；支持 RFC2231 编码文件名。保留顶层命名 inline 正文及 multipart/alternative 正文，避免修复造成空白正文。对任意畸形 MIME 的恢复不作完整保证。
2. **合法邮件日期**：同文件 `NewEmailFromReader`。改用标准库 `net/mail.ParseDate`；支持单数字日、无星期、GMT、带注释等合法形式；真正无效/缺失日期仍保持原来的当前时间回退。本次不改变数据库“投递时间”字段语义，也不恢复已经丢失的历史原始日期。
3. **IDLE 生命周期**：`server/listen/imap_server/session_idle.go: Idle`。依赖库已经异步调用 Idle；应等待 stop 并完成清理后才返回，确保 DONE 应答前旧注册已移除。新增生命周期及真实本地 TLS 会话连续 25 次进出测试。不等于补齐所有 IMAP 协议特性。
4. **同域 MX 分组**：`server/utils/send/send.go: doSend`。等优先级 MX 的顺序可能随机变化；过去每个地址分别查询，以“域名+首 MX”为键分组，又以域名合并错误，可能互相覆盖临时/永久失败。现在先按规范化域名聚合，再每域查一次 MX、执行一个投递任务，保留原有 MX 次序与端口回退策略。未引入队列或修改跨域部分成功语义。

未发现需要恢复旧版立即删除、撤销既有安全加固或新增功能的依据。

## 验证与交付边界

- 新增回归先验证旧逻辑失败，再验证修复；正常 inline 正文、旧 Date 格式等兼容情况另作正向回归。
- 最终验证全部通过（Go 1.26.8）：`go test -count=1 ./...`、`go vet ./...`、`go test -race -count=1 -p 1 ./...`；不是仅复用上轮缓存结果。
- 前端 4 项状态辅助函数回归通过；前端源码没有改变，本次没有重新进行 Safari/iOS 实机验收。
- `git diff --check` 通过。新增测试覆盖正常与错误路径；这些测试不构成生产环境验收。
- 不连接 VPS，不使用真实收件箱，不发送公网测试邮件。数据库测试使用临时 SQLite；协议测试使用本机随机端口；插件测试只启动并终止自己创建的临时进程。
- 审查完成时未修改远程分支或发布镜像；随后用户于 2026-09-22 授权先推送定制分支、同步 master，再发布 Release 与 Docker，计划版本为 `0.08`。发布结果以 GitHub Actions 和 GHCR 核验为准。既有 `0.07` 不覆盖、不包含这些新修复。
- 历史审计报告中的剩余架构限制仍然成立；本报告不表示全部漏洞已排除。

审查采用 `code-review` 的 Standards/Spec 两轴独立核验。仓库没有该技能预期的 `docs/agents/issue-tracker.md`；技能建议使用 `/setup-matt-pocock-skills` 配置追踪流程，但本次已有明确的上游发布说明和提交作为需求来源，无需为审查新增工具配置。
