package main

import (
	"context"
	"fmt"
	"log"
	"micro_market/common"
	common_otel "micro_market/common/otel"
	inventoryv1 "micro_market/gen/inventory/v1"
	"net"
	"net/http"

	"github.com/gorilla/mux"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
)

var (
	serviceName    = common.EnvOrDef("SERVICE_NAME", "inventory-service")
	serviceVersion = common.EnvOrDef("SERVICE_VERSION", "1.0.0")

	HttpServerPort = common.EnvOrDef("HTTP_PORT", "1234")
	GRPCServerPort = common.EnvOrDef("GRPC_PORT", "50051")
)

var telemetry common_otel.TelemetryProvider

func main() {
	ctx := context.Background()

	// Init Telemetry
	cfg := common_otel.TelemetryConfig{ServiceName: serviceName, ServiceVersion: serviceVersion}
	tel, err := common_otel.NewTelemetry(ctx, cfg)
	if err != nil {
		panic(err)
	}
	defer tel.Close(ctx)
	telemetry = tel
	telemetry.LogInfo("telemetry was initialized")

	// Init DB
	if err := initDB(ctx); err != nil {
		panic(err)
	}
	defer closeDB()
	log.Println("DB was initialized")

	// Init GRPC Clients
	initGRPCClients(ctx)
	defer CloseCheckoutClient()

	// Seed DB
	if err := RunSeed(ctx); err != nil {
		panic(err)
	}
	log.Println("DB was seeded")

	// Init GRPC Server
	go initGRPCServer()

	// Init HTTP Server
	go initHTTPServer()

	select {}
}

func initHTTPServer() {
	// HTTP Server
	router := mux.NewRouter()

	RegisterAppRoutes(router)

	p := fmt.Sprintf(":%s", HttpServerPort)
	log.Printf("Running inventory service on: http://localhost%s\n", p)
	telemetry.LogFatalln(http.ListenAndServe(p, router))
}

func initGRPCServer() {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%s", GRPCServerPort))
	if err != nil {
		telemetry.LogFatalln("failed to listen: %v", err)
	}
	grpcServer := grpc.NewServer(grpc.StatsHandler(otelgrpc.NewServerHandler()))
	inventoryv1.RegisterInventoryServiceServer(grpcServer, &InventoryGrpcServer{})
	log.Printf("GRPC server is running on: %s\n", lis.Addr().String())
	telemetry.LogFatalln(grpcServer.Serve(lis))
}

func initGRPCClients(ctx context.Context) {
	if err := InitCheckoutClient(ctx); err != nil {
		panic(err)
	}
	telemetry.LogInfo("Checkout client was initialized")
}
