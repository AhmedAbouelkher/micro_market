package main

import (
	"time"

	"github.com/oklog/ulid/v2"
	"gorm.io/gorm"
)

type ProductModel struct {
	ID  uint   `gorm:"primaryKey,autoIncrement"`
	SID string `gorm:"uniqueIndex;column:sid"` // product id used between services

	Name     string
	Price    int // price in usd
	Currency string

	AvailableQuantity *int  `gorm:"default:0"`
	OutOfStock        *bool `gorm:"default:false"`

	CreatedAt time.Time      `gorm:"autoCreateTime"`
	UpdatedAt time.Time      `gorm:"autoUpdateTime"`
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

func (p *ProductModel) BeforeCreate(tx *gorm.DB) (err error) {
	if p.SID == "" {
		p.SID = ulid.Make().String()
	}
	return
}

func (p *ProductModel) GetAvailableQuantity() int {
	if p.AvailableQuantity == nil {
		return 0
	}
	return *p.AvailableQuantity
}

func (p *ProductModel) IsOutOfStock() bool { return p.OutOfStock != nil && *p.OutOfStock }

// MARK:- Resource
type ProductResource struct {
	InternalID        uint      `json:"internal_id"`
	SID               string    `json:"sid"`
	Name              string    `json:"name"`
	Price             int       `json:"price"`
	Currency          string    `json:"currency"`
	AvailableQuantity int       `json:"available_quantity"`
	OutOfStock        bool      `json:"out_of_stock"`
	CreatedAt         time.Time `json:"created_at"`
}

func (p *ProductModel) ToResource() ProductResource {
	return ProductResource{
		InternalID:        p.ID,
		SID:               p.SID,
		Name:              p.Name,
		Price:             p.Price,
		Currency:          p.Currency,
		AvailableQuantity: p.GetAvailableQuantity(),
		OutOfStock:        p.IsOutOfStock(),
		CreatedAt:         p.CreatedAt,
	}
}
