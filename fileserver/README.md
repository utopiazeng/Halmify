# Halmify 文件服务

**简体中文** | [English](./README.en.md) · [项目首页](../README.md)

独立的 Go 文件服务，使用 Gin、GORM、MySQL 和 Redis，供 CRM 文件网关通过签名内部接口调用。完整的数据库、密钥及 CRM 配置见[安装指南](../docs/INSTALL.md)。

## 配置

从仓库根目录执行：

```bash
cd fileserver
cp -n conf/app.conf.example conf/app.conf
```

编辑 `conf/app.conf`，替换数据库密码与共享密钥。默认配置读取路径为 `conf/app.conf`，可通过 `FILESERVER_CONFIG` 修改；即使通过环境变量提供参数，也需要配置文件存在。

| 参数 | 示例或说明 |
| --- | --- |
| `server_addr` | `127.0.0.1:18080`，内部监听地址 |
| `node_id` | `fs-1`，每个节点使用不同标识 |
| `storage_root` | `data`，相对于进程工作目录，需可写 |
| `db_name` | `ai_crm_files`，与 CRM 数据库分开 |
| `db_socket` | 留空通过 TCP 连接 |
| `app_id` / `app_secret` | 与 CRM 的 `fileserver_app_id` / `fileserver_app_secret` 一致 |
| `redis_host` / `redis_port` | 示例为 `127.0.0.1` / `6379` |
| `max_upload_bytes` | 示例为 1 GiB；CRM 网关另有上传限制，示例为 10 MiB |

程序会创建存储目录，上传数据默认位于 `fileserver/data/` 并被 Git 忽略。多节点部署时 CRM 的 `fileserver_nodes` 可配置多个节点地址。

## 迁移与启动

以下命令在 `fileserver/` 内执行：

```bash
go mod download
go run ./cmd/server migrate
go run ./cmd/server
```

正常启动不会执行数据库迁移。新安装或升级时先运行 `migrate`，使用 MySQL `GET_LOCK` 协调迁移，未获得锁时迁移命令报错退出。

启动需要 MySQL 和 Redis。包括 `/internal/v1/health` 在内的内部接口使用应用 ID、HMAC 签名、时间戳和 nonce 校验，不能用普通匿名请求判断其是否正常；CRM 网关会执行签名健康探测。

## 构建与测试

在 `fileserver/` 内执行：

```bash
go build -o bin/fileserver ./cmd/server
```

生产迁移和启动使用同一个二进制，并保持工作目录能找到配置文件：

```bash
./bin/fileserver migrate
./bin/fileserver
```

单元测试：

```bash
go test ./...
```

文件服务用于内部通信，示例仅监听回环地址；跨主机部署时按网络环境修改监听和节点地址。备份时同时保存 MySQL 文件元数据和存储目录中的文件。
