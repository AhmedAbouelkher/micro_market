package main

import (
	"time"
)

type OrderModel struct {
	ID  uint   `gorm:"primaryKey,autoIncrement"`
	SID string `gorm:"uniqueIndex;column:sid"` // order id used between services

	UserID uint `gorm:"index"`

	ProductID    uint          `gorm:"index"`
	Product      *ProductModel `gorm:"->:false,foreignKey:ProductID;references:ID"`
	PricePerItem int           // price in usd
	Quantity     int
	Total        int

	CreatedAt time.Time `gorm:"autoCreateTime"`
	UpdatedAt time.Time `gorm:"autoUpdateTime"`
}

// MARK:- Resource
type OrderResource struct {
	InternalID  uint      `json:"internal_id"`
	SID         string    `json:"sid"`
	UserID      uint      `json:"user_id"`
	UserName    string    `json:"user_name,omitempty"`
	UserEmail   string    `json:"user_email,omitempty"`
	ProductID   uint      `json:"product_id"`
	ProductName string    `json:"product_name,omitempty"`
	Quantity    int       `json:"quantity"`
	Total       int       `json:"total"`
	CreatedAt   time.Time `json:"created_at"`
}

func (o *OrderModel) ToResource() OrderResource {
	r := OrderResource{
		InternalID: o.ID,
		SID:        o.SID,
		UserID:     o.UserID,
		ProductID:  o.ProductID,
		Quantity:   o.Quantity,
		Total:      o.Total,
		CreatedAt:  o.CreatedAt,
	}
	if p := o.Product; p != nil {
		r.ProductName = p.Name
	}
	return r
}
