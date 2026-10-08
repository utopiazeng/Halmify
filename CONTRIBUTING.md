# 贡献指南

**简体中文** | [English](./CONTRIBUTING.en.md) · [项目首页](./README.md)

欢迎改进 Halmify 的代码、文档与翻译。较大的功能或架构调整请先通过 [Issue](https://github.com/utopiazeng/Halmify/issues) 说明场景和设计；小修复可直接提交 Pull Request。

## 开发流程

1. Fork 仓库并创建功能分支。
2. 按[安装指南](./docs/INSTALL.md)配置独立的本地数据库和服务。
3. 让修改聚焦于一个问题，并同步更新受影响的中英文文档。
4. 执行与修改相关的检查，在 Pull Request 中说明问题、修改后的行为及验证结果。

提交问题时请提供复现步骤、预期与实际行为、运行环境和脱敏后的日志。不要附带账号密码、令牌、API 密钥或客户数据。技术讨论也可加入 QQ 群 **halmify 技术交流1（419223866）**。

## 检查命令

以下命令从仓库根目录执行。后端检查可分别进入对应模块运行：

```bash
cd ai-crm
go test ./...
```

```bash
cd fileserver
go test ./...
```

前端逻辑和构建检查：

```bash
cd ai-crm-frontend
npm ci
npm run test:unit
npm run build -- --base /
```

修改界面文案时遵循[前端多语言流程](./ai-crm-frontend/README.md)，运行同步及静态、运行时检查。只改文档时，检查链接、图片、命令和中英文一致性即可。

端到端测试需要已启动的后端和配置好的测试账号；部分场景会创建或修改数据，应使用独立测试环境。通过 `E2E_USERNAME`、`E2E_PASSWORD` 提供账号，通过 `PLAYWRIGHT_CHROMIUM_PATH` 设置 Chrome 路径；不同场景的权限及即时消息账号要求见 `tests/e2e/` 中对应测试文件：

```bash
cd ai-crm-frontend
PLAYWRIGHT_CHROMIUM_PATH=/path/to/chrome npm run test:e2e -- 'tests/e2e/[^/]+\.spec\.js$'
```

Playwright 开发测试启动前端的端口是 `3000`，正常开发端口是 `3001`。生产冒烟测试有独立配置，不包含在上述命令中。

## 配置与提交内容

只提交 `app.conf.example` 等示例配置。实际的 `conf/app.conf`、`.env`、数据库备份、上传文件和测试报告由根目录 `.gitignore` 排除。依赖清单、锁文件及源代码中的 SQL 脚本应保留。

新增配置请记录用途、默认值和是否支持环境变量。新界面文案使用 `t(key)`；中文和英文项目文档保持对应。提交前检查 `git status --short`，避免将本机数据加入仓库。

## 许可证

提交贡献即表示你有权贡献相应内容，并同意将贡献按项目的 [MIT 许可证](./LICENSE) 提供。第三方代码或资源请注明来源及适用许可证。
