DBURL ?= $(DATABASE_URL)

.PHONY: db_up db_down db_logs psql

db_up:
	@docker compose up -d

db_down:
	@docker compose down

db_logs:
	@docker compose logs -f db

psql:
	@docker exec -it currentz_db psql \
		-U $(POSTGRES_USER) \
		-d $(POSTGRES_DB)
