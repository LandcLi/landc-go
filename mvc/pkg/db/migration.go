package db

import (
	"fmt"
	"sort"
	"time"

	"gorm.io/gorm"
)

// MigrationRecord 迁移记录（存储在数据库中）
type MigrationRecord struct {
	ID        uint      `gorm:"primarykey"`
	Version   string    `gorm:"uniqueIndex;size:255;not null"`
	Name      string    `gorm:"size:255;not null"`
	AppliedAt time.Time `gorm:"not null"`
}

func (MigrationRecord) TableName() string {
	return "schema_migrations"
}

// Migration 定义一次迁移
type Migration struct {
	Version string // 版本号（建议格式：20260525_001）
	Name    string // 迁移名称描述
	Up      func(tx *gorm.DB) error
	Down    func(tx *gorm.DB) error
}

// Migrator 迁移管理器
type Migrator struct {
	db         *gorm.DB
	migrations []*Migration
}

// NewMigrator 创建迁移管理器
func NewMigrator(db *gorm.DB) *Migrator {
	return &Migrator{
		db:         db,
		migrations: make([]*Migration, 0),
	}
}

// NewMigratorFromGlobal 使用全局 DB 创建迁移管理器
func NewMigratorFromGlobal() (*Migrator, error) {
	db := GetDB()
	if db == nil {
		return nil, fmt.Errorf("global database not initialized")
	}
	return NewMigrator(db), nil
}

// Register 注册迁移
func (m *Migrator) Register(migrations ...*Migration) {
	m.migrations = append(m.migrations, migrations...)
}

// Init 初始化迁移表
func (m *Migrator) Init() error {
	return m.db.AutoMigrate(&MigrationRecord{})
}

// Up 执行所有未应用的迁移
func (m *Migrator) Up() error {
	if err := m.Init(); err != nil {
		return fmt.Errorf("failed to init migration table: %w", err)
	}

	applied, err := m.getAppliedVersions()
	if err != nil {
		return err
	}

	pending := m.getPendingMigrations(applied)
	if len(pending) == 0 {
		return nil
	}

	for _, migration := range pending {
		if err := m.applyUp(migration); err != nil {
			return fmt.Errorf("migration %s (%s) failed: %w", migration.Version, migration.Name, err)
		}
	}

	return nil
}

// UpTo 执行到指定版本
func (m *Migrator) UpTo(version string) error {
	if err := m.Init(); err != nil {
		return fmt.Errorf("failed to init migration table: %w", err)
	}

	applied, err := m.getAppliedVersions()
	if err != nil {
		return err
	}

	pending := m.getPendingMigrations(applied)
	for _, migration := range pending {
		if err := m.applyUp(migration); err != nil {
			return fmt.Errorf("migration %s (%s) failed: %w", migration.Version, migration.Name, err)
		}
		if migration.Version == version {
			break
		}
	}

	return nil
}

// Down 回退最近 n 次迁移
func (m *Migrator) Down(steps int) error {
	if err := m.Init(); err != nil {
		return fmt.Errorf("failed to init migration table: %w", err)
	}

	if steps <= 0 {
		steps = 1
	}

	applied, err := m.getAppliedRecords()
	if err != nil {
		return err
	}

	// 按版本号倒序
	sort.Slice(applied, func(i, j int) bool {
		return applied[i].Version > applied[j].Version
	})

	count := 0
	for _, record := range applied {
		if count >= steps {
			break
		}

		migration := m.findMigration(record.Version)
		if migration == nil {
			return fmt.Errorf("migration %s not found in registered migrations", record.Version)
		}

		if migration.Down == nil {
			return fmt.Errorf("migration %s (%s) has no Down function", migration.Version, migration.Name)
		}

		if err := m.applyDown(migration); err != nil {
			return fmt.Errorf("rollback %s (%s) failed: %w", migration.Version, migration.Name, err)
		}

		count++
	}

	return nil
}

// Reset 回退所有迁移
func (m *Migrator) Reset() error {
	if err := m.Init(); err != nil {
		return fmt.Errorf("failed to init migration table: %w", err)
	}

	applied, err := m.getAppliedRecords()
	if err != nil {
		return err
	}

	return m.Down(len(applied))
}

// Status 获取迁移状态
func (m *Migrator) Status() ([]*MigrationStatus, error) {
	if err := m.Init(); err != nil {
		return nil, fmt.Errorf("failed to init migration table: %w", err)
	}

	applied, err := m.getAppliedVersions()
	if err != nil {
		return nil, err
	}

	records, err := m.getAppliedRecords()
	if err != nil {
		return nil, err
	}

	recordMap := make(map[string]*MigrationRecord)
	for _, r := range records {
		recordMap[r.Version] = &r
	}

	sorted := m.getSortedMigrations()
	statuses := make([]*MigrationStatus, 0, len(sorted))

	for _, migration := range sorted {
		status := &MigrationStatus{
			Version: migration.Version,
			Name:    migration.Name,
			Applied: applied[migration.Version],
		}
		if record, ok := recordMap[migration.Version]; ok {
			status.AppliedAt = &record.AppliedAt
		}
		statuses = append(statuses, status)
	}

	return statuses, nil
}

// Pending 获取待执行的迁移数量
func (m *Migrator) Pending() (int, error) {
	if err := m.Init(); err != nil {
		return 0, err
	}
	applied, err := m.getAppliedVersions()
	if err != nil {
		return 0, err
	}
	pending := m.getPendingMigrations(applied)
	return len(pending), nil
}

// MigrationStatus 迁移状态
type MigrationStatus struct {
	Version   string
	Name      string
	Applied   bool
	AppliedAt *time.Time
}

func (m *Migrator) applyUp(migration *Migration) error {
	return m.db.Transaction(func(tx *gorm.DB) error {
		if err := migration.Up(tx); err != nil {
			return err
		}

		record := MigrationRecord{
			Version:   migration.Version,
			Name:      migration.Name,
			AppliedAt: time.Now(),
		}
		return tx.Create(&record).Error
	})
}

func (m *Migrator) applyDown(migration *Migration) error {
	return m.db.Transaction(func(tx *gorm.DB) error {
		if err := migration.Down(tx); err != nil {
			return err
		}

		return tx.Where("version = ?", migration.Version).Delete(&MigrationRecord{}).Error
	})
}

func (m *Migrator) getAppliedVersions() (map[string]bool, error) {
	var records []MigrationRecord
	if err := m.db.Find(&records).Error; err != nil {
		return nil, fmt.Errorf("failed to query migration records: %w", err)
	}

	applied := make(map[string]bool)
	for _, r := range records {
		applied[r.Version] = true
	}
	return applied, nil
}

func (m *Migrator) getAppliedRecords() ([]MigrationRecord, error) {
	var records []MigrationRecord
	if err := m.db.Order("version DESC").Find(&records).Error; err != nil {
		return nil, fmt.Errorf("failed to query migration records: %w", err)
	}
	return records, nil
}

func (m *Migrator) getPendingMigrations(applied map[string]bool) []*Migration {
	sorted := m.getSortedMigrations()
	var pending []*Migration
	for _, migration := range sorted {
		if !applied[migration.Version] {
			pending = append(pending, migration)
		}
	}
	return pending
}

func (m *Migrator) getSortedMigrations() []*Migration {
	sorted := make([]*Migration, len(m.migrations))
	copy(sorted, m.migrations)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Version < sorted[j].Version
	})
	return sorted
}

func (m *Migrator) findMigration(version string) *Migration {
	for _, migration := range m.migrations {
		if migration.Version == version {
			return migration
		}
	}
	return nil
}
