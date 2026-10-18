# My Project

## Run with PostgreSQL

Build and start the API and PostgreSQL with Docker Compose:

```sh
docker compose up --build
```

The API is available at `http://localhost:9090`. To run the API directly on
your machine instead, start PostgreSQL:

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
