package service

import (
	"context"
	"fmt"
	"time"

	"product-management/internal/models"
	"product-management/internal/repository"
	"product-management/internal/queue"
	"product-management/internal/cache"
	"product-management/pkg/logger"

	"github.com/go-playground/validator/v10"
)

type ProductService struct {
	productRepo     *repository.ProductRepository
	validator       *validator.Validate
	logger          *logger.Logger
	rabbitMQPublisher *queue.RabbitMQPublisher
	redisCache      *cache.RedisCache
}

func NewProductService(
	productRepo *repository.ProductRepository,
	logger *logger.Logger,
	rabbitMQPublisher *queue.RabbitMQPublisher,
	redisCache *cache.RedisCache,
) *ProductService {
	return &ProductService{
		productRepo:     productRepo,
		validator:       validator.New(),
		logger:          logger,
		rabbitMQPublisher: rabbitMQPublisher,
		redisCache:      redisCache,
	}
}

func (s *ProductService) CreateProduct(ctx context.Context, req *models.ProductCreateRequest) (*models.Product, error) {
	// Validate input
	if err := s.validator.Struct(req); err != nil {
		return nil, fmt.Errorf("validation error: %w", err)
	}

	// Create product model
	product := &models.Product{
		UserID:             req.UserID,
		ProductName:        req.ProductName,
		ProductDescription: req.ProductDescription,
		ProductPrice:       req.ProductPrice,
		ProductImages:      req.ProductImages,
		CreatedAt:          time.Now(),
		UpdatedAt:          time.Now(),
	}

	// Save to database
	if err := s.productRepo.Create(ctx, product); err != nil {
		return nil, err
	}

	// Publish image processing message
	if err := s.rabbitMQPublisher.PublishImageProcessingTask(product.ID, product.ProductImages); err != nil {
		s.logger.Error("Failed to publish image processing task", logger.Error(err))
		// Log error but don't fail the entire operation
	}

	return product, nil
}

func (s *ProductService) GetProductByID(ctx context.Context, productID int64) (*models.Product, error) {
	// Try cache first
	cacheKey := fmt.Sprintf("product:%d", productID)
	cachedProduct, err := s.redisCache.Get(ctx, cacheKey)
	if err == nil && cachedProduct != nil {
		return cachedProduct, nil
	}

	// Fetch from database
	product, err := s.productRepo.FindByID(ctx, productID)
	if err != nil {
		return nil, err
	}

	// Cache the result
	if err := s.redisCache.Set(ctx, cacheKey, product, 1*time.Hour); err != nil {
		s.logger.Error("Failed to cache product", logger.Error(err))
	}

	return product, nil
}

func (s *ProductService) ListProducts(ctx context.Context, params *models.ProductFilterParams) ([]models.Product, int, error) {
	// Validate input
	if err := s.validator.Struct(params); err != nil {
		return nil, 0, fmt.Errorf("validation error: %w", err)
	}

	// Generate cache key
	cacheKey := fmt.Sprintf("products:%d:%f:%f:%s:%d:%d", 
		params.UserID, 
		params.MinPrice, 
		params.MaxPrice, 
		params.ProductName, 
		params.Page, 
		params.PageSize,
	)

	// Try cache first
	cachedProducts, totalCount, err := s.redisCache.GetList(ctx, cacheKey)
	if err == nil && cachedProducts != nil {
		return cachedProducts, totalCount, nil
	}

	// Fetch from database
	products, total, err := s.productRepo.FindByUserID(ctx, params)
	if err != nil {
		return nil, 0, err
	}

	// Cache the result
	if err := s.redisCache.SetList(ctx, cacheKey, products, total, 30*time.Minute); err != nil {
		s.logger.Error("Failed to cache product list", logger.Error(err))
	}

	return products, total, nil
}

func (s *ProductService) ProcessProductImages(ctx context.Context, productID int64, imageURLs []string) error {
	// Simulate image processing (would typically be in a separate microservice)
	compressedImages := make([]string, len(imageURLs))
	for i, url := range imageURLs {
		// In a real-world scenario, this would call an image processing service
		compressedImages[i] = fmt.Sprintf("compressed_%s", url)
	}

	// Update product with compressed images
	if err := s.productRepo.UpdateCompressedImages(ctx, productID, compressedImages); err != nil {
		return err
	}

	// Invalidate cache
	cacheKey := fmt.Sprintf("product:%d", productID)
	if err := s.redisCache.Delete(ctx, cacheKey); err != nil {
		s.logger.Error("Failed to invalidate product cache", logger.Error(err))
	}

	return nil
}