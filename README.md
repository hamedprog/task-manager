# My Project

## Run with PostgreSQL

Start PostgreSQL:

```sh
docker compose up -d postgres
```

The API uses the default local connection string
`postgres://taskmanager:taskmanager@localhost:5432/taskmanager?sslmode=disable`.
Set `DATABASE_URL` to override it, then start the API:

```sh
go run ./cmd/api
```

The API creates the `tasks` table at startup if it does not already exist.
