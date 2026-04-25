package main

import (
	"time"

	"github.com/oklog/ulid/v2"
	"gorm.io/gorm"
)

type OrderModel struct {
	ID  uint   `gorm:"primaryKey,autoIncrement"`
	SID string `gorm:"uniqueIndex;column:sid"` // order id used between services

	UserID uint       `gorm:"index"`
	User   *UserModel `gorm:"->:false,foreignKey:UserID;references:ID"`

	ProductID    uint          `gorm:"index"`
	Product      *ProductModel `gorm:"->:false,foreignKey:ProductID;references:ID"`
	Quantity     int
	PricePerItem int // price in usd
	Total        int

	CreatedAt time.Time `gorm:"autoCreateTime"`
	UpdatedAt time.Time `gorm:"autoUpdateTime"`
}

func (o *OrderModel) BeforeCreate(tx *gorm.DB) (err error) {
	o.SID = ulid.Make().String()
	return
}

// MARK:- Resource
type OrderResource struct {
	InternalID   uint      `json:"internal_id"`
	SID          string    `json:"sid"`
	UserID       uint      `json:"user_id"`
	UserName     string    `json:"user_name,omitempty"`
	UserEmail    string    `json:"user_email,omitempty"`
	ProductID    uint      `json:"product_id"`
	ProductName  string    `json:"product_name,omitempty"`
	PricePerItem int       `json:"price_per_item"`
	Quantity     int       `json:"quantity"`
	Total        int       `json:"total"`
	CreatedAt    time.Time `json:"created_at"`
}

func (o *OrderModel) ToResource() OrderResource {
	r := OrderResource{
		InternalID:   o.ID,
		SID:          o.SID,
		UserID:       o.UserID,
		ProductID:    o.ProductID,
		Quantity:     o.Quantity,
		PricePerItem: o.PricePerItem,
		Total:        o.Total,
		CreatedAt:    o.CreatedAt,
	}
	if u := o.User; u != nil {
		r.UserName = u.Name
		r.UserEmail = u.Email
	}
	if p := o.Product; p != nil {
		r.ProductName = p.Name
	}
	return r
}
