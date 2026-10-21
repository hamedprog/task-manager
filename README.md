# Task Manager

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
docker compose up -d postgres redis pyroscope
```

Then run:

```sh
PYROSCOPE_SERVER_ADDRESS=http://localhost:4040 go run ./cmd/api
```
## Run Tests

To run unit tests, generate mocks and run:

```sh
make test
```

To run integration test too:

```
DATABASE_URL="postgres://taskmanager:taskmanager@localhost:5432/taskmanager_test?sslmode=disable" make test
```


To run the k6 load test against the API, install k6 and run:

```sh
make k6-test
```

The test ramps up to five virtual users, requests the task list, and reports
latency and request-failure thresholds in the k6 output. Override the API URL
with `make k6-test K6_TEST_BASE_URL=http://localhost:9090`.

test result:
```sh
make k6-test 
k6 run --env BASE_URL="http://localhost:9090" ./test/k6/loadtest.js

         /\      Grafana   /‾‾/  
    /\  /  \     |\  __   /  /   
   /  \/    \    | |/ /  /   ‾‾\ 
  /          \   |   (  |  (‾)  |
 / __________ \  |_|\_\  \_____/ 


     execution: local
        script: ./test/k6/loadtest.js
        output: -

     scenarios: (100.00%) 1 scenario, 5 max VUs, 1m20s max duration (incl. graceful stop):
              * default: Up to 5 looping VUs for 50s over 3 stages (gracefulRampDown: 30s, gracefulStop: 30s)



  █ THRESHOLDS 

    checks
    ✓ 'rate>0.99' rate=100.00%

    http_req_duration
    ✓ 'p(95)<500' p(95)=2.41ms

    http_req_failed
    ✓ 'rate<0.01' rate=0.00%


  █ TOTAL RESULTS 

    checks_total.......: 155292  3106.872012/s
    checks_succeeded...: 100.00% 155292 out of 155292
    checks_failed......: 0.00%   0 out of 155292

    ✓ GET /tasks returns 200
    ✓ GET /tasks returns a JSON array

    HTTP
    http_req_duration..............: avg=2.48ms min=38.93µs med=1.44ms max=4.87s p(90)=2.05ms p(95)=2.41ms
      { expected_response:true }...: avg=2.48ms min=38.93µs med=1.44ms max=4.87s p(90)=2.05ms p(95)=2.41ms
    http_req_failed................: 0.00%  0 out of 77646
    http_reqs......................: 77646  1553.436006/s

    EXECUTION
    iteration_duration.............: avg=2.64ms min=409.8µs med=1.6ms  max=4.87s p(90)=2.25ms p(95)=2.63ms
    iterations.....................: 77646  1553.436006/s
    vus............................: 1      min=1          max=5
    vus_max........................: 5      min=5          max=5

    NETWORK
    data_received..................: 29 MB  579 kB/s
    data_sent......................: 6.5 MB 131 kB/s




running (0m50.0s), 0/5 VUs, 77646 complete and 0 interrupted iterations
default ✓ [======================================] 0/5 VUs  50s
```

The API uses
`postgres://taskmanager:taskmanager@localhost:5432/taskmanager?sslmode=disable`
by default when run locally. Set `DATABASE_URL` to override it. The `tasks`
table is created at API startup if it does not already exist.
Set `REDIS_ADDR` to override the Redis address; it defaults to
`localhost:6379`. Task list responses are cached for 30 seconds, with the cache
version incremented after successful task creates, updates, and deletes.
Pyroscope is available at `http://localhost:4040` when running the full Compose
stack. Set `PYROSCOPE_SERVER_ADDRESS` to enable continuous CPU and heap
profiling; profiling is disabled when the variable is unset.

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

## API Usage (cURL)

The following examples assume the API is running at `http://localhost:9090`.

### Create a task

Create a task with a title and assignee:

```sh
curl -X POST http://localhost:9090/tasks \
  -H "Content-Type: application/json" \
  -d '{"title":"learn gin","assignee":"hamed"}'
```

### Create a task with a query parameter

Create a task while passing an `assignee` query parameter:

```sh
curl -X POST "http://localhost:9090/tasks?assignee=string" \
  -H "Content-Type: application/json" \
  -d '{"title":"learn gin","assignee":"hamed"}'
```

### List tasks

Retrieve all tasks:

```sh
curl http://localhost:9090/tasks
```

Filter tasks by assignee and completion status:

```sh
curl -G http://localhost:9090/tasks \
  --data-urlencode "assignee=hamed" \
  --data-urlencode "completed=false" \
  --data-urlencode "limit=20" \
  --data-urlencode "offset=0"
```

### Update a task

Mark task `1` as completed:

```sh
curl -X PATCH http://localhost:9090/tasks/1 \
  -H "Content-Type: application/json" \
  -d '{"completed":true}'
```

### Get a task by ID

```sh
curl http://localhost:9090/tasks/1
```

### Delete a task

```sh
curl -X DELETE http://localhost:9090/tasks/1
```
