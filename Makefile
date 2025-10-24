ENV_FILE ?= .env/local.env

ifneq (,$(wildcard $(ENV_FILE)))
ENV_FILE_ARGS := --env-file $(ENV_FILE)
-include $(ENV_FILE)
endif

POSTGRES_USER ?= subs_user
POSTGRES_PASSWORD ?= subs_password
POSTGRES_DB ?= subs_db
POSTGRES_HOST ?= postgres
PG_PORT_CONTAINER ?= 5432

export

DC := docker compose $(ENV_FILE_ARGS)

DB_URL := postgres://$(POSTGRES_USER):$(POSTGRES_PASSWORD)@$(POSTGRES_HOST):$(PG_PORT_CONTAINER)/$(POSTGRES_DB)?sslmode=disable
MIG := $(DC) run --rm migrate

.PHONY: up down migrate-up migrate-down

run_service: migrate-up
	@status=0; \
	$(DC) up --build || status=$$?; \
	if [ $$status -ne 0 ] && [ $$status -ne 130 ]; then exit $$status; fi; \
	exit 0

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
