package db

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/LandcLi/landc-go/frame/pkg/config"
	"github.com/glebarez/sqlite"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// SQLite 默认库文件路径。未显式配置 dsn 时使用。
const defaultSQLiteDSN = "data/landc.db"

// sqliteDefaultPragmas 是嵌入式数据库的推荐参数：
//   - busy_timeout  : 写锁竞争时最多等待 5s 而不是立刻报 database is locked
//   - journal_mode  : WAL 提升并发读性能（多读单写）
//   - foreign_keys  : SQLite 默认关闭外键约束，显式打开以保证与 MySQL 行为一致
const sqliteDefaultPragmas = "?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)&_pragma=foreign_keys(1)"

func getMySQLDialector(cfg *config.DatabaseConfig) (gorm.Dialector, error) {
	return mysql.Open(cfg.DSN), nil
}

func getPostgresDialector(cfg *config.DatabaseConfig) (gorm.Dialector, error) {
	return postgres.Open(cfg.DSN), nil
}

// getSQLiteDialector 返回纯 Go 实现的 SQLite dialector（github.com/glebarez/sqlite，
// 基于 modernc.org/sqlite），不依赖 CGO，可直接交叉编译与在无 gcc 环境运行。
func getSQLiteDialector(cfg *config.DatabaseConfig) (gorm.Dialector, error) {
	dsn, err := normalizeSQLiteDSN(cfg.DSN)
	if err != nil {
		return nil, err
	}
	return sqlite.Open(dsn), nil
}

// normalizeSQLiteDSN 规范化 SQLite 连接串：
//   - 为空时回落到 defaultSQLiteDSN；
//   - 文件型数据库自动创建父目录（否则 modernc 会报 unable to open database file）；
//   - 自动追加推荐的 pragma 参数，调用方重复配置时以调用方为准（只补缺失项）。
func normalizeSQLiteDSN(raw string) (string, error) {
	dsn := strings.TrimSpace(raw)
	if dsn == "" {
		dsn = defaultSQLiteDSN
	}

	// 内存库与显式 URI 不做目录处理
	base, query, hasQuery := strings.Cut(dsn, "?")
	if !isMemorySQLite(base) {
		if dir := filepath.Dir(base); dir != "" && dir != "." && dir != string(filepath.Separator) {
			if err := os.MkdirAll(dir, 0o755); err != nil {
				return "", fmt.Errorf("failed to create sqlite directory %q: %w", dir, err)
			}
		}
	}

	if hasQuery {
		// 已有参数：仅补齐缺失的 pragma。
		// 注意 pragma 以函数名区分（busy_timeout / journal_mode / foreign_keys），
		// 不能用 "_pragma=" 作为 key，否则命中一个就会把其余全部跳过。
		existing := sqlitePragmaNames(query)
		for _, p := range strings.Split(strings.TrimPrefix(sqliteDefaultPragmas, "?"), "&") {
			name := sqlitePragmaName(p)
			if name == "" || existing[name] {
				continue
			}
			query += "&" + p
			existing[name] = true
		}
		return base + "?" + query, nil
	}

	return base + sqliteDefaultPragmas, nil
}

// sqlitePragmaName 从 "_pragma=busy_timeout(5000)" 中提取 "busy_timeout"。
func sqlitePragmaName(entry string) string {
	value, ok := strings.CutPrefix(entry, "_pragma=")
	if !ok {
		return ""
	}
	name, _, _ := strings.Cut(value, "(")
	return strings.TrimSpace(name)
}

// sqlitePragmaNames 收集 query 中已声明的 pragma 名称集合。
func sqlitePragmaNames(query string) map[string]bool {
	names := make(map[string]bool)
	for _, part := range strings.Split(query, "&") {
		if name := sqlitePragmaName(part); name != "" {
			names[name] = true
		}
	}
	return names
}

// isMemorySQLite 判断是否为内存/临时库。
func isMemorySQLite(dsn string) bool {
	return strings.Contains(dsn, ":memory:") ||
		strings.HasPrefix(dsn, "file::memory:") ||
		strings.Contains(dsn, "mode=memory")
}
