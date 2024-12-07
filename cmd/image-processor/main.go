package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"product-management/internal/config"
	"product-management/internal/queue"
	"product-management/internal/repository"
	"product-management/internal/service"
	"product-management/pkg/logger"

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

	// RabbitMQ Consumer
	rabbitMQURL := fmt.Sprintf("amqp://%s:%s", cfg.RabbitMQHost, cfg.RabbitMQPort)
	rabbitMQConsumer, err := queue.NewRabbitMQConsumer(rabbitMQURL)
	if err != nil {
		appLogger.Error("RabbitMQ connection failed", logger.Error(err))
		os.Exit(1)
	}
	defer rabbitMQConsumer.Close()

	// Repositories
	productRepo := repository.NewProductRepository(db.DB, appLogger)

	// Services
	productService := service.NewProductService(
		productRepo,
		appLogger,
		nil, // No publisher needed for processor
		nil, // No cache needed for processor
	)

	// Context for cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Image processing message handler
	imageProcessingHandler := func(productID int64, imageURLs []string) error {
		appLogger.Info(fmt.Sprintf("Processing images for product %d", productID))
		return productService.ProcessProductImages(ctx, productID, imageURLs)
	}

	// Start consuming messages
	go func() {
		appLogger.Info("Starting image processing consumer")
		if err := rabbitMQConsumer.ConsumeImageProcessingTasks(ctx, imageProcessingHandler); err != nil {
			appLogger.Error("Image processing consumer failed", logger.Error(err))
			cancel()
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// Block until a signal is received
	<-quit
	appLogger.Info("Shutting down image processor...")

	// Cancel context to stop processing
	cancel()

	appLogger.Info("Image processor stopped")
}