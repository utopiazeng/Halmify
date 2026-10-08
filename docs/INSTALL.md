# 安装与配置

**简体中文** | [English](./INSTALL.en.md) · [项目首页](../README.md)

本文采用单机本地开发配置。每个命令块均说明工作目录；启动服务时使用不同终端。生产前端部署见[构建指南](../ai-crm-frontend/BUILD.md)。

## 1. 准备环境与源码

| 工具或服务 | 要求 |
| --- | --- |
| Go | 1.22 或更高版本，供两个 Go 模块使用 |
| Node.js / npm | Node.js 20，npm 用于安装和构建前端 |
| MySQL | 推荐 MySQL 8；迁移使用 MySQL 专用语句与命名锁 |
| Redis | 推荐 Redis 7；CRM 与文件服务启动时均需连接 |
| Git / OpenSSL | 克隆源码及生成随机密钥 |

先通过操作系统包管理器或服务官方安装说明安装并启动 MySQL、Redis，然后获取源码：

```bash
git clone https://github.com/utopiazeng/Halmify.git
cd Halmify
go version
node --version
npm --version
mysql --version
redis-cli ping
```

本地 Redis 示例使用 `127.0.0.1:6379`，无密码，预期 `ping` 返回 `PONG`。如使用其他地址或认证，请在两个服务的配置中填写相应值。

## 2. 创建空数据库

使用具有建库及创建用户权限的 MySQL 管理账号连接，例如：

```bash
mysql -u root -p
```

执行下列 SQL，先把密码占位值替换为自己的密码。示例账号仅获准访问项目的两个数据库：

```sql
CREATE DATABASE ai_crm CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
CREATE DATABASE ai_crm_files CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
CREATE USER 'ai_crm'@'127.0.0.1' IDENTIFIED BY 'replace-me-db-password';
GRANT ALL PRIVILEGES ON ai_crm.* TO 'ai_crm'@'127.0.0.1';
GRANT ALL PRIVILEGES ON ai_crm_files.* TO 'ai_crm'@'127.0.0.1';
```

如用户或数据库已存在，按实际环境调整；MySQL 连接账号的 host 需匹配实际连接来源。服务会通过代码建立表和迁移数据，不需要导入本地 SQL 备份。CRM 迁移也会创建内置演示租户与样例业务记录。

## 3. 配置服务

从仓库根目录执行：

```bash
cp -n ai-crm/conf/app.conf.example ai-crm/conf/app.conf
cp -n fileserver/conf/app.conf.example fileserver/conf/app.conf
```

`-n` 保留已有配置。如果文件已经存在，请对照示例逐项调整，尤其检查端口、数据库 socket 和凭据。两个实际配置文件都被 Git 忽略。

分别生成独立的随机密钥，每个用途执行一次：

```bash
openssl rand -hex 32
```

填写配置中的所有 `replace-me` 占位值：

| 用途 | 配置位置 |
| --- | --- |
| 数据库密码 | 两个文件的 `db_password`，与 MySQL 账号密码一致 |
| JWT 签名密钥 | CRM 的 `jwt_secret` |
| 官网内部集成令牌 | CRM 的 `official_site_internal_token` |
| IoT 主密钥 | CRM 的 `iot_master_secret` |
| 文件服务共享密钥 | CRM 的 `fileserver_app_secret` 和文件服务的 `app_secret`，两处使用同一个随机值 |

文件服务的 `app_id` 应与 CRM 的 `fileserver_app_id` 一致，示例均为 `ai-crm`。保持两个 `db_socket` 为空即可使用本地 TCP 连接；已有配置中的 socket 值需显式清空，不能仅设置 `DB_HOST` 来切换连接方式。

示例将上传文件保存在 `fileserver/data/`，服务会自动创建目录；进程必须具有写权限。CRM 示例保留磁盘容量保护，要求至少 1 GiB 空闲空间，且使用率低于 85%。请按实际磁盘容量调整。

### 环境变量与可选配置

配置加载器支持的环境变量会覆盖文件配置；程序不会自动加载 `.env`。前端的 `.env` 由 Vite 加载，不能用于存放后端密钥。

| 组件 | 常用环境变量 |
| --- | --- |
| CRM 数据库 | `DB_HOST`、`DB_PORT`、`DB_USER`、`DB_PASSWORD`、`DB_NAME`、`DB_SOCKET`、`DB_RUN_MIGRATIONS` |
| CRM Redis | `REDIS_ADDR`、`REDIS_PASSWORD`、`REDIS_DB` |
| CRM 身份与设备 | `JWT_SECRET`、`IOT_MASTER_SECRET` |
| CRM 文件网关 | `FILESERVER_NODES`、`FILESERVER_APP_ID`、`FILESERVER_APP_SECRET` |
| 文件服务 | `FILESERVER_CONFIG`、`FILESERVER_ADDR`、`FILESERVER_STORAGE_ROOT`、`FILESERVER_NODE_ID` |
| 文件服务数据库 | `FILESERVER_DB_HOST`、`FILESERVER_DB_PORT`、`FILESERVER_DB_USER`、`FILESERVER_DB_PASSWORD`、`FILESERVER_DB_NAME` |
| 文件服务 Redis | `FILESERVER_REDIS_HOST`、`FILESERVER_REDIS_PORT`、`FILESERVER_REDIS_PASSWORD`、`FILESERVER_REDIS_DB` |
| AI / RAG | CRM 的 `AI_API_URL`、`AI_API_KEY`、`AI_MODEL`、`RAG_API_URL`、`RAG_API_KEY` |

文件服务默认读取其工作目录下的 `conf/app.conf`，即使使用环境变量也需要该配置文件存在。CRM 通过 Beego 读取 `conf/app.conf`。

AI 不是启动本地 CRM 的必需条件。需要企业研究或 AI 分析时，填写真实的服务地址、API 密钥与可用模型；可通过平台及租户 AI 设置管理相关能力。示例使用 DeepSeek 接口地址，租户 AI 设置也支持 OpenAI 及兼容接口。API 调用可能产生服务商费用。前端品牌可通过 `VITE_BRAND_NAME`、`VITE_BRAND_TAGLINE` 配置，详见[前端说明](../ai-crm-frontend/README.md)。

## 4. 安装依赖并启动

以下三个步骤均从仓库根目录打开独立终端执行。

### 终端 A：文件服务

```bash
cd fileserver
go mod download
go run ./cmd/server migrate
go run ./cmd/server
```

必须先执行迁移；正常启动不会自动建表。内部接口监听 `127.0.0.1:18080`，包含健康检查在内的 `/internal/v1/` 接口都需要签名，直接请求可能返回认证错误。

### 终端 B：CRM 后端

```bash
cd ai-crm
go mod download
go run .
```

示例配置监听 `8092`，启动时自动执行增量迁移并通过 MySQL 命名锁协调。首次启动保持 `db_run_migrations = true`。升级已有数据库前应备份；需要由指定实例完成迁移时，再为其他实例设置 `DB_RUN_MIGRATIONS=false`。

### 终端 C：前端

```bash
cd ai-crm-frontend
npm ci
npm run dev
```

访问 `http://localhost:3001/`。Vite 将 `/api` 及 WebSocket 请求代理到 `http://localhost:8092`。

## 5. 首次账号初始化

在本地完成初始化后再对外开放服务。只有 `users` 表为空时，首次注册接口才接受请求。先替换下面的用户名、邮箱与密码：

```bash
curl --fail-with-body http://127.0.0.1:8092/api/v1/auth/register \
  -H 'Content-Type: application/json' \
  --data '{"username":"platform-admin","password":"replace-me-unique-admin-password","email":"admin@example.com"}'
```

此账号的类型为平台超级管理员，无预置通用登录密码。之后的注册会返回 `403`，其他账号由平台管理员或企业管理员创建。

1. 打开前端并使用刚创建的账号登录，完成登录滑块验证。
2. 在平台租户管理中创建企业，填写企业名称、管理员用户名和邮箱，并选择相应套餐与模块。
3. 复制创建结果中的一次性激活链接，让企业管理员打开链接设置密码。示例链接有效期为 24 小时；它不是自动发送的邮件，也不是固定初始密码。
4. 使用企业管理员账号登录 CRM，再通过系统管理创建成员和分配角色。

平台管理员负责平台管理，不能直接进入租户业务模块。创建企业的 API 为 `POST /api/v1/tenants`，需平台管理员认证；创建响应包含 `activation_link_once`。

本地激活地址由 `tenant_admin_activation_base_url = http://localhost:3001` 指定。生产部署时改为真实前端地址；部署到 `/demo/` 时例如使用 `https://crm.example.com/demo`。也可通过 `TENANT_ADMIN_ACTIVATION_BASE_URL` 覆盖。

## 6. 常见问题

| 现象 | 排查方式 |
| --- | --- |
| 数据库连接失败 | 检查数据库是否存在、账号授权、密码、TCP 地址及是否残留 `db_socket` |
| Redis 连接失败 | 两个服务都需要 Redis；检查地址、认证和数据库编号 |
| 前端请求 `502` | 确认后端监听 `8092`，并与 Vite 代理目标或静态运行器的 `-api` 参数一致 |
| 上传提示节点不可用 | 检查文件服务迁移、共享 ID/密钥、存储写权限及磁盘空闲容量 |
| 激活链接地址错误 | 修改激活前端地址，包含实际部署路径，然后重新生成链接 |
| 页面刷新后 `404` 或静态资源失败 | 配置 SPA 回退；构建 `base` 必须与部署路径一致，见构建指南 |

数据库备份、真实配置和私有生产审计脚本由 `.gitignore` 排除。发布前检查 `git status --short` 与待提交文件清单；忽略规则不会移除已经跟踪的文件。
