package main

import (
	"context"
	"errors"
	"micro_market/common"
	checkoutv1 "micro_market/gen/checkout/v1"
	commonv1 "micro_market/gen/common/v1"
	"net/http"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"gorm.io/gorm"
)

func GetAllProducts(ctx context.Context) ([]ProductResource, error) {
	sCtx, span := telemetry.TraceStart(ctx, "GetAllProducts")
	defer span.End()

	ctx, cancel := context.WithTimeout(sCtx, 30*time.Second)
	defer cancel()
	products := []ProductModel{}
	if err := dbInstance.WithContext(ctx).
		Model(ProductModel{}).
		Order("created_at DESC").
		Find(&products).Error; err != nil {
		return nil, err
	}
	resources := []ProductResource{}
	for _, product := range products {
		resources = append(resources, product.ToResource())
	}
	return resources, nil
}

func GetProduct(ctx context.Context, id uint) (*ProductResource, error) {
	sCtx, span := telemetry.TraceStart(ctx, "GetProduct")
	defer span.End()

	ctx, cancel := context.WithTimeout(sCtx, 30*time.Second)
	defer cancel()

	product := ProductModel{}
	if err := dbInstance.WithContext(ctx).
		Model(ProductModel{}).
		Where("id = ?", id).
		First(&product).Error; err != nil &&
		!errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	if product.ID == 0 {
		return nil, NewAppError(http.StatusNotFound, "product: %s not found", product.SID)
	}
	resource := product.ToResource()
	return &resource, nil
}

type CreateProductRequest struct {
	Name              string `json:"name"`
	Price             int    `json:"price"`
	Currency          string `json:"currency"`
	AvailableQuantity int    `json:"available_quantity"`
}

func CreateProduct(ctx context.Context, req CreateProductRequest) (*ProductResource, error) {
	sCtx, span := telemetry.TraceStart(ctx, "CreateProduct")
	defer span.End()
	span.SetAttributes(
		attribute.String("product.name", req.Name),
		attribute.Int("product.price", req.Price),
		attribute.String("product.currency", req.Currency),
		attribute.Int("product.available_quantity", req.AvailableQuantity),
	)

	ctx, cancel := context.WithTimeout(sCtx, 30*time.Second)
	defer cancel()

	product := ProductModel{
		Name:              req.Name,
		Price:             req.Price,
		Currency:          req.Currency,
		AvailableQuantity: common.IntPtr(req.AvailableQuantity),
		OutOfStock:        common.BoolPtr(false),
	}
	if err := dbInstance.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&product).Error; err != nil {
			return err
		}
		c := GetCheckoutClient()
		if c == nil {
			return NewAppError(http.StatusInternalServerError, "checkout client not initialized")
		}
		_, err := c.AddNewProduct(ctx, &checkoutv1.AddNewProductRequest{
			Product: &commonv1.Product{
				Sid:        product.SID,
				Name:       product.Name,
				Price:      int32(product.Price),
				Currency:   product.Currency,
				OutOfStock: product.IsOutOfStock(),
			},
		})
		return err
	}); err != nil {
		return nil, err
	}

	resource := product.ToResource()
	return &resource, nil
}

type UpdateProductRequest struct {
	Name              string `json:"name"`
	Price             int    `json:"price"`
	Currency          string `json:"currency"`
	AvailableQuantity int    `json:"available_quantity"`
}

func UpdateProduct(ctx context.Context, id uint, req UpdateProductRequest) (*ProductResource, error) {
	sCtx, span := telemetry.TraceStart(ctx, "UpdateProduct")
	defer span.End()
	span.SetAttributes(
		attribute.Int64("product.id", int64(id)),
		attribute.String("product.name", req.Name),
		attribute.Int("product.price", req.Price),
		attribute.String("product.currency", req.Currency),
		attribute.Int("product.available_quantity", req.AvailableQuantity),
	)

	ctx, cancel := context.WithTimeout(sCtx, 30*time.Second)
	defer cancel()

	product := ProductModel{}
	if err := dbInstance.WithContext(ctx).
		Model(ProductModel{}).
		Where("id = ?", id).
		First(&product).Error; err != nil &&
		!errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	if product.ID == 0 {
		return nil, NewAppError(http.StatusNotFound, "product: %s not found", product.SID)
	}
	product.Name = req.Name
	product.Price = req.Price
	product.Currency = req.Currency
	product.AvailableQuantity = common.IntPtr(req.AvailableQuantity)
	if req.AvailableQuantity > 0 {
		product.OutOfStock = common.BoolPtr(false)
	} else {
		product.OutOfStock = common.BoolPtr(true)
	}

	if err := dbInstance.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(&product).Error; err != nil {
			return err
		}
		c := GetCheckoutClient()
		if c == nil {
			return NewAppError(http.StatusInternalServerError, "checkout client not initialized")
		}
		_, err := c.UpdateProduct(ctx, &checkoutv1.UpdateProductRequest{
			Sid:               product.SID,
			Name:              product.Name,
			Price:             int32(product.Price),
			Currency:          product.Currency,
			AvailableQuantity: int32(product.GetAvailableQuantity()),
			OutOfStock:        product.IsOutOfStock(),
		})
		return err
	}); err != nil {
		return nil, err
	}

	resource := product.ToResource()
	return &resource, nil
}

func DeleteProduct(ctx context.Context, id uint) error {
	sCtx, span := telemetry.TraceStart(ctx, "DeleteProduct")
	defer span.End()
	span.SetAttributes(attribute.Int64("product.id", int64(id)))

	ctx, cancel := context.WithTimeout(sCtx, 30*time.Second)
	defer cancel()

	product := ProductModel{}
	if err := dbInstance.WithContext(ctx).
		Model(ProductModel{}).
		Select("id, sid").
		Where("id = ?", id).
		First(&product).Error; err != nil &&
		!errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	if product.ID == 0 {
		return NewAppError(http.StatusNotFound, "product: %s not found", product.SID)
	}

	span.SetAttributes(attribute.String("product.sid", product.SID))

	return dbInstance.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(ProductModel{}).
			Where("id = ?", id).
			Delete(&ProductModel{}).Error; err != nil {
			return err
		}
		c := GetCheckoutClient()
		if c == nil {
			return NewAppError(http.StatusInternalServerError, "checkout client not initialized")
		}
		_, err := c.DeleteProduct(ctx, &checkoutv1.DeleteProductRequest{Sid: product.SID})
		return err
	})
}

type ProductAvailabilityResponse struct {
	IsAvailable       bool
	AvailableQuantity int
	OutOfStock        bool
}

func CheckProductAvailability(ctx context.Context, sid string, quantity int) (*ProductAvailabilityResponse, error) {
	sCtx, span := telemetry.TraceStart(ctx, "CheckProductAvailability")
	defer span.End()
	span.SetAttributes(
		attribute.String("product.sid", sid),
		attribute.Int("requested_quantity", quantity),
	)

	ctx, cancel := context.WithTimeout(sCtx, 30*time.Second)
	defer cancel()

	product := ProductModel{}
	if err := dbInstance.WithContext(ctx).
		Model(ProductModel{}).
		Where("sid = ?", sid).
		First(&product).Error; err != nil &&
		!errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	if product.ID == 0 {
		return nil, NewAppError(http.StatusNotFound, "product: %s not found", sid)
	}
	return &ProductAvailabilityResponse{
		IsAvailable:       product.GetAvailableQuantity() >= quantity,
		AvailableQuantity: product.GetAvailableQuantity(),
		OutOfStock:        product.IsOutOfStock(),
	}, nil
}
