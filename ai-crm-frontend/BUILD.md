# 前端构建与部署

**简体中文** | [English](./BUILD.en.md) · [前端说明](./README.md)

需要 Node.js 20 与 npm。所有命令默认在仓库的前端目录执行：

```bash
cd ai-crm-frontend
npm ci
```

## 选择部署路径

| 部署方式 | 构建命令 | 访问地址示例 |
| --- | --- | --- |
| 默认子路径 | `npm run build` | `https://crm.example.com/demo/` |
| 域名根路径 | `npm run build -- --base /` | `https://crm.example.com/` |

产物输出到 `dist/`。构建路径影响静态资源 URL 和 Vue Router 的 history 基路径，必须与部署路径一致。改变部署路径后重新构建，同时修改后端的 `tenant_admin_activation_base_url`。

当前 `build` 脚本使用依赖中的 Node 20。需要显式构建时也可使用：

```bash
npm exec --package node@20 -- node ./node_modules/vite/bin/vite.js build --base /
```

## 使用 Go 静态运行器

`serve-dist.go` 支持 SPA 回退，并将 `/api/`（包括 WebSocket）代理到后端。下面的方式适合根路径部署，运行时不需要 Node：

```bash
npm run build -- --base /
CGO_ENABLED=0 go build -o serve-dist serve-dist.go
./serve-dist -addr 127.0.0.1:3001 -api http://127.0.0.1:8092
```

访问 `http://localhost:3001/`。请显式传入 `-api`，因为运行器代码中的默认后端端口仍为 `8091`。可通过 `-dist` 指定产物目录，默认是当前目录的 `dist/`。

复制 `dist/` 和适合目标操作系统、架构的 `serve-dist` 二进制即可在另一台服务器运行。生产环境可在前方配置 TLS 反向代理；本机例子只监听回环地址。

## 使用 Nginx

将新构建的 `dist/` 内容复制到站点目录。根路径部署示例中，将 `/srv/halmify/web` 改成实际的静态目录：

```nginx
server {
    listen 80;
    server_name crm.example.com;
    root /srv/halmify/web;
    index index.html;

    location / {
        try_files $uri $uri/ /index.html;
    }

    location /api/ {
        proxy_pass http://127.0.0.1:8092;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_set_header Host $host;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_read_timeout 300s;
        client_max_body_size 10m;
    }
}
```

该例使用 HTTP，生产域名需按环境配置 TLS。Nginx 上传限制应与 CRM 上传限制一致；需要较大固件或其他文件时，按实际接口调整。

对于默认 `/demo/` 构建，将产物复制到 `/srv/halmify/web/demo/`，用下列块替换上面的 `location /`，保留同级的 `/api/` 代理：

```nginx
location = /demo {
    return 301 /demo/;
}
location /demo/ {
    try_files $uri $uri/ /demo/index.html;
}
```

例如 `dist/assets/` 应成为 `/srv/halmify/web/demo/assets/`。激活链接的基础地址相应设置为 `https://crm.example.com/demo`。

## 验证与常见问题

发布前移除旧构建产物并按目标路径重建：

```bash
rm -rf dist
npm run build -- --base /
```

配置默认不生成 sourcemap。检查新产物时，下面两个命令应没有匹配；`rg` 没有匹配时退出码为 `1`：

```bash
find dist -name '*.map'
rg 'sourceMappingURL' dist
```

上传时替换旧静态产物，避免旧文件残留；执行 `nginx -t` 检查配置后，再按服务器的服务管理方式重载 Nginx。

- 静态资源 `404`：核对构建 `base`、访问 URL 和产物目录位置。
- 子页面刷新 `404`：确认 SPA 回退到对应的 `index.html`。
- API `502`：确认后端在 `8092` 运行，并检查代理目标。
- Vite chunk-size warning：提示部分包体积较大，通常不阻止构建；以退出码判断构建是否成功。
- 服务器 Node 或 `libstdc++` 不兼容：在兼容的构建环境生成产物，使用 Go 运行器或 Nginx 提供静态文件。
