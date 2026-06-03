package db

import (
	"sync"
	"testing"

	"github.com/LandcLi/landc-go/frame/pkg/config"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file::memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to create GORM instance: %v", err)
	}
	InitGlobalDBWithObject(db)
	return db
}

func cleanupTestDB(t *testing.T) {
	t.Helper()
	Close()
}

func TestInitGlobalDBWithObject(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t)

	retrieved := GetDB()
	if retrieved != db {
		t.Error("GetDB should return the same instance")
	}
}

func TestInitGlobalDBWithConfig(t *testing.T) {
	cfg := &config.DatabaseConfig{
		Driver: "sqlite",
		DSN:   "file::memory:?cache=shared",
	}

	err := InitGlobalDBWithConfig(cfg)
	if err != nil {
		t.Fatalf("InitGlobalDBWithConfig failed: %v", err)
	}
	defer cleanupTestDB(t)

	db := GetDB()
	if db == nil {
		t.Error("GetDB should not return nil")
	}
}

func TestInitGlobalDBWithDefault(t *testing.T) {
	t.Skip("Requires MySQL connection, skipping test")
}

func TestGetDBThreadSafety(t *testing.T) {
	setupTestDB(t)
	defer cleanupTestDB(t)

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			db := GetDB()
			if db == nil {
				t.Error("GetDB should not return nil in concurrent access")
			}
		}()
	}
	wg.Wait()
}

func TestClose(t *testing.T) {
	setupTestDB(t)

	err := Close()
	if err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	db := GetDB()
	if db != nil {
		t.Error("GetDB should return nil after Close")
	}
}

func TestAutoMigrate(t *testing.T) {
	setupTestDB(t)
	defer cleanupTestDB(t)

	type TestModel struct {
		ID   uint   `gorm:"primaryKey"`
		Name string `gorm:"size:255"`
	}

	err := AutoMigrate(&TestModel{})
	if err != nil {
		t.Fatalf("AutoMigrate failed: %v", err)
	}
}

func TestInitGlobalDBWithConfig_DuplicateInit(t *testing.T) {
	cfg := &config.DatabaseConfig{
		Driver: "sqlite",
		DSN:   "file::memory:?cache=shared",
	}

	err := InitGlobalDBWithConfig(cfg)
	if err != nil {
		t.Fatalf("First init failed: %v", err)
	}
	defer cleanupTestDB(t)

	err = InitGlobalDBWithConfig(cfg)
	if err != nil {
		t.Fatalf("Second init should not fail (should skip): %v", err)
	}
}

func TestInitGlobalDBWithDefault_NoConfig(t *testing.T) {
	t.Skip("Requires MySQL connection, skipping test")
}
