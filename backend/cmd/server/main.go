package main

import (
	"log"
	"os"

	"github.com/ispcms/backend/internal/config"
	"github.com/ispcms/backend/internal/database"
	"github.com/ispcms/backend/internal/router"
	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables")
	}

	cfg := config.Load()

	db, err := database.Connect(cfg)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// Clean up old orphaned/duplicate rows before AutoMigrate adds new indexes.
	database.PrepareSchema(db)

	if err := database.Migrate(db); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	if err := database.Seed(db, cfg); err != nil {
		log.Fatalf("Failed to seed database: %v", err)
	}

	app, scheduler := router.Setup(db, cfg)

	// Start OLT background sync scheduler
	scheduler.Start()
	log.Println("OLT sync scheduler started")

	port := cfg.ServerPort
	if port == "" {
		port = "8080"
	}
	log.Printf("Server starting on port %s", port)
	if err := app.Listen(":" + port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
		os.Exit(1)
	}
}
