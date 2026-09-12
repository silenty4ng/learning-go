# iGoBlog — Go 博客系统

一个用 Go 语言实现的博客系统：RESTful API + 原生前端界面（零构建），同时作为 Go 语言设计思想与工程实践的学习范本。

## 前端界面

`web/` 目录下的原生前端站点（HTML + CSS + ES Modules，无任何构建工具与运行时依赖），由 Gin 同端口静态托管，前后端同源。

```powershell
# 在项目根目录运行（前端与 API 同端口）
go run ./cmd/server
# 浏览器访问
#   http://localhost:8080/        前台（文章列表 / 详情 / 评论 / 登录注册）
#   http://localhost:8080/admin   管理后台（文章 / 分类 / 标签 / 评论管理）
```

### 功能导览

| 页面 | 说明 |
|------|------|
| 首页 `/#/` | 文章卡片网格、分页、分类/标签筛选、防抖关键词搜索（状态同步到 URL 可分享回退） |
| 文章详情 `/#/article/:id` | 正文排版、标签、浏览量、评论分页与发表（登录后），评论者/作者可删评 |
| 登录/注册 | 弹层 Tab 切换，注册成功自动登录，Token 存 localStorage |
| 后台-文章管理 `/admin#/posts` | 搜索、新建、编辑、删除（确认弹窗） |
| 后台-编辑器 `#/posts/new` | 标题、分类下拉、标签胶囊多选、正文编辑，编辑模式自动回填 |
| 后台-分类/标签 | 行内新增、重命名、删除（分类下有文章时 409 冲突 Toast 提示） |
| 后台-评论管理 | 按文章选择查看全部评论并删除 |

### 前端实现要点

- **hash 路由**（`assets/js/router.js`）：`:param` 路径参数 + query 解析，视图返回清理函数
- **统一 API 层**（`assets/js/api.js`）：自动携带 Bearer Token；解析统一响应；401 清除登录态并广播 `auth:expired`；429 转义为限流提示
- **XSS 防护**：所有动态内容经 `escapeHtml` / `textContent` 渲染
- **设计系统**（`assets/css/base.css`）：CSS 变量设计令牌（靛蓝主色 `#4F46E5`、中性灰阶）、按钮/表单/卡片/弹窗/Toast/分页/骨架屏等通用组件
- **响应式**：桌面双栏卡片网格，≤768px 单栏 + 汉堡导航；后台侧边栏移动端抽屉式
- **微动效**：卡片 hover 上浮、骨架屏 shimmer、Toast/弹窗过渡，全程 150~200ms ease

## 技术栈

| 组件 | 选型 | 说明 |
|------|------|------|
| 语言 | Go 1.22+ | |
| Web 框架 | [Gin](https://github.com/gin-gonic/gin) | 路由、参数绑定校验、中间件、静态托管 |
| 数据库 | SQLite（[modernc.org/sqlite](https://modernc.org/sqlite)） | 纯 Go 驱动，零部署、免 CGO |
| 认证 | JWT（golang-jwt/v5）+ bcrypt | 无状态认证，密码哈希存储 |
| 限流 | golang.org/x/time/rate | 令牌桶，按客户端 IP 维度 |
| 前端 | 原生 HTML + CSS + ES Modules | 零构建零依赖，Gin 托管于 `web/` |
| 测试 | 标准库 testing + httptest | |

## 目录结构（Go 标准布局）

```
iGoBlog/
├── cmd/server/main.go           # 入口：手工依赖注入 + 优雅关停
├── web/                         # 原生前端站点（Gin 静态托管）
│   ├── index.html               # 前台入口（hash 路由：列表/详情 + 登录弹层）
│   ├── admin.html               # 后台入口（侧边栏布局 + 登录门禁）
│   └── assets/
│       ├── css/                 # base.css 设计系统 + front/admin 页面样式
│       └── js/
│           ├── api.js auth.js router.js ui.js format.js   # 公共模块
│           ├── front/           # 前台页面逻辑（main/home/article）
│           └── admin/           # 后台页面逻辑（app/posts/editor/taxonomies/comments）
├── internal/
│   ├── apperr/                  # 统一业务错误类型（携带 HTTP 状态码与错误码）
│   ├── config/                  # 配置加载（环境变量优先）
│   ├── database/                # SQLite 连接、PRAGMA、建表迁移
│   ├── model/                   # 领域模型（各层共享的数据结构）
│   ├── pkg/jwtutil/             # JWT 签发/校验工具
│   ├── repository/              # 数据访问层：接口定义 + SQLite 实现
│   ├── service/                 # 业务逻辑层：接口定义 + 实现（含并发组件）
│   ├── handler/                 # HTTP 层：参数绑定、路由装配
│   └── middleware/              # JWT 认证、令牌桶限流、统一响应
└── Makefile
```

## 架构与 Go 设计思想

### 1. 分层架构与依赖倒置

```
handler（HTTP 参数/响应） → service（业务逻辑） → repository（数据访问） → SQLite
```

- service 与 handler **只依赖接口**（`repository.UserRepository`、`service.ArticleService` 等），
  具体实现通过 `main.go` 手工注入 —— 无 DI 框架，贴合 Go 惯例。
- 单元测试用 mock 实现替换 repository，业务逻辑可脱离数据库验证。

### 2. 错误处理

- `internal/apperr.AppError`：实现 `error` 接口 + `Unwrap()`，支持 `errors.As/Is`；
  携带 HTTP 状态码与业务错误码。
- repository 层把 `sql.ErrNoRows` 转换为 `apperr.NotFound`，把唯一约束冲突转换为
  `apperr.Conflict`；service 层做业务校验与权限判断；handler 层经 `middleware.Fail`
  统一转换为 JSON 错误响应。
- 统一响应格式：

```json
// 成功
{"message": "ok", "data": {...}}
// 失败
{"code": "NOT_FOUND", "message": "文章不存在"}
```

### 3. 并发处理

- **浏览量异步批量写入**（`internal/service/view_counter.go`）：
  - `ViewCounter` 持有带缓冲 channel，`Add` 用 `select + default` 非阻塞投递，
    缓冲满时有损丢弃，绝不拖慢读请求；
  - 后台 goroutine 以「满批 or 定时」双触发条件批量落库（一个事务合并 N 次写）；
  - `Close()` 幂等，关停时排空缓冲、`sync.WaitGroup` 等待退出，数据不丢。
- **令牌桶限流**（`internal/middleware/ratelimit.go`）：`sync.Mutex` 保护的
  按 IP 限流器表，超限返回 429 + `Retry-After`。
- **优雅关停**（`cmd/server/main.go`）：`signal.NotifyContext` 监听 SIGINT/SIGTERM →
  `http.Server.Shutdown` 停止接收请求 → `ViewCounter.Close` 冲刷统计 → 关闭连接池。
- **超时控制**：数据库操作全程传递 `context.Context`，冲刷任务自带 10s 超时。

### 4. 可测试性

| 测试文件 | 层次 | 手段 |
|----------|------|------|
| `internal/service/user_service_test.go` | service | mock repository，验证注册/登录/错误语义 |
| `internal/service/article_service_test.go` | service | mock 三类 repository，验证权限校验与浏览量投递 |
| `internal/service/view_counter_test.go` | 并发组件 | 验证满批/关停冲刷/Close 幂等 |
| `internal/repository/article_repo_test.go` | repository | 临时 SQLite 文件，验证 SQL、筛选、分页、批量自增 |
| `internal/handler/smoke_test.go` | 端到端 | httptest 全链路：注册→登录→发文→评论→删除 |

## 快速开始

```powershell
# 运行（默认端口 8080，数据库文件 blog.db）
go run ./cmd/server

# 后端热重载开发（需先安装：go install github.com/air-verse/air@latest）
# 修改 internal/ 或 cmd/ 下的 Go 代码自动重编译重启；
# web/ 下的前端是静态文件，改完刷新浏览器即生效
air

# 测试（含竞态检测）
go test ./... -count=1
go test ./... -race -count=1

# 静态检查与构建
go vet ./...
go build ./...
```

### 配置（环境变量）

| 变量 | 默认值 | 说明 |
|------|--------|------|
| `BLOG_PORT` | `8080` | HTTP 监听端口 |
| `BLOG_DB_PATH` | `blog.db` | SQLite 文件路径 |
| `BLOG_JWT_SECRET` | `dev-secret-change-me` | JWT 签名密钥（生产必改） |
| `BLOG_JWT_EXPIRE` | `24h` | Token 有效期 |
| `BLOG_RATE_LIMIT` | `50` | 每秒平均请求数 |
| `BLOG_RATE_BURST` | `100` | 令牌桶突发容量 |
| `BLOG_VIEW_FLUSH_INTERVAL` | `5s` | 浏览量批量落库间隔 |
| `BLOG_VIEW_BATCH_SIZE` | `100` | 浏览量满批阈值 |
| `BLOG_ADMIN_USERNAME` | `admin` | 首次建库时的管理员用户名 |
| `BLOG_ADMIN_PASSWORD` | 随机生成 | 首次建库时的管理员密码（随机时打印到控制台） |

## API 文档

所有接口前缀为 `/api/v1`。认证接口使用 `Authorization: Bearer <token>` 请求头。

### 认证

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/api/v1/auth/register` | 注册 `{username, password}` |
| POST | `/api/v1/auth/login` | 登录，返回 `{token}` |
| GET | `/api/v1/users/me` | 当前用户信息（需认证） |

### 文章

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/v1/articles?page=1&page_size=10&category_id=&tag_id=&keyword=` | 分页列表，支持按分类/标签筛选与关键词搜索 |
| GET | `/api/v1/articles/:id` | 详情（异步累加浏览量） |
| POST | `/api/v1/articles` | 创建 `{title, content, category_id, tag_ids}`（需认证） |
| PUT | `/api/v1/articles/:id` | 更新（仅作者） |
| DELETE | `/api/v1/articles/:id` | 删除（仅作者） |

### 分类

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/v1/categories` | 列表 |
| GET | `/api/v1/categories/:id` | 详情 |
| POST | `/api/v1/categories` | 创建 `{name}`（需认证） |
| PUT | `/api/v1/categories/:id` | 更新 |
| DELETE | `/api/v1/categories/:id` | 删除（分类下有文章时返回 409） |

### 标签

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/v1/tags` | 列表 |
| GET | `/api/v1/tags/:id` | 详情 |
| POST | `/api/v1/tags` | 创建 `{name}`（需认证） |
| DELETE | `/api/v1/tags/:id` | 删除 |

### 评论

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/v1/articles/:id/comments?page=&page_size=` | 文章评论分页列表 |
| POST | `/api/v1/articles/:id/comments` | 发表评论 `{content}`（需认证） |
| DELETE | `/api/v1/comments/:id` | 删除（评论者本人或文章作者） |

### 其他

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/healthz` | 健康检查 |
| GET | `/api/v1/site/settings` | 站点设置（公开只读：站点名称/描述/注册开关） |

### 管理员（超管）

首次启动创建数据库时，系统自动生成管理员账号并把**用户名与密码打印到控制台**（密码随机生成，可用环境变量 `BLOG_ADMIN_USERNAME` / `BLOG_ADMIN_PASSWORD` 覆盖；密码仅首次显示，请妥善保管）。

超管能力：
- **管理所有文章与评论**：不受"仅作者本人"限制，可编辑/删除任何人的内容
- **用户管理**：分页查看用户、设置/撤销管理员角色、删除用户（级联删除其全部文章与评论）
- **站点设置**：站点名称与描述、**开关用户注册**（关闭后注册接口返回 403，前台自动隐藏注册入口）

安全约束：不能删除自己的账号、不能自我降级、站点至少保留一名管理员。

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/v1/admin/users?page=&page_size=` | 用户列表 |
| PUT | `/api/v1/admin/users/:id/role` | 修改角色 `{role: "admin"\|"user"}` |
| DELETE | `/api/v1/admin/users/:id` | 删除用户及其全部内容 |
| GET | `/api/v1/admin/settings` | 读取站点设置 |
| PUT | `/api/v1/admin/settings` | 更新站点设置 |

### 角色与权限模型

- `users.role`：`admin`（管理员）/ `user`（普通用户）
- JWT 携带角色信息；认证中间件解析 Token 后**回查数据库**注入最新角色——被删除用户的 Token 立即失效，角色变更即时生效
- 管理端点经 `auth + RequireAdmin` 双重中间件保护

### 错误码

| HTTP | code | 场景 |
|------|------|------|
| 400 | `VALIDATION_ERROR` | 参数校验失败 |
| 401 | `UNAUTHORIZED` | 未认证 / Token 失效 |
| 403 | `FORBIDDEN` | 无权限操作 |
| 404 | `NOT_FOUND` | 资源不存在 |
| 409 | `CONFLICT` | 资源冲突（用户名/分类/标签重复等） |
| 429 | `RATE_LIMITED` | 触发限流 |
| 500 | `INTERNAL_ERROR` | 服务器内部错误 |

## 示例（PowerShell）

```powershell
# 注册
Invoke-RestMethod -Method Post -Uri http://localhost:8080/api/v1/auth/register `
  -ContentType "application/json" -Body '{"username":"alice","password":"password123"}'

# 登录
$resp = Invoke-RestMethod -Method Post -Uri http://localhost:8080/api/v1/auth/login `
  -ContentType "application/json" -Body '{"username":"alice","password":"password123"}'
$token = $resp.data.token

# 创建文章
Invoke-RestMethod -Method Post -Uri http://localhost:8080/api/v1/articles `
  -ContentType "application/json" -Headers @{Authorization = "Bearer $token"} `
  -Body '{"title":"Hello Go","content":"concurrency & interfaces","category_id":1,"tag_ids":[1]}'
```
