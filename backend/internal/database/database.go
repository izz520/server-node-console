package database

import (
	"server-sing-box-2/backend/internal/domain"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func Connect(dsn string) (*gorm.DB, error) {
	return gorm.Open(postgres.Open(dsn), &gorm.Config{})
}

func AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&domain.User{},
		&domain.Server{},
		&domain.NATPortMapping{},
		&domain.ProtocolNode{},
		&domain.Subscription{},
		&domain.ClashTemplate{},
		&domain.SubscriptionNode{},
		&domain.Task{},
		&domain.TaskLog{},
		&domain.OperationLog{},
	)
}
