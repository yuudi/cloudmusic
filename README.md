# Cloud Music

Cloud Music 是一个开源、自托管音乐平台，以移动端为主，同时面向浏览器和 PWA。用户连接自己的存储空间，主动导入歌曲，在线播放，并在后续版本中向好友分享音乐。

项目目前处于设计阶段。客户端选用 Svelte，按需加载 ffmpeg wasm 解码器；服务端选用 Go、Gin 和 GORM，使用 Postgres 与 Redis，通过 rclone 接入存储；初期单机部署，遵循 12-factor，为未来部署到 Kubernetes 保留空间。

## 本地开发

推荐使用 devcontainer，本仓库已包含 devcontainer 和 Postgres 和 Redis 设置。在仓库根目录执行 `just init` 安装前后端依赖，`just db-migrate` 初始化数据库，分别使用 `just dev-server` 和 `just dev-web` 启动服务。

- [前端开发说明](apps/web/README.md)
- [后端开发说明与数据库重置命令](apps/server/README.md)

`just` 列出常用命令：`init`、`dev-web`、`dev-server`、`build-web`、`build-server`、`test-web`、`test-server`、`db-migrate` 和 `db-reset`。`db-reset` 会删除开发数据库中的表并重新执行迁移。

单独安装依赖、格式与静态检查、Redis 重置和完整数据重置的直接命令见上述开发说明。

## 设计文档

- [产品规则与服务架构](design/README.md)
- [技术方案与歌曲数据模型](design/data-model.md)
- [分享会设计](design/party-mode.md)

## 路线图

任务与实现进度在 [GitHub Project：Cloud Music](https://github.com/users/yuudi/projects/2) 中追踪，按版本、类型和状态整理；具体规则及验收条件记录在关联 Issues 中。
