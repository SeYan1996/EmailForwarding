package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	Database DatabaseConfig
	Gmail    GmailConfig
	Server   ServerConfig
	App      AppConfig
}

type DatabaseConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	Name     string
}

type GmailConfig struct {
	CredentialsFile string
	TokenFile       string
	UserEmail       string
}

type ServerConfig struct {
	Port string
	Mode string
}

type AppConfig struct {
	CheckInterval    time.Duration
	Keywords         []string
	MaxEmailsPerBatch int64 // 每批获取的最大邮件数量
	MaxBatches       int    // 最大批次数
}

func LoadConfig() *Config {
	port, _ := strconv.Atoi(getEnv("DB_PORT", "3306"))
	checkInterval, _ := time.ParseDuration(getEnv("CHECK_INTERVAL", "5m"))
	maxEmails, _ := strconv.ParseInt(getEnv("MAX_EMAILS_PER_BATCH", "50"), 10, 64)
	maxBatches, _ := strconv.Atoi(getEnv("MAX_BATCHES", "10"))

	return &Config{
		Database: DatabaseConfig{
			Host:     getEnv("DB_HOST", "localhost"),
			Port:     port,
			User:     getEnv("DB_USER", "root"),
			Password: getEnv("DB_PASSWORD", ""),
			Name:     getEnv("DB_NAME", "email_forwarding"),
		},
		Gmail: GmailConfig{
			CredentialsFile: getEnv("GMAIL_CREDENTIALS_FILE", "credentials.json"),
			TokenFile:       getEnv("GMAIL_TOKEN_FILE", "token.json"),
			UserEmail:       getEnv("GMAIL_USER_EMAIL", ""),
		},
		Server: ServerConfig{
			Port: getEnv("SERVER_PORT", "8080"),
			Mode: getEnv("GIN_MODE", "debug"),
		},
		App: AppConfig{
			CheckInterval:    checkInterval,
			Keywords:         []string{"紧急", "重要", "客户", "投诉"}, // 可配置的关键字
			MaxEmailsPerBatch: maxEmails,
			MaxBatches:       maxBatches,
		},
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}