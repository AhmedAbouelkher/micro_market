FROM golang:1.25.3-bookworm

WORKDIR /usr/src/app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN apt-get update && apt-get install -y --no-install-recommends protobuf-compiler && rm -rf /var/lib/apt/lists/*

RUN go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.36.11 && go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.5.1

ENV PATH="/go/bin:${PATH}"
RUN mkdir -p gen && protoc -I proto --go_out=gen --go_opt=paths=source_relative --go-grpc_out=gen --go-grpc_opt=paths=source_relative proto/common/v1/*.proto proto/checkout/v1/*.proto proto/inventory/v1/*.proto

RUN CGO_ENABLED=1 sh -c 'cd checkout-service && go build -ldflags="-s -w" -o /usr/local/bin/checkout-service .'

EXPOSE 1234
EXPOSE 50051
CMD ["/usr/local/bin/checkout-service"]
