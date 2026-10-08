# Halmify 前端

**简体中文** | [English](./README.en.md) · [项目首页](../README.md)

基于 Vue 3、Vue Router、Element Plus 和 Vite 的 Web 界面。完整环境初始化见[安装指南](../docs/INSTALL.md)，生产运行见[构建与部署指南](./BUILD.md)。

## 本地开发

从仓库根目录进入前端目录：

```bash
cd ai-crm-frontend
npm ci
npm run dev
```

需要 Node.js 20。开发服务器监听 `3001`，通过 `/api` 将 HTTP 和 WebSocket 请求代理到后端 `http://localhost:8092`。

品牌名称和标语默认来自 `src/brand.js`。如需自定义，可复制 `.env.example` 为 `.env` 并修改 `VITE_BRAND_NAME`、`VITE_BRAND_TAGLINE`，然后重启开发服务器或重新构建。`VITE_` 变量会进入浏览器代码，不应包含密钥。

## 多语言开发流程

界面文案的中文原文维护在 `src/translate.md` 的 `messages.zh-CN` 中；同步脚本维护生成文件并补齐其他 11 种语言。

- 新增或修改界面文案时使用 `t(key)`，不要直接在模板或脚本里写中文。
- 中文原文以 `src/translate.md` 为准，不要手工编辑生成文件。
- 修改后运行 `npm run i18n:sync`。脚本可能调用外部翻译服务；失败时输出 pending 报告并使检查失败。
- 菜单、按钮、状态标签使用短文案，长说明放入提示或帮助文本。

在前端目录执行：

```bash
npm run i18n:sync
npm run i18n:sync:check
npm run i18n:scan:ci
npm run i18n:runtime:ci
npm run build
```

`i18n:sync:check` 只检查，不调用翻译接口；同步与检查脚本通过 npm 使用 Node 20，首次执行可能需要网络下载。界面支持简体中文、繁体中文、英语、西班牙语、法语、德语、日语、韩语、葡萄牙语、阿拉伯语、俄语和意大利语。

## 测试与构建

```bash
npm run test:unit
npm run build
```

默认生产构建使用 `/demo/` 路径；部署到域名根路径时使用 `npm run build -- --base /`。无需 Node 的 Go 静态运行器及部署配置见[BUILD.md](./BUILD.md)。端到端测试环境和命令见[贡献指南](../CONTRIBUTING.md)。
