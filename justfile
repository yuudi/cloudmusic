set positional-arguments := true
set dotenv-load := true
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
    go -C apps/server build -o bin/reset-password ./cmd/reset-password

# 本地运行后端
dev-server:
    go -C apps/server run ./cmd/server

# 同时运行前后端，按 Ctrl+C 一起停止
[parallel]
dev: dev-server dev-web

# 顺序运行全部测试，需要 PostgreSQL、Redis 和三个 Playwright 浏览器
test: test-web-unit test-server-unit test-server-integration test-scripts-unit test-web-e2e test-server-e2e test-fullstack-e2e

# 前端单元与组件测试，组件测试使用无头 Chromium
test-web-unit:
    npm --prefix apps/web run test:unit

# 后端单元测试，禁用数据库集成测试
test-server-unit:
    TEST_DATABASE_URL= go -C apps/server test -race ./...

# 后端 Go 数据库集成测试，同时运行单元测试
test-server-integration:
    TEST_DATABASE_URL="${TEST_DATABASE_URL:-${DATABASE_URL:?Set TEST_DATABASE_URL or DATABASE_URL to a PostgreSQL test database}}" go -C apps/server test -race -count=1 ./...

# 公共测试脚本的生命周期单元测试
test-scripts-unit:
    python3 -m unittest discover -s scripts/tests -v

# 跨浏览器界面与 PWA 测试，账号 API 使用模拟响应
test-web-e2e:
    CLOUDMUSIC_ACCOUNT_E2E= npm --prefix apps/web run test:e2e

# 后端 API E2E，使用 Hurl、真实后端及临时数据库 schema
test-server-e2e:
    TEST_DATABASE_URL="${TEST_DATABASE_URL:-${DATABASE_URL:?Set TEST_DATABASE_URL or DATABASE_URL to a PostgreSQL test database}}" bash apps/server/tests/hurl/run.sh

# 前后端联动 E2E，使用 Chromium、真实后端及临时数据库 schema
test-fullstack-e2e:
    TEST_DATABASE_URL="${TEST_DATABASE_URL:-${DATABASE_URL:?Set TEST_DATABASE_URL or DATABASE_URL to a PostgreSQL test database}}" bash apps/web/tests/e2e/run.sh

# 兼容原后端测试入口，运行不依赖数据库的单元测试

alias test-server := test-server-unit

# 兼容原前端测试入口，运行单元、组件及模拟 API 的跨浏览器测试
test-web: test-web-unit test-web-e2e

# 执行数据库迁移（可使用 up、down 或 status）
db-migrate command="up":
    go -C apps/server run ./cmd/migrate "$1"

# 清空开发数据库并重新建表，仅用于 cloudmusic_dev
db-reset:
    bash .devcontainer/scripts/reset-postgres.sh

# 构建前端，输出至 apps/web/build
build-web *args:
    npm --prefix apps/web run build -- "$@"

# 本地运行前端，可追加 --port 等参数
dev-web *args:
    npm --prefix apps/web run dev -- --host 0.0.0.0 "$@"
