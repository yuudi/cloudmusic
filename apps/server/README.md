# Cloud Music 服务端

Go 1.27.3、Gin、Postgres + GORM、Redis。Go 依赖通过 `go.mod` 和 `go.sum` 固定。提供基础账号认证；歌曲、存储接入和音频转发业务尚未实现。

## 本地开发

在 VS Code 执行 **Dev Containers: Rebuild and Reopen in Container**。`.devcontainer/compose.yaml` 启动开发容器、Postgres 18.6 和 Redis 8.10.3，数据库就绪后才启动开发容器。保留 Node.js 26、Go 和 just，增加 psql 与 redis-cli。

在仓库根目录执行：

```sh
go -C apps/server mod download
just db-migrate
just dev-server
```

另开终端执行 `just dev-web`，或直接使用 `just dev` 同时启动前后端。Vite 默认将 `/api` 代理到 `http://localhost:8080`，浏览器使用同源 API，不需要开放 CORS。生产环境同样通过同源 `/api` 访问；当前 Go 服务不托管前端静态文件。just 的版本要求见[根目录开发说明](../../README.md#本地开发)。

Compose 自动注入 `DATABASE_URL` 和 `REDIS_URL`，使用 `postgres`、`redis` 服务名；不必创建 `.env`。数据库和 Redis 使用独立命名卷，重建开发容器保留数据，容器启动不自动执行迁移或重置。

容器外运行 Go 时，先转发数据库与 Redis 端口，将 `.env.example` 复制为 `apps/server/.env`，再从根目录使用 `just`。just 自动读取该文件，已有环境变量优先。Go 程序和直接执行的脚本只读取环境变量；容器外使用下文的直接命令前，可在 Bash 中加载本地配置：

```sh
set -a
source apps/server/.env
set +a
```

开发数据库为 `cloudmusic_dev`，用户名和密码均为 `cloudmusic`；仅用于开发。开发容器端口转发包括 5173、8080、Postgres 5432 和 Redis 6379；Compose 不向宿主机直接发布数据库端口。

## 配置

| 环境变量                   | 默认值  | 用途                                                |
| -------------------------- | ------- | --------------------------------------------------- |
| `DATABASE_URL`             | 必填    | Postgres URL，支持 `postgres://`、`postgresql://`   |
| `REDIS_URL`                | 必填    | Redis URL，支持 `redis://`、`rediss://`             |
| `HTTP_ADDR`                | `:8080` | HTTP 监听地址                                       |
| `TRUSTED_PROXIES`          | 空      | 可信反向代理 IP 或 CIDR，逗号分隔；空表示不信任代理 |
| `LOG_LEVEL`                | `info`  | slog 日志等级                                       |
| `DB_MAX_OPEN_CONNS`        | `10`    | 数据库最大连接数，必须大于 0                        |
| `DB_MAX_IDLE_CONNS`        | `5`     | 空闲连接数，0 到最大连接数                          |
| `DB_CONN_MAX_LIFETIME`     | `30m`   | 连接最长存活时间                                    |
| `DEPENDENCY_TIMEOUT`       | `3s`    | 启动时每项依赖及完整就绪探测的超时                  |
| `HTTP_READ_HEADER_TIMEOUT` | `5s`    | HTTP 请求头读取超时                                 |
| `HTTP_IDLE_TIMEOUT`        | `60s`   | HTTP keep-alive 空闲超时                            |
| `SHUTDOWN_TIMEOUT`         | `10s`   | 等待 HTTP 请求结束的退出超时                        |

时间参数必须为正数，格式如 `500ms`、`3s`、`30m`。配置错误或启动时依赖不可达会退出并返回非零状态。日志为 stdout 上的 JSON；请求日志不记录查询参数、请求体和请求头。进程收到 SIGINT/SIGTERM 后停止接收新连接，等待已有请求完成，再关闭数据库和 Redis 连接。反向代理信任默认关闭；使用代理时通过 `TRUSTED_PROXIES` 显式配置可信代理的 IP 或 CIDR。仅指定受控代理，代理必须清理或正确重写客户端提供的转发头，并确保不能绕过代理访问服务；否则客户端可能伪造来源 IP 并绕过限流。

## 健康检查

- `GET /api/v1/health/live`：进程正常即返回 200，不检查外部依赖。
- `GET /api/v1/health/ready`：检查 Postgres、Redis，两者可达时返回 200，否则返回 503；响应只包含依赖状态，不暴露内部错误。

```sh
curl http://localhost:8080/api/v1/health/live
curl http://localhost:8080/api/v1/health/ready
```

就绪检查只验证连接，不验证迁移版本；部署流程应先执行迁移，再启动服务。

## 数据库迁移与重置

迁移使用 Goose，SQL 位于 `migrations/` 并嵌入迁移二进制，与 GORM 的业务查询职责分开。服务启动不调用 `AutoMigrate`。迁移工具只要求数据库配置，不依赖 Redis。

```sh
just db-migrate          # 应用所有待执行迁移
just db-migrate status   # 查看版本
just db-migrate down     # 回滚一个版本
```

`00001_placeholder.sql` 保留原有迁移历史；`00002_accounts.sql` 定义账号、密码凭据和会话。后续通过新增编号 SQL 文件演进表结构，保留已执行的迁移历史；每个文件包含 `-- +goose Up` 与 `-- +goose Down`。

下面的重置命令会删除开发数据，只能手动执行：

```sh
just db-reset                              # 删除 cloudmusic_dev 的 public schema，再重新执行迁移
bash .devcontainer/scripts/reset-redis.sh   # FLUSHDB，仅清空本地 Redis 的数据库 0
bash .devcontainer/scripts/reset-data.sh    # 依次重置 Postgres 与 Redis
```

脚本位于 `.devcontainer/scripts/`，数据库重置 SQL 位于 `.devcontainer/sql/`。Postgres 重置会检查库名为 `cloudmusic_dev`，保留数据库和角色；Redis 重置只接受 `redis`、`localhost`、`127.0.0.1` 上的 6379 端口、数据库 0。重置过程中有任何命令失败，后续步骤停止。

## 工程命令与目录

以下命令在仓库根目录执行：

| 命令                | 用途                                    |
| ------------------- | --------------------------------------- |
| `just init`         | 安装前后端依赖                          |
| `just dev`          | 同时启动前后端                          |
| `just dev-server`   | 本地启动                                |
| `just build-server` | 构建服务、迁移工具到 `apps/server/bin/` |
| `just test-server`  | 带 race detector 的测试，无需外部数据库 |

仅下载后端依赖或格式化代码时，直接执行：

```sh
go -C apps/server mod download
go -C apps/server fmt ./...
```

完整检查按顺序执行格式检查、静态检查、测试与构建：

```sh
test -z "$(gofmt -l apps/server)"  # 不通过时用 gofmt -l apps/server 查看文件
go -C apps/server vet ./...
just test-server
just build-server
```

- `cmd/server`：服务入口及进程生命周期。
- `cmd/migrate`：独立迁移入口。
- `cmd/reset-password`：管理员交互式重置密码。
- `internal/auth`：账号、密码凭据、会话与游客清理。
- `internal/config`：环境变量读取与校验。
- `internal/database`：GORM、Postgres 连接池。
- `internal/cache`：Redis 连接。
- `internal/httpserver`：路由、日志、panic 恢复与健康检查。
- `migrations`：版本化 SQL。

测试覆盖无效配置、连接串脱敏、依赖故障时的存活/就绪状态、探测超时及 HTTP panic 恢复。账号数据库集成测试另覆盖游客升级、资料与会话、密码重置并发及游客清理；设置 `TEST_DATABASE_URL` 后会创建和清理随机临时 schema，未设置时跳过。开发容器重建需要 Docker。

## 账号 API

接口类型、认证方式和响应状态详见 [OpenAPI 规范](openapi/openapi.yaml)。

所有接口位于 `/api/v1/auth`，请求和响应使用 JSON。`POST /guest` 创建游客，`POST /register` 接收 `username`、`password`、`display_name`，`POST /login` 接收 `username`、`password`。成功返回 `user`、`token`、`token_type`、`expires_in`。用户名为 3–32 位 ASCII 字母、数字或下划线，比较忽略大小写；密码为 8–1024 字节，显示名最多 100 个 Unicode 字符。密码采用 Argon2id PHC 格式（版本 19，内存 64 MiB，迭代 3 次，并行度 4），记录算法、参数与随机盐；旧格式在成功登录后自动升级。数据库只保存随机会话令牌的 SHA-256 摘要。

下列接口须带 `Authorization: Bearer <token>`：

- `GET /me`：当前身份。
- `PATCH /me`：可选提供 `display_name`、`username`（非空用户名仅正式账号可设置）；省略或传 null 的字段保持不变，空显示名会清空，空用户名保持不变。计为用户活动，更新最后活跃时间并延长当前会话有效期；成功返回 204。
- `POST /upgrade`：游客提供 `username`、`password`，保留用户 ID 和当前会话。
- `POST /logout` 和 `POST /logout-all`：分别撤销当前会话和全部会话。
- `POST /password`：正式账号提供 `old_password`、`new_password`，成功后撤销全部会话，需重新登录。

会话在认证成功的活动请求后续期 30 天；`GET /me` 用于检查会话，不计为活动。游客超过 30 天没有活动即不可使用；服务启动后及每小时清理过期游客，撤销其会话并匿名化用户记录，保留 UUID。后续业务接入须在同一清理流程中处理个人数据，并区分用户活动与后台轮询；播放活动也需更新活跃时间。用户数据访问必须按当前用户 ID 限制，游客不能连接外部网盘，此约束须在网盘功能实现时执行。

匿名创建与登录接口在单个服务进程中按来源地址合计限制每分钟 10 次；反向代理信任默认关闭；代理部署时需正确设置 `TRUSTED_PROXIES`，并在入口按真实客户端地址限流。生产环境须使用 HTTPS，保护 Bearer 令牌；退出游客会失去该设备的游客身份。服务端不在日志中记录令牌或密码。

管理员在服务器的交互终端运行 `go -C apps/server run ./cmd/reset-password USERNAME`，输入并确认新密码。该命令仅要求数据库配置，重置后撤销全部会话。构建产物也可通过 `apps/server/bin/reset-password USERNAME` 运行。

数据库集成测试使用 `TEST_DATABASE_URL`，会在目标库创建并删除名称随机的独立测试 schema；未设置时跳过。普通测试命令不需要数据库。

账号 Hurl 端到端测试要求已安装 `hurl`、`psql`、`curl` 和 Python 3，并提供 PostgreSQL 与 Redis：

```sh
TEST_DATABASE_URL="$DATABASE_URL" bash apps/server/tests/hurl/run.sh
```

脚本在指定 PostgreSQL 数据库中创建随机名称的临时 schema，执行迁移并在本地随机端口启动服务；结束时停止服务并删除该 schema。Redis 仅供服务连接及健康检查，不会清空数据。`TEST_DATABASE_URL` 指向的数据库用户需要创建 schema 的权限。
