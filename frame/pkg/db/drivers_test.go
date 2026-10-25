package db

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/LandcLi/landc-go/frame/pkg/config"
	"gorm.io/gorm"
)

func TestNormalizeSQLiteDSN_Default(t *testing.T) {
	dsn, err := normalizeSQLiteDSN("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.HasPrefix(dsn, defaultSQLiteDSN) {
		t.Errorf("expected default dsn prefix %q, got %q", defaultSQLiteDSN, dsn)
	}
	if !strings.Contains(dsn, "busy_timeout") || !strings.Contains(dsn, "journal_mode(WAL)") {
		t.Errorf("expected default pragmas appended, got %q", dsn)
	}
}

func TestNormalizeSQLiteDSN_CreatesParentDir(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "nested", "deep", "sso.db")

	dsn, err := normalizeSQLiteDSN(target)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := os.Stat(filepath.Dir(target)); err != nil {
		t.Fatalf("expected parent dir created: %v", err)
	}
	if !strings.HasPrefix(dsn, filepath.ToSlash(target)) && !strings.HasPrefix(dsn, target) {
		t.Errorf("dsn should keep file path, got %q", dsn)
	}
}

func TestNormalizeSQLiteDSN_MemoryUntouched(t *testing.T) {
	dsn, err := normalizeSQLiteDSN("file::memory:?cache=shared")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(dsn, "cache=shared") {
		t.Errorf("existing query must be preserved, got %q", dsn)
	}
	if !strings.Contains(dsn, "busy_timeout") {
		t.Errorf("missing pragmas should be appended, got %q", dsn)
	}
}

func TestNormalizeSQLiteDSN_KeepsCallerPragma(t *testing.T) {
	target := filepath.Join(t.TempDir(), "x.db")
	dsn, err := normalizeSQLiteDSN(target + "?_pragma=busy_timeout(1000)")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.Count(dsn, "busy_timeout") != 1 {
		t.Errorf("caller pragma must not be duplicated, got %q", dsn)
	}
	if !strings.Contains(dsn, "journal_mode(WAL)") {
		t.Errorf("missing pragmas should still be appended, got %q", dsn)
	}
}

// TestSQLiteDialectorWorksWithoutCGO 验证纯 Go SQLite 驱动在 CGO_ENABLED=0 下可用。
// 这是框架级关键能力：无 gcc 环境也必须能跑起嵌入式数据库。
func TestSQLiteDialectorWorksWithoutCGO(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "sub", "runtime.db")
	dialector, err := getSQLiteDialector(&config.DatabaseConfig{Driver: "sqlite", DSN: dbPath})
	if err != nil {
		t.Fatalf("getSQLiteDialector failed: %v", err)
	}

	db, err := gorm.Open(dialector, &gorm.Config{})
	if err != nil {
		t.Fatalf("gorm.Open failed: %v", err)
	}
	defer func() {
		if sqlDB, err := db.DB(); err == nil {
			_ = sqlDB.Close()
		}
	}()

	type probe struct {
		ID   uint
		Name string
	}
	if err := db.AutoMigrate(&probe{}); err != nil {
		t.Fatalf("AutoMigrate failed: %v", err)
	}
	if err := db.Create(&probe{Name: "ok"}).Error; err != nil {
		t.Fatalf("insert failed: %v", err)
	}

	var got probe
	if err := db.First(&got).Error; err != nil {
		t.Fatalf("query failed: %v", err)
	}
	if got.Name != "ok" {
		t.Errorf("roundtrip mismatch: %+v", got)
	}

	// WAL 生效时会在库文件旁生成 -wal 文件
	if _, err := os.Stat(dbPath + "-wal"); err != nil {
		t.Logf("warning: -wal file not found (%v), journal_mode may not be WAL", err)
	}
}
