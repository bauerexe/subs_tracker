package config

import (
	"fmt"
	"github.com/spf13/viper"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
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

// LoadConfig - load config from ENV_FILE if present, falling back to the environment
func LoadConfig() (*Config, error) {
	cfg := &Config{
		Env: "local",
		Server: ServerConfig{
			Host:    "0.0.0.0",
			Port:    8080,
			Timeout: 5 * time.Second,
		},
		Pg: PgConfig{
			Host:     "postgres",
			Port:     5432,
			User:     "subs_user",
			Password: "subs_password",
			Db:       "subs_db",
			SSLMode:  "disable",
		},
	}

	p := os.Getenv("ENV_FILE")
	if p == "" {
		p = "local.env"
	}

	if fi, err := os.Stat(p); err == nil && !fi.IsDir() {
		v := viper.New()
		v.SetConfigFile(p)
		ext := strings.ToLower(filepath.Ext(p))

		if ext == ".env" || ext == "" {
			v.SetConfigType("env")
		}

		if err = v.ReadInConfig(); err != nil {
			return nil, fmt.Errorf("read config %q: %w", p, err)
		}

		lookup := func(key string) (string, bool) {
			if !v.IsSet(key) {
				return "", false
			}
			return v.GetString(key), true
		}

		if err = applyOverrides(cfg, lookup, fmt.Sprintf("config file %q", p)); err != nil {
			return nil, err
		}
	} else if err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("stat config %q: %w", p, err)
	} else {
		lookup := func(key string) (string, bool) {
			return os.LookupEnv(key)
		}

		if err = applyOverrides(cfg, lookup, "environment"); err != nil {
			return nil, err
		}
	}

	return cfg, nil
}

func applyOverrides(cfg *Config, lookup func(string) (string, bool), source string) error {
	if v, ok := lookup("APP_ENV"); ok && strings.TrimSpace(v) != "" {
		cfg.Env = strings.TrimSpace(v)
	}

	if v, ok := lookup("HTTP_HOST"); ok {
		cfg.Server.Host = strings.TrimSpace(v)
	}

	if v, ok := lookup("HTTP_PORT"); ok {
		port, err := strconv.Atoi(strings.TrimSpace(v))
		if err != nil {
			return fmt.Errorf("parse %s HTTP_PORT: %w", source, err)
		}
		cfg.Server.Port = port
	}

	if v, ok := lookup("HTTP_TIMEOUT"); ok {
		timeout, err := time.ParseDuration(strings.TrimSpace(v))
		if err != nil {
			return fmt.Errorf("parse %s HTTP_TIMEOUT: %w", source, err)
		}
		cfg.Server.Timeout = timeout
	}

	if v, ok := lookup("HTTP_CORS_ORIGINS"); ok {
		raw := strings.TrimSpace(v)
		if raw == "" {
			cfg.Server.CORSOrigins = nil
		} else {
			parts := strings.Split(raw, ",")
			cors := make([]string, 0, len(parts))
			for _, part := range parts {
				if s := strings.TrimSpace(part); s != "" {
					cors = append(cors, s)
				}
			}
			cfg.Server.CORSOrigins = cors
		}
	}

	if v, ok := lookup("POSTGRES_HOST"); ok {
		cfg.Pg.Host = strings.TrimSpace(v)
	}

	if v, ok := lookup("POSTGRES_PORT"); ok {
		port, err := strconv.Atoi(strings.TrimSpace(v))
		if err != nil {
			return fmt.Errorf("parse %s POSTGRES_PORT: %w", source, err)
		}
		cfg.Pg.Port = port
	}

	if v, ok := lookup("POSTGRES_USER"); ok {
		cfg.Pg.User = strings.TrimSpace(v)
	}

	if v, ok := lookup("POSTGRES_PASSWORD"); ok {
		cfg.Pg.Password = v
	}

	if v, ok := lookup("POSTGRES_DB"); ok {
		cfg.Pg.Db = strings.TrimSpace(v)
	}

	if v, ok := lookup("POSTGRES_SSLMODE"); ok {
		cfg.Pg.SSLMode = strings.TrimSpace(v)
	}

	return nil
}
