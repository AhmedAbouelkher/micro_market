package main

import (
	"errors"
	"log"
	"os"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var dbInstance *gorm.DB

func initDB() error {
	if err := os.MkdirAll("data", 0o755); err != nil {
		return err
	}
	db, err := gorm.Open(sqlite.Open("data/checkout.db"), &gorm.Config{})
	if err != nil {
		return err
	}
	dbInstance = db

	telemetry.UseGormPlugin(db)

	err = db.AutoMigrate(&UserModel{})
	err = db.AutoMigrate(&ProductModel{})
	err = db.AutoMigrate(&OrderModel{})
	if err != nil {
		return err
	}

	return nil
}

func closeDB() error {
	if dbInstance == nil {
		return errors.New("database not initialized")
	}
	db, err := dbInstance.DB()
	if err != nil {
		log.Fatal(err)
	}
	return db.Close()
}