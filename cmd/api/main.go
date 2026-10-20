package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	"task-manager/internal/cache"
	"task-manager/internal/handler"
	"task-manager/internal/repository"
	"task-manager/internal/service"
	"task-manager/internal/telemetry"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/redis/go-redis/v9"
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

	shutdownProfiling, err := telemetry.InitProfiling("task-manager", os.Getenv("PYROSCOPE_SERVER_ADDRESS"))
	if err != nil {
		return fmt.Errorf("initialize profiling: %w", err)
	}
	defer func() {
		if err := shutdownProfiling(); err != nil {
			runErr = errors.Join(runErr, fmt.Errorf("shutdown profiling: %w", err))
		}
	}()

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		databaseURL = "postgres://taskmanager:taskmanager@localhost:5432/taskmanager?sslmode=disable"
	}
	sqlDB, err := sql.Open("pgx", databaseURL)
	if err != nil {
		return err
	}
	defer sqlDB.Close()

	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		redisAddr = "localhost:6379"
	}
	redisClient := redis.NewClient(&redis.Options{Addr: redisAddr})
	taskListCache := cache.NewRedisTaskListCache(redisClient)
	defer func() {
		if err := taskListCache.Close(); err != nil {
			runErr = errors.Join(runErr, fmt.Errorf("close Redis client: %w", err))
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := sqlDB.PingContext(ctx); err != nil {
		return err
	}

	taskRepository := repository.NewTaskRepository(sqlDB)
	taskService := service.NewTaskService(taskRepository, taskListCache)
	httpMetrics, shutdownMetrics, err := telemetry.InitMetrics(context.Background(), "task-manager")
	if err != nil {
		return fmt.Errorf("initialize metrics: %w", err)
	}
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := shutdownMetrics(ctx); err != nil {
			runErr = errors.Join(runErr, fmt.Errorf("shutdown metrics: %w", err))
		}
	}()

	taskHandler := handler.NewTaskHandler(taskService, httpMetrics.TaskCount)
	router := gin.New()
	router.Use(gin.Logger())
	router.Use(cors.Default())
	router.Use(cors.Default())
	router.Use(otelgin.Middleware("task-manager"))
	router.Use(gin.Recovery())
	router.Use(httpMetrics.Middleware())
	taskHandler.RegisterRoutes(router)

	return router.Run(":9090")
}
