.PHONY: generate migrate-up migrate-down migrate-status run stop test

generate:
	sqlboiler psql --config db/sqlboiler.toml

migrate:
	go run ./cmd/migrator --config=config/local.yaml

run:
	docker-compose up -d --build

stop:
	docker-compose down

test:
	go test -race -count=1 -v ./internal/... -coverprofile=coverage.out

cover: test
	go tool cover -html=coverage.out