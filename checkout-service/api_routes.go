package main

import (
	"encoding/json"
	"micro_market/common"
	"net/http"

	"github.com/go-playground/validator/v10"
	"github.com/gorilla/mux"
)

var (
	validate = validator.New()
)

func RegisterAppRoutes(router *mux.Router) {
	router.HandleFunc("/health", handleHealth).Methods(http.MethodGet)

	r := router.PathPrefix("/api/v1").Subrouter()

	r.HandleFunc("/users", handleGetUsers).Methods(http.MethodGet)
	r.HandleFunc("/products", handleGetProducts).Methods(http.MethodGet)

	{
		orders := r.PathPrefix("/orders").Subrouter()
		orders.HandleFunc("", handleGetOrders).Methods(http.MethodGet)
		orders.HandleFunc("", handlePlaceOrder).Methods(http.MethodPost)
	}

	r.Use(common.JSONMiddleware)
	r.Use(telemetry.LogRequest,
		telemetry.MeterRequestDuration,
		telemetry.MeterRequestInFlight,
		telemetry.MeterRequestStatus)

	r.Use(telemetry.MuxMiddleware)
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	_, span := telemetry.TraceStart(r.Context(), "GET /health")
	defer span.End()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]any{"message": "OK"})
}

func handleGetUsers(w http.ResponseWriter, r *http.Request) {
	// ctx, span := telemetry.TraceStart(r.Context(), "GET /api/v1/users")
	// defer span.End()

	users, err := GetAllUsers(r.Context())
	if err != nil {
		common.SendJsonError(w, http.StatusInternalServerError, err)
		return
	}
	json.NewEncoder(w).Encode(users)
}

func handleGetOrders(w http.ResponseWriter, r *http.Request) {
	// ctx, span := telemetry.TraceStart(r.Context(), "GET /api/v1/orders")
	// defer span.End()

	orders, err := GetAllOrders(r.Context())
	if err != nil {
		common.SendJsonError(w, http.StatusInternalServerError, err)
		return
	}
	json.NewEncoder(w).Encode(orders)
}

func handleGetProducts(w http.ResponseWriter, r *http.Request) {
	// ctx, span := telemetry.TraceStart(r.Context(), "GET /api/v1/products")
	// defer span.End()

	products, err := GetAllProducts(r.Context())
	if err != nil {
		common.SendJsonError(w, http.StatusInternalServerError, err)
		return
	}
	json.NewEncoder(w).Encode(products)
}

func handlePlaceOrder(w http.ResponseWriter, r *http.Request) {
	// ctx, span := telemetry.TraceStart(r.Context(), "POST /api/v1/orders")
	// defer span.End()

	var req PlaceOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		common.SendJsonError(w, http.StatusBadRequest, err)
		return
	}
	if err := validate.Struct(req); err != nil {
		common.SendJsonError(w, http.StatusUnprocessableEntity, err)
		return
	}
	order, err := PlaceNewOrder(r.Context(), req)
	if err != nil {
		common.SendJsonError(w, http.StatusInternalServerError, err)
		return
	}
	json.NewEncoder(w).Encode(order)
}
