# Contributing to landc-go

感谢您对 **landc-go** 的兴趣！我们欢迎各种形式的贡献，包括但不限于提交 Bug 报告、功能建议、代码贡献、文档改进等。

## 目录

- [行为准则](#行为准则)
- [如何开始](#如何开始)
- [开发环境搭建](#开发环境搭建)
- [代码规范](#代码规范)
- [提交 PR](#提交-pr)
- [提交信息规范](#提交信息规范)
- [报告问题](#报告问题)

## 行为准则

本项目采用了 [Contributor Covenant 行为准则](CODE_OF_CONDUCT.md)。请确保您的行为符合该准则。简而言之：**善待彼此**。

## 如何开始

1. **Fork** 本仓库
2. 创建您的特性分支 (`git checkout -b feat/amazing-feature`)
3. 提交您的变更 (`git commit -m 'feat: add amazing feature'`)
4. 推送分支 (`git push origin feat/amazing-feature`)
5. 打开一个 **Pull Request**

## 开发环境搭建

### 前置条件

- Go 1.24+（通过 `go version` 检查）
- Make（通过 `make --version` 检查）
- golangci-lint（可选，用于本地 lint）

### 快速启动

```bash
# 克隆项目
git clone https://github.com/LandcLi/landc-go.git
cd landc-go

# 使用 go.work 进行多模块开发
go work use ./api ./frame ./log ./tools ./workflow ./saas

# 运行所有测试
make test

# 格式化代码
make fmt

# 运行 lint
make lint
```

### 常用命令

| 命令 | 说明 |
|------|------|
| `make test` | 运行所有模块测试（含竞态检测） |
| `make fmt` | 格式化所有 Go 代码 |
| `make vet` | 静态检查所有模块 |
| `make lint` | 运行 golangci-lint |
| `make build` | 编译所有模块 |
| `make tidy` | 清理各模块 go.mod |
| `make coverage` | 显示测试覆盖率 |

## 代码规范

- **Go 版本**: 1.24
- **格式化**: 所有代码必须通过 `go fmt`
- **导入分组**: 标准库 → 第三方库 → 内部库，每组之间空行分隔
- **注释**: 所有导出类型、函数、常量必须有 Go 风格的注释
- **错误处理**: 不要忽略错误（避免 `_ =`），使用项目统一的错误码体系
- **测试**: 新功能必须有对应的测试文件，测试覆盖率达到 70%+
- **并发安全**: 共享资源使用 `sync.RWMutex` 或 channel 保护

### Lint 检查

项目使用 `golangci-lint`，配置见 `.golangci.yml`。提交前请确保：

```bash
golangci-lint run ./...
```

## 提交 PR

1. 确保 PR 标题清晰描述变更内容
2. 在 PR 描述中说明 **为什么** 要做这个变更
3. 如果 PR 修复了某个 Issue，请引用它：`Fixes #123`
4. 确保 CI（lint + test + build）全部通过
5. Review 后至少需要一位维护者批准才能合并

### PR 标题格式

```
<type>(<scope>): <description>
```

例如：
- `feat(frame): add Redis Cluster support`
- `fix(tools): correct UUID v7 generation`
- `docs(log): update README with new API`
- `test(workflow): add DAG execution tests`
- `chore: update dependencies`

## 提交信息规范

我们采用 [Conventional Commits](https://www.conventionalcommits.org/) 规范：

```
<type>[optional scope]: <description>

[optional body]

[optional footer(s)]
```

**type 类型**：
- `feat` — 新功能
- `fix` — Bug 修复
- `docs` — 文档变更
- `style` — 代码格式（不影响功能）
- `refactor` — 重构（既不修复 Bug 也不添加功能）
- `test` — 添加或修改测试
- `chore` — 构建过程或辅助工具变更

**scope 范围** — 对应子模块名：`api`, `frame`, `log`, `tools`, `workflow`, `saas`

## 报告问题

- **Bug 报告**: 请使用 GitHub Issues 的 Bug 报告模板，包含复现步骤、预期行为、实际行为、环境信息
- **功能请求**: 请使用 Feature Request 模板，描述使用场景和期望行为
- **安全问题**: 请**不要**公开提交 Issue，参考 [SECURITY.md](SECURITY.md)

## 分支策略

- `main` — 稳定版本，保持随时可发布状态
- `develop` — 开发分支，接收特性合并
- `feat/*` — 特性分支，从 develop 创建
- `fix/*` — 修复分支
- `docs/*` — 文档分支

## 许可

通过贡献代码，您同意您的贡献将按照项目的 [MIT 许可证](LICENSE) 进行许可。
