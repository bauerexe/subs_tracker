package config

import (
	"github.com/joho/godotenv"
	"gopkg.in/yaml.v3"
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
}

func resolvePath(cwd, p string) string {
	if p == "" {
		return ""
	}
	if filepath.IsAbs(p) {
		return p
	}
	if up, ok := findUp(cwd, p, 8); ok { // ищем вверх: cwd/../../p
		return up
	}
	return filepath.Join(cwd, p) // как есть (на случай, если кладёшь рядом с пакетом)
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

	// 1) .env
	envPath := os.Getenv("CONFIG_PG_PATH")
	if envPath == "" {
		if up, ok := findUp(cwd, ".env/local_pg.env", 8); ok {
			envPath = up
		}
	} else {
		envPath = resolvePath(cwd, envPath)
	}
	if envPath != "" {
		_ = godotenv.Overload(envPath)
	}

	// 2) YAML
	path := os.Getenv("CONFIG_PATH")
	if path == "" {
		if up, ok := findUp(cwd, "configs/local.yaml", 8); ok {
			path = up
		} else if up, ok := findUp(cwd, ".env/local.yaml", 8); ok {
			path = up
		} else {
			log.Fatal("CONFIG_PATH not set and local.yaml not found")
		}
	} else {
		path = resolvePath(cwd, path)
	}

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
