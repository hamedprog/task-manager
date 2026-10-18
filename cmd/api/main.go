package main

import (
	"log"
	"task-manager/internal/handler"
	"task-manager/internal/repository"

	"github.com/gin-gonic/gin"
)

func main() {
	taskRepository := repository.NewTaskRepository()
	taskHandler := handler.NewTaskHandler(taskRepository)
	router := gin.Default()
	taskHandler.RegisterRoutes(router)

	if err := router.Run(":9090"); err != nil {
		log.Fatal(err)
	}
}
