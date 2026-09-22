# 本 Fork 的默认发布流程

用户于 2026-09-22 指定：实际修改通过验证后，默认先提交并推送
`codex/admin-wildcard-catchall`，再同步到 `master`，最后生成新 Release 和 Docker。
只读检查、没有需要修改的问题时，不制造空提交或无意义版本。

## 顺序与安全边界

1. 检查工作区，保留用户未完成的改动；确认没有配置、数据库、私钥等敏感文件进入提交。
2. 执行相应测试、`go vet`、必要的 race 检查和前端构建/测试。
3. 提交并推送定制分支到 `willamblack/PMail`，等待回归检查通过。
4. 检查远程 `master`。能快进则快进同步；有独立改动先保留并处理，不强推覆盖。
5. 从两分支同步后的明确提交创建尚未使用的版本标签（如 `0.08`），创建正式 Release。
6. 等待 Docker 与二进制发布流程完成，核对源码 SHA、镜像版本、多架构、匿名拉取权限和 Release 附件。

本地 `fork` 通常是用户仓库，`origin` 是作者上游；操作前始终核对 remote URL。
不得推送上游仓库，不移动已发布的版本标签，不自动修改 VPS。

## Release 与镜像触发关系

- 发布正式 Release（`released`）触发 `docker_build_admin_catchall.yml`。
- 同一次 Docker 构建发布 `ghcr.io/willamblack/pmail:<版本>` 与 `:latest`。
- 支持 `linux/386`、`linux/amd64`、`linux/arm/v7`、`linux/arm64`。
- `release.yml` 同时生成 Windows、Linux、macOS 的 amd64/arm64 ZIP 附件。
- 旧 `docker_build.yml` 在本 Fork 中跳过，避免同一个版本被两套流程重复构建、覆盖。
- 只推送 Git 标签不会自动构建镜像；后续需要发布正式 Release。
- 手动运行 catch-all Docker 流程仍可用，默认不改 `latest`；不要对已发布版本重复运行。
- 预发布不是上述默认流程，原有预发布行为保持不变；不要把预发布当正式 Release。

标签/Release 已创建后，如果代码需要再次修改，使用新的版本号，不偷偷移动旧标签。
基础设施临时失败可以重跑同一源码的失败任务；成功不等于 VPS 已升级。

## 升级提醒

生产升级先停止写入并完整备份挂载目录，再用固定版本重建容器。
保持原 `/work/config` 挂载，不重新初始化数据库或轮换 DKIM 私钥。
老版本镜像和版本标签保留用于回退；`latest` 会随正式 Release 更新，因此生产建议固定版本。
