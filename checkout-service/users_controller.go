package main

import (
	"context"
	"time"
)

func GetAllUsers(ctx context.Context) ([]UserResource, error) {
	sCtx, span := telemetry.TraceStart(ctx, "GetAllUsers")
	defer span.End()

	ctx, cancel := context.WithTimeout(sCtx, 30*time.Second)
	defer cancel()

	users := []UserModel{}
	if err := dbInstance.WithContext(ctx).
		Model(UserModel{}).
		Order("created_at ASC").
		Find(&users).Error; err != nil {
		return nil, err
	}
	resources := []UserResource{}
	for _, user := range users {
		resources = append(resources, user.ToResource())
	}
	return resources, nil
}
