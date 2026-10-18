package main

import (
	"context"
	"database/sql"
	"log"
	"os"
	"time"

	"task-manager/internal/handler"
	"task-manager/internal/repository"
	"task-manager/internal/service"

	"github.com/gin-gonic/gin"
	_ "github.com/jackc/pgx/v5/stdlib"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		databaseURL = "postgres://taskmanager:taskmanager@localhost:5432/taskmanager?sslmode=disable"
	}

	db, err := sql.Open("pgx", databaseURL)
	if err != nil {
		return err
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		return err
	}

	taskRepository := repository.NewTaskRepository(db)
	if err := taskRepository.Initialize(ctx); err != nil {
		return err
	}

	taskService := service.NewTaskService(taskRepository)
	taskHandler := handler.NewTaskHandler(taskService)
	router := gin.Default()
	taskHandler.RegisterRoutes(router)

	return router.Run(":9090")
}
