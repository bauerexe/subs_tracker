ENV_FILE := .env/local.env
DC := docker compose --env-file $(ENV_FILE)

include $(ENV_FILE)
export

DB_URL := postgres://$(POSTGRES_USER):$(POSTGRES_PASSWORD)@postgres:$(PG_PORT_CONTAINER)/$(POSTGRES_DB)?sslmode=disable
MIG := $(DC) run --rm migrate

.PHONY: up down migrate-up migrate-down

run_service:
	$(DC) up --build

up:
	$(DC) up -d postgres adminer

down:
	$(DC) down

down-v:
	$(DC) down -v

restart:
	$(MAKE) down
	$(MAKE) up

migrate-up:
	$(MIG) -path /migrations -database "$(DB_URL)" up

migrate-down:
	$(MIG) -path /migrations -database "$(DB_URL)" down 1
