# Makefile

include .env
export $(shell sed 's/=.*//' .env)

MIGRATIONS_DIR = internal/db/migrations
DBURL = $(DATABASE_URL)

.PHONY: db_up db_down db_logs psql migrate_up migrate_down migrate_status migrate_create

db_up:
	docker compose up -d

db_down:
	docker compose down

db_logs:
	docker compose logs -f db

psql:
	docker exec -it currentz_db psql -U $(POSTGRES_USER) -d $(POSTGRES_DB)

migrate_up:
	goose -dir $(MIGRATIONS_DIR) postgres "$(DBURL)" up

migrate_down:
	goose -dir $(MIGRATIONS_DIR) postgres "$(DBURL)" down

migrate_status:
	goose -dir $(MIGRATIONS_DIR) postgres "$(DBURL)" status

migrate_create:
	@read -p "Migration name: " name; goose -dir $(MIGRATIONS_DIR) create $$name sql
