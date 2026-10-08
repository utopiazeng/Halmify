# Halmify file service

[简体中文](./README.md) | **English** · [Project home](../README.en.md)

A dedicated Go file service using Gin, GORM, MySQL, and Redis. The CRM file gateway calls its signed internal APIs. See the [installation guide](../docs/INSTALL.en.md) for database, secret, and CRM configuration.

## Configuration

From the repository root:

```bash
cd fileserver
cp -n conf/app.conf.example conf/app.conf
```

Edit `conf/app.conf` to replace the database password and shared secret. The default configuration path is `conf/app.conf`; override it with `FILESERVER_CONFIG`. The file must exist even when supplying settings through environment variables.

| Setting | Example or purpose |
| --- | --- |
| `server_addr` | `127.0.0.1:18080`, internal listen address |
| `node_id` | `fs-1`; use a different identifier for each node |
| `storage_root` | `data`, relative to the working directory and writable by the process |
| `db_name` | `ai_crm_files`, separate from the CRM database |
| `db_socket` | Empty for TCP connections |
| `app_id` / `app_secret` | Match CRM `fileserver_app_id` / `fileserver_app_secret` |
| `redis_host` / `redis_port` | Example: `127.0.0.1` / `6379` |
| `max_upload_bytes` | 1 GiB in the example; the CRM gateway has a separate limit of 10 MiB in its example |

The service creates storage directories. Uploads go to `fileserver/data/` by default and are ignored by Git. For multiple nodes, configure their addresses in CRM `fileserver_nodes`.

## Migrations and startup

Run inside `fileserver/`:

```bash
go mod download
go run ./cmd/server migrate
go run ./cmd/server
```

Normal startup does not run migrations. Run `migrate` first for a new installation or upgrade. MySQL `GET_LOCK` coordinates migrations; the migration command exits with an error if it cannot acquire the lock.

Startup requires MySQL and Redis. Internal endpoints, including `/internal/v1/health`, validate the application ID, HMAC signature, timestamp, and nonce. An anonymous request is not a valid health check; the CRM gateway performs signed probes.

## Build and tests

Run inside `fileserver/`:

```bash
go build -o bin/fileserver ./cmd/server
```

Use the same binary for production migrations and startup, with a working directory that contains the configuration file:

```bash
./bin/fileserver migrate
./bin/fileserver
```

Unit tests:

```bash
go test ./...
```

The file service is intended for internal communication; the example listens only on loopback. For deployments across hosts, adjust the listen and node addresses to your network. Back up both the MySQL file metadata and files in the storage directory.
