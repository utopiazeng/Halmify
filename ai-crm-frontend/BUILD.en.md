# Frontend build and deployment

[简体中文](./BUILD.md) | **English** · [Frontend guide](./README.en.md)

Use Node.js 20 and npm. Run commands from the repository's frontend directory:

```bash
cd ai-crm-frontend
npm ci
```

## Choose the deployment path

| Deployment | Build command | Example URL |
| --- | --- | --- |
| Default subpath | `npm run build` | `https://crm.example.com/demo/` |
| Domain root | `npm run build -- --base /` | `https://crm.example.com/` |

Output goes to `dist/`. The build path controls asset URLs and the Vue Router history base and must match the deployment path. Rebuild when changing the path and update the backend's `tenant_admin_activation_base_url`.

The current `build` script uses the Node 20 dependency. An explicit alternative is:

```bash
npm exec --package node@20 -- node ./node_modules/vite/bin/vite.js build --base /
```

## Go static runner

`serve-dist.go` provides an SPA fallback and proxies `/api/`, including WebSocket connections, to the backend. This example serves the domain root without requiring Node at runtime:

```bash
npm run build -- --base /
CGO_ENABLED=0 go build -o serve-dist serve-dist.go
./serve-dist -addr 127.0.0.1:3001 -api http://127.0.0.1:8092
```

Visit `http://localhost:3001/`. Explicitly pass `-api`: the runner's code still defaults to backend port `8091`. Use `-dist` to select another output directory; it defaults to `dist/` in the working directory.

To run on another server, copy `dist/` and a `serve-dist` binary built for its OS and architecture. Place a TLS reverse proxy in front for production; this local example listens only on loopback.

## Nginx

Copy the contents of the new `dist/` build into the site directory. For domain-root deployment, replace `/srv/halmify/web` with your actual static directory:

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

This example uses HTTP; configure TLS for your production domain. Match Nginx's upload limit to the CRM limit; adjust it for the relevant endpoints if larger firmware or other files are needed.

For the default `/demo/` build, copy the output into `/srv/halmify/web/demo/` and replace `location /` above with these blocks. Keep the separate `/api/` proxy:

```nginx
location = /demo {
    return 301 /demo/;
}
location /demo/ {
    try_files $uri $uri/ /demo/index.html;
}
```

For example, `dist/assets/` becomes `/srv/halmify/web/demo/assets/`. Set the activation link base URL to `https://crm.example.com/demo`.

## Verification and troubleshooting

Remove old build output and rebuild for the target path before publishing:

```bash
rm -rf dist
npm run build -- --base /
```

The configuration disables sourcemaps by default. These checks should find no matches in the new output; `rg` returns exit code `1` when there are no matches:

```bash
find dist -name '*.map'
rg 'sourceMappingURL' dist
```

Replace old static output during upload to avoid leftover files. Validate configuration with `nginx -t`, then reload Nginx using your server's service manager.

- Asset `404`: check the build `base`, URL, and output directory placement.
- Page refresh `404`: configure the SPA fallback to the correct `index.html`.
- API `502`: confirm the backend is running on `8092` and check the proxy target.
- Vite chunk-size warning: indicates larger bundles and usually does not block the build; check the exit code for success.
- Incompatible server Node or `libstdc++`: build in a compatible environment and serve the output with the Go runner or Nginx.
