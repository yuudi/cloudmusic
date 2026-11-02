# Cloud Music

Cloud Music 是一个开源、自托管音乐平台，以移动端为主，同时面向浏览器和 PWA。目标是让用户连接自己的存储空间，主动导入、播放歌曲，并向好友分享音乐。

项目处于早期开发阶段，已实现基础账号的前后端流程：注册、登录、游客身份与升级、个人资料修改、密码修改、会话撤销，以及游客连续 30 天不活跃清理。客户端已具备 PWA 应用入口、黑白灰与深红配色、系统深浅色适配和中英文切换（英文回退）。

客户端采用 Svelte 5、SvelteKit、Tailwind CSS 和 Skeleton；服务端采用 Go、Gin 和 GORM，使用 Postgres 与 Redis。歌曲、存储接入、音频播放和好友分享尚未实现；rclone 接入与 ffmpeg wasm 解码回退见设计文档。初期面向单机自托管部署。

## 本地开发

推荐使用 devcontainer，本仓库已包含 devcontainer、Postgres 和 Redis 配置。在仓库根目录执行 `just init` 安装前后端依赖，`just db-migrate` 初始化数据库，然后使用 `just dev` 同时启动前后端。浏览器访问 `http://localhost:5173`，按 Ctrl+C 停止两个服务。也可以分别使用 `just dev-server` 和 `just dev-web` 启动服务。

- [前端开发说明](apps/web/README.md)
- [后端开发说明与数据库重置命令](apps/server/README.md)

`just` 列出常用命令：`init`、`dev`、`dev-web`、`dev-server`、`build-web`、`build-server`、`test-web`、`test-server`、`db-migrate` 和 `db-reset`。`db-reset` 会删除开发数据库中的表并重新执行迁移。

单独安装依赖、格式与静态检查、Redis 重置和完整数据重置的直接命令见上述开发说明。

浏览器测试环境按需安装，不会在创建 devcontainer 时自动执行。先运行 `just init`，再在容器中执行：

```sh
# 安装 Chromium 及系统依赖，供前端组件测试使用
bash .devcontainer/scripts/install-playwright.sh

# 运行全部 E2E 测试时，安装三个浏览器及系统依赖
bash .devcontainer/scripts/install-playwright.sh chromium firefox webkit
```

系统依赖安装可能需要 sudo，请以开发用户 `vscode` 运行脚本，让 Playwright 按需提权；浏览器下载到当前用户的缓存目录。重建容器或升级 Playwright 后可重新运行。日常手动查看页面只需宿主机浏览器，无需安装这些测试浏览器。

安装脚本用于补齐容器的浏览器测试环境，使用项目锁定的 Playwright 版本；需要网络下载浏览器及系统包，不安装 Node.js 或 npm 项目依赖。用法及测试限制见[前端开发说明](apps/web/README.md#工程检查)。

## 验证

以下命令在仓库根目录执行。数据库测试使用随机临时 schema，不重置开发数据；需要可达的 PostgreSQL 和 Redis，以及对应的测试工具。

```sh
# 前端类型、格式、lint、单元/组件测试及跨浏览器 E2E，内含生产构建
npm --prefix apps/web run validate

# 后端测试，包含数据库状态、事务和并发检查
TEST_DATABASE_URL="$DATABASE_URL" go -C apps/server test -race ./...

# 后端账号 API 流程
TEST_DATABASE_URL="$DATABASE_URL" bash apps/server/tests/hurl/run.sh

# Chromium 对接独立后端的真实账号流程
TEST_DATABASE_URL="$DATABASE_URL" bash apps/web/tests/e2e/run.sh
```

普通前端 E2E 中的账号界面测试使用模拟 API，真实后端用例会跳过；最后一条命令单独验证前后端联动。没有设置 `TEST_DATABASE_URL` 时，Go 的数据库集成测试也会跳过，不能视为数据库验证通过。

## 设计文档

- [产品规则与服务架构](design/README.md)
- [技术方案与歌曲数据模型](design/data-model.md)
- [分享会设计](design/party-mode.md)

## 路线图

任务与实现进度在 [GitHub Project：Cloud Music](https://github.com/users/yuudi/projects/2) 中追踪，按版本、类型和状态整理；具体规则及验收条件记录在关联 Issues 中。
