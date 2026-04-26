package main

import (
	"context"
	"errors"
	"math/rand"
	"micro_market/common"
	checkoutv1 "micro_market/gen/checkout/v1"
	commonv1 "micro_market/gen/common/v1"

	"github.com/go-faker/faker/v4"
	"github.com/oklog/ulid/v2"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
)

func RunSeed(ctx context.Context) error {
	checkoutClient := GetCheckoutClient()
	if checkoutClient == nil {
		return errors.New("checkout client not initialized")
	}

	return dbInstance.WithContext(ctx).
		Transaction(func(tx *gorm.DB) error {
			var pCount int64
			if err := tx.Model(ProductModel{}).Count(&pCount).Error; err != nil {
				return err
			}
			products := []ProductModel{}
			if pCount <= 0 {
				// seeding products
				for range 10 {
					products = append(products, ProductModel{
						SID:               ulid.Make().String(),
						Name:              faker.FirstName(),
						Price:             10 + rand.Intn(100),
						AvailableQuantity: common.IntPtr(rand.Intn(10)),
						OutOfStock:        common.BoolPtr(rand.Float32() < 0.2),
						Currency:          "USD",
					})
				}
				if err := tx.Transaction(func(tx *gorm.DB) error {
					if err := tx.Create(&products).Error; err != nil {
						return err
					}
					for _, product := range products {
						_, err := checkoutClient.AddNewProduct(ctx, &checkoutv1.AddNewProductRequest{
							Product: &commonv1.Product{
								Sid:        product.SID,
								Name:       product.Name,
								Price:      int32(product.Price),
								Currency:   product.Currency,
								OutOfStock: product.IsOutOfStock(),
							},
						})
						println("added product to checkout", product.SID)
						if err != nil {
							if st, ok := status.FromError(err); ok {
								telemetry.LogErrorlnf("product: %s, error: %s, status: %s", product.SID, st.Message(), st.Code())
								return err
							}
							telemetry.LogErrorlnf("product: %s, error: %v", product.SID, err)
						}
					}
					return nil
				}); err != nil {
					return err
				}
			}

			return nil
		})
}
