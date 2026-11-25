![Go CI](https://img.shields.io/github/actions/workflow/status/ozaitsev92/bms/ci.yaml?branch=main&label=go)

# Building Management System API

REST API for managing buildings and apartments.

## Stack

- Go 1.27
- Fiber (HTTP)
- SQLBoiler (ORM)
- PostgreSQL 19
- golang-migrate (schema migrations)
- Testcontainers (repository integration tests)

## Data Model

### building
- id: serial primary key
- name: unique, required
- address: required
- created_at, updated_at: timestamptz

### apartment
- id: serial primary key
- building_id: foreign key to building.id (ON DELETE CASCADE)
- number: required
- floor: required, any integer (0 = ground floor, negative = basement)
- sq_meters: required, > 0 (CHECK constraint)
- unique constraint: (building_id, number)
- created_at, updated_at: timestamptz

## API

Base URL: http://localhost:8080

### Buildings
- `GET /buildings`: paginated list, optional `with_apartments=true|false`
- `GET /buildings/{id}`: includes the building's apartments
- `GET /buildings/{id}/apartments`: paginated, 404 if the building does not exist
- `POST /buildings`: upsert (see below)
- `DELETE /buildings/{id}`: 204, also deletes its apartments

### Apartments
- `GET /apartments`: paginated list
- `GET /apartments/{id}`
- `GET /apartments/building/{buildingId}`: alias of `GET /buildings/{id}/apartments`
- `POST /apartments`: upsert (see below)
- `DELETE /apartments/{id}`: 204

### Operations
- `GET /healthz`: liveness, always 204
- `GET /readyz`: readiness, 204 if the database answers a ping within 1s, otherwise 503

### Upsert semantics
`POST` creates or updates, and the status code reports what actually happened:

| Request body | Behaviour | Status |
|---|---|---|
| no `id`, new natural key | inserts | `201` + `Location` header |
| no `id`, existing natural key (building `name`, apartment `building_id`+`number`) | updates that row | `200` |
| with `id` of an existing row | updates it | `200` |
| with `id` that does not exist | nothing is written | `404` |

### Pagination
List endpoints use keyset pagination ordered by `id`: `?limit=` (1–200, default 50) and `?after_id=` (the last `id` of the previous page). The response is a JSON array.

### Errors
Every error has the shape `{"error": "...", "details": ["..."]}`. `details` is present only for validation errors and lists every violation.

| Status | Meaning |
|---|---|
| 400 | malformed JSON, unknown field, invalid path id or query parameter |
| 404 | resource not found |
| 409 | unique key already taken (for example, renaming a building to an existing name) |
| 413 | body larger than `http_server.body_limit_bytes` (default 1 MiB) |
| 415 | `Content-Type` is not `application/json` |
| 422 | validation failed, or `building_id` refers to a building that does not exist |
| 500 / 503 | internal error / dependency unavailable or request timed out. The cause is logged with the request id, not returned. |

### Example Payloads

Create/Upsert building:
```json
{
  "name": "Tower A",
  "address": "123 Main St"
}
```

Create/Upsert apartment:
```json
{
  "building_id": 1,
  "number": "101",
  "floor": 2,
  "sq_meters": 60
}
```

## Run Locally

Prerequisites: Docker (with Compose v2), Go 1.27+

Start the full stack:
```bash
make run
```

Stop the stack:
```bash
make stop
```

## Configuration

The YAML file is passed with `--config` or `CONFIG_PATH` (see `config/local.yaml`). Environment variables override the file:

| Variable | Overrides |
|---|---|
| `APP_ENV` | `env` (`local` = text logs, debug level; `prod` = JSON logs, info level) |
| `HTTP_ADDRESS` | `http_server.address` |
| `POSTGRES_DSN` | `postgres.dsn` |
| `MIGRATIONS_PATH` | `postgres.migrations_path` |

## Migrations and Models

Run migrations from the host against the compose database (after `make run`):
```bash
make migrate           # apply all pending
make migrate-down      # roll back one step
make migrate-version   # print current version
# migrator also supports: force N (clear a dirty state)
```

Regenerate SQLBoiler models (needs a migrated database on `localhost:5432`):
```bash
make generate
```

## Testing

```bash
make test        # all tests, including Postgres integration tests (requires Docker)
make test-unit   # unit tests only (-short, no Docker)
make cover       # HTML coverage report
make lint        # golangci-lint (config in .golangci.yml)
make vuln        # govulncheck
```

## Postman

Import `postman_collection.json` into Postman, then:
1. Set base_url (default: http://localhost:8080).
2. Run "Upsert Building (Create)" first to populate building_id.
3. Run "Upsert Apartment (Create)" to populate apartment_id.
4. Use the remaining requests for get/list/update/delete flows.

## Project Structure

```text
cmd/
  api/            # HTTP server entrypoint (composition root)
  migrator/       # DB migration entrypoint
config/
  local.yaml      # local configuration
db/
  migrations/     # SQL migrations
  sqlboiler.toml  # SQLBoiler configuration
internal/
  adapter/
    httpapi/      # Fiber server, middleware, handlers, DTOs, error mapping
    postgres/     # Repository implementations
  app/            # Use cases / services
  domain/         # Entities, invariants, domain errors
  port/           # Repository interfaces and their error contract
models/           # SQLBoiler generated models (do not edit)
```

## Design decisions

- **Hexagonal layering.** `domain` imports nothing. `port` declares the repository interfaces *and* their error contract (`domain.ErrNotFound`, `ErrConflict`, `ErrInvalidReference`). Adapters translate driver errors at the boundary, so `app` never sees `database/sql`.
- **One error-to-status mapping.** Handlers return errors and `httpapi.ErrorHandler` maps them. 5xx causes are logged with the request id; clients get stable messages.
- **Upsert by natural key** is kept because the API contract requires it. Upserts are hand-written SQL (sqlboiler can't report insert vs update) using `RETURNING (xmax = 0)`, and updates by id never insert.
- **No authentication** is configured. That is out of scope for this exercise.
