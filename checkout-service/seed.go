package main

import (
	"context"

	"github.com/go-faker/faker/v4"
	"gorm.io/gorm"
)

func RunSeed(ctx context.Context) error {
	return dbInstance.WithContext(ctx).
		Transaction(func(tx *gorm.DB) error {
			var uCount int64
			if err := tx.Model(UserModel{}).Count(&uCount).Error; err != nil {
				return err
			}
			users := []UserModel{}
			if uCount <= 0 {
				// seeding users
				for range 20 {
					users = append(users, UserModel{
						Name:  faker.FirstName(),
						Email: faker.Email(),
					})
				}
				if err := tx.Create(&users).Error; err != nil {
					return err
				}
			}
			return nil
		})
}
