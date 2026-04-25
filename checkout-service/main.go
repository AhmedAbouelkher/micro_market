package main

import (
	"context"
	"fmt"
	"log"
	"micro_market/common"
	common_otel "micro_market/common/otel"
	checkoutv1 "micro_market/gen/checkout/v1"
	"net"
	"net/http"
	"os"

	"github.com/gorilla/mux"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
)

var (
	serviceName    = common.EnvOrDef("SERVICE_NAME", "checkout-service")
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
		log.Printf("failed to create telemetry: %v\n", err)
		os.Exit(1)
	}
	defer tel.Close(ctx)
	telemetry = tel
	log.Println("telemetry was initialized")

	// Init DB
	if err := initDB(); err != nil {
		panic(err)
	}
	defer closeDB()
	log.Println("DB was initialized")

	// Init GRPC Clients
	initGRPCClients()
	defer CloseInventoryClient()

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
	log.Printf("REST server is running on: http://localhost%s\n", p)
	telemetry.LogFatalln(http.ListenAndServe(p, router))
}

func initGRPCServer() {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%s", GRPCServerPort))
	if err != nil {
		telemetry.LogFatalln("failed to listen: %v", err)
	}
	grpcServer := grpc.NewServer(grpc.StatsHandler(otelgrpc.NewServerHandler()))
	checkoutv1.RegisterCheckoutServiceServer(grpcServer, &CheckoutGrpcServer{})
	log.Printf("GRPC server is running on: %s\n", lis.Addr().String())
	telemetry.LogFatalln(grpcServer.Serve(lis))
}

func initGRPCClients() {
	if err := InitInventoryClient(); err != nil {
		panic(err)
	}
	telemetry.LogInfo("Inventory client was initialized")
}
