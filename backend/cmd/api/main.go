package main

import (
	"log"

	"server-sing-box-2/backend/internal/config"
	"server-sing-box-2/backend/internal/database"
	"server-sing-box-2/backend/internal/router"
)

func main() {
	cfg := config.Load()

	db, err := database.Connect(cfg.DatabaseDSN)
	if err != nil {
		log.Fatalf("connect database: %v", err)
	}

	if err := database.AutoMigrate(db); err != nil {
		log.Fatalf("auto migrate: %v", err)
	}

	app := router.New(router.Dependencies{
		Config: cfg,
		DB:     db,
	})

	if err := app.Run(cfg.ServerAddr); err != nil {
		log.Fatalf("start server: %v", err)
	}
}
