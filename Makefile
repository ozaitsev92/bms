GOLANGCI_LINT_VERSION := v2.15.0
SQLBOILER_VERSION     := v4.19.7
LOCAL_DSN             ?= postgres://postgres:postgres@localhost:5432/bms?sslmode=disable

.PHONY: build lint vuln test test-unit cover migrate migrate-down migrate-version generate run stop

build:
	go build -o bin/ ./cmd/...

lint:
	go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION) run ./...

vuln:
	go run golang.org/x/vuln/cmd/govulncheck@latest ./...

test:
	go test -race -count=1 ./... -coverprofile=coverage.out

test-unit:
	go test -race -count=1 -short ./...

cover: test
	go tool cover -html=coverage.out

migrate:
	POSTGRES_DSN='$(LOCAL_DSN)' MIGRATIONS_PATH=db/migrations go run ./cmd/migrator --config=config/local.yaml up

migrate-down:
	POSTGRES_DSN='$(LOCAL_DSN)' MIGRATIONS_PATH=db/migrations go run ./cmd/migrator --config=config/local.yaml down

migrate-version:
	POSTGRES_DSN='$(LOCAL_DSN)' MIGRATIONS_PATH=db/migrations go run ./cmd/migrator --config=config/local.yaml version

generate:
	GOBIN=$(CURDIR)/bin go install github.com/aarondl/sqlboiler/v4@$(SQLBOILER_VERSION) \
		github.com/aarondl/sqlboiler/v4/drivers/sqlboiler-psql@$(SQLBOILER_VERSION)
	PATH="$(CURDIR)/bin:$$PATH" sqlboiler psql --config db/sqlboiler.toml

run:
	docker compose up -d --build

stop:
	docker compose down
