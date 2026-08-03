# 依赖基线

记录 landc-go 各模块的关键依赖版本（作为发布与后续升级的比对基准）。

> 最后更新：2026-08-03（Go 1.24 → 1.26 安全升级后）
> 生成方式：`go list -m all` 各模块核对，与本仓库 `go.mod`/`go.sum` 保持一致。

## Go 版本

| 项 | 值 |
|---|---|
| go 指令 | `1.26`（所有 go.mod + go.work 统一） |
| toolchain | `go1.26.5`（CI `setup-go@v5` 固定 `go-version: "1.26"`） |
| 说明 | go 1.24 已于 2026-02-10 停止安全支持，标准库漏洞修复需 go1.25.9+ |

## 核心依赖

| 依赖 | 版本 | 主要使用者 |
|---|---|---|
| `google.golang.org/grpc` | v1.82.1 | frame |
| `github.com/dop251/goja` | v0.0.0-20240816181238-8130cadc5774 | workflow（JS 执行器） |
| `gorm.io/gorm` | v1.31.1 | frame / saas / workflow |
| `gorm.io/driver/mysql` | v1.6.0 | frame（可选） |
| `gorm.io/driver/postgres` | v1.6.0 | frame（可选） |
| `gorm.io/driver/sqlite` | v1.6.0 | frame（测试）/ saas / workflow |
| `github.com/gin-gonic/gin` | v1.11.0 | frame / api |
| `go.uber.org/zap` | v1.26.0 | log |
| `github.com/sirupsen/logrus` | v1.9.3 | log |
| `github.com/redis/go-redis/v9` | v9.19.0 | workflow（可选） |
| `github.com/jackc/pgx/v5` | v5.9.2 | frame（间接，postgres 驱动） |
| `github.com/quic-go/quic-go` | v0.59.1 | frame（间接，gin→http3） |
| `golang.org/x/text` | v0.39.0 | 多模块（间接） |
| `golang.org/x/sys` | v0.46.0 | 多模块（间接） |

## 安全工具链

| 工具 | 版本 | 说明 |
|---|---|---|
| golangci-lint | v2.12.2 | CI `lint` job；`.golangci.yml` 为 v2 格式（`linters.default: none` 白名单语义） |
| gosec | v2.22.2 | CI `security` job（`-severity high`） |
| govulncheck | v1.6.0 | CI `security` job；当前 6 模块 0 漏洞 |

## 版本升级门槛

- 依赖 `go` 指令 ≤ 1.26：可自动合入 minor/patch
- 依赖 `go` 指令 > 1.26：走人工 review，核对 toolchain 兼容性
- **历史教训**：grpc v1.83 / goja 2026-06 / otel v1.44 均要求更高 go 版本导致 toolchain 漂移（经历 1.25→1.24 回退、1.24→1.26 升级两次），升级依赖时务必核对 go 指令
