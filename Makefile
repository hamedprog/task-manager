.PHONY: generate generate-api generate-mocks

generate: generate-api generate-mocks

generate-api:
	mkdir -p internal/api
	go tool github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen --config=codgen.yaml api/openapi.yaml

generate-mocks:
	go tool go.uber.org/mock/mockgen -source=internal/service/task_service.go -destination=internal/service/mock_task_repository_test.go -package=service
