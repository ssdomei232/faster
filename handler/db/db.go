package db

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/ssdomei232/faster/configs"
)

func GetDB() (*sql.DB, error) {
	config, err := configs.GetConfig()
	if err != nil {
		return nil, err
	}

	// 验证配置是否完整
	if config.DB.Host == "" || config.DB.User == "" || config.DB.DBName == "" {
		return nil, fmt.Errorf("mysql configuration is incomplete: host=%s, user=%s, dbname=%s",
			config.DB.Host, config.DB.User, config.DB.DBName)
	}

	// 构建连接字符串
	dsn := fmt.Sprintf("%s:%s@tcp(%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		config.DB.User,
		config.DB.Password,
		config.DB.Host,
		config.DB.DBName)

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database connection: %w", err)
	}

	// 测试连接
	if err = db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// 配置连接池
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	return db, nil
}
