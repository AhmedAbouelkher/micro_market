package main

import (
	"context"
	"micro_market/common"
	checkoutv1 "micro_market/gen/checkout/v1"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type CheckoutGrpcServer struct {
	checkoutv1.UnimplementedCheckoutServiceServer
}

func (s *CheckoutGrpcServer) AddNewProduct(ctx context.Context, req *checkoutv1.AddNewProductRequest) (*checkoutv1.AddNewProductResponse, error) {
	_, span := telemetry.TraceStart(ctx, "grpc:AddNewProduct")
	defer span.End()

	if err := AddNewProduct(ctx, req); err != nil {
		return nil, handleGRPCError(err)
	}
	return &checkoutv1.AddNewProductResponse{Success: true}, nil
}

func (s *CheckoutGrpcServer) UpdateProduct(ctx context.Context, req *checkoutv1.UpdateProductRequest) (*checkoutv1.UpdateProductResponse, error) {
	_, span := telemetry.TraceStart(ctx, "grpc:UpdateProduct")
	defer span.End()

	if err := UpdateProduct(ctx, req); err != nil {
		return nil, handleGRPCError(err)
	}
	return &checkoutv1.UpdateProductResponse{Success: true}, nil
}

func (s *CheckoutGrpcServer) DeleteProduct(ctx context.Context, req *checkoutv1.DeleteProductRequest) (*checkoutv1.DeleteProductResponse, error) {
	_, span := telemetry.TraceStart(ctx, "grpc:DeleteProduct")
	defer span.End()

	if err := DeleteProduct(ctx, req); err != nil {
		return nil, handleGRPCError(err)
	}
	return &checkoutv1.DeleteProductResponse{Success: true}, nil
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
