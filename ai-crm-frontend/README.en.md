# Halmify frontend

[简体中文](./README.md) | **English** · [Project home](../README.en.md)

The Web interface uses Vue 3, Vue Router, Element Plus, and Vite. See the [installation guide](../docs/INSTALL.en.md) for the complete environment and the [build guide](./BUILD.en.md) for production deployment.

## Local development

From the repository root:

```bash
cd ai-crm-frontend
npm ci
npm run dev
```

Node.js 20 is required. The development server listens on `3001` and proxies HTTP and WebSocket requests under `/api` to `http://localhost:8092`.

Default branding comes from `src/brand.js`. To customize it, copy `.env.example` to `.env`, edit `VITE_BRAND_NAME` and `VITE_BRAND_TAGLINE`, then restart the development server or rebuild. `VITE_` variables are included in browser code and must not contain secrets.

## Translation workflow

Maintain source Chinese text in `messages.zh-CN` within `src/translate.md`. The synchronization script maintains generated files and fills translations for the other 11 languages.

- Use `t(key)` for new or changed UI text instead of writing Chinese directly in templates or scripts.
- Treat `src/translate.md` as the source of truth; do not edit generated files manually.
- Run `npm run i18n:sync` after changes. It may call an external translation service; failures produce a pending report and fail the checks.
- Keep menu, button, and status labels short; use tooltips or help text for longer explanations.

Run in the frontend directory:

```bash
npm run i18n:sync
npm run i18n:sync:check
npm run i18n:scan:ci
npm run i18n:runtime:ci
npm run build
```

`i18n:sync:check` only checks files and does not call translation services. Synchronization and checking scripts use Node 20 through npm; the first run may require a network download. The interface supports Simplified Chinese, Traditional Chinese, English, Spanish, French, German, Japanese, Korean, Portuguese, Arabic, Russian, and Italian.

## Tests and builds

```bash
npm run test:unit
npm run build
```

The default production build uses `/demo/`. For deployment at the domain root, use `npm run build -- --base /`. See [BUILD.en.md](./BUILD.en.md) for the Go static runner, which does not require Node at runtime, and deployment configuration. The [contribution guide](../CONTRIBUTING.en.md) describes end-to-end test requirements and commands.
