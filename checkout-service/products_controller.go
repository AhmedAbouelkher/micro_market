package main

import (
	"context"
	"errors"
	"micro_market/common"
	checkoutv1 "micro_market/gen/checkout/v1"
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

func AddNewProduct(ctx context.Context, req *checkoutv1.AddNewProductRequest) error {
	sCtx, span := telemetry.TraceStart(ctx, "AddNewProduct")
	defer span.End()
	span.SetAttributes(
		attribute.String("product.sid", req.Product.Sid),
		attribute.String("product.name", req.Product.Name),
		attribute.Int64("product.price", int64(req.Product.Price)),
		attribute.String("product.currency", req.Product.Currency),
		attribute.Bool("product.out_of_stock", req.Product.OutOfStock),
	)

	ctx, cancel := context.WithTimeout(sCtx, 30*time.Second)
	defer cancel()

	rp := req.Product
	product := ProductModel{
		SID:        rp.Sid,
		Name:       rp.Name,
		Price:      int(rp.Price),
		Currency:   rp.Currency,
		OutOfStock: common.BoolPtr(rp.OutOfStock),
	}
	if err := dbInstance.WithContext(ctx).
		Create(&product).Error; err != nil {
		return err
	}
	return nil
}

func UpdateProduct(ctx context.Context, req *checkoutv1.UpdateProductRequest) error {
	sCtx, span := telemetry.TraceStart(ctx, "UpdateProduct")
	defer span.End()
	span.SetAttributes(
		attribute.String("product.sid", req.Sid),
		attribute.String("product.name", req.Name),
		attribute.Int64("product.price", int64(req.Price)),
		attribute.String("product.currency", req.Currency),
		attribute.Bool("product.out_of_stock", req.OutOfStock),
	)

	ctx, cancel := context.WithTimeout(sCtx, 30*time.Second)
	defer cancel()

	product := ProductModel{}
	if err := dbInstance.WithContext(ctx).
		Model(ProductModel{}).
		Where("sid = ?", req.Sid).
		First(&product).Error; err != nil &&
		!errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	if product.ID == 0 {
		return common.NewAppError(http.StatusNotFound, "product: %s not found", req.Sid)
	}
	product.Name = req.Name
	product.Price = int(req.Price)
	product.Currency = req.Currency
	product.OutOfStock = common.BoolPtr(req.OutOfStock)
	if err := dbInstance.WithContext(ctx).
		Save(&product).Error; err != nil {
		return err
	}
	return nil
}

func DeleteProduct(ctx context.Context, req *checkoutv1.DeleteProductRequest) error {
	sCtx, span := telemetry.TraceStart(ctx, "DeleteProduct")
	defer span.End()
	span.SetAttributes(attribute.String("product.sid", req.Sid))

	ctx, cancel := context.WithTimeout(sCtx, 30*time.Second)
	defer cancel()

	product := ProductModel{}
	if err := dbInstance.WithContext(ctx).
		Model(ProductModel{}).
		Where("sid = ?", req.Sid).
		First(&product).Error; err != nil {
		return err
	}
	if err := dbInstance.WithContext(ctx).
		Delete(&product).Error; err != nil {
		return err
	}
	return nil
}
