# Halmify CRM

[简体中文](./README.md) | **English**

<img src="./ai-crm-frontend/public/halmify-logo-horizontal.svg" alt="Halmify CRM" width="320">

Halmify is an open-source CRM built with Vue 3 and Go for teams that want to host and extend their customer management and sales workflows. It includes multi-tenant management, CRM workflows, AI analysis, instant messaging, and manufacturing and IoT extensions.

[Installation and configuration](./docs/INSTALL.en.md) · [Contributing](./CONTRIBUTING.en.md) · [Report an issue](https://github.com/utopiazeng/Halmify/issues) · [Community](#technical-community) · [Support the project](./docs/DONATE.md)

## Features

| Capability | Description |
| --- | --- |
| CRM workflows | Workspace, companies and contacts, leads, opportunities, deals, orders, products, projects, campaigns, and tickets |
| Tenants and permissions | Platform administration, tenants and plans, tenant administrators, roles, object access control, and audit logs |
| AI analysis | Company research, tenant analysis jobs, provider and model settings, and usage records; an external AI service must be configured |
| Collaboration | Instant messaging, teams, approvals, reminders, and sales dashboards |
| File management | Dedicated file service, signed internal APIs, tenant usage tracking, and an upload/download gateway |
| Manufacturing | Customer profiles, RFQs and quotations, BOMs and costs, sample projects, production progress, and after-sales workflows |
| IoT | Device management, device APIs, firmware releases, and OTA updates |
| Multilingual interface | 12 languages, including Simplified Chinese, Traditional Chinese, and English; see the frontend guide for the translation workflow |

Available modules depend on account permissions, tenant settings, and enabled industry packages. The MIT license permits hosting, modification, and commercial use under its terms.

## Technology and structure

| Component | Technology | Purpose |
| --- | --- | --- |
| `ai-crm-frontend/` | Vue 3, Vue Router, Element Plus, Vite | Browser interface, development port `3001` |
| `ai-crm/` | Go, Beego, GORM | CRM APIs, tenants, and business logic, port `8092` |
| `fileserver/` | Go, Gin, GORM | File storage and signed internal APIs, port `18080` |

MySQL stores business data and file metadata. Redis supports login security, caching, real-time collaboration, and distributed job locks. The frontend calls the CRM backend through `/api/`; the backend sends signed requests to the file service.

```text
Halmify/
├── ai-crm-frontend/     # Web interface and frontend tests
├── ai-crm/              # CRM service and database migrations
├── fileserver/          # Dedicated file service
├── docs/                # Installation and donation guides
├── CONTRIBUTING.md      # Contribution guide
├── alipay.jpg           # Alipay donation QR code
└── LICENSE              # MIT license
```

## Getting started

Install Go 1.22 or later, Node.js 20, and npm. MySQL 8 and Redis 7 are recommended.

```bash
git clone https://github.com/utopiazeng/Halmify.git
cd Halmify
```

Follow the [installation guide](./docs/INSTALL.en.md) to create two empty databases, copy and fill in the example configuration files, and start the file service, CRM backend, and frontend in that order. The CRM runs database migrations on startup; the file service requires a separate migration command first.

The development interface is available at `http://localhost:3001/`. On a fresh installation, create a platform administrator through the registration API, then use that account to create tenants and tenant administrators. The installation guide includes the commands and steps.

## Documentation

- [Installation, configuration, and initialization](./docs/INSTALL.en.md)
- [Frontend development and translations](./ai-crm-frontend/README.en.md)
- [Frontend production build and deployment](./ai-crm-frontend/BUILD.en.md)
- [File service configuration and migrations](./fileserver/README.en.md)
- [IoT device API (Chinese technical reference)](./ai-crm/IOT_DEVICE_API.md)
- [IoT OTA updates (Chinese technical reference)](./ai-crm/IOT_OTA_UPGRADE.md)
- [Contributing](./CONTRIBUTING.en.md)

## Contributing

Use [Issues](https://github.com/utopiazeng/Halmify/issues) for bug reports and suggestions, and [Pull Requests](https://github.com/utopiazeng/Halmify/pulls) for code, translation, and documentation improvements. Read the [contribution guide](./CONTRIBUTING.en.md) before getting started.

## Technical community

Join the QQ group **halmify 技术交流1** (Halmify Technical Discussion 1) to discuss deployment, development, features, and technical questions.

**QQ group number: 419223866**. Search for this group number in QQ and request to join.

For bugs or feature requests that need tracking, also open a [GitHub issue](https://github.com/utopiazeng/Halmify/issues) so the discussion and resolution remain easy to follow.

## Support and donations

If Halmify helps you, consider starring or sharing the project, contributing, or making a voluntary donation through Alipay to support maintenance. You choose the amount.

<img src="./alipay.jpg" alt="Alipay donation QR code; scan with the Alipay app" width="240">

Scan with Alipay, or save the image and select it from the app's photo scanner. [Full-size image](./alipay.jpg) · [Bilingual donation guide](./docs/DONATE.md)

## License

Licensed under the [MIT License](./LICENSE), Copyright (c) 2026 Utopia Zeng. Retain the copyright and permission notices required by the license when using or distributing the project. Third-party dependencies retain their own licenses.
