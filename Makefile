PROTO_DIR=./proto
GEN_DIR=./gen
BINARY_DIR=./bin

.PHONY: run_checkout run_inventory build_checkout build_inventory gen docker-build-checkout docker-build-inventory

run_checkout: gen tidy build_checkout
	cd checkout-service && ./$(BINARY_DIR)/service

run_inventory: gen tidy build_inventory
	cd inventory-service && ./$(BINARY_DIR)/service

build_checkout:
	cd checkout-service && go build -o $(BINARY_DIR)/service .

build_inventory:
	cd inventory-service && go build -o $(BINARY_DIR)/service .

docker-build-checkout: gen
	docker build -f checkout-service/Dockerfile -t micro_market-checkout .

docker-build-inventory: gen
	docker build -f inventory-service/Dockerfile -t micro_market-inventory .

tidy:
	go mod tidy
	cd checkout-service && go mod tidy
	cd inventory-service && go mod tidy

gen:
	mkdir -p $(GEN_DIR)
	protoc \
	-I $(PROTO_DIR) \
	--go_out=$(GEN_DIR) --go_opt=paths=source_relative \
	--go-grpc_out=$(GEN_DIR) --go-grpc_opt=paths=source_relative \
	$(PROTO_DIR)/common/v1/*.proto \
	$(PROTO_DIR)/checkout/v1/*.proto \
	$(PROTO_DIR)/inventory/v1/*.proto

