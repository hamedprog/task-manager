# My Project

## Run with PostgreSQL

Build and start the API and PostgreSQL with Docker Compose:

```sh
docker compose up --build
```

The API is available at `http://localhost:9090`. To run the API directly on
your machine instead, start PostgreSQL:

Swagger UI is available at `http://localhost:8080` and loads its API definition
from `api/openapi.yaml`.

```sh
docker compose up -d postgres
```

Then run:

```sh
go run ./cmd/api
```

The API uses
`postgres://taskmanager:taskmanager@localhost:5432/taskmanager?sslmode=disable`
by default when run locally. Set `DATABASE_URL` to override it. The `tasks`
table is created at API startup if it does not already exist.

## API definition and code generation

The OpenAPI 3 contract is maintained in `api/openapi.yaml`. Generated request
and response types and Gin route bindings are written to `internal/api/` using
`codgen.yaml`.

Regenerate API code and repository mocks with:

```sh
make generate
```

To regenerate only the API types and Gin bindings:

```sh
make generate-api
```
