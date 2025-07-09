# Stars Admin 后端系统

基于 Go + Gin + GORM 的后台管理系统后端服务

## 项目特性

- 🚀 基于 Gin Web 框架，性能优异
- 🗄️ 使用 GORM 作为 ORM 框架
- 🔐 JWT 身份验证
- 📝 完整的 RBAC 权限管理
- 📊 操作日志记录
- 🐳 Docker 部署支持
- 📖 Swagger API 文档

## 技术栈

- **框架**: Gin
- **ORM**: GORM
- **数据库**: MySQL/PostgreSQL
- **认证**: JWT
- **配置**: Viper
- **日志**: Logrus
- **文档**: Swagger

## 项目结构

```
backend/
├── cmd/                    # 入口文件
│   ├── main.go            # 主服务入口
│   └── migrate/           # 数据库迁移
│       └── main.go        
├── config/                # 配置文件
│   └── config.yaml        
├── internal/              # 内部包
│   ├── api/               # API 相关
│   │   ├── handlers/      # 处理器
│   │   ├── middleware/    # 中间件
│   │   └── routes/        # 路由
│   ├── config/            # 配置管理
│   ├── database/          # 数据库连接
│   ├── models/            # 数据模型
│   ├── services/          # 业务逻辑
│   └── utils/             # 工具函数
├── migrations/            # 数据库迁移脚本
├── pkg/                   # 公共包
├── scripts/               # 脚本文件
├── go.mod                 # Go 模块文件
└── go.sum                 # Go 模块校验文件
```

## 快速开始

### 环境要求

- Go 1.19+
- MySQL 5.7+ / PostgreSQL 9.6+
- Redis 6.0+

### 安装依赖

```bash
go mod tidy
```

### 配置文件

复制配置文件模板：

```bash
cp config/config.yaml.example config/config.yaml
```

修改配置文件中的数据库连接信息：

```yaml
database:
  host: localhost
  port: 3306
  username: root
  password: your_password
  database: stars_admin
  charset: utf8mb4
```

### 数据库迁移

运行数据库迁移脚本：

```bash
go run cmd/migrate/main.go
```

### 启动服务

```bash
go run cmd/main.go
```

服务将在 `http://localhost:8080` 启动

### 默认账户

- 用户名: `admin`
- 密码: `admin123`

## API 文档

启动服务后，可以通过以下地址访问 API 文档：

- Swagger UI: `http://localhost:8080/swagger/index.html`

## 数据库表结构

### 用户相关表

- `xc_users` - 用户表
- `xc_roles` - 角色表
- `xc_user_roles` - 用户角色关联表

### 权限相关表

- `xc_menus` - 菜单表
- `xc_permissions` - 权限表
- `xc_role_menus` - 角色菜单关联表

### 日志表

- `xc_operation_logs` - 操作日志表

## 开发指南

### 添加新的 API 接口

1. 在 `internal/models/` 中定义数据模型
2. 在 `internal/services/` 中实现业务逻辑
3. 在 `internal/api/handlers/` 中创建处理器
4. 在 `internal/api/routes/` 中注册路由

### 中间件

项目内置了以下中间件：

- 认证中间件 (`Auth`)
- 日志中间件 (`Logger`)
- 操作日志中间件 (`OperationLogger`)
- 错误处理中间件 (`ErrorHandler`)
- 跨域中间件 (`CORS`)
- 限流中间件 (`RateLimiter`)

## 部署

### Docker 部署

构建镜像：

```bash
docker build -t stars-admin-backend .
```

运行容器：

```bash
docker run -p 8080:8080 stars-admin-backend
```

### Docker Compose 部署

```bash
docker-compose up -d
```

## 环境变量

| 变量名 | 描述 | 默认值 |
|--------|------|--------|
| `GIN_MODE` | 运行模式 | `debug` |
| `PORT` | 服务端口 | `8080` |
| `DB_HOST` | 数据库主机 | `localhost` |
| `DB_PORT` | 数据库端口 | `3306` |
| `DB_USER` | 数据库用户名 | `root` |
| `DB_PASSWORD` | 数据库密码 | - |
| `DB_NAME` | 数据库名称 | `stars_admin` |
| `JWT_SECRET` | JWT 密钥 | - |
| `REDIS_HOST` | Redis 主机 | `localhost` |
| `REDIS_PORT` | Redis 端口 | `6379` |

## 贡献指南

1. Fork 本仓库
2. 创建功能分支 (`git checkout -b feature/AmazingFeature`)
3. 提交更改 (`git commit -m 'Add some AmazingFeature'`)
4. 推送到分支 (`git push origin feature/AmazingFeature`)
5. 创建 Pull Request

## 许可证

本项目使用 MIT 许可证。详情请查看 [LICENSE](LICENSE) 文件。

## 联系方式

如有问题或建议，请提交 Issue 或联系维护者。