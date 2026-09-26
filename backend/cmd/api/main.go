package main

import (
	"log"
	"os"
	"strings"

	"github.com/gofiber/fiber/v2"

	"deftersystem/backend/internal/handler"
	"deftersystem/backend/internal/repository"
	"deftersystem/backend/internal/service"
	"deftersystem/backend/pkg/scheduler"
	"deftersystem/backend/pkg/telegram"
)

func main() {
	app := fiber.New(fiber.Config{
		AppName:        "Deftersystem API v1.0",
		ErrorHandler:   handler.CustomErrorHandler,
		ReadBufferSize: 16384, // 16KB header buffer to support modern JWT and auth cookies safely
	})

	// Load all variables from .env / .env.local file into runtime environment
	envPaths := []string{".env.local", "../.env.local", ".env", "../.env", "backend/.env"}
	for _, envPath := range envPaths {
		if data, err := os.ReadFile(envPath); err == nil {
			for _, line := range strings.Split(string(data), "\n") {
				line = strings.TrimSpace(line)
				if line == "" || strings.HasPrefix(line, "#") {
					continue
				}
				parts := strings.SplitN(line, "=", 2)
				if len(parts) == 2 {
					key := strings.TrimSpace(parts[0])
					val := strings.Trim(strings.TrimSpace(parts[1]), `"'`)
					if os.Getenv(key) == "" {
						_ = os.Setenv(key, val)
					}
				}
			}
			break
		}
	}

	// Initialize Telegram Alert & Notification Notifier
	tg := telegram.InitGlobal()
	if tg.IsEnabled() {
		log.Println("🔔 Telegram Alerting System is ENABLED (Real-time alarm dispatch active)")
	} else {
		log.Println("ℹ️  Telegram Alerting System is DISABLED (Set TELEGRAM_BOT_TOKEN & TELEGRAM_CHAT_ID to enable)")
	}

	dbURL := os.Getenv("DATABASE_URL")

	log.Printf("Connecting to live PostgreSQL database...")
	pool, err := repository.NewPostgresPool(dbURL)
	if err != nil || pool == nil {
		log.Fatalf("FATAL: Failed to connect to PostgreSQL database pool: %v. Server will not start in mock fallback mode.", err)
	}
	defer pool.Close()
	log.Println("Successfully connected to PostgreSQL database pool!")

	periodRepo := repository.NewPostgresPeriodRepository(pool)
	txRepo := repository.NewPostgresTransactionRepository(pool)
	tenantRepo := repository.NewPostgresTenantRepository(pool)
	idemRepo := repository.NewPostgresIdempotencyRepository(pool)
	secRepo := repository.NewPostgresUserSecurityRepository(pool)
	supplierRepo := repository.NewPostgresSupplierRepository(pool)

	periodSvc := service.NewPeriodService(periodRepo, tenantRepo, txRepo)
	txSvc := service.NewTransactionService(txRepo, periodRepo)
	tenantSvc := service.NewTenantService(tenantRepo)
	supplierSvc := service.NewSupplierService(supplierRepo, periodRepo)

	handler.SetupRouter(app, periodSvc, txSvc, periodRepo, txRepo, idemRepo, tenantSvc, tenantRepo, secRepo, supplierSvc, supplierRepo)

	// Start Automated 3-2-1 Daily Backup Scheduler (Runs daily at 03:00)
	backupOrchestrator := scheduler.NewBackupOrchestrator(pool)
	backupOrchestrator.StartDailyCron()

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Live PostgreSQL API Server listening on :%s", port)
	if err := app.Listen(":" + port); err != nil {
		log.Fatalf("Error starting server: %v", err)
	}
}
