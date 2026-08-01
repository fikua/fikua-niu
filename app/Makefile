.PHONY: test lint typecheck build bootstrap up down killtest

# Aligned with fikua.yml `commands.*` (manifest is the source of truth;
# this Makefile is a convenience wrapper for local developer use).

test:
	go test ./...

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

# killtest (ADR-04, T-31): repeatedly SIGKILL the running binary mid-write
# and verify SQLite integrity + clean restart. NOT run in CI by default —
# manual, mandatory procedure before closing NIU-1 (REL-01/NFR-07).
# Usage: make killtest N=10
N ?= 10
killtest:
	go run ./tests/killtest -n $(N)
