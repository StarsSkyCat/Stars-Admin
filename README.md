# Stars Admin 管理系统

一个基于 React + TypeScript + Go 的现代化后台管理系统，采用前后端分离架构。

## 项目概述

Stars Admin 是一个功能完整的后台管理系统，包含用户管理、角色权限、菜单管理、操作日志等核心功能。

### 技术栈

**前端**
- React 18 + TypeScript
- Vite 构建工具
- Ant Design UI 框架
- Redux Toolkit 状态管理
- React Router 路由管理
- Axios HTTP 客户端

**后端**
- Go + Gin Web 框架
- GORM ORM 框架
- JWT 身份验证
- MySQL/PostgreSQL 数据库
- Redis 缓存
- Swagger API 文档

## 项目结构

```
Stars Admin/
├── frontend/              # 前端项目
│   ├── src/
│   │   ├── components/    # 公共组件
│   │   ├── pages/         # 页面组件
│   │   ├── hooks/         # 自定义钩子
│   │   ├── services/      # API 服务
│   │   ├── store/         # 状态管理
│   │   ├── types/         # 类型定义
│   │   └── utils/         # 工具函数
│   ├── public/            # 静态资源
│   ├── package.json
│   └── vite.config.ts
├── backend/               # 后端项目
│   ├── cmd/               # 入口文件
│   ├── internal/          # 内部包
│   │   ├── api/           # API 处理
│   │   ├── models/        # 数据模型
│   │   ├── services/      # 业务逻辑
│   │   └── utils/         # 工具函数
│   ├── config/            # 配置文件
│   ├── migrations/        # 数据库迁移
│   ├── go.mod
│   └── go.sum
├── docs/                  # 项目文档
├── docker-compose.yml     # Docker 编排文件
├── .gitignore            # Git 忽略文件
└── README.md             # 项目说明
```

## 功能特性

### 🔐 权限管理
- 用户管理：增删改查用户信息
- 角色管理：角色分配和权限控制
- 菜单管理：动态菜单配置
- 权限控制：基于 RBAC 的权限系统

### 📊 系统监控
- 操作日志：详细记录用户操作
- 系统监控：性能指标监控
- 错误追踪：异常信息记录

### 🛡️ 安全防护
- JWT 身份验证
- 接口权限验证
- 数据加密存储
- 防 XSS 攻击

### 🚀 开发体验
- 热重载开发
- TypeScript 类型安全
- 代码规范检查
- 自动化测试

## 快速开始

### 环境要求

- Node.js 16.0+
- Go 1.19+
- MySQL 5.7+ / PostgreSQL 9.6+
- Redis 6.0+

### 安装依赖

**前端**
```bash
cd frontend
npm install
```

**后端**
```bash
cd backend
go mod tidy
```

### 配置文件

1. 复制后端配置文件：
```bash
cp backend/config/config.yaml.example backend/config/config.yaml
```

2. 修改数据库连接信息：
```yaml
database:
  host: localhost
  port: 3306
  username: root
  password: your_password
  database: stars_admin
```

### 数据库初始化

```bash
cd backend
go run cmd/migrate/main.go
```

### 启动服务

**启动后端服务**
```bash
cd backend
go run cmd/main.go
```

**启动前端服务**
```bash
cd frontend
npm run dev
```

### 访问系统

- 前端地址：http://localhost:3000
- 后端地址：http://localhost:8080
- API 文档：http://localhost:8080/swagger/index.html

### 默认账户

- 用户名：`admin`
- 密码：`admin123`

## 项目启动

### 使用根目录脚本启动

1. 安装所有依赖：
```bash
npm run install:all
```

2. 启动开发服务（前后端同时启动）：
```bash
npm run dev
```

3. 分别启动前后端：
```bash
# 启动前端
npm run dev:frontend

# 启动后端
npm run dev:backend
```

### 构建生产版本

```bash
npm run build
```

## 开发指南

### 前端开发

1. 组件开发规范
2. 状态管理最佳实践
3. API 接口调用
4. 路由配置
5. 样式管理

### 后端开发

1. API 接口开发
2. 数据模型定义
3. 业务逻辑实现
4. 中间件使用
5. 错误处理

### 代码规范

- 使用 ESLint 和 Prettier 格式化前端代码
- 使用 gofmt 格式化后端代码
- 遵循 Git 提交规范
- 编写单元测试

## 部署说明

### 生产环境部署

1. 环境准备：确保安装了 Node.js 和 Go 环境
2. 配置文件设置：根据 `.env.example` 和 `config.yaml.example` 配置文件
3. 数据库迁移：运行 `npm run migrate` 执行数据库迁移
4. 构建项目：运行 `npm run build` 构建生产版本
5. 启动服务：部署构建后的文件到服务器

### 监控和维护

1. 日志管理
2. 性能监控
3. 备份策略
4. 安全更新

## 贡献指南

### 提交规范

```
feat: 新功能
fix: 修复 bug
docs: 文档更新
style: 代码格式化
refactor: 代码重构
test: 测试相关
chore: 构建过程或辅助工具的变动
```

### 开发流程

1. Fork 项目
2. 创建功能分支
3. 提交代码
4. 创建 Pull Request
5. 代码审查
6. 合并分支

## 常见问题

### 前端相关

Q: 如何添加新的页面？
A: 在 `src/pages/` 目录下创建新的组件，并在路由中配置。

Q: 如何调用后端 API？
A: 使用 `src/services/` 目录下的 API 服务函数。

### 后端相关

Q: 如何添加新的 API 接口？
A: 在 `internal/api/handlers/` 中创建处理器，在 `internal/api/routes/` 中注册路由。

Q: 如何进行数据库迁移？
A: 运行 `go run cmd/migrate/main.go` 执行数据库迁移。

## 更新日志

### v1.0.0 (2024-01-01)
- 初始版本发布
- 完整的用户权限管理系统
- 前后端分离架构
- Docker 部署支持

## 许可证

本项目采用 MIT 许可证。详情请查看 [LICENSE](LICENSE) 文件。

## 联系我们

- 项目地址：https://github.com/your-username/stars-admin
- 问题反馈：https://github.com/your-username/stars-admin/issues
- 邮箱：admin@example.com

## 致谢

感谢所有为这个项目做出贡献的开发者。

---

⭐ 如果这个项目对你有帮助，请给我们一个 Star！