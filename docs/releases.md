# 发布流程

landc-go 为多 Module Monorepo，各模块可独立发版。本文档说明发布流程、tag 规范与回滚方案。

## Tag 规范

| 目标 | Tag 格式 | 示例 |
|---|---|---|
| 库模块（log/tools/api/frame/workflow/saas） | `<module>/vX.Y.Z` | `frame/v0.5.0` |
| 整体 / CLI | `vX.Y.Z` | `v0.5.0` |

- **严格语义化版本**：`vX.Y.Z`，无预发布/构建元数据后缀（`v0.5.0-rc1` 暂不支持，CI `validate-tag` 会拒绝）
- 版本号必须递增，遵循 [SemVer](https://semver.org)
- CI `release.yml` 的 `validate-tag` job 会校验：模块前缀白名单 + 严格 semver

## 发布顺序

依赖者后发（拓扑序）：

```
tools → log → api → frame → workflow → saas
```

即：先发被依赖的基础模块（tools/log），最后发依赖最多的模块（workflow/saas）。

## 发布前检查

运行预发布脚本（30 分钟内完成）：

```bash
GOLANGCI=/path/to/golangci-lint GOVULNCHECK=/path/to/govulncheck \
  bash scripts/release-check.sh
```

要求全部通过：
1. `git status` 干净
2. `make build && make vet && make test` 全绿
3. `golangci-lint run ./...` 0 error（v2.12.2）
4. `govulncheck ./...` 无 Critical/High
5. `go mod verify` 通过
6. go 指令全部 = 1.26
7. 无未关闭 release-blocker issue

## 发布步骤

1. 收敛 `CHANGELOG.md`：`[Unreleased]` → `[X.Y.Z] - YYYY-MM-DD`，分类 Added/Changed/Fixed/Security
2. 更新 `docs/dependencies.md` 依赖基线（如有变更）
3. 移除开发期 `replace` 指令（仅发布模块的 go.mod；本地开发仍可用 go.work）
4. 提交并推送 `dev` 分支
5. 按拓扑序打 tag 并推送：

```bash
git tag -a tools/v0.5.0 -m "tools v0.5.0"
git tag -a log/v0.5.0   -m "log v0.5.0"
# ... 依序 frame / workflow / saas
git push origin --tags
```

6. CI 自动执行 `release.yml`：validate-tag → test → 创建 GitHub Release（含 release notes）
7. 验证：
   - `go list -m -json github.com/LandcLi/landc-go/frame@v0.5.0` 可解析
   - 干净环境 `go get github.com/LandcLi/landc-go/frame@v0.5.0` 成功
   - pkg.go.dev 索引成功（发布后数小时）

## 回滚方案

| 场景 | 操作 |
|---|---|
| tag 打错（尚未被下游引用） | `git tag -d <tag> && git push origin :refs/tags/<tag>`（需仓库权限，尽快） |
| 已被下游引用但有严重 bug | 发布 **patch 版本**（`vX.Y.Z+1`）修复，**不删已发布的 tag**（Go module proxy 缓存不可删除） |
| CI 发布失败 | 修复 workflow 后重跑 job；若已创建 Release 则编辑补充 |

## 预发布版本策略

- v0.x 为预发布阶段，**不承诺 API 冻结**，minor 版本可含破坏性变更
- v1.0.0 前置条件：SubWorkflow 实现（或移除）、scheduler/lock 测试完备、下游 ≥2-4 周真实使用、API 冻结评审
