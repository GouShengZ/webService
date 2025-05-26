# WebService - Golang Web框架项目

这是一个基于Gin框架构建的完整Golang Web服务项目，集成了日志管理、链路追踪、JWT认证、GORM数据库操作等功能。

## 🚀 功能特性

- **Web框架**: 使用Gin框架构建高性能Web服务
- **数据库**: 集成GORM V2，支持MySQL数据库
- **中间件系统**:
  - 日志中间件：完整的请求日志记录
  - 响应格式化：统一的API响应格式
  - 链路追踪：基于Jaeger的分布式追踪
  - JWT认证：完整的用户认证和授权系统
  - 请求ID：为每个请求生成唯一标识
  - CORS：跨域资源共享支持
- **日志系统**: 基于Logrus的结构化日志，支持日志分割和轮转
- **配置管理**: 使用Viper进行配置管理
- **数据库迁移**: 自动数据库表结构迁移和种子数据
- **优雅关闭**: 支持服务的优雅关闭

## 📁 项目结构

```
webService/
├── main.go                    # 程序入口
├── config.yaml               # 配置文件
├── go.mod                     # Go模块文件
├── README.md                  # 项目文档
└── internal/                  # 内部包
    ├── config/                # 配置管理
    │   └── config.go
    ├── database/              # 数据库连接
    │   └── database.go
    ├── logger/                # 日志管理
    │   └── logger.go
    ├── tracer/                # 链路追踪
    │   └── tracer.go
    ├── middleware/            # 中间件
    │   ├── auth.go           # JWT认证中间件
    │   ├── logger.go         # 日志中间件
    │   ├── request_id.go     # 请求ID中间件
    │   ├── response.go       # 响应格式化中间件
    │   └── tracing.go        # 链路追踪中间件
    ├── models/                # 数据模型
    │   └── user.go
    ├── service/               # 业务逻辑层
    │   └── user.go
    ├── handler/               # 处理器层
    │   └── handler.go
    ├── router/                # 路由配置
    │   └── router.go
    └── migration/             # 数据库迁移
        └── migration.go
```

## 🛠️ 安装和运行

### 前置要求

- Go 1.21+
- MySQL 5.7+
- Jaeger (可选，用于链路追踪)

### 1. 克隆项目

```bash
git clone <repository-url>
cd webService
```

### 2. 安装依赖

```bash
go mod tidy
```

### 3. 配置数据库

修改 `config.yaml` 文件中的数据库配置：

```yaml
database:
  host: localhost
  port: 3306
  username: your_username
  password: your_password
  database: your_database
```

### 4. 启动服务

```bash
go run main.go
```

服务将在 `http://localhost:8080` 启动。

## 📚 API文档

### 健康检查

```http
GET /health
GET /ping
```

### 用户认证

#### 用户注册
```http
POST /api/v1/public/register
Content-Type: application/json

{
  "username": "testuser",
  "email": "test@example.com",
  "password": "password123",
  "nickname": "Test User"
}
```

#### 用户登录
```http
POST /api/v1/public/login
Content-Type: application/json

{
  "username": "testuser",
  "password": "password123"
}
```

#### 刷新Token
```http
POST /api/v1/public/refresh
Content-Type: application/json

{
  "token": "your_jwt_token"
}
```

### 用户管理（需要认证）

#### 获取个人资料
```http
GET /api/v1/auth/profile
Authorization: Bearer your_jwt_token
```

#### 更新个人资料
```http
PUT /api/v1/auth/profile
Authorization: Bearer your_jwt_token
Content-Type: application/json

{
  "nickname": "New Nickname",
  "avatar": "avatar_url",
  "email": "new@example.com"
}
```

#### 用户登出
```http
POST /api/v1/auth/logout
Authorization: Bearer your_jwt_token
```

### 管理员功能（需要管理员权限）

#### 获取用户列表
```http
GET /api/v1/admin/users?page=1&page_size=10&role=user&status=1
Authorization: Bearer admin_jwt_token
```

#### 获取用户详情
```http
GET /api/v1/admin/users/{id}
Authorization: Bearer admin_jwt_token
```

#### 更新用户信息
```http
PUT /api/v1/admin/users/{id}
Authorization: Bearer admin_jwt_token
Content-Type: application/json

{
  "nickname": "Updated Nickname",
  "role": "admin",
  "status": 1
}
```

#### 删除用户
```http
DELETE /api/v1/admin/users/{id}
Authorization: Bearer admin_jwt_token
```

### 公开用户信息

#### 获取公开用户列表
```http
GET /api/v1/users?page=1&page_size=10
```

#### 获取公开用户详情
```http
GET /api/v1/users/{id}
```

## 🔧 配置说明

### 服务器配置
```yaml
server:
  port: 8080              # 服务端口
  mode: debug             # 运行模式: debug, release, test
  read_timeout: 60s       # 读取超时
  write_timeout: 60s      # 写入超时
```

### 数据库配置
```yaml
database:
  driver: mysql           # 数据库驱动
  host: localhost         # 数据库主机
  port: 3306             # 数据库端口
  username: root         # 用户名
  password: password     # 密码
  database: webservice   # 数据库名
  charset: utf8mb4       # 字符集
  parse_time: true       # 解析时间
  loc: Local             # 时区
  max_idle_conns: 10     # 最大空闲连接数
  max_open_conns: 100    # 最大打开连接数
  conn_max_lifetime: 3600s # 连接最大生存时间
```

### 日志配置
```yaml
log:
  level: info            # 日志级别: debug, info, warn, error
  format: json           # 日志格式: json, text
  output: file           # 输出方式: console, file, both
  file_path: ./logs/app.log # 日志文件路径
  max_size: 100          # 单个日志文件最大大小(MB)
  max_backups: 30        # 保留的日志文件数量
  max_age: 7             # 日志文件保留天数
  compress: true         # 是否压缩旧日志文件
```

### JWT配置
```yaml
jwt:
  secret: your-secret-key # JWT密钥
  expire_time: 24h       # Token过期时间
  issuer: webservice     # 签发者
```

### Jaeger配置
```yaml
jaeger:
  service_name: webservice # 服务名称
  agent_host: localhost    # Jaeger Agent主机
  agent_port: 6831        # Jaeger Agent端口
  sampler_type: const     # 采样类型
  sampler_param: 1        # 采样参数
```

## 🔐 默认用户

项目启动时会自动创建以下默认用户：

- **管理员用户**:
  - 用户名: `admin`
  - 密码: `password`
  - 角色: `admin`

- **测试用户**:
  - 用户名: `testuser`
  - 密码: `password`
  - 角色: `user`

## 📝 响应格式

所有API响应都遵循统一的格式：

```json
{
  "code": 0,
  "message": "success",
  "data": {},
  "timestamp": 1640995200,
  "request_id": "uuid-string"
}
```

- `code`: 响应码，0表示成功，其他值表示错误
- `message`: 响应消息
- `data`: 响应数据（可选）
- `timestamp`: 响应时间戳
- `request_id`: 请求唯一标识

## 🚀 部署

### Docker部署

1. 构建镜像：
```bash
docker build -t webservice .
```

2. 运行容器：
```bash
docker run -p 8080:8080 -v $(pwd)/config.yaml:/app/config.yaml webservice
```

### 生产环境配置

1. 修改配置文件中的运行模式：
```yaml
server:
  mode: release
```

2. 设置环境变量：
```bash
export WEBSERVICE_JWT_SECRET=your-production-secret
export WEBSERVICE_DATABASE_PASSWORD=your-production-password
```

## 🤝 贡献

欢迎提交Issue和Pull Request来改进这个项目。

## 📄 许可证

本项目采用MIT许可证。