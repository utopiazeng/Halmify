# Installation and configuration

[简体中文](./INSTALL.md) | **English** · [Project home](../README.en.md)

This guide uses a local, single-machine development setup. Each command block identifies its working directory; use separate terminals for long-running services. See the [build guide](../ai-crm-frontend/BUILD.en.md) for production frontend deployment.

## 1. Prepare the environment and source

| Tool or service | Requirement |
| --- | --- |
| Go | 1.22 or later for both Go modules |
| Node.js / npm | Node.js 20, with npm for frontend dependencies and builds |
| MySQL | MySQL 8 recommended; migrations use MySQL-specific statements and named locks |
| Redis | Redis 7 recommended; both services connect during startup |
| Git / OpenSSL | Clone the source and generate random secrets |

Install and start MySQL and Redis using your OS package manager or the services' official installation instructions, then obtain the source:

```bash
git clone https://github.com/utopiazeng/Halmify.git
cd Halmify
go version
node --version
npm --version
mysql --version
redis-cli ping
```

The local Redis example uses `127.0.0.1:6379` without a password; `ping` should return `PONG`. For a different address or authentication, update both services' configuration files.

## 2. Create empty databases

Connect using a MySQL administrator account that can create databases and users, for example:

```bash
mysql -u root -p
```

Run this SQL after replacing the password placeholder with your own password. The example account has access only to the two project databases:

```sql
CREATE DATABASE ai_crm CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
CREATE DATABASE ai_crm_files CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
CREATE USER 'ai_crm'@'127.0.0.1' IDENTIFIED BY 'replace-me-db-password';
GRANT ALL PRIVILEGES ON ai_crm.* TO 'ai_crm'@'127.0.0.1';
GRANT ALL PRIVILEGES ON ai_crm_files.* TO 'ai_crm'@'127.0.0.1';
```

Adapt these commands if the user or databases already exist. The MySQL account host must match the connection source. The services create tables and run migrations from code; importing local SQL backups is unnecessary. CRM migrations also create a built-in demo tenant and sample business records.

## 3. Configure the services

From the repository root:

```bash
cp -n ai-crm/conf/app.conf.example ai-crm/conf/app.conf
cp -n fileserver/conf/app.conf.example fileserver/conf/app.conf
```

`-n` preserves existing files. If a file already exists, compare it with the example, especially ports, database sockets, and credentials. Git ignores both actual configuration files.

Generate a separate random secret for each purpose by running this command once per secret:

```bash
openssl rand -hex 32
```

Replace every `replace-me` placeholder:

| Purpose | Setting |
| --- | --- |
| Database password | `db_password` in both files, matching the MySQL account password |
| JWT signing secret | CRM `jwt_secret` |
| Internal website integration token | CRM `official_site_internal_token` |
| IoT master secret | CRM `iot_master_secret` |
| Shared file service secret | CRM `fileserver_app_secret` and file service `app_secret`; use the same random value in both |

The file service `app_id` must match CRM `fileserver_app_id`; both examples use `ai-crm`. Leave both `db_socket` settings empty for local TCP connections. Explicitly clear any socket value in an existing configuration; setting `DB_HOST` alone does not switch it to TCP.

The example stores uploads under `fileserver/data/`; the service creates the directory and needs write access. The CRM example keeps disk protection enabled and requires at least 1 GiB of free disk space and usage below 85%. Adjust these thresholds to your storage capacity.

### Environment variables and optional settings

Environment variables supported by the configuration loaders override file settings. The Go services do not automatically load `.env` files. Vite loads frontend `.env` files; do not put backend secrets in them.

| Component | Common environment variables |
| --- | --- |
| CRM database | `DB_HOST`, `DB_PORT`, `DB_USER`, `DB_PASSWORD`, `DB_NAME`, `DB_SOCKET`, `DB_RUN_MIGRATIONS` |
| CRM Redis | `REDIS_ADDR`, `REDIS_PASSWORD`, `REDIS_DB` |
| CRM identity and devices | `JWT_SECRET`, `IOT_MASTER_SECRET` |
| CRM file gateway | `FILESERVER_NODES`, `FILESERVER_APP_ID`, `FILESERVER_APP_SECRET` |
| File service | `FILESERVER_CONFIG`, `FILESERVER_ADDR`, `FILESERVER_STORAGE_ROOT`, `FILESERVER_NODE_ID` |
| File service database | `FILESERVER_DB_HOST`, `FILESERVER_DB_PORT`, `FILESERVER_DB_USER`, `FILESERVER_DB_PASSWORD`, `FILESERVER_DB_NAME` |
| File service Redis | `FILESERVER_REDIS_HOST`, `FILESERVER_REDIS_PORT`, `FILESERVER_REDIS_PASSWORD`, `FILESERVER_REDIS_DB` |
| AI / RAG | CRM `AI_API_URL`, `AI_API_KEY`, `AI_MODEL`, `RAG_API_URL`, `RAG_API_KEY` |

The file service reads `conf/app.conf` relative to its working directory by default; this file must exist even when using environment variables. The CRM reads `conf/app.conf` through Beego.

AI configuration is optional for starting the local CRM. To use company research or AI analysis, configure a real service URL, API key, and available model. Platform and tenant AI settings manage these capabilities. The example uses the DeepSeek endpoint; tenant AI settings also support OpenAI and compatible endpoints. API calls may incur provider charges. Set frontend branding with `VITE_BRAND_NAME` and `VITE_BRAND_TAGLINE`; see the [frontend guide](../ai-crm-frontend/README.en.md).

## 4. Install dependencies and start services

Open three separate terminals at the repository root for these steps.

### Terminal A: file service

```bash
cd fileserver
go mod download
go run ./cmd/server migrate
go run ./cmd/server
```

Run the migration first; normal startup does not create tables. Internal APIs listen on `127.0.0.1:18080`. Every `/internal/v1/` endpoint, including health checks, requires a signature; an unsigned request may return an authentication error.

### Terminal B: CRM backend

```bash
cd ai-crm
go mod download
go run .
```

The example listens on `8092`. Startup runs incremental migrations with MySQL named locks for coordination. Keep `db_run_migrations = true` on the first startup. Back up existing databases before upgrading; if one designated instance performs migrations, set `DB_RUN_MIGRATIONS=false` for the other instances afterward.

### Terminal C: frontend

```bash
cd ai-crm-frontend
npm ci
npm run dev
```

Visit `http://localhost:3001/`. Vite proxies `/api` and WebSocket requests to `http://localhost:8092`.

## 5. Initialize the first account

Complete initialization locally before exposing the service. Registration accepts requests only while the `users` table is empty. Replace the username, email, and password in this example first:

```bash
curl --fail-with-body http://127.0.0.1:8092/api/v1/auth/register \
  -H 'Content-Type: application/json' \
  --data '{"username":"platform-admin","password":"replace-me-unique-admin-password","email":"admin@example.com"}'
```

This creates a platform super administrator. There is no preset shared login password. Subsequent registrations return `403`; platform or tenant administrators create additional accounts.

1. Open the frontend, log in with the new account, and complete the slider verification.
2. In platform tenant management, create a company with an administrator username and email, and select the appropriate plan and modules.
3. Copy the one-time activation link from the creation result. The tenant administrator opens it to set a password. The example link expires after 24 hours; it is not an automatically sent email or a fixed initial password.
4. Log in with the tenant administrator account, then create members and assign roles in system management.

Platform administrators manage the platform and cannot directly enter tenant business modules. The tenant creation API is `POST /api/v1/tenants` and requires platform administrator authentication; its response includes `activation_link_once`.

Local activation links use `tenant_admin_activation_base_url = http://localhost:3001`. Set the actual frontend URL for production, including the deployment path: for example, `https://crm.example.com/demo` for `/demo/`. `TENANT_ADMIN_ACTIVATION_BASE_URL` can override this setting.

## 6. Troubleshooting

| Symptom | Check |
| --- | --- |
| Database connection fails | Database existence, account grants, password, TCP address, and any remaining `db_socket` |
| Redis connection fails | Both services need Redis; check its address, authentication, and database number |
| Frontend API returns `502` | Backend must listen on `8092`, matching the Vite proxy or static runner's `-api` flag |
| Upload reports no available node | File service migration, shared ID/secret, writable storage, and available disk capacity |
| Activation link uses the wrong address | Update the frontend activation URL, including the deployment path, then regenerate the link |
| Refresh returns `404` or assets fail to load | Configure an SPA fallback and match the build `base` to the deployment path; see the build guide |

Git ignores database backups, actual configuration, and the private production audit script. Review `git status --short` and the file list before publishing; ignore rules do not remove files already tracked by Git.
