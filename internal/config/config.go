package config

import (
	"errors"
	"github.com/joho/godotenv"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Env    string       `yaml:"env" env-default:"development"`
	Server ServerConfig `yaml:"http_server"`
	Pg     PgConfig     `yaml:"postgres"`
}

type ServerConfig struct {
	Host        string        `yaml:"host"`
	Port        int           `yaml:"port"`
	Timeout     time.Duration `yaml:"timeout"`
	CORSOrigins []string      `yaml:"cors_origins"`
}

type PgConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	Db       string `yaml:"db"`
	SSLMode  string `yaml:"sslmode"`
}

func resolvePath(cwd, p string) string {
	if p == "" {
		return ""
	}
	if filepath.IsAbs(p) {
		return p
	}
	if up, ok := findUp(cwd, p, 8); ok {
		return up
	}
	return filepath.Join(cwd, p)
}

func findUp(start, rel string, max int) (string, bool) {
	dir := start
	for i := 0; i <= max; i++ {
		p := filepath.Join(dir, rel)
		if _, err := os.Stat(p); err == nil {
			return p, true
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return "", false
}

func LoadConfig() *Config {
	cwd, _ := os.Getwd()

	envFile := os.Getenv("ENV_FILE")
	if envFile == "" {
		envFile = ".env/local.env"
	}
	envPath := resolvePath(cwd, envFile)
	if envPath != "" {
		if err := godotenv.Overload(envPath); err != nil {
			var pathErr *fs.PathError
			switch {
			case errors.Is(err, fs.ErrNotExist):
				// ignore missing file to allow running with predefined environment variables
			case errors.As(err, &pathErr) && errors.Is(pathErr.Err, fs.ErrNotExist):
				// ignore missing file
			default:
				log.Fatalf("load env: %v", err)
			}
		}
	}

	cfg := Config{
		Env: getEnv("APP_ENV", "development"),
		Server: ServerConfig{
			Host:        getEnv("HTTP_HOST", "0.0.0.0"),
			Port:        getEnvInt("HTTP_PORT", 8080),
			Timeout:     getEnvDuration("HTTP_TIMEOUT", 5*time.Second),
			CORSOrigins: getEnvStringSlice("HTTP_CORS_ORIGINS"),
		},
		Pg: PgConfig{
			Host:     getEnv("POSTGRES_HOST", "localhost"),
			Port:     getEnvInt("POSTGRES_PORT", 5432),
			User:     getEnvRequired("POSTGRES_USER"),
			Password: getEnvRequired("POSTGRES_PASSWORD"),
			Db:       getEnvRequired("POSTGRES_DB"),
			SSLMode:  getEnv("POSTGRES_SSLMODE", "disable"),
		},
	}
	return &cfg
}

func getEnv(key, def string) string {
	if value, ok := os.LookupEnv(key); ok {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" {
			return trimmed
		}
	}
	return def
}

func getEnvRequired(key string) string {
	if value, ok := os.LookupEnv(key); ok {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" {
			return trimmed
		}
	}
	log.Fatalf("missing required env %s", key)
	return ""
}

func getEnvInt(key string, def int) int {
	value, ok := os.LookupEnv(key)
	if !ok || strings.TrimSpace(value) == "" {
		return def
	}
	parsed, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil {
		log.Fatalf("parse int env %s: %v", key, err)
	}
	return parsed
}

func getEnvDuration(key string, def time.Duration) time.Duration {
	value, ok := os.LookupEnv(key)
	if !ok || strings.TrimSpace(value) == "" {
		return def
	}
	parsed, err := time.ParseDuration(strings.TrimSpace(value))
	if err != nil {
		log.Fatalf("parse duration env %s: %v", key, err)
	}
	return parsed
}

func getEnvStringSlice(key string) []string {
	value, ok := os.LookupEnv(key)
	if !ok {
		return nil
	}
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}
	parts := strings.Split(trimmed, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		item := strings.TrimSpace(part)
		if item != "" {
			result = append(result, item)
		}
	}
	if len(result) == 0 {
		return nil
	}
	return result
}
