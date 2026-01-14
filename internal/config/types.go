package config

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// Config 应用配置
type Config struct {
	Server        ServerConfig        `yaml:"server"`
	Database      DatabaseConfig      `yaml:"database"`
	Monitor       MonitorConfig       `yaml:"monitor"`
	Logger        LoggerConfig        `yaml:"logger"`
	Elasticsearch ElasticsearchConfig `yaml:"elasticsearch"`
	Alert         AlertConfig         `yaml:"alert"`
	SNMP          SNMPConfig          `yaml:"snmp"`
}

type ServerConfig struct {
	HTTPPort int    `yaml:"http_port"`
	GRPCPort int    `yaml:"grpc_port"`
	Host     string `yaml:"host"`
}

type DatabaseConfig struct {
	Driver   string `yaml:"driver"`
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	DBName   string `yaml:"dbname"`
	SSLMode  string `yaml:"sslmode"`
}

type MonitorConfig struct {
	CheckInterval int `yaml:"check_interval"` // seconds
	Workers       int `yaml:"workers"`
}

type LoggerConfig struct {
	Level  string `yaml:"level"`  // debug, info, warn, error
	Output string `yaml:"output"` // stdout, stderr, or file path
}

type ElasticsearchConfig struct {
	Enabled  bool   `yaml:"enabled"`  // 是否启用 Elasticsearch
	Addresses []string `yaml:"addresses"` // ES 节点地址，如 ["http://localhost:9200"]
	Username string `yaml:"username"` // ES 用户名
	Password string `yaml:"password"` // ES 密码
	IndexPrefix string `yaml:"index_prefix"` // 索引前缀，如 "monitor-logs"
}

type AlertConfig struct {
	Enabled          bool `yaml:"enabled"`            // 是否启用告警
	CooldownSeconds  int  `yaml:"cooldown_seconds"`   // 告警冷却时间（秒）
	RetryTimes       int  `yaml:"retry_times"`        // 失败重试次数
	RetryInterval    int  `yaml:"retry_interval"`    // 重试间隔（秒）
}

type SNMPConfig struct {
	DefaultCommunity string `yaml:"default_community"` // 默认 SNMP community string
	DefaultVersion   string `yaml:"default_version"`    // 默认 SNMP version: v1, v2c, v3
	DefaultTimeout   int    `yaml:"default_timeout"`    // 默认超时时间（毫秒）
}

// Load 从文件加载配置
func LoadFromFile(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// 设置默认值
	setDefaults(&config)

	return &config, nil
}

// SaveToFile 保存配置到文件
func SaveToFile(path string, config *Config) error {
	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// Load 从环境变量加载配置
func Load() *Config {
	return &Config{
		Server: ServerConfig{
			HTTPPort: getEnvInt("HTTP_PORT", 8080),
			GRPCPort: getEnvInt("GRPC_PORT", 9090),
			Host:     getEnv("HOST", "0.0.0.0"),
		},
		Database: DatabaseConfig{
			Driver:   getEnv("DB_DRIVER", "sqlite"),
			Host:     getEnv("DB_HOST", "localhost"),
			Port:     getEnvInt("DB_PORT", 3306),
			User:     getEnv("DB_USER", "root"),
			Password: getEnv("DB_PASSWORD", ""),
			DBName:   getEnv("DB_NAME", "monitor.db"),
			SSLMode:  getEnv("DB_SSLMODE", "disable"),
		},
		Monitor: MonitorConfig{
			CheckInterval: getEnvInt("MONITOR_INTERVAL", 60),
			Workers:       getEnvInt("MONITOR_WORKERS", 10),
		},
		Logger: LoggerConfig{
			Level:  getEnv("LOG_LEVEL", "info"),
			Output: getEnv("LOG_OUTPUT", "stdout"),
		},
		Elasticsearch: ElasticsearchConfig{
			Enabled:     getEnvBool("ES_ENABLED", false),
			Addresses:   getEnvSlice("ES_ADDRESSES", []string{"http://localhost:9200"}),
			Username:    getEnv("ES_USERNAME", ""),
			Password:    getEnv("ES_PASSWORD", ""),
			IndexPrefix: getEnv("ES_INDEX_PREFIX", "monitor-logs"),
		},
		Alert: AlertConfig{
			Enabled:         getEnvBool("ALERT_ENABLED", true),
			CooldownSeconds: getEnvInt("ALERT_COOLDOWN", 300),
			RetryTimes:      getEnvInt("ALERT_RETRY_TIMES", 3),
			RetryInterval:   getEnvInt("ALERT_RETRY_INTERVAL", 60),
		},
		SNMP: SNMPConfig{
			DefaultCommunity: getEnv("SNMP_COMMUNITY", "public"),
			DefaultVersion:   getEnv("SNMP_VERSION", "v2c"),
			DefaultTimeout:   getEnvInt("SNMP_TIMEOUT", 5000),
		},
	}
}

// setDefaults 设置默认值
func setDefaults(config *Config) {
	if config.Server.HTTPPort == 0 {
		config.Server.HTTPPort = 8080
	}
	if config.Server.GRPCPort == 0 {
		config.Server.GRPCPort = 9090
	}
	if config.Server.Host == "" {
		config.Server.Host = "0.0.0.0"
	}
	if config.Database.Driver == "" {
		config.Database.Driver = "sqlite"
	}
	if config.Database.DBName == "" {
		config.Database.DBName = "monitor.db"
	}
	if config.Monitor.CheckInterval == 0 {
		config.Monitor.CheckInterval = 60
	}
	if config.Monitor.Workers == 0 {
		config.Monitor.Workers = 10
	}
	if config.Logger.Level == "" {
		config.Logger.Level = "info"
	}
	if config.Logger.Output == "" {
		config.Logger.Output = "stdout"
	}
	if config.Alert.CooldownSeconds == 0 {
		config.Alert.CooldownSeconds = 300
	}
	if config.Alert.RetryTimes == 0 {
		config.Alert.RetryTimes = 3
	}
	if config.Alert.RetryInterval == 0 {
		config.Alert.RetryInterval = 60
	}
	if config.SNMP.DefaultCommunity == "" {
		config.SNMP.DefaultCommunity = "public"
	}
	if config.SNMP.DefaultVersion == "" {
		config.SNMP.DefaultVersion = "v2c"
	}
	if config.SNMP.DefaultTimeout == 0 {
		config.SNMP.DefaultTimeout = 5000
	}
}

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

func getEnvInt(key string, defaultVal int) int {
	if val := os.Getenv(key); val != "" {
		var intVal int
		if _, err := fmt.Sscanf(val, "%d", &intVal); err == nil {
			return intVal
		}
	}
	return defaultVal
}

func getEnvBool(key string, defaultVal bool) bool {
	if val := os.Getenv(key); val != "" {
		if val == "true" || val == "1" || val == "yes" {
			return true
		}
		return false
	}
	return defaultVal
}

func getEnvSlice(key string, defaultVal []string) []string {
	if val := os.Getenv(key); val != "" {
		// 支持逗号分隔的字符串
		var result []string
		for _, v := range splitAndTrim(val, ",") {
			if v != "" {
				result = append(result, v)
			}
		}
		if len(result) > 0 {
			return result
		}
	}
	return defaultVal
}

// splitAndTrim 分割字符串并去除空白
func splitAndTrim(s, sep string) []string {
	var result []string
	for _, part := range strings.Split(s, sep) {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

// Validate 验证配置的有效性
func (c *Config) Validate() error {
	// 验证服务器配置
	if c.Server.HTTPPort < 1 || c.Server.HTTPPort > 65535 {
		return fmt.Errorf("invalid HTTP port: %d", c.Server.HTTPPort)
	}
	if c.Server.GRPCPort < 1 || c.Server.GRPCPort > 65535 {
		return fmt.Errorf("invalid gRPC port: %d", c.Server.GRPCPort)
	}
	if c.Server.Host == "" {
		return fmt.Errorf("server host cannot be empty")
	}

	// 验证数据库配置
	validDrivers := map[string]bool{
		"sqlite":  true,
		"mysql":   true,
		"postgres": true,
	}
	if !validDrivers[c.Database.Driver] {
		return fmt.Errorf("invalid database driver: %s", c.Database.Driver)
	}

	if c.Database.Driver != "sqlite" {
		if c.Database.Host == "" {
			return fmt.Errorf("database host cannot be empty for %s", c.Database.Driver)
		}
		if c.Database.Port < 1 || c.Database.Port > 65535 {
			return fmt.Errorf("invalid database port: %d", c.Database.Port)
		}
		if c.Database.User == "" {
			return fmt.Errorf("database user cannot be empty for %s", c.Database.Driver)
		}
		if c.Database.DBName == "" {
			return fmt.Errorf("database name cannot be empty")
		}
	} else {
		if c.Database.DBName == "" {
			return fmt.Errorf("database file path cannot be empty for sqlite")
		}
	}

	// 验证监控配置
	if c.Monitor.CheckInterval < 1 {
		return fmt.Errorf("monitor check interval must be at least 1 second")
	}
	if c.Monitor.Workers < 1 {
		return fmt.Errorf("monitor workers must be at least 1")
	}

	// 验证日志配置
	validLogLevels := map[string]bool{
		"debug": true,
		"info":  true,
		"warn":  true,
		"error": true,
	}
	if !validLogLevels[c.Logger.Level] {
		return fmt.Errorf("invalid log level: %s", c.Logger.Level)
	}

	// 验证Elasticsearch配置
	if c.Elasticsearch.Enabled {
		if len(c.Elasticsearch.Addresses) == 0 {
			return fmt.Errorf("elasticsearch addresses cannot be empty when enabled")
		}
	}

	// 验证告警配置
	if c.Alert.Enabled {
		if c.Alert.CooldownSeconds < 0 {
			return fmt.Errorf("alert cooldown seconds cannot be negative")
		}
		if c.Alert.RetryTimes < 0 {
			return fmt.Errorf("alert retry times cannot be negative")
		}
		if c.Alert.RetryInterval < 0 {
			return fmt.Errorf("alert retry interval cannot be negative")
		}
	}

	// 验证SNMP配置
	validSNMPVersions := map[string]bool{
		"v1":  true,
		"v2c": true,
		"v3":  true,
	}
	if !validSNMPVersions[c.SNMP.DefaultVersion] {
		return fmt.Errorf("invalid SNMP version: %s", c.SNMP.DefaultVersion)
	}
	if c.SNMP.DefaultTimeout < 0 {
		return fmt.Errorf("SNMP timeout cannot be negative")
	}

	return nil
}