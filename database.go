package main

import (
	"fmt"
	"os"
	"strconv"
	"time"

	gormMysql "gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func openDatabase() (*gorm.DB, error) {
	user := env("NS_MARIADB_USER", "auxilia_user")
	password := env("NS_MARIADB_PASSWORD", "auxilia_password")
	host := env("NS_MARIADB_HOSTNAME", "127.0.0.1")
	port := env("NS_MARIADB_PORT", "3306")
	database := env("NS_MARIADB_DATABASE", "auxilia_web")
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=UTC", user, password, host, port, database)
	db, err := gorm.Open(gormMysql.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent), TranslateError: true})
	if err != nil {
		return nil, err
	}
	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}
	maxOpen := envInt("DB_MAX_OPEN_CONNS", 5)
	sqlDB.SetMaxOpenConns(maxOpen)
	sqlDB.SetMaxIdleConns(min(maxOpen, 2))
	sqlDB.SetConnMaxIdleTime(5 * time.Minute)
	sqlDB.SetConnMaxLifetime(30 * time.Minute)
	if err := sqlDB.Ping(); err != nil {
		return nil, err
	}
	return db, nil
}

func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
func envInt(key string, fallback int) int {
	value, err := strconv.Atoi(os.Getenv(key))
	if err != nil || value < 1 {
		return fallback
	}
	return value
}
