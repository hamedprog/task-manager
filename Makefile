.PHONY: generate

generate:
	go tool go.uber.org/mock/mockgen -source=internal/service/task_service.go -destination=internal/service/mock_task_repository_test.go -package=service
