# 开发容器中的数据库连接（仅当使用 devcontainer 开发时参考）

- `.devcontainer/compose.yaml` 为工作容器注入 `DATABASE_URL` 和 `REDIS_URL`，分别通过 Compose 服务名 `postgres` 和 `redis` 连接。PostgreSQL 数据库为 `cloudmusic_dev`，Redis 使用数据库 0。
- 命令执行沙箱可能隔离网络，即使开发容器和数据库服务运行正常，也会出现服务名无法解析或连接失败。遇到此情况时，先检查执行环境的网络隔离，并在获准访问容器网络的环境中验证，再判断 Compose 配置是否有问题。
- 可用 `psql "$DATABASE_URL"` 和 `redis-cli -u "$REDIS_URL" --no-auth-warning PING` 验证连接；连接正常不代表读写及迁移已验证。
- `just db-reset` 会删除 `cloudmusic_dev` 的 `public` schema 并重新运行迁移；`bash .devcontainer/scripts/reset-redis.sh` 会清空 Redis 数据库 0；`bash .devcontainer/scripts/reset-data.sh` 依次执行两者。重置会删除数据，执行前确认目标和影响范围。

# 项目设计

- 歌曲资源、用户持有记录、存储副本、转存授权与 Redis 职责见 [design/data-model.md](design/data-model.md)。该文档是设计约定，不代表业务表已实现；业务迁移随对应功能添加，保留已执行迁移历史。

# 后端 API 开发与验证

- 实现或修改后端 API 时，依据实际路由、认证、请求与响应类型和错误行为同步更新 [OpenAPI 定义](apps/server/openapi/openapi.yaml)，并用 OpenAPI 校验工具检查规范及引用。接口行为有特殊限制时应在定义中写明。
- 为新增或修改的业务规则编写有意义的 Go 单元测试，覆盖正常情况与关键边界；涉及数据库状态、事务或并发时补充数据库集成测试。运行 `just test-server`；需要验证数据库的测试通过 `TEST_DATABASE_URL="$DATABASE_URL" go -C apps/server test -race ./...` 运行。未设置 `TEST_DATABASE_URL` 时数据库集成测试会跳过，应在结果中说明。
- 新增或修改 API 时，同步添加或更新 `apps/server/tests/hurl/` 中的 Hurl 测试，覆盖主要调用流程、认证和关键错误响应，并运行 `TEST_DATABASE_URL="$DATABASE_URL" bash apps/server/tests/hurl/run.sh`。脚本需要 PostgreSQL、Redis、Hurl、psql、curl 和 Python 3；它创建并清理独立的临时 PostgreSQL schema，不重置现有数据。测试入口及环境要求见 [后端开发说明](apps/server/README.md)。
- 如测试因环境或依赖不可用而无法执行，报告未执行的检查和原因，不把跳过视为通过。
