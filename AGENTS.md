# 开发容器中的数据库连接（仅当使用 devcontainer 开发时参考）

- `.devcontainer/compose.yaml` 为工作容器注入 `DATABASE_URL` 和 `REDIS_URL`，分别通过 Compose 服务名 `postgres` 和 `redis` 连接。PostgreSQL 数据库为 `cloudmusic_dev`，Redis 使用数据库 0。
- 首次启动服务前，以及新增或拉取迁移后，需手动执行 `just db-migrate`。
- 命令执行沙箱可能隔离网络，即使开发容器和数据库服务运行正常，也会出现服务名无法解析或连接失败。遇到此情况时，先检查执行环境的网络隔离，并在获准访问容器网络的环境中验证，再判断 Compose 配置是否有问题。
- 可用 `psql "$DATABASE_URL"` 和 `redis-cli -u "$REDIS_URL" --no-auth-warning PING` 验证连接；连接正常不代表读写及迁移已验证。
- `just db-reset` 会删除 `cloudmusic_dev` 的 `public` schema 并重新运行迁移；`bash .devcontainer/scripts/reset-redis.sh` 会清空 Redis 数据库 0；`bash .devcontainer/scripts/reset-data.sh` 依次执行两者。重置会删除数据，执行前确认目标和影响范围。

# 项目设计

- 歌曲资源、用户持有记录、存储副本、转存授权与 Redis 职责见 [design/data-model.md](design/data-model.md)。该文档是设计约定，不代表业务表已实现；业务迁移随对应功能添加，保留已执行迁移历史。

# 脚本与命令

- 各种脚本的用法应记录在对应应用的 `apps/*/README.md` 中，并在脚本用法变更时同步更新。
- 根目录的 `justfile` 仅存放供人直接使用的命令。
- 测试统一从根目录的 just 命令进入：`test-web-unit`、`test-server-unit`、`test-server-integration`、`test-server-e2e`、`test-web-e2e`、`test-fullstack-e2e` 和 `test-scripts-unit`；`just test` 顺序运行全部测试，任一步失败即停止。数据库相关入口优先使用 `TEST_DATABASE_URL`，为空或未设置时回退到 `DATABASE_URL`，两者均未配置时失败；API E2E 和前后端联动另需 `REDIS_URL`。

# 后端 API 开发与验证

- 实现或修改后端 API 时，依据实际路由、认证、请求与响应类型和错误行为同步更新 [OpenAPI 定义](apps/server/openapi/openapi.yaml)，并用 OpenAPI 校验工具检查规范及引用。接口行为有特殊限制时应在定义中写明。
- 为新增或修改的业务规则编写有意义的 Go 单元测试，覆盖正常情况与关键边界；涉及数据库状态、事务或并发时补充数据库集成测试。分别运行 `just test-server-unit` 和 `just test-server-integration`。单元测试入口显式禁用数据库集成测试，不能替代数据库验证；直接运行 Go 测试且未设置 `TEST_DATABASE_URL` 时，数据库集成用例会跳过，应在结果中说明。
- 新增或修改 API 时，同步添加或更新 `apps/server/tests/hurl/` 中的 Hurl 测试，覆盖主要调用流程、认证和关键错误响应，并运行 `just test-server-e2e`。脚本需要 Go、PostgreSQL、Redis、Hurl、psql、curl 和 Python 3；它创建并清理独立的临时 PostgreSQL schema，不重置现有数据。测试入口及环境要求见 [后端开发说明](apps/server/README.md)。
- 如测试因环境或依赖不可用而无法执行，报告未执行的检查和原因，不把跳过视为通过。

# 前端账号与验证

- 前端采用 Tailwind CSS 和 Skeleton，使用黑白灰底色、深红强调色，深浅色跟随系统。API 类型由 OpenAPI 生成，修改 API 后在 `apps/web` 执行 `npm run generate:api`，不要手动修改 `src/lib/api/schema.d.ts`。
- 前端国际化使用 `svelte-i18n`，支持 `en` 和 `zh-CN`，英文为回退语言。新增 UI 文案须同步维护 `apps/web/src/lib/i18n/messages/` 中的两份字典；错误和通知保存消息键，在渲染时翻译。语言切换不得刷新页面、重置身份或表单，也不改变路由。用户内容不翻译；国际化规则及测试见前端 README。
- `just test-web-unit` 包含账号客户端和浏览器会话测试；`just test-web-e2e` 使用模拟 API 验证账号边界。交付账号功能时还应在仓库根目录运行 `just test-fullstack-e2e`，验证真实后端，脚本仅创建和清理随机临时 PostgreSQL schema。不要把普通 E2E 中跳过的真实后端用例视为联调通过。
