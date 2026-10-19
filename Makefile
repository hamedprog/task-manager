.DEFAULT_GOAL := help

MIGRATE_DB_NAME := taskmanager
MIGRATE_DSN := "postgres://taskmanager:taskmanager@localhost:5432/$(MIGRATE_DB_NAME)?sslmode=disable"
MIGRATE_VER := development
MIGRATE_SOURCE := file://db/$(MIGRATE_DB_NAME)/$(MIGRATE_VER)

.PHONY: generate
generate: ## Generate
generate: generate-api
generate: generate-mocks

.PHONY: generate-api
generate-api: ## Generate API
	mkdir -p internal/api
	go tool github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen --config=codgen.yaml api/openapi.yaml

.PHONY: generate-mocks
generate-mocks: ## Generate Mocks
	go tool go.uber.org/mock/mockgen -source=internal/service/task_service.go -destination=internal/service/mock_task_repository_test.go -package=service
	go tool go.uber.org/mock/mockgen -source=internal/handler/task_handler.go -destination=internal/handler/mock_task_service_test.go -package=handler

.PHONY: help
help: ## Help
	@grep --extended-regexp "^[a-zA-Z_-]+:.*?## .*$$" $(MAKEFILE_LIST) \
| sort \
| awk 'BEGIN { FS = ":.*?## " }; { printf "\033[36m%-33s\033[0m %s\n", $$1, $$2 }'

.PHONY: test
test: ## Test
	go test ./... -count=1 -covermode=atomic

.PHONY: migrate-up
migrate-up: ## Migrate Up
	migrate -database=$(MIGRATE_DSN) -source=$(MIGRATE_SOURCE) -verbose up

.PHONY: migrate-down
migrate-down: ## Migrate Down
	migrate -database=$(MIGRATE_DSN) -source=$(MIGRATE_SOURCE) -verbose down 1