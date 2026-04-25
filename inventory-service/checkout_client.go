package main

import (
	"context"
	"errors"
	checkoutv1 "micro_market/gen/checkout/v1"
	"os"

	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var (
	clientAddress = os.Getenv("CHECKOUT_SERVICE_ADDRESS")

	checkoutConn   *grpc.ClientConn
	checkoutClient checkoutv1.CheckoutServiceClient
)

func GetCheckoutClient() checkoutv1.CheckoutServiceClient {
	if checkoutClient == nil {
		return nil
	}
	return checkoutClient
}

func InitCheckoutClient(ctx context.Context) error {
	if clientAddress == "" {
		return errors.New("CHECKOUT_SERVICE_ADDRESS is not set")
	}
	conn, err := grpc.NewClient(clientAddress,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithStatsHandler(otelgrpc.NewClientHandler()),
	)
	if err != nil {
		return err
	}
	checkoutClient = checkoutv1.NewCheckoutServiceClient(conn)
	checkoutConn = conn
	return nil
}

func CloseCheckoutClient() error {
	if checkoutConn == nil {
		return errors.New("closing checkout connection: checkout connection not initialized")
	}
	return checkoutConn.Close()
}
