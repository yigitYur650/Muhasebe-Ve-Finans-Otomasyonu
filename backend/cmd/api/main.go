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
	if dbURL == "" || dbURL == "postgres://postgres:[YOUR-DB-PASSWORD]@db.[YOUR-PROJECT-REF].supabase.co:5432/postgres?sslmode=require" || len(dbURL) < 10 {
		dbURL = "postgres://postgres.xtmfsdvwlminlchpustb:6uNlbk0wlN5TuSDZ@aws-0-ap-northeast-1.pooler.supabase.com:6543/postgres?sslmode=require"
	}
	// Fallback IPv6 to IPv4 pooler replacement
	if strings.Contains(dbURL, "db.xtmfsdvwlminlchpustb.supabase.co") {
		dbURL = "postgres://postgres.xtmfsdvwlminlchpustb:6uNlbk0wlN5TuSDZ@aws-0-ap-northeast-1.pooler.supabase.com:6543/postgres?sslmode=require"
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
