package config

import (
	"github.com/stretchr/testify/assert"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLoadConfig(t *testing.T) {
	f, err := os.Create("./example.yaml")
	if err != nil {
		t.Fatalf("failed to create tempfile: %v", err)
	}
	f_pg, err := os.Create("./example.env")
	if err != nil {
		t.Fatalf("failed to create tempfile: %v", err)
	}
	abs, err := filepath.Abs(f.Name())
	abs_pg, _ := filepath.Abs(f_pg.Name())

	defer os.RemoveAll(abs)
	defer os.RemoveAll(abs_pg)

	f.WriteString("env: \"local\"\nhttp_server:\n  host: \"localhost\"\n  port: 8080\n  timeout: 4s\npostgres:\n  host: \"localhost\"\n  port: 5432\n  user: ${POSTGRES_USER}\n  password: ${POSTGRES_PASSWORD}\n  db: ${POSTGRES_DB}\n  sslmode: \"disable\"")
	f_pg.WriteString("POSTGRES_USER=subs_user\nPOSTGRES_PASSWORD=subs_password\nPOSTGRES_DB=subs_db\n\nPG_PORT_HOST=5432\nPG_PORT_CONTAINER=5432\n\nADMINER_PORT_HOST=8080\nADMINER_PORT_CONTAINER=8080\n")

	t.Setenv("CONFIG_PATH", abs)
	t.Setenv("CONFIG_PG_PATH", abs_pg)
	cfg := LoadConfig()
	assert.Equal(t, Config{
		Env: "local",
		Server: ServerConfig{
			Host:    "localhost",
			Port:    8080,
			Timeout: 4 * time.Second,
		},
		Pg: PgConfig{
			Host:     "localhost",
			Port:     5432,
			User:     "subs_user",
			Password: "subs_password",
			Db:       "subs_db",
		},
	}, *cfg)

}
