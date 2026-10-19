package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	"task-manager/internal/handler"
	"task-manager/internal/repository"
	"task-manager/internal/service"
	"task-manager/internal/telemetry"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	_ "github.com/jackc/pgx/v5/stdlib"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() (runErr error) {
	shutdownTracing, err := telemetry.InitTracing(context.Background(), "task-manager")
	if err != nil {
		return fmt.Errorf("initialize tracing: %w", err)
	}
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := shutdownTracing(ctx); err != nil {
			runErr = errors.Join(runErr, fmt.Errorf("shutdown tracing: %w", err))
		}
	}()

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
	taskService := service.NewTaskService(taskRepository)
	taskHandler := handler.NewTaskHandler(taskService)
	router := gin.Default()
	router.Use(cors.Default())
	router.Use(otelgin.Middleware("task-manager"))
	taskHandler.RegisterRoutes(router)

	return router.Run(":9090")
}
