package main

import (
	"context"
	"errors"
	"micro_market/common"
	"net/http"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"gorm.io/gorm"
)

func GetAllOrders(ctx context.Context) ([]OrderResource, error) {
	sCtx, span := telemetry.TraceStart(ctx, "GetAllOrders")
	defer span.End()

	ctx, cancel := context.WithTimeout(sCtx, 30*time.Second)
	defer cancel()
	orders := []OrderModel{}
	if err := dbInstance.WithContext(ctx).
		Model(OrderModel{}).
		Joins("Product").
		Order("order_models.created_at DESC").
		Find(&orders).Error; err != nil {
		return nil, err
	}
	resources := []OrderResource{}
	for _, order := range orders {
		resources = append(resources, order.ToResource())
	}
	return resources, nil
}

type PlaceOrderRequest struct {
	OrderSID   string
	UserID     uint
	ProductSID string
	Quantity   int
}

func PlaceNewOrder(ctx context.Context, req PlaceOrderRequest) (*OrderResource, error) {
	sCtx, span := telemetry.TraceStart(ctx, "PlaceNewOrder")
	defer span.End()
	if err := common.MaybeError("inventory.PlaceNewOrder"); err != nil {
		return nil, err
	}
	span.SetAttributes(
		attribute.Int64("order.user_id", int64(req.UserID)),
		attribute.String("order.product_sid", req.ProductSID),
		attribute.Int("order.quantity", req.Quantity),
	)

	ctx, cancel := context.WithTimeout(sCtx, 30*time.Second)
	defer cancel()

	product := ProductModel{}
	if err := dbInstance.WithContext(ctx).
		Model(ProductModel{}).
		Where("sid = ?", req.ProductSID).
		First(&product).Error; err != nil &&
		!errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	if product.ID == 0 {
		return nil, common.NewAppError(http.StatusNotFound, "product not found")
	}
	if product.IsOutOfStock() || product.GetAvailableQuantity() < req.Quantity {
		return nil, common.NewAppError(http.StatusBadRequest, "product out of stock")
	}
	order := OrderModel{
		SID:          req.OrderSID,
		UserID:       req.UserID,
		ProductID:    product.ID,
		Quantity:     req.Quantity,
		PricePerItem: product.Price,
		Total:        req.Quantity * product.Price,
	}
	if err := dbInstance.WithContext(ctx).
		Create(&order).Error; err != nil {
		return nil, err
	}
	r := order.ToResource()
	return &r, nil
}
