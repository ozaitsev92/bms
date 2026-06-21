# Building Management System API

REST API for managing buildings and apartments.

## Stack

- Go 1.25
- Fiber (HTTP)
- SQLBoiler (ORM)
- PostgreSQL
- golang-migrate (schema migrations)
- Testcontainers (repository integration tests)

## Data Model

### building
- id: serial primary key
- name: unique, required
- address: required

### apartment
- id: serial primary key
- building_id: foreign key to building.id (ON DELETE CASCADE)
- number: required
- floor: required
- sq_meters: required
- unique constraint: (building_id, number)

## API

Base URL: http://localhost:8080

### Buildings
- GET /buildings
  - Optional query param: with_apartments=true|false
- GET /buildings/{id}
- POST /buildings
  - Upsert semantics
  - Returns 201 when created, 200 when request body includes id and updates
- DELETE /buildings/{id}

### Apartments
- GET /apartments
- GET /apartments/{id}
- GET /apartments/building/{buildingId}
- POST /apartments
  - Upsert semantics
  - Returns 201 when created, 200 when request body includes id and updates
- DELETE /apartments/{id}

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

Prerequisites: Docker, Docker Compose, Go 1.25+

Start full stack (Postgres + migrator + API):
```bash
make run
```

Stop stack:
```bash
make stop
```

## Migrations and Models

Apply migrations manually:
```bash
make migrate
```

Regenerate SQLBoiler models:
```bash
make generate
```

## Testing

Run project tests:
```bash
make test
```

Generate HTML coverage report:
```bash
make cover
```

## Postman

This repository includes a ready-to-import Postman collection:

- postman_collection.json

How to use:
1. Import postman_collection.json into Postman.
2. Set base_url (default: http://localhost:8080).
3. Run "Upsert Building (Create)" first to populate building_id.
4. Run "Upsert Apartment (Create)" to populate apartment_id.
5. Use the remaining requests for get/list/update/delete flows.

## Project Structure

```text
cmd/
  api/            # HTTP server entrypoint
  migrator/       # DB migration entrypoint
config/
  local.yaml      # local configuration
db/
  migrations/     # SQL migrations
  sqlboiler.toml  # SQLBoiler configuration
internal/
  adapter/
    http/         # Fiber handlers and routing
    postgres/     # Repository implementations
  app/            # Use-case/services
  domain/         # Core entities
  port/           # Repository interfaces
models/           # SQLBoiler generated models
```

## Notes

- No authentication middleware is configured in the current API.
- Configuration is loaded from CONFIG_PATH (or --config), with POSTGRES_DSN override support.
