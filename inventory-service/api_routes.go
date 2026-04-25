package main

import (
	"encoding/json"
	"micro_market/common"
	"net/http"
	"strconv"

	"github.com/go-playground/validator/v10"
	"github.com/gorilla/mux"
)

var (
	validate = validator.New()
)

func RegisterAppRoutes(router *mux.Router) {
	router.HandleFunc("/health", handleHealth).Methods(http.MethodGet)

	r := router.PathPrefix("/api/v1").Subrouter()

	r.HandleFunc("/orders", handleGetOrders).Methods(http.MethodGet)

	{
		products := r.PathPrefix("/products").Subrouter()
		products.HandleFunc("", handleGetProducts).Methods(http.MethodGet)
		products.HandleFunc("", handleCreateProduct).Methods(http.MethodPost)
		products.HandleFunc("/{id}", handleGetProduct).Methods(http.MethodGet)
		products.HandleFunc("/{id}", handleUpdateProduct).Methods(http.MethodPut)
		products.HandleFunc("/{id}", handleDeleteProduct).Methods(http.MethodDelete)
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

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]any{"message": "OK"})
}

func handleGetOrders(w http.ResponseWriter, r *http.Request) {
	ctx, span := telemetry.TraceStart(r.Context(), "GET /api/v1/orders")
	defer span.End()

	orders, err := GetAllOrders(ctx)
	if err != nil {
		common.SendJsonError(w, http.StatusInternalServerError, err)
		return
	}
	json.NewEncoder(w).Encode(orders)
}

func handleGetProducts(w http.ResponseWriter, r *http.Request) {
	ctx, span := telemetry.TraceStart(r.Context(), "GET /api/v1/products")
	defer span.End()

	products, err := GetAllProducts(ctx)
	if err != nil {
		common.SendJsonError(w, http.StatusInternalServerError, err)
		return
	}
	json.NewEncoder(w).Encode(products)
}

func handleGetProduct(w http.ResponseWriter, r *http.Request) {
	ctx, span := telemetry.TraceStart(r.Context(), "GET /api/v1/products/{id}")
	defer span.End()

	vars := mux.Vars(r)
	id, err := strconv.ParseUint(vars["id"], 10, 64)
	if err != nil {
		common.SendJsonError(w, http.StatusBadRequest, err)
		return
	}
	product, err := GetProduct(ctx, uint(id))
	if err != nil {
		common.SendJsonError(w, http.StatusInternalServerError, err)
		return
	}
	json.NewEncoder(w).Encode(product)
}

func handleCreateProduct(w http.ResponseWriter, r *http.Request) {
	ctx, span := telemetry.TraceStart(r.Context(), "POST /api/v1/products")
	defer span.End()

	var req CreateProductRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		common.SendJsonError(w, http.StatusBadRequest, err)
		return
	}
	if err := validate.Struct(req); err != nil {
		common.SendJsonError(w, http.StatusUnprocessableEntity, err)
		return
	}
	product, err := CreateProduct(ctx, req)
	if err != nil {
		common.SendJsonError(w, http.StatusInternalServerError, err)
		return
	}
	json.NewEncoder(w).Encode(product)
}

func handleUpdateProduct(w http.ResponseWriter, r *http.Request) {
	ctx, span := telemetry.TraceStart(r.Context(), "PUT /api/v1/products/{id}")
	defer span.End()

	vars := mux.Vars(r)
	id, err := strconv.ParseUint(vars["id"], 10, 64)
	if err != nil {
		common.SendJsonError(w, http.StatusBadRequest, err)
		return
	}
	var req UpdateProductRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		common.SendJsonError(w, http.StatusBadRequest, err)
		return
	}
	if err := validate.Struct(req); err != nil {
		common.SendJsonError(w, http.StatusUnprocessableEntity, err)
		return
	}
	product, err := UpdateProduct(ctx, uint(id), req)
	if err != nil {
		common.SendJsonError(w, http.StatusInternalServerError, err)
		return
	}
	json.NewEncoder(w).Encode(product)
}

func handleDeleteProduct(w http.ResponseWriter, r *http.Request) {
	ctx, span := telemetry.TraceStart(r.Context(), "DELETE /api/v1/products/{id}")
	defer span.End()

	vars := mux.Vars(r)
	id, err := strconv.ParseUint(vars["id"], 10, 64)
	if err != nil {
		common.SendJsonError(w, http.StatusBadRequest, err)
		return
	}
	if err := DeleteProduct(ctx, uint(id)); err != nil {
		common.SendJsonError(w, http.StatusInternalServerError, err)
		return
	}
	json.NewEncoder(w).Encode(map[string]any{"message": "Product deleted successfully"})
}
