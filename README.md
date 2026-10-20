# My Project

## Run with PostgreSQL

Build and start the API and PostgreSQL with Docker Compose:

```sh
docker compose up --build
```

The API is available at `http://localhost:9090`. Compose also starts Redis for
task-list caching. To run the API directly on your machine, start PostgreSQL
and Redis:

Swagger UI is available at `http://localhost:8080` and loads its API definition
from `api/openapi.yaml`.
Jaeger is available at `http://localhost:16686` for viewing collected traces.
Prometheus is available at `http://localhost:9091`. It scrapes metrics exposed
by the OpenTelemetry Collector, which also forwards traces to Jaeger.

```sh
docker compose up -d postgres redis
```

Then run:

```sh
go run ./cmd/api
```

The API uses
`postgres://taskmanager:taskmanager@localhost:5432/taskmanager?sslmode=disable`
by default when run locally. Set `DATABASE_URL` to override it. The `tasks`
table is created at API startup if it does not already exist.
Set `REDIS_ADDR` to override the Redis address; it defaults to
`localhost:6379`. Task list responses are cached for 30 seconds, with the cache
version incremented after successful task creates, updates, and deletes.

Incoming HTTP requests are traced and measured with OpenTelemetry and exported
over OTLP/HTTP to the Collector. Prometheus receives the total request counter,
request duration histogram, and cumulative number of tasks returned by task-list
requests through the Collector.
When running the API outside Compose, start the Collector and set
`OTEL_EXPORTER_OTLP_ENDPOINT` to its OTLP/HTTP base URL (for example,
`http://localhost:4318`).

Example Prometheus queries:

```promql
sum(rate(task_manager_http_requests_total[5m]))
```

```promql
histogram_quantile(
  0.95,
  sum by (le) (rate(task_manager_http_request_duration_seconds_bucket[5m]))
)
```

```promql
task_manager_tasks_count_total
```

## API definition and code generation

The OpenAPI 3 contract is maintained in `api/openapi.yaml`. Generated request
and response types and Gin route bindings are written to `internal/api/` using
`codgen.yaml`.
The task list endpoint accepts optional `assignee`, `completed`, `limit`, and
`offset` query parameters, for example
`GET /tasks?assignee=Alex&completed=false&limit=20&offset=0`.

Regenerate API code and repository mocks with:

```sh
make generate
```

To regenerate only the API types and Gin bindings:

```sh
make generate-api
```
