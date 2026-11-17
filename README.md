# Stars-Admin

基于 Go、Gin、GORM、PostgreSQL 和 Redis 的模块化单体中后台后端框架。

当前已实现 M1 框架和 M2 认证与权限：JWT 双令牌、Redis 原子会话、Casbin、用户/角色/菜单/部门管理、登录审计及幂等 seed。M3 数据范围过滤验收前，系统管理入口仅有效 admin 可用，普通用户可使用本人认证接口。

本地格式/lint/build、race 单元和集成测试、非 root 生产镜像冒烟均已通过；平台结果见 [GitHub Actions](https://github.com/StarsSkyCat/Stars-Admin/actions/workflows/ci.yaml)。

## 快速开始

宿主机只需 Docker、支持 `!reset` 的 Docker Compose 和 Shell。当前验证环境为 Linux x86_64、Docker 29.7.2、Compose 5.5.0；Go、Make、GCC、Air、swag 和 golangci-lint 都在开发容器内。

在项目根目录执行：

```sh
./scripts/dev.sh dev-env
./scripts/dev.sh migrate-up
# 首次初始化前，通过受保护环境变量注入 STARS_SEED_ADMIN_PASSWORD。
./scripts/dev.sh seed
./scripts/dev.sh run
```

首次运行会构建工具链并下载依赖。开发容器映射宿主机 UID/GID，不挂载 Docker socket。基础服务使用独立的 stars-admin-dev 项目，端口仅绑定 127.0.0.1：PostgreSQL 5432、Redis 6379、HTTP 8080。

seed 首次从环境变量读取至少 12 字符、最多 72 字节的初始密码，创建用户名 admin（用户和角色 ID 均为 1）；不会打印密码。重复执行允许省略初始密码，且不重置已有密码、状态和授权。seed 仅连接数据库，在迁移后、开放流量前显式运行；HTTP 启动不自动初始化账号。

登录前获取 `/api/sys/auth/captcha`，使用返回的 uuid/image 完成人工验证码，再提交 login 的 captchaUuid/captchaCode。首登只允许 profile/password/logout；修改本人密码会撤销所有会话，须重新登录后在 Swagger Authorize 填入 `Bearer <accessToken>`。

- [ping](http://127.0.0.1:8080/api/ping)：统一响应，含 X-Request-Id 响应头。
- [存活探针](http://127.0.0.1:8080/api/health/live)：仅检查进程。
- [就绪探针](http://127.0.0.1:8080/api/health/ready)：检查迁移版本、PostgreSQL 和 Redis；失败返回 503，`/api/health` 为别名。
- [Swagger](http://127.0.0.1:8080/swagger/index.html)：开发环境的 Swagger 2.0 文档。

运行入口使用 Air 热重载；Ctrl+C 停止应用，`./scripts/dev.sh dev-down` 停止基础服务并保留开发数据卷。可用 STARS_HTTP_PORT、STARS_PG_PORT、STARS_REDIS_PORT 更改宿主机映射端口。

## 常用命令

| 命令 | 行为 |
|---|---|
| `./scripts/dev.sh tools` | 构建或更新锁定的开发工具镜像 |
| `./scripts/dev.sh build` | 生成 Swagger 并编译 bin/stars-admin |
| `./scripts/dev.sh swag` | 从注解生成 docs/swagger/ |
| `./scripts/dev.sh fmt` | 主动格式化 Go 源码 |
| `./scripts/dev.sh lint` | 检查格式、goimports、静态分析及导入边界，不改写源码 |
| `./scripts/dev.sh test` | 执行带 race 的单元测试 |
| `./scripts/dev.sh test-integration` | 独立 PG/Redis 中执行集成测试，结束后清理测试资源 |
| `./scripts/dev.sh verify` | lint → build → test → test-integration，与 CI 使用同一入口 |
| `./scripts/dev.sh migrate-up` | 升级内嵌的版本化迁移 |
| `./scripts/dev.sh migrate-down` | 仅回退一版，会删除该版结构和数据 |
| `./scripts/dev.sh migrate-status` | 输出当前版本、期望版本和 dirty 状态 |
| `./scripts/dev.sh seed` | 初始化缺失基础数据；首次密码从环境变量读取 |
| `./scripts/smoke.sh` | 独立生产镜像验证迁移、seed、改密、认证、幂等及停机 |

Makefile 的对应目标也可在已准备 Go/Make 和锁定工具的环境使用。集成测试会回退数据库，因此入口固定使用独立的 stars-admin-test 库和项目，拒绝非测试库；单元测试不要求运行数据库。

## 配置与运行约定

配置按基础 YAML → APP_ENV 选定的环境 YAML → STARS_ 环境变量覆盖。APP_ENV 只接受 dev/test/prod，嵌套键转为下划线，例如 STARS_DATABASE_HOST、STARS_DATABASE_PASSWORD、STARS_REDIS_ADDR。STARS_CONFIG_DIR 可选择配置目录，空环境变量按显式空值处理。

生产使用 configs/config.prod.yaml，必须通过环境变量提供非开发默认值的数据库密码及至少 32 字节的 JWT 密钥。生产默认 JSON 日志、关闭 Swagger、CORS 明确白名单，数据库 TLS 默认为 verify-full。密码哈希成本由 STARS_AUTH_BCRYPT_COST 设置，默认 10、范围 4—14，生产范围 10—14。

所有环境都关闭 AutoMigrate；必须先运行迁移再启动 HTTP，当前最新版本为 2。dirty、未升级或超前版本会拒绝启动，不自动 force 或回退。二进制支持 `stars-admin [serve]`、`stars-admin migrate up|down|status`、`stars-admin seed`。迁移只读取数据库/日志配置；seed 额外读取哈希/节点配置，均不连接 Redis 或启动 HTTP。

普通 JSON 响应为 `{code,msg,data}`；ID 使用十进制字符串，时间统一 UTC/RFC3339，错误不透出内部细节。默认每个 IP/路由每分钟 120 次请求，探针和 Swagger 静态资源豁免；Redis 故障返回 503。请求日志仅记录固定元信息，不采集请求头、查询参数或请求体。

雪花节点默认 0、范围 0—1023，不同应用实例必须由部署分配不同编号；本框架不提供节点自动分配或跨实例租约。

M2 的一致性验收范围为单应用实例、单 Redis，不支持应用多副本策略同步或 Redis Cluster 跨槽 Lua。JWT 仅接受 HS256；Access 默认 30 分钟，会话从首次登录起最长七天，刷新不延长期限。每次认证核对数据库版本和 Redis；密码、状态或角色成员变化提升 auth_version，旧令牌在 Redis 清理失败及应用重启后仍不可用。策略同步失败时鉴权与 readiness 返回 503，后台重试恢复。

认证业务参数存放 sys_config：captchaEnabled=true、multiLogin=true、maxSessions=5、passwordRetryLimit=5、lockMinutes=10，键均以 `sys.account.` 开头。超过会话上限淘汰最早会话；关闭多点登录则新登录替换全部旧会话。缺失参数采用上述默认值，非法已有参数拒绝认证；参数管理接口在 M3 交付。

除全局限流外，登录按 IP 20 次/分钟、账号 10 次/分钟；验证码按 IP 30 次/分钟；刷新按 IP 60 次/分钟、sid 30 次/分钟。只信任已配置的代理来源，验证码错误/过期/使用后失效，密码失败达到阈值锁定十分钟。

## 生产镜像与验证

```sh
docker build -f deployments/Dockerfile -t stars-admin:m2 .
./scripts/smoke.sh
```

生产镜像使用静态二进制、TLS 根证书、配置及内嵌迁移，默认用户为 65532:65532。smoke 使用公开的测试凭据及隔离网络，测试数据库关闭 TLS；此项测试设置不用于真实生产部署。

官方基础镜像标签和摘要记录在 deployments/images.lock，依赖固定在 go.mod/go.sum，工具版本固定在开发 Dockerfile。GitHub Actions 在 push/PR 时调用同样的容器入口，成功后将生成的 Swagger JSON/YAML 上传为 swagger artifact。

## 工程导读与本地文档

业务使用 `internal/modules/{module}/{feature}` 的两级结构；sys 已实现 M2 功能，job/gen 仍为占位模块。框架能力集中在 internal/core，可复用工具位于 pkg，正式结构变更位于 migrations，容器配置位于 deployments。岗位、导入导出、文件、在线用户管理和五档数据过滤留在 M3，任务调度和代码生成留在 M4。

按用户要求，docs/ 和 AGENTS.md 只保留在本地，不纳入 Git。Swagger 生成物同样忽略，所有构建和 CI 必须先运行 make swag，不能依赖已提交的生成文件。

当前工作区的设计和进度文档：

- [协作规则](./AGENTS.md)
- [技术选型](./docs/前期开发与技术选型文档.md)、[数据库设计](./docs/数据库设计文档.md)、[编码规范](./docs/编码规范.md)
- [进度文档](./docs/进度文档.md)、[M1 开发文档](./docs/开发文档/M1-框架骨架.md)
- [M1 验证记录](./docs/M1验证记录.md)、[文档审核记录](./docs/文档审核记录.md)
- [M2 开发文档](./docs/开发文档/M2-认证与权限.md)、[M2 验证记录](./docs/M2验证记录.md)
