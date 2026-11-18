# Stars-Admin

Stars-Admin 是基于 Go、Gin、GORM、PostgreSQL 和 Redis 的模块化中后台后端框架，提供认证授权、系统管理、定时任务和代码生成能力，可作为业务后台的开发基础。

Stars-Admin 在同一个仓库中维护 Go 后端和 `web/` 下的 Vben Admin 前端。后端支持 PostgreSQL，应用、任务调度和授权同步按单实例设计。Ant Design Vue 应用已接入登录、改密、个人中心、用户与部门管理，可在 `web/` 运行 `pnpm dev:antd`，默认访问 `http://localhost:5666`。本地 `docs/` 提供开发与接入说明。

## 功能

| 模块 | 能力 |
|---|---|
| 认证与权限 | 验证码、登录锁定、JWT 双令牌、原子刷新、首登改密、在线会话、持久撤销、Casbin RBAC |
| 系统管理 | 用户、角色、菜单、部门、岗位、五档数据范围、字典、参数配置 |
| 文件与审计 | 文件上传/下载/删除及所有者校验、操作日志、登录日志、在线用户与强制下线 |
| 数据处理 | 用户 Excel 导入导出、范围过滤、批量操作事务校验 |
| 定时任务 | UTC 六位 Cron、白名单函数、启停、手动执行、并发门禁、执行日志 |
| 代码生成 | PG 业务表元数据、字段配置/同步、CRUD 模板、预览、ZIP 下载及受控本地输出 |
| 工程质量 | 统一响应、请求日志、健康检查、Swagger、版本化迁移、race 测试、覆盖率与漏洞扫描 |

## 环境要求

两种运行方式使用相同的配置与迁移：

- **Docker 开发**：Docker Engine、Docker Compose（支持 `!reset`）和 Shell；Go、Make、GCC 及开发工具在容器内，无需宿主安装。
- **本机开发**：Go **1.27.4**、GNU Make、GCC 或兼容 C 编译器（race 测试需要）、PostgreSQL **17**、Redis **7**。以下命令使用 Bash，Windows 可在 WSL 中执行。

容器和工具的精确版本见 [Dockerfile.dev](./deployments/Dockerfile.dev) 与 [images.lock](./deployments/images.lock)，不使用 `latest` 镜像。

```sh
git clone https://github.com/StarsSkyCat/Stars-Admin.git
cd Stars-Admin
git switch dev
```

## Docker 启动

首次运行会构建工具镜像并下载依赖。在项目根目录依次执行：

```sh
# 启动开发 PostgreSQL 和 Redis，仅绑定宿主 127.0.0.1。
./scripts/dev.sh dev-env

# 启动开发 API，自动执行 SQL 迁移，使用 Air 热重载。
./scripts/dev.sh run
```

本地开发首次登录使用 `admin` / `123456`，按页面提示填写验证码并修改初始密码。也可在首次初始化前通过 `STARS_SEED_ADMIN_PASSWORD` 提供自定义初始密码，至少 12 个字符、最多 72 字节。重复初始化保留已有密码、状态和授权。

只准备开发库时，可执行 `./scripts/dev.sh migrate-up`。初始管理员、菜单与基础配置由 [数据迁移](./migrations/000006_data.up.sql) 写入。

开发端口默认 HTTP `8080`、PostgreSQL `5432`、Redis `6379`；可通过 `STARS_HTTP_PORT`、`STARS_PG_PORT`、`STARS_REDIS_PORT` 调整宿主映射。`./scripts/dev.sh dev-down` 停止开发基础服务并保留数据卷。

### 浏览器提示“未发送任何数据”时

先看启动终端，确认出现 `HTTP 服务已启动`。Air 的 `running...` 只表示尝试启动子进程；如果随后出现 `Process Exit with Code: 1`，后端已经退出，容器仍可能占着8080，浏览器因此收到空响应。

若启动失败，先用 Ctrl+C 停止这次启动并执行 `./scripts/dev.sh migrate-status` 核对版本。开发库为空或版本落后时，可单独执行初始化再启动：

```sh
./scripts/dev.sh migrate-up
./scripts/dev.sh run
```

开发环境的 `run` 在监听 HTTP 前完成迁移。`dev-env` 启动 PostgreSQL 和 Redis。前台运行时请保持终端打开。服务启动后可用 `curl -fsS http://127.0.0.1:8080/api/health/ready` 验证就绪，再打开 Swagger。若接口正常而 Swagger 返回 404，检查当前环境的 Swagger 配置。

## 不使用 Docker 启动

先在本机安装并启动 PostgreSQL、Redis，创建开发账号与数据库；例如以 PG 管理员身份执行以下 SQL，密码自行设置：

```sql
CREATE ROLE stars LOGIN PASSWORD '替换为本机开发密码';
CREATE DATABASE stars_admin OWNER stars;
```

安装与容器一致的开发工具，并将 Go 的 bin 目录加入 PATH：

```sh
export GOTOOLCHAIN=local
export PATH="$(go env GOPATH)/bin:$PATH"
go mod download
go install github.com/swaggo/swag/cmd/swag@v1.16.6
go install github.com/air-verse/air@v1.67.4
go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.15.0
go install golang.org/x/tools/cmd/goimports@v0.52.0
go install golang.org/x/vuln/cmd/govulncheck@v1.9.0
```

在项目根目录配置连接并启动：

```sh
export APP_ENV=dev
export STARS_APP_ADDR=127.0.0.1:8080
export STARS_DATABASE_HOST=127.0.0.1
export STARS_DATABASE_PORT=5432
export STARS_DATABASE_USERNAME=stars
export STARS_DATABASE_DBNAME=stars_admin
read -r -s -p '本机数据库密码：' STARS_DATABASE_PASSWORD; printf '\n'
export STARS_DATABASE_PASSWORD
export STARS_REDIS_ADDR=127.0.0.1:6379
# Redis 启用密码时，额外通过 STARS_REDIS_PASSWORD 提供。

# 启动时执行版本化迁移并初始化基础数据。
make run
```

以上启动命令不调用 Docker。也可通过 `make build` 生成二进制，再执行 `./bin/stars-admin serve`；直接使用 Go 入口时先执行 `make swag`，再运行 `go run ./cmd/server serve`。Swagger 生成包不入库，不能跳过生成步骤直接编译新克隆的源码。

## 登录与接口

启动成功后访问：

| 地址 | 用途 |
|---|---|
| `http://127.0.0.1:8080/api/ping` | 基础连通性 |
| `http://127.0.0.1:8080/api/health/live` | 进程存活 |
| `http://127.0.0.1:8080/api/health/ready` | 数据库/迁移、Redis、授权与调度就绪，失败返回 503 |
| `http://127.0.0.1:8080/swagger/index.html` | 开发环境 API 文档和调试入口 |

1. 调用 `GET /api/sys/auth/captcha` 获取图片验证码。
2. 向 `POST /api/sys/auth/login` 提交 `username`、`password`、`captchaUuid`、`captchaCode`。
3. 首次登录后调用 `PUT /api/sys/auth/password` 修改密码，再重新登录。首登令牌仅允许本人 profile/password/logout；改密会撤销全部会话。
4. 后续请求携带 `Authorization: Bearer <accessToken>`；Swagger 中填入完整的 `Bearer <accessToken>`。Refresh 令牌只用于刷新接口。

API 使用 `/api` 前缀，JSON 字段为小驼峰，普通响应格式为 `{code,msg,data}`。ID 与 ID 列表使用十进制字符串，时间采用 UTC/RFC3339；文件、Excel、ZIP 成功时返回二进制流。分页默认 `page=1&pageSize=20`，上限 100；排序及查询运算符由服务端白名单控制。

用户/部门管理对持相应权限的普通角色开放，并应用数据范围；全局角色、菜单、岗位、字典、参数、审计、job/gen 管理仅有效 admin 可用。按类型查询启用字典项允许登录用户访问。

## 测试与检查

### Docker 测试

```sh
./scripts/dev.sh test              # race 单元测试，无需启动 PG/Redis
./scripts/dev.sh test-integration  # 自动创建并清理隔离 PG/Redis 测试项目
./scripts/dev.sh verify-m5         # lint/build、覆盖率、迁移、Swagger、漏洞检查
./scripts/smoke.sh                 # 独立生产镜像冒烟与备份恢复演练
```

### 不使用 Docker 测试

安装上述 Go 工具后，一条命令执行格式检查、lint、构建和 race 单元测试：

```sh
make verify-local
```

该命令不调用 Docker，也不要求数据库或 Redis。单独执行时可用 `make lint`、`make test`、`make build`；`make fmt` 会主动格式化源码。

本机集成测试需额外准备**专用测试数据库和 Redis**。以 PG 管理员身份创建测试账号与数据库，不能使用开发或生产库：

```sql
CREATE ROLE stars_test LOGIN PASSWORD '替换为本机测试密码';
CREATE DATABASE stars_admin_test OWNER stars_test;
```

在单独终端中运行，避免覆盖开发终端的环境变量：

```sh
export APP_ENV=test
export STARS_INTEGRATION=1
export STARS_DATABASE_HOST=127.0.0.1
export STARS_DATABASE_PORT=5432
export STARS_DATABASE_USERNAME=stars_test
export STARS_DATABASE_DBNAME=stars_admin_test
read -r -s -p '测试数据库密码：' STARS_DATABASE_PASSWORD; printf '\n'
export STARS_DATABASE_PASSWORD
export STARS_REDIS_ADDR=127.0.0.1:6379
# 这里必须指向专用测试 Redis；启用密码时另设 STARS_REDIS_PASSWORD。

make test-integration
# 收集与合并 service 语句覆盖率，门槛为整体 >=60%。
make test-cover
make integration-cover
make coverage
# 其他独立检查。
make test-migrate
make check-swag
make vuln
```

集成测试会执行迁移回退和数据清理，只允许 `APP_ENV=test`、库名 `stars_admin_test` 的隔离环境。生成器验收会在临时模块实际执行 build/lint/race/swag 与 PG/Casbin 行为测试。

CI 定义 lint/test/migrate/swag/vuln 五个 job，并保存覆盖率与 Swagger artifact；工作区及可见 Git 历史的敏感信息规则扫描入口为 `./scripts/secret-scan.sh`。

## 配置

- [config.yaml](./configs/config.yaml)：基础配置及逐项中文说明，默认用于本机开发。
- [config.prod.yaml](./configs/config.prod.yaml)：生产覆盖项，不包含真实密钥。
- [config.test.yaml](./configs/config.test.yaml)：测试覆盖项。
- [casbin_model.conf](./configs/casbin_model.conf)：RBAC 模型；用户和角色使用独立命名空间。

加载顺序为基础 YAML → `APP_ENV` 对应环境 YAML → `STARS_` 环境变量。`APP_ENV` 仅允许 dev/test/prod；例如 `database.host` 对应 `STARS_DATABASE_HOST`，列表变量使用逗号分隔。`STARS_CONFIG_DIR` 可更改配置目录，空环境变量也会覆盖默认值。命令不会自动读取 `.env`。

所有环境都关闭 GORM AutoMigrate，服务启动时检查正式迁移版本。空库依次应用结构基线和数据迁移，已在第 5 版的库只执行数据迁移。第 1—4 版的库先用包含旧迁移的版本升级到第 5 版，例如提交 `b4f8e3e`。[000005_schema](./migrations/000005_schema.up.sql) · [000006_data](./migrations/000006_data.up.sql)

开发环境的 `serve` 自动执行 `migrate up`，生产环境由部署流程显式执行。迁移无需 Redis。`migrate down` 每次回退一版。从第 6 版回退时保留现有数据，从第 5 版回退会删除业务表及数据。

## 生产部署

构建生产镜像：

```sh
docker build -f deployments/Dockerfile -t stars-admin:local .
```

[compose.prod.yaml](./deployments/compose.prod.yaml) 提供连接外部 PostgreSQL/Redis 的单实例示例。使用前需提供以下环境变量：

| 变量 | 内容 |
|---|---|
| `STARS_IMAGE` | 经过验收的镜像标签或摘要 |
| `STARS_DATABASE_HOST/PORT/USERNAME/DBNAME/PASSWORD` | PostgreSQL 连接，PORT 可省略（默认5432） |
| `STARS_PG_CA_FILE` | PG 根证书绝对路径；使用 verify-full 且证书主机名必须匹配 |
| `STARS_REDIS_ADDR/PASSWORD` | 专用受保护网络中的 Redis 及密码；当前客户端无 TLS 配置 |
| `STARS_JWT_SECRET` | 至少 32 字节的环境注入密钥 |
| `STARS_ID_NODE` | 0—1023 的雪花节点编号 |
| `STARS_CORS_ORIGINS` | 明确的来源白名单，禁止 `*` |
| `STARS_UPLOADS_DIR`、`STARS_LOGS_DIR` | 上传/日志绝对目录，UID/GID 65532 可写 |

在目标环境准备好目录、证书及受保护变量后：

```sh
docker compose -p stars-admin-prod -f deployments/compose.prod.yaml config --quiet
# 先停止旧实例、备份数据库和上传目录，并在隔离环境确认备份可恢复。
# 首次创建管理员前，在当前环境注入并导出 STARS_SEED_ADMIN_PASSWORD。
docker compose -p stars-admin-prod -f deployments/compose.prod.yaml run --rm --no-deps -e STARS_SEED_ADMIN_PASSWORD api migrate up
unset STARS_SEED_ADMIN_PASSWORD
docker compose -p stars-admin-prod -f deployments/compose.prod.yaml up -d --wait api
```

生产镜像为非 root scratch，提供内置 readiness probe，关闭 Swagger 和本地代码生成；上传/日志持久化，临时文件写入 tmpfs。反向代理负责 HTTPS，默认 HTTP 端口只绑定 127.0.0.1。升级须区分应用回滚与结构回退，`down` 无法恢复已删除的数据。

## 项目结构

```text
cmd/                 应用、探针、冒烟和覆盖率命令
configs/             YAML 配置与 Casbin 模型
internal/app/        路由、依赖装配与生命周期
internal/core/       配置、数据库、认证基础能力、中间件、调度等
internal/modules/    sys / job / gen 业务模块
pkg/                 通用工具
migrations/          PostgreSQL 版本化 SQL 迁移
scripts/             开发、测试与验证入口
deployments/         Dockerfile 与 Compose
```

业务采用 `internal/modules/{module}/{feature}` 结构，按 handler/service/repository 分层。数据库、Redis、策略同步、调度与日志 worker 按生命周期统一装配。

## 使用边界

- 单应用/调度实例、单 Redis；不支持多副本策略同步、跨实例任务防重或 Redis Cluster 跨槽 Lua。
- 任务仅调用显式注册的函数，必须配合 context 取消；日志为有界队列的尽力审计。
- 普通字典/参数缓存有 TTL 陈旧窗口，安全认证参数不依赖普通缓存放行。
- 生成器只支持已登记 Base、类型及单 bigint 雪花主键的业务表；凭证字段名称拦截不能替代人工审查。默认 ZIP，local 默认关闭且生产禁用，产物需审核后手工接入。
- 文件按所有者校验，同 hash 只在同一所有者内去重；admin 的管理权限仍需经过认证与首登改密。

版本功能及已知限制见 [CHANGELOG](./CHANGELOG.md)，许可见 [LICENSE](./LICENSE)。
