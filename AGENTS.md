# 开发容器中的数据库连接（仅当使用 devcontainer 开发时参考）

- `.devcontainer/compose.yaml` 为工作容器注入 `DATABASE_URL` 和 `REDIS_URL`，分别通过 Compose 服务名 `postgres` 和 `redis` 连接。PostgreSQL 数据库为 `cloudmusic_dev`，Redis 使用数据库 0。
- 命令执行沙箱可能隔离网络，即使开发容器和数据库服务运行正常，也会出现服务名无法解析或连接失败。遇到此情况时，先检查执行环境的网络隔离，并在获准访问容器网络的环境中验证，再判断 Compose 配置是否有问题。
- 可用 `psql "$DATABASE_URL"` 和 `redis-cli -u "$REDIS_URL" --no-auth-warning PING` 验证连接；连接正常不代表读写及迁移已验证。
- `just db-reset` 会删除 `cloudmusic_dev` 的 `public` schema 并重新运行迁移；`bash .devcontainer/scripts/reset-redis.sh` 会清空 Redis 数据库 0；`bash .devcontainer/scripts/reset-data.sh` 依次执行两者。重置会删除数据，执行前确认目标和影响范围。
