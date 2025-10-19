package config

import (
	"errors"
	"github.com/joho/godotenv"
	"gopkg.in/yaml.v3"
	"io/fs"
	"log"
	"os"
	"path/filepath"
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
	var cfg Config
	cwd, _ := os.Getwd()

	envFile := os.Getenv("ENV_FILE")
	if envFile == "" {
		envFile = ".env"
	}
	envPath := resolvePath(cwd, envFile)
	if envPath != "" {
		if err := godotenv.Overload(envPath); err != nil && !errors.Is(err, fs.ErrNotExist) {
			log.Fatalf("load env: %v", err)
		}
	}

	path := os.Getenv("CONFIG_PATH")
	if path == "" {
		path = "configs/local.yaml"
	}
	path = resolvePath(cwd, path)

	raw, err := os.ReadFile(path)
	if err != nil {
		log.Fatalf("read config: %v", err)
	}

	expanded := os.ExpandEnv(string(raw))
	if err := yaml.Unmarshal([]byte(expanded), &cfg); err != nil {
		log.Fatalf("unmarshal config: %v", err)
	}
	return &cfg
}
