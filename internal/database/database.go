package database

import (
	"fmt"
	"monitor/internal/models"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type Config struct {
	Driver   string
	Host     string
	Port     int
	User     string
	Password string
	DBName   string
	SSLMode  string
}

var DB *gorm.DB

func InitDB(config Config) error {
	var dialector gorm.Dialector

	switch config.Driver {
	case "mysql":
		dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
			config.User, config.Password, config.Host, config.Port, config.DBName)
		dialector = mysql.Open(dsn)
	case "postgres":
		dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
			config.Host, config.Port, config.User, config.Password, config.DBName, config.SSLMode)
		dialector = postgres.Open(dsn)
	case "sqlite":
		dialector = sqlite.Open(config.DBName)
	default:
		return fmt.Errorf("unsupported database driver: %s", config.Driver)
	}

	db, err := gorm.Open(dialector, &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("failed to get database instance: %w", err)
	}

	// Optimized connection pool settings for high concurrency
	sqlDB.SetMaxIdleConns(50)                // Increased idle connections
	sqlDB.SetMaxOpenConns(200)               // Increased max open connections
	sqlDB.SetConnMaxLifetime(time.Hour)      // Connection lifetime
	sqlDB.SetConnMaxIdleTime(5 * time.Minute) // Reduce idle time

	DB = db

	if err := DB.AutoMigrate(
		&models.MonitorTarget{},
		&models.MonitorStatus{},
		&models.MonitorHistory{},
		&models.IPGeoCache{},
		&models.DNSProvider{},
		&models.AlertChannel{},
		&models.AlertRule{},
		&models.AlertCondition{},
		&models.AlertRuleGroup{},
	); err != nil {
		return fmt.Errorf("failed to migrate database: %w", err)
	}

	// Initialize default DNS providers
	if err := initDefaultDNSProviders(); err != nil {
		return fmt.Errorf("failed to initialize default DNS providers: %w", err)
	}

	return nil
}

// initDefaultDNSProviders initializes default DNS providers if none exist
func initDefaultDNSProviders() error {
	var count int64
	if err := DB.Model(&models.DNSProvider{}).Count(&count).Error; err != nil {
		return err
	}

	// Only initialize if no providers exist
	if count > 0 {
		return nil
	}

	providers := []models.DNSProvider{
		{
			Name:       "Google DNS (UDP)",
			Server:     "8.8.8.8:53",
			ServerType: "udp",
			IsDefault:  true,
		},
		{
			Name:       "Google DNS (DoH)",
			Server:     "https://dns.google/resolve",
			ServerType: "doh",
			IsDefault:  false,
		},
		{
			Name:       "Cloudflare DNS (UDP)",
			Server:     "1.1.1.1:53",
			ServerType: "udp",
			IsDefault:  false,
		},
		{
			Name:       "Cloudflare DNS (DoH)",
			Server:     "https://cloudflare-dns.com/dns-query",
			ServerType: "doh",
			IsDefault:  false,
		},
		{
			Name:       "Quad9 DNS (UDP)",
			Server:     "9.9.9.9:53",
			ServerType: "udp",
			IsDefault:  false,
		},
		{
			Name:       "Aliyun DNS (UDP)",
			Server:     "223.5.5.5:53",
			ServerType: "udp",
			IsDefault:  false,
		},
	}

	for _, provider := range providers {
		if err := DB.Create(&provider).Error; err != nil {
			return err
		}
	}

	return nil
}

func GetDB() *gorm.DB {
	return DB
}