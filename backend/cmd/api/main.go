package main

import (
	"log"
	"os"

	"github.com/gofiber/fiber/v2"

	"deftersystem/backend/internal/handler"
	"deftersystem/backend/internal/repository"
	"deftersystem/backend/internal/service"
)

func main() {
	app := fiber.New(fiber.Config{
		AppName:      "Deftersystem API v1.0",
		ErrorHandler: handler.CustomErrorHandler,
	})

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" || dbURL == "postgres://postgres:[YOUR-DB-PASSWORD]@db.[YOUR-PROJECT-REF].supabase.co:5432/postgres?sslmode=require" {
		dbURL = "postgres://postgres:postgres@localhost:5432/deftersystem?sslmode=disable"
	}

	log.Printf("Connecting to live PostgreSQL database...")
	pool, err := repository.NewPostgresPool(dbURL)
	if err != nil {
		log.Printf("Warning: Failed to connect to PostgreSQL: %v", err)
		log.Println("Starting API server in standalone mode...")
	} else {
		defer pool.Close()
		log.Println("Successfully connected to PostgreSQL database pool!")
	}

	// Direct PostgreSQL Repositories (No-Mock-Data Principle)
	var periodRepo = repository.NewPostgresPeriodRepository(pool)
	var txRepo = repository.NewPostgresTransactionRepository(pool)
	var tenantRepo = repository.NewPostgresTenantRepository(pool)
	var idemRepo = repository.NewPostgresIdempotencyRepository(pool)

	periodSvc := service.NewPeriodService(periodRepo, tenantRepo, txRepo)
	txSvc := service.NewTransactionService(txRepo, periodRepo)
	tenantSvc := service.NewTenantService(tenantRepo)

	handler.SetupRouter(app, periodSvc, txSvc, periodRepo, txRepo, idemRepo, tenantSvc)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Live PostgreSQL API Server listening on :%s", port)
	if err := app.Listen(":" + port); err != nil {
		log.Fatalf("Error starting server: %v", err)
	}
}
