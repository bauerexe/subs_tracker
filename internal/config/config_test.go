package config

import (
	"github.com/stretchr/testify/assert"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLoadConfig(t *testing.T) {
	dir := t.TempDir()

	cfgPath := filepath.Join(dir, "config.yaml")
	envPath := filepath.Join(dir, "app.env")

	if err := os.WriteFile(cfgPath, []byte("env: \"local\"\nhttp_server:\n  host: \"localhost\"\n  port: ${APP_PORT}\n  timeout: 4s\n  cors_origins:\n    - \"http://localhost:3000\"\n    - \"http://127.0.0.1:3000\"\npostgres:\n  host: \"localhost\"\n  port: ${PG_PORT}\n  user: ${POSTGRES_USER}\n  password: ${POSTGRES_PASSWORD}\n  db: ${POSTGRES_DB}\n  sslmode: ${POSTGRES_SSLMODE}\n"), 0o600); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	if err := os.WriteFile(envPath, []byte("POSTGRES_USER=subs_user\nPOSTGRES_PASSWORD=subs_password\nPOSTGRES_DB=subs_db\nPOSTGRES_SSLMODE=disable\nAPP_PORT=8080\nPG_PORT=5432\n"), 0o600); err != nil {
		t.Fatalf("failed to write env: %v", err)
	}

	t.Setenv("CONFIG_PATH", cfgPath)
	t.Setenv("ENV_FILE", envPath)

	cfg := LoadConfig()

	assert.Equal(t, Config{
		Env: "local",
		Server: ServerConfig{
			Host:        "localhost",
			Port:        8080,
			Timeout:     4 * time.Second,
			CORSOrigins: []string{"http://localhost:3000", "http://127.0.0.1:3000"},
		},
		Pg: PgConfig{
			Host:     "localhost",
			Port:     5432,
			User:     "subs_user",
			Password: "subs_password",
			Db:       "subs_db",
			SSLMode:  "disable",
		},
	}, *cfg)
}
