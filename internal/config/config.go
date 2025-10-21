package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// Config - structure with all info about db
type Config struct {
	Env    string `mapstructure:"APP_ENV"`
	Server ServerConfig
	Pg     PgConfig
}

// ServerConfig - structure with fields about server
type ServerConfig struct {
	Host        string        `mapstructure:"HTTP_HOST"`
	Port        int           `mapstructure:"HTTP_PORT"`
	Timeout     time.Duration `mapstructure:"HTTP_TIMEOUT"`
	CORSOrigins []string      `mapstructure:"HTTP_CORS_ORIGINS"`
}

// PgConfig - structure with fields about postgres db
type PgConfig struct {
	Host     string `mapstructure:"POSTGRES_HOST"`
	Port     int    `mapstructure:"POSTGRES_PORT"`
	User     string `mapstructure:"POSTGRES_USER"`
	Password string `mapstructure:"POSTGRES_PASSWORD"`
	Db       string `mapstructure:"POSTGRES_DB"`
	SSLMode  string `mapstructure:"POSTGRES_SSLMODE"`
}

// LoadConfig - load config from ENV_FILE in env variable or default - .env/local.env
func LoadConfig() (*Config, error) {
	v := viper.New()

	p := os.Getenv("ENV_FILE")
	if p == "" {
		p = "local.env"
	}

	v.SetConfigFile(p)
	ext := strings.ToLower(filepath.Ext(p))

	if ext == ".env" || ext == "" {
		v.SetConfigType("env")
	}

	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("read config %q: %w", p, err)
	}

	v.AutomaticEnv()

	timeout, err := time.ParseDuration(v.GetString("HTTP_TIMEOUT"))
	if err != nil {
		return nil, fmt.Errorf("parse HTTP_TIMEOUT: %w", err)
	}

	var cors []string
	if s := v.GetString("HTTP_CORS_ORIGINS"); s != "" {
		for _, it := range strings.Split(s, ",") {
			cors = append(cors, strings.TrimSpace(it))
		}
	}

	cfg := &Config{
		Env: v.GetString("APP_ENV"),
		Server: ServerConfig{
			Host:        v.GetString("HTTP_HOST"),
			Port:        v.GetInt("HTTP_PORT"),
			Timeout:     timeout,
			CORSOrigins: cors,
		},
		Pg: PgConfig{
			Host:     v.GetString("POSTGRES_HOST"),
			Port:     v.GetInt("POSTGRES_PORT"),
			User:     v.GetString("POSTGRES_USER"),
			Password: v.GetString("POSTGRES_PASSWORD"),
			Db:       v.GetString("POSTGRES_DB"),
			SSLMode:  v.GetString("POSTGRES_SSLMODE"),
		},
	}
	return cfg, nil
}
