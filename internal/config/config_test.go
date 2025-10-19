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

	envPath := filepath.Join(dir, "app.env")

	if err := os.WriteFile(envPath, []byte("APP_ENV=local\nHTTP_HOST=localhost\nHTTP_PORT=8080\nHTTP_TIMEOUT=4s\nHTTP_CORS_ORIGINS=http://localhost:3000,http://127.0.0.1:3000\nPOSTGRES_HOST=localhost\nPOSTGRES_PORT=5432\nPOSTGRES_USER=subs_user\nPOSTGRES_PASSWORD=subs_password\nPOSTGRES_DB=subs_db\nPOSTGRES_SSLMODE=disable\n"), 0o600); err != nil {
		t.Fatalf("failed to write env: %v", err)
	}

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
