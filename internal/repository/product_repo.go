package repository

import (
	"context"
	"database/sql"
	"fmt"

	"product-management/internal/models"
	"product-management/pkg/logger"

	"github.com/lib/pq"
)

type ProductRepository struct {
	db     *sql.DB
	logger *logger.Logger
}

func NewProductRepository(db *sql.DB, logger *logger.Logger) *ProductRepository {
	return &ProductRepository{
		db:     db,
		logger: logger,
	}
}

func (r *ProductRepository) Create(ctx context.Context, product *models.Product) error {
	query := `
		INSERT INTO products 
		(user_id, product_name, product_description, product_price, product_images) 
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at, updated_at
	`

	err := r.db.QueryRowContext(
		ctx, 
		query, 
		product.UserID, 
		product.ProductName, 
		product.ProductDescription, 
		product.ProductPrice, 
		pq.Array(product.ProductImages),
	).Scan(&product.ID, &product.CreatedAt, &product.UpdatedAt)

	if err != nil {
		r.logger.Error("Failed to create product", logger.Error(err))
		return fmt.Errorf("failed to create product: %w", err)
	}

	return nil
}

func (r *ProductRepository) FindByID(ctx context.Context, id int64) (*models.Product, error) {
	query := `
		SELECT id, user_id, product_name, product_description, 
		       product_price, product_images, compressed_product_images, 
		       created_at, updated_at 
		FROM products 
		WHERE id = $1
	`

	product := &models.Product{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&product.ID, 
		&product.UserID, 
		&product.ProductName, 
		&product.ProductDescription, 
		&product.ProductPrice, 
		pq.Array(&product.ProductImages),
		pq.Array(&product.CompressedProductImages),
		&product.CreatedAt, 
		&product.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("product not found")
		}
		r.logger.Error("Failed to find product", logger.Error(err))
		return nil, fmt.Errorf("failed to retrieve product: %w", err)
	}

	return product, nil
}

func (r *ProductRepository) FindByUserID(ctx context.Context, params *models.ProductFilterParams) ([]models.Product, int, error) {
	baseQuery := `
		SELECT id, user_id, product_name, product_description, 
		       product_price, product_images, compressed_product_images, 
		       created_at, updated_at 
		FROM products 
		WHERE user_id = $1
	`
	countQuery := `
		SELECT COUNT(*) 
		FROM products 
		WHERE user_id = $1
	`

	var conditions []string
	var args []interface{}
	args = append(args, params.UserID)
	argPos := 2

	if params.MinPrice > 0 {
		conditions = append(conditions, fmt.Sprintf("product_price >= $%d", argPos))
		args = append(args, params.MinPrice)
		argPos++
	}

	if params.MaxPrice > 0 {
		conditions = append(conditions, fmt.Sprintf("product_price <= $%d", argPos))
		args = append(args, params.MaxPrice)
		argPos++
	}

	if params.ProductName != "" {
		conditions = append(conditions, fmt.Sprintf("product_name ILIKE $%d", argPos))
		args = append(args, "%"+params.ProductName+"%")
		argPos++
	}

	// Apply conditions
	if len(conditions) > 0 {
		baseQuery += " AND " + strings.Join(conditions, " AND ")
		countQuery += " AND " + strings.Join(conditions, " AND ")
	}

	// Pagination
	if params.Page < 1 {
		params.Page = 1
	}
	if params.PageSize < 1 {
		params.PageSize = 10
	}
	offset := (params.Page - 1) * params.PageSize
	baseQuery += fmt.Sprintf(" LIMIT $%d OFFSET $%d", argPos, argPos+1)
	args = append(args, params.PageSize, offset)

	// Get total count
	var totalCount int
	err := r.db.QueryRowContext(ctx, countQuery, args[:len(args)-2]...).Scan(&totalCount)
	if err != nil {
		r.logger.Error("Failed to count products", logger.Error(err))
		return nil, 0, fmt.Errorf("failed to count products: %w", err)
	}

	// Execute query
	rows, err := r.db.QueryContext(ctx, baseQuery, args...)
	if err != nil {
		r.logger.Error("Failed to find products", logger.Error(err))
		return nil, 0, fmt.Errorf("failed to retrieve products: %w", err)
	}
	defer rows.Close()

	var products []models.Product
	for rows.Next() {
		var product models.Product
		err := rows.Scan(
			&product.ID, 
			&product.UserID, 
			&product.ProductName, 
			&product.ProductDescription, 
			&product.ProductPrice, 
			pq.Array(&product.ProductImages),
			pq.Array(&product.CompressedProductImages),
			&product.CreatedAt, 
			&product.UpdatedAt,
		)
		if err != nil {
			r.logger.Error("Failed to scan product", logger.Error(err))
			return nil, 0, fmt.Errorf("failed to scan product: %w", err)
		}
		products = append(products, product)
	}

	return products, totalCount, nil
}

func (r *ProductRepository) UpdateCompressedImages(ctx context.Context, productID int64, compressedImages []string) error {
	query := `
		UPDATE products 
		SET compressed_product_images = $1, updated_at = CURRENT_TIMESTAMP 
		WHERE id = $2
	`

	_, err := r.db.ExecContext(ctx, query, pq.Array(compressedImages), productID)
	if err != nil {
		r.logger.Error("Failed to update compressed images", logger.Error(err))
		return fmt.Errorf("failed to update compressed images: %w", err)
	}

	return nil
}