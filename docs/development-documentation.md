# 开发文档

## 项目概述

本项目是一个基于现代技术栈的后台快速开发框架，包含完整的前后端分离架构。

## 技术栈

### 后端技术栈
- **Go 1.21+**: 主要后端开发语言
- **MySQL 5.7**: 主数据库
- **Redis**: 缓存和会话存储
- **Gin**: HTTP Web框架
- **GORM**: ORM框架
- **JWT**: 认证授权
- **Viper**: 配置管理
- **Logrus**: 日志管理
- **Swagger**: API文档生成

### 前端技术栈
- **React 18**: UI框架
- **Ant Design**: UI组件库
- **TypeScript**: 类型安全
- **React Router**: 路由管理
- **Axios**: HTTP客户端
- **Redux Toolkit**: 状态管理
- **Vite**: 构建工具

## 架构设计

### 后端架构

```
backend/
├── cmd/                    # 应用入口
│   └── main.go
├── internal/               # 内部代码
│   ├── api/               # API路由和处理器
│   │   ├── handlers/      # 请求处理器
│   │   ├── middleware/    # 中间件
│   │   └── routes/        # 路由定义
│   ├── config/            # 配置管理
│   ├── database/          # 数据库连接和配置
│   ├── models/            # 数据模型
│   ├── services/          # 业务逻辑层
│   └── utils/             # 工具函数
├── pkg/                   # 公共包
├── scripts/               # 脚本文件
├── migrations/            # 数据库迁移
├── go.mod
└── go.sum
```

### 前端架构

```
frontend/
├── src/
│   ├── components/        # 公共组件
│   ├── pages/            # 页面组件
│   ├── services/         # API服务
│   ├── utils/            # 工具函数
│   ├── hooks/            # 自定义Hook
│   ├── assets/           # 静态资源
│   ├── styles/           # 样式文件
│   ├── types/            # TypeScript类型定义
│   └── store/            # Redux状态管理
├── public/               # 静态文件
├── package.json
└── vite.config.ts
```

## 核心模块

### 1. 用户认证模块
- JWT Token认证
- 角色权限管理
- 登录/注册功能
- 密码加密存储

### 2. 权限管理模块
- RBAC权限模型
- 菜单权限控制
- 接口权限验证
- 数据权限过滤

### 3. 系统管理模块
- 用户管理
- 角色管理
- 菜单管理
- 操作日志
- 系统配置

### 4. 通用功能模块
- 文件上传/下载
- 数据导入/导出
- 消息通知
- 定时任务

## 数据库设计

### 核心表结构

#### 用户表 (users)
- id: 主键
- username: 用户名
- password: 密码哈希
- email: 邮箱
- phone: 手机号
- status: 状态
- created_at: 创建时间
- updated_at: 更新时间

#### 角色表 (roles)
- id: 主键
- name: 角色名称
- description: 角色描述
- permissions: 权限列表
- created_at: 创建时间
- updated_at: 更新时间

#### 菜单表 (menus)
- id: 主键
- parent_id: 父级菜单ID
- name: 菜单名称
- path: 路径
- icon: 图标
- sort: 排序
- status: 状态
- created_at: 创建时间
- updated_at: 更新时间

## API规范

### RESTful API设计

#### 统一响应格式
```json
{
  "code": 200,
  "message": "success",
  "data": {},
  "timestamp": "2024-01-01T00:00:00Z"
}
```

#### 状态码规范
- 200: 成功
- 400: 请求参数错误
- 401: 未授权
- 403: 禁止访问
- 404: 资源不存在
- 500: 服务器内部错误

### 请求规范
- GET: 获取资源
- POST: 创建资源
- PUT: 更新资源
- DELETE: 删除资源
- PATCH: 部分更新

## 安全规范

### 1. 身份验证
- JWT Token机制
- Token过期时间控制
- 刷新Token机制

### 2. 数据验证
- 输入参数验证
- SQL注入防护
- XSS攻击防护
- CSRF防护

### 3. 权限控制
- 接口权限验证
- 数据权限过滤
- 操作日志记录

## 性能优化

### 1. 数据库优化
- 索引优化
- 查询优化
- 连接池配置

### 2. 缓存策略
- Redis缓存
- 查询结果缓存
- 静态资源缓存

### 3. 前端优化
- 代码分割
- 懒加载
- 图片压缩
- CDN加速

## 部署规范

### 1. 环境配置
- 开发环境
- 测试环境
- 生产环境

### 2. 容器化部署
- Docker容器
- Docker Compose
- Kubernetes

### 3. 监控告警
- 系统监控
- 性能监控
- 错误告警

## 代码规范

### 1. Go代码规范
- 遵循Go官方编码规范
- 使用gofmt格式化代码
- 添加必要的注释
- 错误处理规范

### 2. 前端代码规范
- 遵循ESLint规则
- 使用Prettier格式化
- 组件命名规范
- Hook使用规范

### 3. 数据库规范
- 表名统一使用复数
- 字段命名使用下划线
- 索引命名规范
- 外键约束规范

## 测试规范

### 1. 单元测试
- 函数测试
- 模块测试
- 覆盖率要求

### 2. 集成测试
- API测试
- 数据库测试
- 缓存测试

### 3. 端到端测试
- UI测试
- 功能测试
- 性能测试

## 版本控制

### 1. Git规范
- 分支管理策略
- 提交信息规范
- 代码审查流程

### 2. 版本发布
- 版本号规范
- 发布流程
- 回滚策略