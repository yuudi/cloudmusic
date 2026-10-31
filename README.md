# Cloud Music

Cloud Music 是一个开源、自托管音乐平台，以移动端为主，同时面向浏览器和 PWA。用户连接自己的存储空间，主动导入歌曲，在线播放，并在后续版本中向好友分享音乐。

项目目前处于设计阶段。客户端选用 Flutter，服务端选用 Go，通过 rclone 接入存储；初期单机部署，使用数据库和 Redis，遵循 12-factor，为未来部署到 Kubernetes 保留空间。

## 设计文档

- [产品规则与服务架构](design/README.md)
- [分享会设计](design/party-mode.md)

## 路线图

任务与实现进度在 [GitHub Project：Cloud Music](https://github.com/users/yuudi/projects/2) 中追踪，按版本、类型和状态整理；具体规则及验收条件记录在关联 Issues 中。
