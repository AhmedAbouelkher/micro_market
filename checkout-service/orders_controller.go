package main

import (
	"context"
	"errors"
	"micro_market/common"
	inventoryv1 "micro_market/gen/inventory/v1"
	"net/http"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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
		Joins("User").
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
	UserID    uint `json:"user_id" validate:"required,min=1"`
	ProductID uint `json:"product_id" validate:"required,min=1"`
	Quantity  int  `json:"quantity" validate:"required,min=1"`
}

func PlaceNewOrder(ctx context.Context, req PlaceOrderRequest) (*OrderResource, error) {
	sCtx, span := telemetry.TraceStart(ctx, "PlaceNewOrder")
	defer span.End()
	if err := common.MaybeError("checkout.PlaceNewOrder"); err != nil {
		return nil, err
	}
	span.SetAttributes(
		attribute.Int64("order.user_id", int64(req.UserID)),
		attribute.Int64("order.product_id", int64(req.ProductID)),
		attribute.Int("order.quantity", req.Quantity),
	)

	ctx, cancel := context.WithTimeout(sCtx, time.Minute)
	defer cancel()

	var userCount int64
	if err := dbInstance.WithContext(ctx).
		Model(UserModel{}).
		Where("id = ?", req.UserID).
		Count(&userCount).Error; err != nil {
		return nil, err
	}
	if userCount <= 0 {
		return nil, common.NewAppError(http.StatusBadRequest, "user: %d not found", req.UserID)
	}

	product := ProductModel{}
	if err := dbInstance.WithContext(ctx).
		Model(ProductModel{}).
		Where("id = ?", req.ProductID).
		First(&product).Error; err != nil &&
		!errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	if product.ID == 0 {
		return nil, common.NewAppError(http.StatusBadRequest, "product: %s not found", product.SID)
	}

	inventoryClient := GetInventoryClient()
	if inventoryClient == nil {
		return nil, common.NewAppError(http.StatusInternalServerError, "inventory client not initialized")
	}

	resp, err := inventoryClient.ReserveProduct(ctx, &inventoryv1.ReserveProductRequest{
		Sid:      product.SID,
		Quantity: int32(req.Quantity),
	})
	if err != nil {
		if st, ok := status.FromError(err); ok {
			switch st.Code() {
			case codes.NotFound:
				return nil, common.NewAppError(http.StatusBadRequest, "product not found")
			case codes.FailedPrecondition, codes.ResourceExhausted:
				return nil, common.NewAppError(http.StatusBadRequest, "product out of stock")
			case codes.DeadlineExceeded, codes.Unavailable:
				return nil, common.NewAppError(http.StatusServiceUnavailable, "inventory unavailable")
			}
		}
		return nil, err
	}

	if resp.OutOfStock {
		return nil, common.NewAppError(http.StatusBadRequest, "product: %s is out of stock", product.SID)
	}
	if !resp.IsAvailable {
		return nil, common.NewAppError(http.StatusBadRequest, "product: %s is not available in the quantity of %d", product.SID, req.Quantity)
	}

	order := OrderModel{
		UserID:    req.UserID,
		ProductID: req.ProductID,
		Quantity:  req.Quantity,
	}
	txErr := dbInstance.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&order).Error; err != nil {
			return err
		}

		_, err := inventoryClient.RegisterOrder(ctx, &inventoryv1.RegisterOrderRequest{
			OrderSid:   order.SID,
			UserId:     int32(req.UserID),
			ProductSid: product.SID,
			Quantity:   int32(req.Quantity),
		})
		if err != nil {
			if st, ok := status.FromError(err); ok {
				switch st.Code() {
				case codes.NotFound:
					return common.NewAppError(http.StatusBadRequest, "product not found")
				case codes.FailedPrecondition, codes.ResourceExhausted:
					return common.NewAppError(http.StatusBadRequest, "product out of stock")
				case codes.DeadlineExceeded, codes.Unavailable:
					return common.NewAppError(http.StatusServiceUnavailable, "inventory unavailable")
				}
			}
			return err
		}

		return err
	})

	r := order.ToResource()
	return &r, txErr
}
