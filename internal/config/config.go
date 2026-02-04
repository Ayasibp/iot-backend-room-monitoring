package config

import (
	"fmt"
	"os"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	Database DatabaseConfig
	JWT      JWTConfig
	Server   ServerConfig
	CORS     CORSConfig
}

type DatabaseConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	Database string
}

type JWTConfig struct {
	AccessSecret       string
	RefreshSecret      string
	AccessTokenExpiry  time.Duration
	RefreshTokenExpiry time.Duration
}

type ServerConfig struct {
	Port    string
	GinMode string
}

type CORSConfig struct {
	AllowedOrigins []string
}

func LoadConfig() *Config {
	// Load .env file if it exists
	_ = godotenv.Load()

	config := &Config{
		Database: DatabaseConfig{
			Host:     getEnv("DB_HOST", "localhost"),
			Port:     getEnv("DB_PORT", "3306"),
			User:     getEnv("DB_USER", "root"),
			Password: getEnv("DB_PASSWORD", ""),
			Database: getEnv("DB_NAME", "iot_theater_monitoring"),
		},
		JWT: JWTConfig{
			AccessSecret:       getEnv("JWT_ACCESS_SECRET", "your-access-secret-key"),
			RefreshSecret:      getEnv("JWT_REFRESH_SECRET", "your-refresh-secret-key"),
			AccessTokenExpiry:  parseDuration(getEnv("ACCESS_TOKEN_EXPIRY", "15m")),
			RefreshTokenExpiry: parseDuration(getEnv("REFRESH_TOKEN_EXPIRY", "168h")),
		},
		Server: ServerConfig{
			Port:    getEnv("PORT", "8080"),
			GinMode: getEnv("GIN_MODE", "debug"),
		},
		CORS: CORSConfig{
			AllowedOrigins: parseOrigins(getEnv("ALLOWED_ORIGINS", "http://localhost:3000,http://localhost:5173")),
		},
	}

	return config
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func parseDuration(s string) time.Duration {
	duration, err := time.ParseDuration(s)
	if err != nil {
		fmt.Printf("Warning: Invalid duration format '%s', using default\n", s)
		return 15 * time.Minute
	}
	return duration
}

func parseOrigins(s string) []string {
	if s == "" {
		return []string{}
	}

	origins := []string{}
	current := ""
	for _, char := range s {
		if char == ',' {
			if current != "" {
				origins = append(origins, current)
				current = ""
			}
		} else {
			current += string(char)
		}
	}
	if current != "" {
		origins = append(origins, current)
	}

	return origins
}
