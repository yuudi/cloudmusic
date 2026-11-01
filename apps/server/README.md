# Cloud Music 服务端

Go 1.27.3、Gin、Postgres + GORM、Redis。Go 依赖通过 `go.mod` 和 `go.sum` 固定。当前提供工程骨架，不包含认证、歌曲、存储接入或音频转发业务。

## 本地开发

在 VS Code 执行 **Dev Containers: Rebuild and Reopen in Container**。`.devcontainer/compose.yaml` 启动开发容器、Postgres 18.6 和 Redis 8.10.3，数据库就绪后才启动开发容器。保留 Node.js 26、Go 和 just，增加 psql 与 redis-cli。

在仓库根目录执行：

```sh
go -C apps/server mod download
just db-migrate
just dev-server
```

另开终端执行 `just dev-web`。Vite 将 `/api` 代理到 `http://localhost:8080`，浏览器使用同源 API，不需要开放 CORS。生产环境同样通过同源 `/api` 访问；当前 Go 服务不托管前端静态文件。

Compose 自动注入 `DATABASE_URL` 和 `REDIS_URL`，使用 `postgres`、`redis` 服务名；不必创建 `.env`。数据库和 Redis 使用独立命名卷，重建开发容器保留数据，容器启动不自动执行迁移或重置。

容器外运行 Go 时，先转发数据库与 Redis 端口，将 `.env.example` 复制为 `apps/server/.env`，再从根目录使用 `just`。just 自动读取该文件，已有环境变量优先。Go 程序和直接执行的脚本只读取环境变量；容器外使用下文的直接命令前，可在 Bash 中加载本地配置：

```sh
set -a
source apps/server/.env
set +a
```

开发数据库为 `cloudmusic_dev`，用户名和密码均为 `cloudmusic`；仅用于开发。开发容器端口转发包括 5173、8080、Postgres 5432 和 Redis 6379；Compose 不向宿主机直接发布数据库端口。

## 配置

| 环境变量                   | 默认值  | 用途                                              |
| -------------------------- | ------- | ------------------------------------------------- |
| `DATABASE_URL`             | 必填    | Postgres URL，支持 `postgres://`、`postgresql://` |
| `REDIS_URL`                | 必填    | Redis URL，支持 `redis://`、`rediss://`           |
| `HTTP_ADDR`                | `:8080` | HTTP 监听地址                                     |
| `LOG_LEVEL`                | `info`  | slog 日志等级                                     |
| `DB_MAX_OPEN_CONNS`        | `10`    | 数据库最大连接数，必须大于 0                      |
| `DB_MAX_IDLE_CONNS`        | `5`     | 空闲连接数，0 到最大连接数                        |
| `DB_CONN_MAX_LIFETIME`     | `30m`   | 连接最长存活时间                                  |
| `DEPENDENCY_TIMEOUT`       | `3s`    | 启动时每项依赖及完整就绪探测的超时                |
| `HTTP_READ_HEADER_TIMEOUT` | `5s`    | HTTP 请求头读取超时                               |
| `HTTP_IDLE_TIMEOUT`        | `60s`   | HTTP keep-alive 空闲超时                          |
| `SHUTDOWN_TIMEOUT`         | `10s`   | 等待 HTTP 请求结束的退出超时                      |

时间参数必须为正数，格式如 `500ms`、`3s`、`30m`。配置错误或启动时依赖不可达会退出并返回非零状态。日志为 stdout 上的 JSON；请求日志不记录查询参数、请求体和请求头。进程收到 SIGINT/SIGTERM 后停止接收新连接，等待已有请求完成，再关闭数据库和 Redis 连接。反向代理信任默认关闭，部署时应根据实际代理地址显式配置。

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

`00001_placeholder.sql` 只执行 `CREATE TABLE bootstrap_placeholder ();`，暂不定义业务字段。后续通过新增编号 SQL 文件演进表结构，保留已执行的迁移历史；每个文件包含 `-- +goose Up` 与 `-- +goose Down`。

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
- `internal/config`：环境变量读取与校验。
- `internal/database`：GORM、Postgres 连接池。
- `internal/cache`：Redis 连接。
- `internal/httpserver`：路由、日志、panic 恢复与健康检查。
- `migrations`：版本化 SQL。

测试覆盖无效配置、连接串脱敏、依赖故障时的存活/就绪状态、探测超时及 HTTP panic 恢复；真实数据库迁移和开发容器启动需在具备 Docker 的环境中验证。
