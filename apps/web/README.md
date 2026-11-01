# Cloud Music 客户端

Svelte 5 + SvelteKit 3，TypeScript strict，SCSS，npm。客户端使用文件路由和 History URL，以纯 SPA 运行，静态文件由 Go 服务或反向代理托管。

## 本地开发

使用 Node.js 26 或更新版本，与仓库的 devcontainer 保持一致。所有命令在 `apps/web` 执行，无需在仓库根目录建立 npm workspace。

```sh
npm ci
npm run dev -- --host 0.0.0.0
```

首页 `/` 和设置页 `/settings` 用于验证路由。当前只提供工程骨架，不包含认证、曲库、存储接入或音频播放业务。

- 页面放在 `src/routes`，可复用组件放在 `src/lib/components`。
- 全局样式位于 `src/lib/styles/global.scss`，组件样式使用 `<style lang="scss">`。
- UI 文案暂用简体中文，未引入国际化库。
- 后续客户端通过同源 `/api` 访问 Go 服务；生产部署须优先将 API 和音频路径交给后端。
- 开发服务器已将 `/api` 代理到 `http://localhost:8080`，配合根目录的 `just dev-server` 使用。

## 工程检查

首次运行浏览器测试前安装浏览器。Linux 容器还需要 Playwright 的系统库：

```sh
npm run test:install
# Linux 容器按需执行，安装系统库时可能需要 sudo：
npx playwright install-deps chromium firefox webkit
```

| 命令                      | 用途                                                |
| ------------------------- | --------------------------------------------------- |
| `npm run check`           | Svelte 和 TypeScript 类型检查，包括端到端测试代码   |
| `npm run lint`            | Prettier 格式检查和 ESLint                          |
| `npm run format`          | 格式化代码                                          |
| `npm run test:unit`       | Vitest 单次执行，包含 Chromium 中的 Svelte 组件测试 |
| `npm run test:unit:watch` | Vitest 监听模式                                     |
| `npm run test:e2e`        | 构建生产版本并执行 Playwright 测试                  |
| `npm test`                | Vitest 和 Playwright                                |
| `npm run validate`        | 类型、格式、lint 和全部测试；端到端测试内含生产构建 |
| `npm run build`           | 输出静态部署文件到 `build/`                         |
| `npm run preview`         | 本地预览生产版本，不作为生产服务使用                |

Vitest 为 Node 单元测试和浏览器组件测试分别保留项目；当前组件测试验证更新必须经过用户确认。Playwright 检查客户端导航、前进后退、深层链接刷新、安装清单与图标、离线冷启动，以及 API 和音频请求不会自动缓存。测试使用生产构建，避免开发服务器掩盖 Service Worker 问题。

自动化目标包含 Chromium、Firefox、WebKit，以及 Android Chrome 和 iPhone Safari 的设备模拟。产品目标覆盖 iOS Safari / 主屏幕 PWA、Android Chrome 和桌面主流浏览器。设备模拟不能替代真机验收；PWA 安装、后台连续播放、锁屏和耳机控制、音频焦点与来电打断需在播放器实现后验证。

## PWA

使用 `@vite-pwa/sveltekit` 的 `withPwa` 集成和 `generateSW` 策略：

- 缓存应用入口、页面脚本、样式和图标。首次成功联网并完成 Service Worker 安装后，可以离线打开界面和路由。
- 不做业务 API、歌曲文件或其他运行时资源缓存。离线播放和歌曲下载尚未实现。
- `/api`、`/audio` 和文件路径不使用 SPA 导航回退；新增后端路径时也应更新 `vite.config.ts` 中的排除规则。
- 新版本等待用户点击“刷新更新”后激活并刷新页面；“稍后”关闭本次提示，避免自动中断播放。所有应用标签页关闭后，浏览器也可以自然激活等待中的版本。
- 返回前台或恢复联网时检查更新，注册和检查失败记录到控制台，不阻断正常页面。
- 默认仅在生产构建注册 Service Worker。通过 `npm run build` 和 `npm run preview` 验证离线行为。

图标源文件是 `static/icon.svg`，已提交标准 PNG、maskable PNG 和 Apple 触屏图标。修改源文件后运行 `npm run icons` 重新生成，需要已安装 Chromium。正式品牌图标可在后续替换。

## 部署约定

部署到域名根路径 `/`。`build/200.html` 是 SPA 回退入口；静态服务必须把页面深层链接（例如 `/settings`）回退到该文件，同时让 `/` 返回该入口。API、音频以及缺失的静态资源不能回退为 HTML。

以 Nginx 的静态部分为例，另行配置 API 和音频到 Go 的代理规则，并确保优先匹配：

```nginx
root /path/to/web/build;
index 200.html;

location / {
    try_files $uri /200.html;
}

location ~* \.[^/]+$ {
    try_files $uri =404;
}

location = /200.html {
    add_header Cache-Control "no-cache";
}

location = /sw.js {
    add_header Cache-Control "no-cache";
}

location ^~ /_app/immutable/ {
    add_header Cache-Control "public, max-age=31536000, immutable";
}
```

HTML、Service Worker 和 web manifest 应使用可及时重新验证的 HTTP 缓存策略，带内容哈希的静态资源可以长期缓存。生产环境使用 HTTPS，localhost 可用于开发。若改为子路径部署，需同时调整 SvelteKit base、路由链接、manifest 的 id/start_url/scope、图标 URL 与 Service Worker 回退路径，不能只改反向代理。

依赖版本通过 `package-lock.json` 固定。初始化时 npm audit 仍报告部分构建与检查工具的间接依赖存在低危或中危告警；未采用建议中的破坏性版本降级。升级工具链时应复查审计和兼容性。
