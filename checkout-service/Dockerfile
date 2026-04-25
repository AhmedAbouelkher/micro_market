FROM golang:1.25.3-bookworm

WORKDIR /usr/src/app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=1 sh -c 'cd checkout-service && go build -ldflags="-s -w" -o /usr/local/bin/checkout-service .'

EXPOSE 1234
EXPOSE 50051
CMD ["/usr/local/bin/checkout-service"]
