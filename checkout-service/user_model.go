package main

import (
	"time"
)

type UserModel struct {
	ID uint `gorm:"primaryKey,autoIncrement"`

	Name  string
	Email string `gorm:"uniqueIndex"`

	CreatedAt time.Time `gorm:"autoCreateTime"`
	UpdatedAt time.Time `gorm:"autoUpdateTime"`
}

// MARK:- Resource

type UserResource struct {
	ID        uint      `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
}

func (u *UserModel) ToResource() UserResource {
	return UserResource{
		ID:        u.ID,
		Name:      u.Name,
		Email:     u.Email,
		CreatedAt: u.CreatedAt,
	}
}
