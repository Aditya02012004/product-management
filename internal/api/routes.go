package routes

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/product-management/internal/service"
	"github.com/product-management/pkg/logger"
)

func SetupProductRoutes(router *gin.Engine, productService *service.ProductService, logger *logger.Logger) {
	// Product group routes
	v1 := router.Group("/api/v1")
	{
		// Create a new product
		v1.POST("/products", func(c *gin.Context) {
			var req struct {
				UserID             int64    `json:"user_id" binding:"required"`
				ProductName        string   `json:"product_name" binding:"required"`
				ProductDescription string   `json:"product_description"`
				ProductPrice       float64  `json:"product_price" binding:"required,min=0"`
				ProductImages      []string `json:"product_images" binding:"required"`
			}

			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}

			product, err := productService.CreateProduct(c.Request.Context(), &req)
			if err != nil {
				logger.Error("Product creation failed", err)
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}

			c.JSON(http.StatusCreated, product)
		})

		// Get product by ID
		v1.GET("/products/:id", func(c *gin.Context) {
			productID, err := strconv.ParseInt(c.Param("id"), 10, 64)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid product ID"})
				return
			}

			product, err := productService.GetProductByID(c.Request.Context(), productID)
			if err != nil {
				logger.Error("Product retrieval failed", err)
				c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
				return
			}

			c.JSON(http.StatusOK, product)
		})

		// List products
		v1.GET("/products", func(c *gin.Context) {
			var params service.ProductFilterParams
			if err := c.ShouldBindQuery(&params); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}

			products, total, err := productService.ListProducts(c.Request.Context(), &params)
			if err != nil {
				logger.Error("Products listing failed", err)
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}

			c.JSON(http.StatusOK, gin.H{
				"products":    products,
				"total_count": total,
				"page":        params.Page,
				"page_size":   params.PageSize,
			})
		})
	}

	// Health check endpoint
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":    "healthy",
			"timestamp": time.Now().Format(time.RFC3339),
		})
	})
}
