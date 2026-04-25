package main

import (
	"context"
	"micro_market/common"
	inventoryv1 "micro_market/gen/inventory/v1"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type InventoryGrpcServer struct {
	inventoryv1.UnimplementedInventoryServiceServer
}

func (s *InventoryGrpcServer) ReserveProduct(ctx context.Context, req *inventoryv1.ReserveProductRequest) (*inventoryv1.ReserveProductResponse, error) {
	sCtx, span := telemetry.TraceStart(ctx, "ReserveProduct")
	defer span.End()

	out, err := CheckProductAvailability(sCtx, req.Sid, int(req.Quantity))
	if err != nil {
		return nil, handleGRPCError(err)
	}
	return &inventoryv1.ReserveProductResponse{
		AvailableQuantity: int32(out.AvailableQuantity),
		OutOfStock:        out.OutOfStock,
		IsAvailable:       out.IsAvailable,
	}, nil
}

func (s *InventoryGrpcServer) RegisterOrder(ctx context.Context, req *inventoryv1.RegisterOrderRequest) (*inventoryv1.RegisterOrderResponse, error) {
	sCtx, span := telemetry.TraceStart(ctx, "RegisterOrder")
	defer span.End()

	_, err := PlaceNewOrder(sCtx, PlaceOrderRequest{
		OrderSID:   req.OrderSid,
		UserID:     uint(req.UserId),
		ProductSID: req.ProductSid,
		Quantity:   int(req.Quantity),
	})
	if err != nil {
		return nil, handleGRPCError(err)
	}
	return &inventoryv1.RegisterOrderResponse{Success: true}, nil
}

func handleGRPCError(err error) error {
	if err == nil {
		return nil
	}
	if appErr, ok := err.(*common.AppError); ok {
		return appErr.GRPCStatus().Err()
	}
	return status.New(codes.Unknown, err.Error()).Err()
}
