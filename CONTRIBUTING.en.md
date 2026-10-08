# Contributing

[简体中文](./CONTRIBUTING.md) | **English** · [Project home](./README.en.md)

Contributions to Halmify's code, documentation, and translations are welcome. For larger features or architectural changes, open an [issue](https://github.com/utopiazeng/Halmify/issues) describing the use case and design first. Small fixes can go directly to a pull request.

## Development workflow

1. Fork the repository and create a feature branch.
2. Follow the [installation guide](./docs/INSTALL.en.md) using separate local databases and services.
3. Keep the change focused on one problem and update affected Chinese and English documentation together.
4. Run relevant checks and describe the problem, resulting behavior, and validation in the pull request.

For bug reports, include reproduction steps, expected and actual behavior, your environment, and sanitized logs. Do not include passwords, tokens, API keys, or customer data. You can also discuss technical questions in the QQ group **halmify 技术交流1 (419223866)**.

## Checks

Start these command blocks from the repository root, then enter the relevant module:

```bash
cd ai-crm
go test ./...
```

```bash
cd fileserver
go test ./...
```

For frontend logic and build changes:

```bash
cd ai-crm-frontend
npm ci
npm run test:unit
npm run build -- --base /
```

For UI text changes, follow the [frontend translation workflow](./ai-crm-frontend/README.en.md) and run synchronization, static checks, and runtime checks. For documentation-only changes, check links, images, commands, and consistency between languages.

End-to-end tests require a running backend and configured test accounts. Some scenarios create or modify data, so use a separate test environment. Supply credentials through `E2E_USERNAME` and `E2E_PASSWORD`, and set `PLAYWRIGHT_CHROMIUM_PATH` to your Chrome executable. Scenario-specific permissions and messaging account requirements are documented in the corresponding files under `tests/e2e/`:

```bash
cd ai-crm-frontend
PLAYWRIGHT_CHROMIUM_PATH=/path/to/chrome npm run test:e2e -- 'tests/e2e/[^/]+\.spec\.js$'
```

The Playwright development server uses port `3000`; normal development uses `3001`. Production smoke tests have a separate configuration and are excluded from the command above.

## Configuration and commit contents

Commit example configuration such as `app.conf.example`. Root `.gitignore` rules exclude actual `conf/app.conf` files, `.env` files, database dumps, uploads, and test reports. Keep dependency manifests, lockfiles, and SQL source scripts.

Document each new setting's purpose, default, and environment variable support. Use `t(key)` for new interface text and keep Chinese and English project documentation aligned. Check `git status --short` before committing to avoid including local data.

## License

By submitting a contribution, you confirm that you have the right to contribute the content and agree to provide it under the project's [MIT License](./LICENSE). Identify the source and license of any third-party code or assets.
