# Cloud Music

Cloud Music 是一个开源、自托管音乐平台，以移动端为主，同时面向浏览器和 PWA。目标是让用户连接自己的存储空间，主动导入、播放歌曲，并向好友分享音乐。

项目处于早期开发阶段，已实现基础账号的前后端流程：注册、登录、游客身份与升级、个人资料修改、密码修改、会话撤销，以及游客连续 30 天不活跃清理。客户端已具备 PWA 应用入口、黑白灰与深红配色、系统深浅色适配和中英文切换（英文回退）。

客户端采用 Svelte 5、SvelteKit、Tailwind CSS 和 Skeleton；服务端采用 Go、Gin 和 pgx，通过 database/sql 访问 Postgres，并使用 Redis。歌曲、存储接入、音频播放和好友分享尚未实现；rclone 接入与 ffmpeg wasm 解码回退见设计文档。初期面向单机自托管部署。

## 本地开发

推荐使用 devcontainer，本仓库已包含开发容器、Postgres 和 Redis 配置。

- [前端开发说明](apps/web/README.md)
- [后端开发说明与数据库重置命令](apps/server/README.md)
- [OpenAPI 定义、前端类型生成与接口维护流程](apps/web/README.md#api-与-openapi)

运行 `just` 列出开发、构建、测试和数据库命令。`db-reset` 会删除开发数据库中的表并重新执行迁移。

单独安装依赖、格式与静态检查、Redis 重置和完整数据重置的直接命令见上述开发说明。

### 启动开发服务器

1. 使用 devcontainer 管理器打开本项目。例如：在 VS Code 中使用 **Dev Containers: Reopen in Container** 打开项目。
2. 在仓库根目录执行 `just init`，安装前后端依赖。
3. 执行 `just db-migrate`，初始化数据库。
4. 执行 `just dev`，访问 `http://localhost:5173`；按 Ctrl+C 停止服务。

新增或拉取迁移后，执行 `just db-migrate`。容器外开发的配置方法见[后端开发说明](apps/server/README.md#本地开发)。

### 浏览器测试环境

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

以下命令在仓库根目录执行：

| 命令                           | 测试范围                                                     | 环境要求                                                       |
| ------------------------------ | ------------------------------------------------------------ | -------------------------------------------------------------- |
| `just test-web-unit`           | 前端 Vitest 单元与组件测试                                   | Node.js、Chromium                                              |
| `just test-server-unit`        | 后端 Go 单元测试，启用 race detector                         | Go                                                             |
| `just test-server-e2e`         | Hurl API E2E，真实 HTTP 服务和数据库，无浏览器               | Go、PostgreSQL、Redis、Hurl、psql、curl、Python 3              |
| `just test-fullstack-e2e`      | Playwright 前后端联动，目前为 Chromium 账号流程              | Node.js、Chromium、Go、PostgreSQL、Redis、psql、curl、Python 3 |
| `just test-server-integration` | Go 数据库集成测试，同时运行单元测试，启用 race detector      | Go、PostgreSQL                                                 |
| `just test-web-e2e`            | 五种浏览器／设备配置的界面及 PWA 测试，账号 API 使用模拟响应 | Node.js、Chromium、Firefox、WebKit                             |
| `just test-scripts-unit`       | 公共测试脚本的生命周期测试                                   | Bash、Python 3、Linux `/proc`                                  |
| `just test`                    | 顺序运行以上全部测试，任一步失败即停止                       | 上述全部环境                                                   |

数据库相关的 just 命令优先使用 `TEST_DATABASE_URL`，为空或未设置时回退到 `DATABASE_URL`；两者都没有时明确失败。API E2E 和前后端联动另需 `REDIS_URL`。just 自动加载 `apps/server/.env`，已有环境变量优先。数据库测试仅创建和清理随机临时 schema，不重置开发数据，也不清空 Redis；数据库用户须有创建和删除 schema 的权限。

`test-server-unit` 显式清空 `TEST_DATABASE_URL`，其中数据库集成用例按设计跳过；`test-server-integration` 才验证数据库行为。`test-web-e2e` 显式禁用真实后端账号用例，这些跳过不代表联调通过；`test-fullstack-e2e` 单独验证真实后端。两种 Playwright E2E 都会构建生产版本并占用 4173 端口，不能同时运行。

兼容入口 `just test-server` 等同于 `just test-server-unit`；`just test-web` 运行前端单元／组件测试及模拟 API 的跨浏览器 E2E。

完整测试和静态检查分别运行：

```sh
just test
npm --prefix apps/web run check
npm --prefix apps/web run lint
test -z "$(gofmt -l apps/server)"
go -C apps/server vet ./...
```

测试工具安装、直接脚本调用和详细限制见[前端说明](apps/web/README.md#工程检查)、[后端说明](apps/server/README.md#账号-api)及[公共脚本说明](scripts/README.md)。

## 设计文档

- [产品规则与服务架构](design/README.md)
- [技术方案与歌曲数据模型](design/data-model.md)
- [分享会设计](design/party-mode.md)

## 路线图

任务与实现进度在 [GitHub Project：Cloud Music](https://github.com/users/yuudi/projects/2) 中追踪，按版本、类型和状态整理；具体规则及验收条件记录在关联 Issues 中。
