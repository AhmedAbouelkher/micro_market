package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/brianvoe/gofakeit/v6"
)

type product struct {
	ID                uint   `json:"internal_id"`
	SID               string `json:"sid"`
	Name              string `json:"name"`
	Price             int    `json:"price"`
	Currency          string `json:"currency"`
	AvailableQuantity int    `json:"available_quantity"`
	OutOfStock        bool   `json:"out_of_stock"`
}

type order struct {
	ID        uint `json:"id"`
	UserID    uint `json:"user_id"`
	ProductID uint `json:"product_id"`
	Quantity  int  `json:"quantity"`
}

type config struct {
	CheckoutURL  string
	InventoryURL string
	Duration     time.Duration
	Interval     time.Duration
	Concurrency  int
	SeedProducts int
	UserIDs      []uint
}

func main() {
	var cfg config
	var userIDs string
	flag.StringVar(&cfg.CheckoutURL, "checkout-url", envOr("CHECKOUT_URL", "http://localhost:8888"), "checkout service base URL")
	flag.StringVar(&cfg.InventoryURL, "inventory-url", envOr("INVENTORY_URL", "http://localhost:9999"), "inventory service base URL")
	flag.DurationVar(&cfg.Duration, "duration", mustDuration(envOr("DURATION", "30s")), "how long to run")
	flag.DurationVar(&cfg.Interval, "interval", mustDuration(envOr("INTERVAL", "500ms")), "delay between requests")
	flag.IntVar(&cfg.Concurrency, "concurrency", envOrInt("CONCURRENCY", 1), "max in-flight requests")
	flag.IntVar(&cfg.SeedProducts, "seed-products", envOrInt("SEED_PRODUCTS", 5), "products to create if list is empty")
	flag.StringVar(&userIDs, "user-ids", envOr("USER_IDS", ""), "comma-separated user ids used for orders")
	flag.Parse()
	if cfg.Concurrency < 1 {
		cfg.Concurrency = 1
	}
	var err error
	cfg.UserIDs, err = parseUserIDs(userIDs)
	if err != nil {
		fail(err)
	}

	log.Printf("load-generator start checkout=%s inventory=%s duration=%s interval=%s concurrency=%d seed_products=%d user_ids=%v", cfg.CheckoutURL, cfg.InventoryURL, cfg.Duration, cfg.Interval, cfg.Concurrency, cfg.SeedProducts, cfg.UserIDs)

	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	gofakeit.Seed(time.Now().UnixNano())
	client := &http.Client{Timeout: 10 * time.Second}

	ctx, cancel := context.WithTimeout(context.Background(), cfg.Duration)
	defer cancel()

	products, err := ensureProducts(ctx, client, cfg, rng)
	if err != nil {
		fail(err)
	}
	log.Printf("seed ready products=%d", len(products))

	ops := []func(context.Context, *http.Client, config, *rand.Rand, []product) error{
		getProducts,
		getOrders,
		createProductOp,
		updateProduct,
		deleteProduct,
		placeOrder,
	}

	ticker := time.NewTicker(cfg.Interval)
	defer ticker.Stop()

	var (
		sem = make(chan struct{}, cfg.Concurrency)
		wg  sync.WaitGroup
		mu  sync.Mutex
	)

	for {
		select {
		case <-ctx.Done():
			wg.Wait()
			log.Printf("load-generator done")
			return
		case <-ticker.C:
			select {
			case sem <- struct{}{}:
				wg.Add(1)
				go func() {
					defer wg.Done()
					defer func() { <-sem }()

					mu.Lock()
					op := ops[rng.Intn(len(ops))]
					opRng := rand.New(rand.NewSource(rng.Int63()))
					mu.Unlock()

					if err := op(ctx, client, cfg, opRng, products); err != nil {
						log.Printf("op error: %v", err)
					}
				}()
			default:
				// do nothing
			}
		}
	}
}

func ensureProducts(ctx context.Context, client *http.Client, cfg config, rng *rand.Rand) ([]product, error) {
	products, err := fetchProducts(ctx, client, cfg.InventoryURL)
	if err != nil {
		return nil, err
	}
	log.Printf("fetched products=%d", len(products))
	for len(products) < cfg.SeedProducts {
		p, err := createProductOnce(ctx, client, cfg, rng)
		if err != nil {
			return nil, err
		}
		products = append(products, p)
		log.Printf("seed create product id=%d name=%q", p.ID, p.Name)
	}
	return products, nil
}

func getProducts(ctx context.Context, client *http.Client, cfg config, _ *rand.Rand, _ []product) error {
	log.Printf("GET inventory products")
	_, err := doJSON(ctx, client, http.MethodGet, cfg.InventoryURL+"/api/v1/products", nil, nil)
	return err
}

func getOrders(ctx context.Context, client *http.Client, cfg config, _ *rand.Rand, _ []product) error {
	log.Printf("GET checkout orders")
	_, err := doJSON(ctx, client, http.MethodGet, cfg.CheckoutURL+"/api/v1/orders", nil, nil)
	return err
}

func createProductOp(ctx context.Context, client *http.Client, cfg config, rng *rand.Rand, _ []product) error {
	log.Printf("POST inventory product")
	p, err := createProductOnce(ctx, client, cfg, rng)
	if err == nil {
		log.Printf("created product id=%d name=%q", p.ID, p.Name)
	}
	return err
}

func createProductOnce(ctx context.Context, client *http.Client, cfg config, rng *rand.Rand) (product, error) {
	req := map[string]any{
		"name":               gofakeit.ProductName(),
		"price":              10 + rng.Intn(100),
		"currency":           "USD",
		"available_quantity": 5 + rng.Intn(20),
	}
	var out product
	_, err := doJSON(ctx, client, http.MethodPost, cfg.InventoryURL+"/api/v1/products", req, &out)
	return out, err
}

func updateProduct(ctx context.Context, client *http.Client, cfg config, rng *rand.Rand, products []product) error {
	if len(products) == 0 {
		return nil
	}
	p := products[rng.Intn(len(products))]
	log.Printf("PUT inventory product id=%d", p.ID)
	req := map[string]any{
		"name":               gofakeit.ProductName(),
		"price":              p.Price + 1,
		"currency":           p.Currency,
		"available_quantity": p.AvailableQuantity + 1,
	}
	_, err := doJSON(ctx, client, http.MethodPut, fmt.Sprintf("%s/api/v1/products/%d", cfg.InventoryURL, p.ID), req, nil)
	return err
}

func deleteProduct(ctx context.Context, client *http.Client, cfg config, rng *rand.Rand, products []product) error {
	if len(products) == 0 || rng.Intn(5) != 0 {
		return nil
	}
	p := products[rng.Intn(len(products))]
	log.Printf("DELETE inventory product id=%d", p.ID)
	_, err := doJSON(ctx, client, http.MethodDelete, fmt.Sprintf("%s/api/v1/products/%d", cfg.InventoryURL, p.ID), nil, nil)
	return err
}

func placeOrder(ctx context.Context, client *http.Client, cfg config, rng *rand.Rand, products []product) error {
	if len(products) == 0 || len(cfg.UserIDs) == 0 {
		return nil
	}
	p := products[rng.Intn(len(products))]
	qty := 1 + rng.Intn(3)
	userID := cfg.UserIDs[rng.Intn(len(cfg.UserIDs))]
	log.Printf("POST checkout order user_id=%d product_id=%d qty=%d", userID, p.ID, qty)
	req := map[string]any{
		"user_id":    userID,
		"product_id": p.ID,
		"quantity":   qty,
	}
	_, err := doJSON(ctx, client, http.MethodPost, cfg.CheckoutURL+"/api/v1/orders", req, nil)
	return err
}

func fetchProducts(ctx context.Context, client *http.Client, baseURL string) ([]product, error) {
	var out []product
	_, err := doJSON(ctx, client, http.MethodGet, baseURL+"/api/v1/products", nil, &out)
	return out, err
}

func doJSON(ctx context.Context, client *http.Client, method, url string, body any, out any) (*http.Response, error) {
	var r io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		r = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, url, r)
	if err != nil {
		return nil, err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		log.Printf("request failed method=%s url=%s status=%s", method, url, resp.Status)
		return resp, fmt.Errorf("%s %s: %s", method, url, resp.Status)
	}
	log.Printf("request ok method=%s url=%s status=%s", method, url, resp.Status)
	if out != nil {
		if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
			return resp, err
		}
	}
	return resp, nil
}

func envOr(key, def string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return def
}

func envOrInt(key string, def int) int {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		var n int
		if _, err := fmt.Sscanf(v, "%d", &n); err == nil {
			return n
		}
	}
	return def
}

func parseUserIDs(v string) ([]uint, error) {
	if strings.TrimSpace(v) == "" {
		return nil, fmt.Errorf("user ids required: set --user-ids or USER_IDS")
	}
	parts := strings.Split(v, ",")
	ids := make([]uint, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		var id uint
		if _, err := fmt.Sscanf(part, "%d", &id); err != nil {
			return nil, fmt.Errorf("invalid user id %q", part)
		}
		ids = append(ids, id)
	}
	if len(ids) == 0 {
		return nil, fmt.Errorf("user ids required: set --user-ids or USER_IDS")
	}
	return ids, nil
}

func mustDuration(v string) time.Duration {
	d, err := time.ParseDuration(v)
	if err != nil {
		panic(err)
	}
	return d
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
