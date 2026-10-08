# Halmify CRM

**简体中文** | [English](./README.en.md)

<img src="./ai-crm-frontend/public/halmify-logo-horizontal.svg" alt="Halmify CRM" width="320">

Halmify 是采用 Vue 3 和 Go 构建的开源 CRM，面向希望自行部署、扩展客户管理与销售协作流程的团队。项目包含多租户管理、CRM 业务流程、AI 分析、即时消息，以及制造业与 IoT 扩展。

[安装与配置](./docs/INSTALL.md) · [贡献指南](./CONTRIBUTING.md) · [报告问题](https://github.com/utopiazeng/Halmify/issues) · [技术交流群](#技术交流群) · [支持项目](./docs/DONATE.md)

## 主要功能

| 能力 | 说明 |
| --- | --- |
| CRM 业务 | 工作台、企业与联系人、线索、商机、交易、订单、产品、项目、市场活动和工单 |
| 多租户与权限 | 平台管理、租户与套餐管理、企业管理员、角色权限、对象访问控制和操作日志 |
| AI 分析 | 企业研究、租户分析任务、模型与服务商配置、用量记录；使用时需配置外部 AI 服务 |
| 团队协作 | 即时消息、团队管理、审批流程、提醒和销售看板 |
| 文件管理 | 独立文件服务、签名内部接口、租户用量管理和上传下载网关 |
| 制造业扩展 | 客户画像、询报价、BOM 与成本、样品项目、生产进度和售后流程 |
| IoT 扩展 | 设备管理、设备接口、固件发布和 OTA 升级 |
| 多语言界面 | 简体中文、繁体中文、英语等 12 种语言；文案维护流程见前端说明 |

具体可用模块取决于账号权限、租户配置及已启用的行业包。项目采用 MIT 许可证，允许依照许可证自行部署、修改和商业使用。

## 技术栈与结构

| 子项目 | 技术 | 用途 |
| --- | --- | --- |
| `ai-crm-frontend/` | Vue 3、Vue Router、Element Plus、Vite | 浏览器界面，开发端口 `3001` |
| `ai-crm/` | Go、Beego、GORM | CRM API、多租户与业务逻辑，端口 `8092` |
| `fileserver/` | Go、Gin、GORM | 文件存储和内部签名接口，端口 `18080` |

MySQL 存储业务与文件元数据；Redis 用于登录安全、缓存、实时协作和分布式任务锁。前端通过 `/api/` 访问 CRM 后端，后端通过签名请求访问文件服务。

```text
Halmify/
├── ai-crm-frontend/     # Web 界面与前端测试
├── ai-crm/              # CRM 服务与数据库迁移
├── fileserver/          # 独立文件服务
├── docs/                # 安装配置与捐赠说明
├── CONTRIBUTING.md      # 贡献指南
├── alipay.jpg           # 支付宝捐赠二维码
└── LICENSE              # MIT 许可证
```

## 开始使用

准备 Go 1.22 或更高版本、Node.js 20 与 npm，推荐使用 MySQL 8 和 Redis 7。

```bash
git clone https://github.com/utopiazeng/Halmify.git
cd Halmify
```

按照[安装与配置指南](./docs/INSTALL.md)创建两个空数据库，复制并填写示例配置，依次启动文件服务、CRM 后端和前端。CRM 会执行数据库迁移，文件服务需先单独执行迁移。

开发界面地址为 `http://localhost:3001/`。首次安装通过注册 API 创建平台管理员，再由平台管理员创建租户和企业管理员。详细步骤和命令见安装指南。

## 文档

- [安装、配置与首次初始化](./docs/INSTALL.md)
- [前端开发与多语言维护](./ai-crm-frontend/README.md)
- [前端生产构建与部署](./ai-crm-frontend/BUILD.md)
- [文件服务配置与迁移](./fileserver/README.md)
- [IoT 设备 API（中文技术参考）](./ai-crm/IOT_DEVICE_API.md)
- [IoT OTA 升级（中文技术参考）](./ai-crm/IOT_OTA_UPGRADE.md)
- [贡献指南](./CONTRIBUTING.md)

## 参与贡献

欢迎通过 [Issues](https://github.com/utopiazeng/Halmify/issues) 提交问题和建议，通过 [Pull Requests](https://github.com/utopiazeng/Halmify/pulls) 改进代码、翻译和文档。开始前请阅读[贡献指南](./CONTRIBUTING.md)。

## 技术交流群

欢迎加入 QQ 群 **halmify 技术交流1**，交流项目部署、二次开发、功能使用和技术问题。

**QQ群号：419223866**。打开 QQ 搜索群号申请加入。

需要跟踪的问题或功能建议，请同时提交到 [GitHub Issues](https://github.com/utopiazeng/Halmify/issues)，方便后续讨论和处理。

## 支持与捐赠

如果 Halmify 对你有帮助，欢迎 Star、分享项目或参与贡献，也可以通过支付宝自愿捐赠，支持项目维护。金额由你自行决定。

<img src="./alipay.jpg" alt="支付宝捐赠二维码，使用支付宝扫一扫" width="240">

打开支付宝扫一扫，或保存图片后从支付宝相册识别。[查看原图](./alipay.jpg) · [双语捐赠说明](./docs/DONATE.md)

## 许可证

本项目使用 [MIT License](./LICENSE)，Copyright (c) 2026 Utopia Zeng。使用或分发时请保留许可证要求的版权和许可声明。第三方依赖遵循各自的许可证。
