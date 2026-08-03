# 性能基准

记录关键路径的性能基线，用于发布后的回归对比。

> 最后更新：2026-08-03
> 环境：macOS（Apple Silicon，12 核），Go 1.26.5
> 命令：`go test -bench=. -benchtime=1000x -run='^$' <pkg>`

## JWT（`frame/pkg/auth`）

| Benchmark | 耗时/op | 说明 |
|---|---|---|
| `BenchmarkGenerateTokenHS256` | ~5.2 µs | HMAC-SHA256 签发 |
| `BenchmarkParseTokenHS256` | ~7.5 µs | HMAC-SHA256 校验 |
| `BenchmarkGenerateTokenRS256` | ~838 µs | RSA-2048 签名（非对称开销明显） |

## 内存 Store（`workflow/pkg/store`）

| Benchmark | 耗时/op | 说明 |
|---|---|---|
| `BenchmarkMemoryStoreExecutionWriteRead` | ~649 ns | 执行写+读往返 |
| `BenchmarkMemoryStoreTaskWrite` | ~333 ns | 任务写入 |

## saas 租户 Scope 查询（`saas/pkg`，SQLite 内存库，100 条数据）

| Benchmark | 耗时/op | 说明 |
|---|---|---|
| `BenchmarkTenantScopeQuery` | ~57 µs | TenantScope 子查询 + 业务表联查 |

## 说明

- 以上为**开发机绝对数值**，仅用于**相对回归**（同机、同基准命令对比），不跨机器比较绝对值
- 复现：`go test -bench=. -benchtime=1000x -run='^$' ./pkg/auth/`（workflow 用 `./pkg/store/`，saas 用 `./pkg/`）
- 如需正式对比，用 `benchstat` 对多轮结果做统计
