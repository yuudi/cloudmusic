set positional-arguments
set dotenv-load
set dotenv-path := "apps/server/.env"

# 列出可用命令
default:
    @just --list

# 初始化前后端
init:
    npm --prefix apps/web ci
    go -C apps/server mod download

# 构建后端服务与迁移工具，输出至 apps/server/bin
build-server:
    go -C apps/server build -o bin/server ./cmd/server
    go -C apps/server build -o bin/migrate ./cmd/migrate

# 本地运行后端
dev-server:
    go -C apps/server run ./cmd/server

# 运行后端测试
test-server:
    go -C apps/server test -race ./...

# 执行数据库迁移（可使用 up、down 或 status）
db-migrate command="up":
    go -C apps/server run ./cmd/migrate "$1"

# 清空开发数据库并重新建表，仅用于 cloudmusic_dev
db-reset:
    bash .devcontainer/scripts/reset-postgres.sh

# 构建前端，输出至 apps/web/build
build-web *args:
    npm --prefix apps/web run build -- "$@"

# 运行前端全部测试，内含生产构建
test-web:
    npm --prefix apps/web test

# 本地运行前端，可追加 --port 等参数
dev-web *args:
    npm --prefix apps/web run dev -- --host 0.0.0.0 "$@"
