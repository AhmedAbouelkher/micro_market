package main

import (
	"errors"
	inventoryv1 "micro_market/gen/inventory/v1"
	"os"

	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var (
	clientAddress = os.Getenv("INVENTORY_SERVICE_ADDRESS")

	inventoryConn   *grpc.ClientConn
	inventoryClient inventoryv1.InventoryServiceClient
)

func GetInventoryClient() inventoryv1.InventoryServiceClient {
	if inventoryClient == nil {
		return nil
	}
	return inventoryClient
}

func InitInventoryClient() error {
	if clientAddress == "" {
		return errors.New("INVENTORY_SERVICE_ADDRESS is not set")
	}
	conn, err := grpc.NewClient(clientAddress,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithStatsHandler(otelgrpc.NewClientHandler()),
	)
	if err != nil {
		return err
	}
	inventoryClient = inventoryv1.NewInventoryServiceClient(conn)
	inventoryConn = conn
	return nil
}

func CloseInventoryClient() error {
	if inventoryConn == nil {
		return errors.New("closing inventory connection: inventory connection not initialized")
	}
	return inventoryConn.Close()
}
