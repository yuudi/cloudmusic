# Cloud Music 客户端

Svelte 5 + SvelteKit 3，TypeScript strict，Tailwind CSS 4 + Skeleton，npm。客户端使用文件路由和 History URL，以纯 SPA 运行，静态文件由 Go 服务或反向代理托管。

## 本地开发

使用 Node.js 26 或更新版本，与仓库的 devcontainer 保持一致。所有命令在 `apps/web` 执行，无需在仓库根目录建立 npm workspace。

```sh
npm ci
npm run dev -- --host 0.0.0.0
```

首页 `/` 提供登录、注册和游客入口；`/login`、`/register` 提供账号表单；`/settings` 支持资料修改、游客升级、改密及当前或全部会话退出。曲库、存储接入和音频播放尚未实现。

- 页面放在 `src/routes`，可复用组件放在 `src/lib/components`。
- 全局样式位于 `src/lib/styles/global.css`，组件可使用 Tailwind 和普通 CSS。Skeleton 保留 mint 的尺寸和排版，覆盖为黑白灰底色与深红强调色；深浅色跟随系统设置。
- UI 使用 `svelte-i18n` 支持英文和简体中文，英文为回退语言。
- 客户端通过同源 `/api` 访问 Go 服务；生产部署须优先将 API 和音频路径交给后端。
- 开发服务器已将 `/api` 代理到 `http://localhost:8080`，配合根目录的 `just dev-server` 使用。

账号使用后端返回的 Bearer 令牌，保存在本站 `localStorage`，刷新后通过 `/api/v1/auth/me` 恢复身份，并同步同源标签页的身份变更。不要在共享设备保留登录；游客退出或清除浏览器数据后无法找回。断网或服务异常时保留凭据以便重试，只有确认会话无效后才清除。游客升级保持 UUID 和当前令牌；切换到已有账号不合并数据。当前修改密码和“退出所有设备”均会撤销全部会话。

资料与密码表单分别保留草稿。同一账号重新验证会话（例如当前密码输入错误）不会覆盖未保存的用户名和显示名；资料保存成功后再同步服务端返回的最新资料。切换到不同账号时重新初始化表单。游客升级后填入正式用户名，同时保留未保存的显示名草稿。

API 类型由 `apps/server/openapi/openapi.yaml` 生成到 `src/lib/api/schema.d.ts`；修改 API 后执行 `npm run generate:api` 并提交生成文件。生成工具使用固定版本的独立 npm CLI 环境，不加入前端依赖树；首次运行需要网络下载工具。开发代理默认连接 8080，也可通过 `CLOUDMUSIC_API_URL` 指定测试后端。

## 国际化

语言选择器位于页头。首次访问按浏览器的语言优先级匹配英文或中文；未匹配时使用英文。手动选择保存在独立的 `localStorage` 项 `cloudmusic.locale`，优先于浏览器设置，并同步到其他标签页。存储被禁用时仍可在当前页面切换。

两份字典位于 `src/lib/i18n/messages/en.json` 和 `zh-CN.json`，随应用一起打包，首次成功缓存应用后离线也可切换。保持 `/login`、`/settings` 等路径不变。切换语言即时更新界面、标题、描述和 `<html lang>`，不会整页刷新或重置表单、身份及应用状态。PWA 安装清单是静态资源，其描述与语言使用英文回退，不随页面切换。

新增文案时同时更新两份字典，使用语义化消息键和 ICU 参数。组件从 `#lib/i18n/index.js` 导入 `t`，例如 `{$t('home.welcome', { name })}`；消息键由英文词典提供类型约束。错误和通知保存 `MessageState` 消息键，在渲染时翻译，避免切换语言后保留旧语言文本。用户输入、用户名、歌曲和歌单内容不翻译。日期和数字按当前语言使用 `Intl` 或库提供的格式化函数。

`npm run test:unit` 检查语言优先级、英文回退、字典键和 ICU 参数一致性，以及 PWA 失败提示的即时翻译。E2E 默认明确设置中文浏览器环境；`e2e/i18n.spec.ts` 另验证英文和不支持的语言、表单保留、标签页同步、存储不可用和离线切换。

## 工程检查

首次运行浏览器测试前按需安装浏览器。devcontainer 中可使用脚本同时安装系统库（以下命令在 `apps/web` 执行）：

```sh
# 只运行组件测试时，默认安装 Chromium
bash ../../.devcontainer/scripts/install-playwright.sh

# 完整 E2E 测试需要三个浏览器
bash ../../.devcontainer/scripts/install-playwright.sh chromium firefox webkit
```

先执行 `npm ci` 安装项目依赖。脚本使用项目中的 Playwright 版本，需要联网下载浏览器和系统包，安装系统库时可能需要 sudo；请以 `vscode` 用户运行脚本，浏览器保存在该用户的缓存目录。安装按需手动执行，不加入 devcontainer 的自动初始化流程。脚本不安装 Node.js、npm 项目依赖或桌面环境；重建容器或升级 Playwright 后可重新运行。其他环境可使用 `npm run test:install` 安装浏览器，Linux 上另用 `npx playwright install-deps chromium firefox webkit` 安装系统依赖。

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
| `npm run generate:api`    | 从后端 OpenAPI 重新生成前端 API 类型                |

Vitest 为 Node 单元测试和浏览器组件测试分别保留项目，覆盖 API 客户端、会话恢复、独立表单草稿、字典完整性和 PWA 更新确认。Playwright 检查客户端导航、前进后退、深层链接刷新、安装清单与图标、离线冷启动，以及 API 和音频请求不会自动缓存。测试使用生产构建，避免开发服务器掩盖 Service Worker 问题。模拟 API 的用例禁用 Service Worker，保证网络模拟覆盖新标签页；离线用例保持 Service Worker 启用。

账号测试另覆盖输入规则、会话恢复、断网重试、游客升级和过期处理。默认 E2E 中的账号界面测试使用模拟 API；真实后端流程通过以下脚本运行（在仓库根目录执行，需要 PostgreSQL、Redis、Go、psql、curl、Python 3 和已安装的 Chromium）：

```sh
TEST_DATABASE_URL="$DATABASE_URL" bash apps/web/tests/e2e/run.sh
```

脚本创建随机临时 PostgreSQL schema，执行迁移，启动独立端口的后端，再运行 Chromium 账号测试；结束后停止后端并删除该 schema，不重置开发数据。覆盖游客升级的身份继承、资料隔离、大小写登录、当前会话退出、改密撤销所有会话和退出所有设备。普通 `npm run test:e2e` 跳过此真实后端用例；运行上述脚本才算完成联调验证。

`TEST_DATABASE_URL` 对应的用户须有创建和删除测试 schema 的权限；`REDIS_URL` 必须指向可达的 Redis，脚本不清空 Redis。脚本直接读取环境变量，不加载 `apps/server/.env`；容器外使用时先按后端 README 加载配置。预览服务器固定使用 4173 端口，不能与另一轮前端 E2E 同时运行。真实账号测试固定使用 Chromium 和单个 worker，避免触发匿名账号接口每分钟 10 次的限流；跨浏览器界面行为由普通 E2E 覆盖。

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
