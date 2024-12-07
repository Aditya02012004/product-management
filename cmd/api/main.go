package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"product-management/internal/api/routes"
	"product-management/internal/cache"
	"product-management/internal/config"
	"product-management/internal/queue"
	"product-management/internal/repository"
	"product-management/internal/service"
	"product-management/pkg/logger"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

func main() {
	// Load configuration
	cfg := config.LoadConfig()

	// Initialize logger
	appLogger := logger.NewLogger()

	// Database connection
	dbConnStr := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		cfg.DBHost, cfg.DBPort, cfg.DBUser, cfg.DBPassword, cfg.DBName,
	)
	db, err := sqlx.Connect("postgres", dbConnStr)
	if err != nil {
		appLogger.Error("Database connection failed", logger.Error(err))
		os.Exit(1)
	}
	defer db.Close()

	// Redis cache
	redisURL := fmt.Sprintf("redis://%s:%s", cfg.RedisHost, cfg.RedisPort)
	redisCache := cache.NewRedisCache(redisURL)

	// RabbitMQ Publisher
	rabbitMQURL := fmt.Sprintf("amqp://%s:%s", cfg.RabbitMQHost, cfg.RabbitMQPort)
	rabbitMQPublisher, err := queue.NewRabbitMQPublisher(rabbitMQURL)
	if err != nil {
		appLogger.Error("RabbitMQ connection failed", logger.Error(err))
		os.Exit(1)
	}
	defer rabbitMQPublisher.Close()

	// Repositories
	productRepo := repository.NewProductRepository(db.DB, appLogger)

	// Services
	productService := service.NewProductService(
		productRepo,
		appLogger,
		rabbitMQPublisher,
		redisCache,
	)

	// Setup Gin router
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()

	// Middleware
	router.Use(gin.Logger())
	router.Use(gin.Recovery())

	// Setup routes
	routes.SetupProductRoutes(router, productService, appLogger)

	// HTTP Server
	srv := &http.Server{
		Addr:    ":8080",
		Handler: router,
	}

	// Graceful shutdown
	go func() {
		appLogger.Info("Starting server on :8080")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			appLogger.Error("Server start failed", logger.Error(err))
			os.Exit(1)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	appLogger.Info("Shutting down server...")

	// Context with timeout for graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		appLogger.Error("Server forced to shutdown", logger.Error(err))
		os.Exit(1)
	}

	appLogger.Info("Server exiting")
}