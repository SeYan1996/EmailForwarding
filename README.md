# 邮件拉取与主题转发系统

一个基于Go语言开发的智能邮件转发系统，能够自动拉取Gmail邮箱中的邮件，根据邮件主题中的关键字和转发目标进行智能分发。

## 功能特性

- 📧 **Gmail集成**: 支持Gmail API，安全读取和发送邮件
- 🎯 **智能转发**: 根据邮件主题中的关键字和目标名称自动转发
- 💾 **数据持久化**: 使用MySQL存储转发目标和邮件处理记录
- ⏰ **定时任务**: 支持定时检查新邮件并自动处理
- 🌐 **REST API**: 提供完整的API接口进行管理
- 📊 **日志记录**: 详细的处理日志和错误跟踪
- 🔧 **灵活配置**: 支持环境变量配置

## 技术栈

- **语言**: Go 1.21+
- **Web框架**: Gin
- **ORM**: GORM
- **数据库**: MySQL
- **邮件服务**: Gmail API
- **日志**: Logrus

## 项目结构

```
EmailForwarding/
├── main.go                 # 程序入口
├── config/                 # 配置管理
│   └── config.go
├── models/                 # 数据模型
│   ├── email.go
│   └── forward_target.go
├── services/               # 业务逻辑
│   ├── gmail_service.go
│   └── email_service.go
├── handlers/               # HTTP处理器
│   └── email_handler.go
├── database/               # 数据库连接
│   └── database.go
├── utils/                  # 工具类
│   └── logger.go
├── go.mod                  # Go模块文件
└── config.example          # 配置示例
```

## 安装部署

### 1. 环境要求

- Go 1.21+
- MySQL 5.7+
- Gmail账号（需要开启API访问）

### 2. 克隆项目

```bash
git clone <repository-url>
cd EmailForwarding
```

### 3. 安装依赖

```bash
go mod tidy
```

### 4. 配置Gmail API

1. 访问 [Google Cloud Console](https://console.cloud.google.com/)
2. 创建新项目或选择现有项目
3. 启用Gmail API
4. 创建OAuth 2.0凭据
5. 下载凭据文件并命名为 `credentials.json`，放到项目根目录

### 5. 配置数据库

创建MySQL数据库：

```sql
CREATE DATABASE email_forwarding CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
```

### 6. 环境配置

复制配置示例文件并修改：

```bash
cp config.example .env
```

编辑 `.env` 文件，填入正确的配置信息：

```env
# 数据库配置
DB_HOST=localhost
DB_PORT=3306
DB_USER=root
DB_PASSWORD=your_password
DB_NAME=email_forwarding

# Gmail配置
GMAIL_CREDENTIALS_FILE=credentials.json
GMAIL_TOKEN_FILE=token.json
GMAIL_USER_EMAIL=your-email@gmail.com

# 服务器配置
SERVER_PORT=8080
GIN_MODE=release

# 应用配置
CHECK_INTERVAL=5m
```

### 7. 运行程序

```bash
go run main.go
```

首次运行会要求OAuth授权，按照提示在浏览器中完成授权。

## 使用说明

### 邮件标题格式

系统会解析符合以下格式的邮件主题：

```
关键字 - 转发目标名称
```

**示例**：
- `客户投诉 - 客服部门`
- `技术故障 - 技术支持`
- `商务合作 - 销售部门`

### API接口

#### 1. 健康检查

```http
GET /health
```

#### 2. 手动处理邮件

```http
POST /api/v1/emails/process
```

#### 3. 获取邮件日志

```http
GET /api/v1/emails/logs?page=1&page_size=20&status=success
```

参数：
- `page`: 页码（默认1）
- `page_size`: 每页大小（默认20，最大100）
- `status`: 状态筛选（pending/success/failed）

#### 4. 获取转发目标列表

```http
GET /api/v1/targets
```

#### 5. 创建转发目标

```http
POST /api/v1/targets
Content-Type: application/json

{
  "name": "客服部门",
  "email": "customer-service@company.com",
  "keywords": "客户,投诉,咨询",
  "is_active": true
}
```

#### 6. 更新转发目标

```http
PUT /api/v1/targets/:id
Content-Type: application/json

{
  "name": "客服部门",
  "email": "new-email@company.com",
  "keywords": "客户,投诉,咨询,服务",
  "is_active": true
}
```

#### 7. 删除转发目标

```http
DELETE /api/v1/targets/:id
```

## 数据库表结构

### 转发目标表 (forward_targets)

| 字段 | 类型 | 说明 |
|------|------|------|
| id | uint | 主键ID |
| name | string | 转发目标名称 |
| email | string | 转发目标邮箱 |
| keywords | string | 关联关键字（逗号分隔） |
| is_active | bool | 是否启用 |
| created_at | datetime | 创建时间 |
| updated_at | datetime | 更新时间 |

### 邮件日志表 (email_logs)

| 字段 | 类型 | 说明 |
|------|------|------|
| id | uint | 主键ID |
| gmail_message_id | string | Gmail消息ID |
| subject | string | 邮件主题 |
| from_email | string | 发件人 |
| to_email | string | 收件人 |
| content | text | 邮件内容 |
| keyword | string | 匹配的关键字 |
| forward_target | string | 转发目标名称 |
| forward_email | string | 转发目标邮箱 |
| forward_status | string | 转发状态 |
| error_message | text | 错误信息 |
| processed_at | datetime | 处理时间 |
| created_at | datetime | 创建时间 |
| updated_at | datetime | 更新时间 |

## 系统特性

### 健壮性设计

1. **重复处理防护**: 使用Gmail消息ID防止重复处理同一邮件
2. **错误处理**: 完善的错误捕获和日志记录
3. **事务保护**: 数据库操作使用事务确保一致性
4. **优雅关闭**: 支持优雅停机，确保正在处理的任务完成

### 边界条件处理

1. **邮件格式验证**: 严格验证邮件主题格式
2. **转发目标验证**: 检查转发目标是否存在且有效
3. **关键字匹配**: 支持模糊匹配和精确匹配
4. **网络异常**: 自动重试机制和错误降级
5. **权限验证**: Gmail API权限检查和token刷新

### 扩展性

1. **插件化设计**: 易于扩展新的邮件服务提供商
2. **规则引擎**: 可扩展更复杂的转发规则
3. **多租户**: 支持多个Gmail账号管理
4. **监控告警**: 集成监控和告警机制

## 故障排除

### 常见问题

1. **OAuth授权失败**
   - 检查Gmail API是否已启用
   - 确认credentials.json文件正确
   - 检查Google账号安全设置

2. **数据库连接失败**
   - 验证数据库配置信息
   - 确认数据库服务已启动
   - 检查网络连接

3. **邮件处理失败**
   - 查看日志了解具体错误
   - 检查Gmail API配额
   - 验证邮件格式是否正确

### 日志级别

- `INFO`: 正常运行信息
- `WARN`: 警告信息
- `ERROR`: 错误信息
- `DEBUG`: 调试信息

## 开发计划

- [ ] 支持多种邮件服务提供商
- [ ] Web管理界面
- [ ] 邮件模板定制
- [ ] 批量操作功能
- [ ] 性能监控面板
- [ ] 邮件内容关键字匹配
- [ ] 自动化测试覆盖

## 贡献

欢迎提交Issue和Pull Request来改进项目。

## 许可证

MIT License
