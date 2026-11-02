# 测试后端公共脚本

`test-server.sh` 为 Hurl 和前端真实后端测试复用临时 PostgreSQL schema、后端编译、迁移、启动、就绪等待和清理流程。两个原有测试入口及运行命令保持不变。

## 使用

脚本需要 Bash、Go、psql、curl、Python 3，以及可达的 PostgreSQL 和 Redis。先设置 `TEST_DATABASE_URL` 和 `REDIS_URL`；脚本只读取环境变量，不加载 `.env`。数据库用户须有创建和删除 schema 的权限。

在 Bash 入口中加载脚本，然后直接调用公共函数：

```bash
source "$repo_root/scripts/test-server.sh"

run_tests() {
    hurl --test --variable "base_url=$CLOUDMUSIC_API_URL" \
        "$repo_root/apps/server/tests/hurl/accounts.hurl"
}

with_test_server hurl run_tests
```

`with_test_server PREFIX COMMAND [ARG...]` 接受 Bash 函数或命令，参数按原样传递。前缀须为小写字母开头、只包含小写字母、数字和下划线的 1–31 个字符；随机后缀避免不同测试使用同一个 schema。回调从环境变量 `CLOUDMUSIC_API_URL` 获取本轮后端地址。各测试入口自行检查 Hurl、npm 等专用工具。

加载脚本只定义函数。环境搭建和回调在子 shell 中运行，使用严格错误处理；应作为独立命令调用，不要放在 `if`、`!` 或 `&&`/`||` 条件中，以免 Bash 禁用函数内的自动错误退出。公共函数临时转发调用方收到的 SIGINT/SIGTERM，等待清理完成后恢复调用方的信号处理及 shell 选项。回调中的目录、环境变量和清理钩子不会改变调用方。

## 生命周期与限制

- 每次只在指定数据库创建和删除本轮随机命名的 schema，不重置 `public`，不清空 Redis。
- 数据库 URL 保留其他查询参数，替换 `search_path`；后端只连接本轮 schema。
- 后端绑定 `127.0.0.1` 的随机空闲端口，就绪等待约 10 秒，每次 HTTP 探测最多 1 秒。选择端口后其他进程仍可能抢占，发生时本轮测试失败。
- 迁移和后端启动失败会输出对应日志。结束后删除临时二进制和日志。
- 后端和回调各用独立进程组。正常结束、测试失败及收到 SIGINT/SIGTERM 时，停止测试进程组和后端，再删除 schema 和临时目录。回调不要脱离进程组运行后台服务。
- 测试失败保留其退出码，中断分别返回 130/143。清理失败仍尝试后续清理；测试原本成功时，清理失败会使整体失败，并在 schema 删除失败时打印其名称供人工处理。SIGKILL、容器强制停止等无法捕获的退出仍可能留下临时 schema。
- 前端入口仍使用固定的预览端口 4173，不能同时运行多轮前端 E2E。

现有入口的环境和测试范围见[后端说明](../apps/server/README.md#账号-api)和[前端说明](../apps/web/README.md#工程检查)。

## 验证公共脚本

在仓库根目录运行 `python3 -m unittest discover -s scripts/tests -v`。测试使用标准库和临时伪工具验证环境隔离、退出码、启动日志、清理失败及 SIGINT/SIGTERM 下的子进程清理，不连接数据库、不启动真实 HTTP 服务。进程存活检查使用 Linux `/proc`，适用于项目的 devcontainer。

真实后端验证仍分别运行 `TEST_DATABASE_URL="$DATABASE_URL" bash apps/server/tests/hurl/run.sh` 和 `TEST_DATABASE_URL="$DATABASE_URL" bash apps/web/tests/e2e/run.sh`，环境要求见上述开发说明。
