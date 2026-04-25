package main

import (
	"time"

	"gorm.io/gorm"
)

type ProductModel struct {
	ID  uint   `gorm:"primaryKey,autoIncrement"`
	SID string `gorm:"uniqueIndex;column:sid"` // product id used between services

	Name       string
	Price      int // price in usd
	Currency   string
	OutOfStock *bool `gorm:"default:false"`

	CreatedAt time.Time      `gorm:"autoCreateTime"`
	UpdatedAt time.Time      `gorm:"autoUpdateTime"`
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

func (p *ProductModel) IsOutOfStock() bool { return p.OutOfStock != nil && *p.OutOfStock }

// MARK:- Resource
type ProductResource struct {
	InternalID uint      `json:"internal_id"`
	SID        string    `json:"sid"`
	Name       string    `json:"name"`
	Price      int       `json:"price"`
	Currency   string    `json:"currency"`
	CreatedAt  time.Time `json:"created_at"`
}

func (p *ProductModel) ToResource() ProductResource {
	return ProductResource{
		InternalID: p.ID,
		SID:        p.SID,
		Name:       p.Name,
		Price:      p.Price,
		Currency:   p.Currency,
		CreatedAt:  p.CreatedAt,
	}
}
