package main

import (
	"log"
	"task-manager/internal/handler"
	"task-manager/internal/repository"
	"task-manager/internal/service"

	"github.com/gin-gonic/gin"
)

func main() {
	taskRepository := repository.NewTaskRepository()
	taskService := service.NewTaskService(taskRepository)
	taskHandler := handler.NewTaskHandler(taskService)
	router := gin.Default()
	taskHandler.RegisterRoutes(router)

	if err := router.Run(":9090"); err != nil {
		log.Fatal(err)
	}
}
