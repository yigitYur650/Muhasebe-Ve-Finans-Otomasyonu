package main

import (
	"log"
	"os"
	"strings"

	"github.com/gofiber/fiber/v2"

	"deftersystem/backend/internal/domain"
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
	if dbURL == "" {
		// Try reading from .env or ../.env or backend/.env
		data, err := os.ReadFile(".env")
		if err != nil {
			data, err = os.ReadFile("../.env")
		}
		if err != nil {
			data, err = os.ReadFile("backend/.env")
		}
		if err == nil {
			for _, line := range strings.Split(string(data), "\n") {
				line = strings.TrimSpace(line)
				if strings.HasPrefix(line, "DATABASE_URL=") {
					dbURL = strings.TrimPrefix(line, "DATABASE_URL=")
					dbURL = strings.Trim(dbURL, `"'`)
					break
				}
			}
		}
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

	var periodRepo domain.PeriodRepository
	var txRepo domain.TransactionRepository
	var tenantRepo domain.TenantRepository
	var idemRepo domain.IdempotencyRepository

	if pool != nil {
		periodRepo = repository.NewPostgresPeriodRepository(pool)
		txRepo = repository.NewPostgresTransactionRepository(pool)
		tenantRepo = repository.NewPostgresTenantRepository(pool)
		idemRepo = repository.NewPostgresIdempotencyRepository(pool)
	} else {
		log.Println("PostgreSQL connection unavailable; initializing in-memory fallback repositories.")
		periodRepo = repository.NewMockPeriodRepo()
		txRepo = repository.NewMockTransactionRepo()
		tenantRepo = repository.NewMockTenantRepo()
		idemRepo = repository.NewMockIdemRepo()
	}

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
