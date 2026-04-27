# Checkout Service

## Description

`checkout-service` is the order entry service in `micro_market`. It exposes HTTP APIs for browsing users, products, and orders, and also exposes a gRPC server for product management. The service stores data in SQLite, talks to `inventory-service` over gRPC, and publishes invoice creation events to Redis.

## Table of Contents

- [Description](#description)
- [Table of Contents](#table-of-contents)
- [Motivation & Why in Golang](#motivation--why-in-golang)
- [Dependencies Used](#dependencies-used)
- [Telemetry](#telemetry)
- [gRPC](#grpc)
- [Run Locally](#run-locally)
- [Build & Run with Docker](#build--run-with-docker)
- [Functions](#functions)

## Motivation & Why in Golang

Go fits this service well because it is small, fast, and easy to deploy as one static binary. The codebase uses goroutines for concurrent HTTP and gRPC servers, gRPC for low-latency service-to-service calls, and strong standard library support for networking, context, and timeouts. That keeps the checkout flow simple and efficient while staying easy to maintain.

## Dependencies Used

- `github.com/gorilla/mux` for HTTP routing.
- `google.golang.org/grpc` for gRPC server and client calls.
- `gorm.io/gorm` and `gorm.io/driver/sqlite` for persistence.
- `github.com/redis/go-redis/v9` for Redis publish/subscribe integration.
- `github.com/go-playground/validator/v10` for request validation.
- `go.opentelemetry.io/*` and `gorm.io/plugin/opentelemetry/tracing` for telemetry.
- `github.com/go-faker/faker/v4` for seed data.

## Telemetry

Telemetry is a core part of this service.

### What it does

The shared telemetry package in `common/otel` creates and configures:

- OTLP log exporter
- OTLP metric exporter
- OTLP trace exporter
- OpenTelemetry propagators for trace context and baggage
- GORM tracing plugin
- Runtime instrumentation

It also wires `logrus` into OpenTelemetry, so service logs are exported with the rest of the signal set.

### How it is used

`checkout-service` initializes telemetry in `main.go` before any DB or server setup:

- `NewTelemetry(...)` creates the providers.
- `UseGormPlugin(db)` enables DB tracing.
- `TraceStart(...)` wraps domain operations and HTTP/gRPC handlers.
- `LogRequest`, `MeterRequestDuration`, `MeterRequestInFlight`, and `MeterRequestStatus` instrument HTTP requests.
- `MuxMiddleware(...)` adds router-level OpenTelemetry middleware.

### Important metrics

The service emits these HTTP metrics:

- `micro_market.http.server.request.duration`
- `micro_market.http.server.active_requests`
- `micro_market.http.server.response.count`

### Important traces

The service traces:

- HTTP requests
- gRPC server calls
- gRPC client calls to inventory
- DB operations through GORM

### Collector config

By default telemetry sends data to `OTEL_COLLECTOR_ENDPOINT`, which falls back to `localhost:4317`.

In Docker Compose, the checkout container uses:

- `OTEL_COLLECTOR_ENDPOINT=grafana-otel:4317`

This is the recommended setup because it exports logs, metrics, and traces in one place.

## gRPC

`checkout-service` runs a gRPC server on `GRPC_PORT` and registers `CheckoutService`.

### Server methods

- `AddNewProduct`
- `UpdateProduct`
- `DeleteProduct`

These methods are thin wrappers around the domain functions in the service. They start a trace span, call the business logic, and translate app errors into gRPC status codes.

### gRPC client usage

The service also acts as a gRPC client to `inventory-service`:

- `InitInventoryClient()` connects using `INVENTORY_SERVICE_ADDRESS`
- `ReserveProduct(...)` checks stock before order creation
- `RegisterOrder(...)` confirms the order after DB write

The client uses OpenTelemetry gRPC stats handlers, so traces flow across services.

## Run Locally

The `Makefile` is the easiest way to run the service locally.

### Checkout service

```bash
make run_checkout
```

This runs:

1. `go mod tidy`
2. `go build -o bin/service .`
3. `cd checkout-service && ./bin/service`

### Useful env vars

- `HTTP_PORT` default `1234`
- `GRPC_PORT` default `50051`
- `SERVICE_NAME` default `checkout-service`
- `SERVICE_VERSION` default `1.0.0`
- `INVENTORY_SERVICE_ADDRESS`
- `REDIS_HOST`
- `REDIS_PORT`
- `REDIS_CHANNEL`
- `OTEL_COLLECTOR_ENDPOINT`

### Notes

- SQLite DB file lives in `checkout-service/data/checkout.db`
- Redis must be reachable before startup
- Inventory service must be reachable if you want to place orders

## Build & Run with Docker

Docker is the recommended way to run this service.

### Build

```bash
make docker-build-checkout
```

This uses `checkout-service/Dockerfile`, which:

- starts from `golang:1.25.3-bookworm`
- installs `protoc`
- installs the Go protobuf plugins
- generates Go code from `proto/*.proto`
- builds the service binary

### Run

Use the root `docker-compose.yml`.

```bash
docker compose up checkout redis inventory grafana-otel
```

Checkout is exposed on host port `8888` and maps to container port `1234`.

### Recommended env setup

The compose file already shows the intended wiring:

- `CHECKOUT_SERVICE_ADDRESS=checkout:50051`
- `INVENTORY_SERVICE_ADDRESS=inventory:50051`
- `REDIS_HOST=redis`
- `OTEL_COLLECTOR_ENDPOINT=grafana-otel:4317`

## Functions

| Function | Description |
| --- | --- |
| `main()` | Boots telemetry, DB, Redis, gRPC client, seed data, HTTP server, and gRPC server. |
| `initHTTPServer()` | Starts HTTP server and registers routes. |
| `initGRPCServer()` | Starts gRPC server and registers `CheckoutService`. |
| `initGRPCClients()` | Initializes downstream gRPC clients. |
| `initDB()` | Opens SQLite DB, enables GORM telemetry plugin, runs migrations. |
| `closeDB()` | Closes SQLite DB connection. |
| `InitRedisDB(ctx)` | Connects to Redis and verifies availability. |
| `CloseRedisDB()` | Closes Redis client. |
| `RunSeed(ctx)` | Seeds users when DB is empty. |
| `RegisterAppRoutes(router)` | Registers HTTP routes and middleware. |
| `handleHealth()` | Health check endpoint. |
| `handleGetUsers()` | Returns all users. |
| `handleGetOrders()` | Returns all orders. |
| `handleGetProducts()` | Returns all products. |
| `handlePlaceOrder()` | Validates request and creates a new order. |
| `GetAllUsers(ctx)` | Loads users from DB and maps them to API resources. |
| `GetAllProducts(ctx)` | Loads products from DB and maps them to API resources. |
| `AddNewProduct(ctx, req)` | Creates a new product. |
| `UpdateProduct(ctx, req)` | Updates an existing product by SID. |
| `DeleteProduct(ctx, req)` | Deletes a product by SID. |
| `GetAllOrders(ctx)` | Loads orders with user and product data. |
| `PlaceNewOrder(ctx, req)` | Checks inventory, writes order, publishes invoice event. |
| `GetInventoryClient()` | Returns cached inventory gRPC client. |
| `InitInventoryClient()` | Connects to inventory gRPC service. |
| `CloseInventoryClient()` | Closes inventory gRPC connection. |
| `handleGRPCError(err)` | Maps app errors to gRPC status errors. |
