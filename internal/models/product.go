package models

import (
	"time"

	"github.com/lib/pq"
)

type Product struct {
	ID                    int64         `json:"id" db:"id"`
	UserID                int64         `json:"user_id" db:"user_id"`
	ProductName           string        `json:"product_name" db:"product_name"`
	ProductDescription    string        `json:"product_description" db:"product_description"`
	ProductPrice          float64       `json:"product_price" db:"product_price"`
	ProductImages         pq.StringArray `json:"product_images" db:"product_images"`
	CompressedProductImages pq.StringArray `json:"compressed_product_images" db:"compressed_product_images"`
	CreatedAt             time.Time     `json:"created_at" db:"created_at"`
	UpdatedAt             time.Time     `json:"updated_at" db:"updated_at"`
}

type ProductCreateRequest struct {
	UserID             int64    `json:"user_id" validate:"required"`
	ProductName        string   `json:"product_name" validate:"required,max=255"`
	ProductDescription string   `json:"product_description"`
	ProductPrice       float64  `json:"product_price" validate:"required,min=0"`
	ProductImages      []string `json:"product_images" validate:"required"`
}

type ProductFilterParams struct {
	UserID      int64   `json:"user_id"`
	MinPrice    float64 `json:"min_price"`
	MaxPrice    float64 `json:"max_price"`
	ProductName string  `json:"product_name"`
	Page        int     `json:"page"`
	PageSize    int     `json:"page_size"`
}