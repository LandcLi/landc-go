package db

import (
	"github.com/LandcLi/landc-go/frame/pkg/config"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func getMySQLDialector(cfg *config.DatabaseConfig) (gorm.Dialector, error) {
	return mysql.Open(cfg.DSN), nil
}

func getPostgresDialector(cfg *config.DatabaseConfig) (gorm.Dialector, error) {
	return postgres.Open(cfg.DSN), nil
}

func getSQLiteDialector(cfg *config.DatabaseConfig) (gorm.Dialector, error) {
	return sqlite.Open(cfg.DSN), nil
}
