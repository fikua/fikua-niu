.PHONY: test lint typecheck build bootstrap up down db db-down

# Aligned with fikua.yml `commands.*` (manifest is the source of truth;
# this Makefile is a convenience wrapper for local developer use).

# Throwaway local PostgreSQL. The platform uses its shared cluster (one schema
# and role per service); here a plain database is enough.
LOCAL_DB_URL ?= postgres://niu:niu@localhost:54321/niu?sslmode=disable

db:
	@docker inspect niu-dev-db >/dev/null 2>&1 || docker run -d --name niu-dev-db \
		-e POSTGRES_USER=niu -e POSTGRES_PASSWORD=niu -e POSTGRES_DB=niu -p 54321:5432 postgres:17-alpine
	@docker start niu-dev-db >/dev/null
	@until docker exec niu-dev-db pg_isready -U niu -q; do sleep 1; done

db-down:
	-docker rm -f niu-dev-db

# The integration tests need PostgreSQL (skipped without TEST_DATABASE_URL).
test: db
	TEST_DATABASE_URL=$(LOCAL_DB_URL) go test -count=1 ./...

# lint accepts a trailing path argument, per the manifest contract
# (commands.lint: "gofmt -l"). Bare `make lint` checks the whole module.
lint:
	gofmt -l $(if $(FILE),$(FILE),.)

typecheck:
	go vet ./...

build:
	go build ./...

bootstrap:
	go mod download

# up/down are placeholders until NIU-2 completes the real Docker Compose
# deployment — see design.md §5 out-of-scope note (tasks.md §5).
up:
	docker compose up -d

down:
	docker compose down
