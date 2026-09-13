package handler

import (
	"os"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"

	"deftersystem/backend/internal/domain"
	"deftersystem/backend/internal/handler/middleware"
	"deftersystem/backend/internal/repository"
	"deftersystem/backend/internal/service"
)

// SetupRouter registers middleware, error handler, and API routes on the Fiber instance.
func SetupRouter(
	app *fiber.App,
	periodSvc domain.PeriodService,
	txSvc domain.TransactionService,
	periodRepo domain.PeriodRepository,
	txRepo domain.TransactionRepository,
	idemRepo domain.IdempotencyRepository,
	tenantServices ...interface{},
) {
	app.Use(recover.New())
	app.Use(logger.New(logger.Config{
		Format: "[${time}] ${status} - ${latency} ${method} ${path}\n",
	}))

	// Secure CORS configuration with restricted allowed origins
	allowedOrigins := os.Getenv("ALLOWED_ORIGINS")
	defaultOrigins := "http://localhost:3000,http://127.0.0.1:3000,http://localhost:8080,https://muhasebe-ve-finans-otomasyonu-2.onrender.com,https://www.oncuotogazmuhasebe.com.tr,https://oncuotogazmuhasebe.com.tr"
	if allowedOrigins == "" {
		allowedOrigins = defaultOrigins
	} else if !strings.Contains(allowedOrigins, "oncuotogazmuhasebe.com.tr") {
		allowedOrigins = allowedOrigins + ",https://www.oncuotogazmuhasebe.com.tr,https://oncuotogazmuhasebe.com.tr"
	}

	app.Use(cors.New(cors.Config{
		AllowOrigins:     allowedOrigins,
		AllowHeaders:     "Origin, Content-Type, Accept, Authorization, Idempotency-Key, X-Tenant-ID, X-User-ID, X-User-Role",
		AllowMethods:     "GET, POST, HEAD, PUT, DELETE, PATCH, OPTIONS",
		AllowCredentials: true,
		ExposeHeaders:    "X-Total-Count, X-Page, X-Limit, Content-Disposition",
	}))

	app.Get("/", func(c *fiber.Ctx) error {
		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"status":  "ok",
			"service": "deftersystem-backend",
			"message": "Deftersystem API is running",
		})
	})

	app.Get("/health", func(c *fiber.Ctx) error {
		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"status":  "ok",
			"service": "deftersystem-backend",
		})
	})

	periodH := NewPeriodHandler(periodSvc)
	txH := NewTransactionHandler(txSvc, txRepo)

	exportH := NewExportHandler(txRepo, periodRepo)
	importSvc := service.NewImportService(txRepo, periodRepo)
	importH := NewImportHandler(importSvc)

	var tenantSvc domain.TenantService
	var tenantRepo domain.TenantRepository
	var secRepo domain.UserSecurityRepository
	for _, arg := range tenantServices {
		if ts, ok := arg.(domain.TenantService); ok {
			tenantSvc = ts
		}
		if tr, ok := arg.(domain.TenantRepository); ok {
			tenantRepo = tr
		}
		if sr, ok := arg.(domain.UserSecurityRepository); ok {
			secRepo = sr
		}
	}

	if secRepo == nil {
		secRepo = repository.NewMockUserSecurityRepository()
	}
	authSvc := service.NewAuthService(secRepo)
	authH := NewAuthHandler(authSvc)

	api := app.Group("/api/v1")

	// Supabase JWT authentication & tenant membership verification middleware
	jwtSecret := os.Getenv("SUPABASE_JWT_SECRET")
	api.Use(middleware.AuthMiddleware(jwtSecret, tenantRepo))

	// Period routes
	periodsGroup := api.Group("/periods")
	periodsGroup.Get("/", periodH.ListPeriods)
	periodsGroup.Get("/history", periodH.GetPeriodHistory)
	periodsGroup.Get("/template/csv", exportH.DownloadSampleCSVTemplate)
	periodsGroup.Get("/:id/export/csv", exportH.ExportTransactionsCSV)
	periodsGroup.Get("/:id/export/excel", exportH.ExportTransactionsExcel)
	periodsGroup.Post("/:id/import/csv", importH.ImportTransactionsCSV)
	periodsGroup.Post("/open", middleware.IdempotencyMiddleware(idemRepo), periodH.OpenNextPeriod)
	periodsGroup.Post("/open-next", middleware.IdempotencyMiddleware(idemRepo), periodH.OpenNextPeriod)
	periodsGroup.Post("/:id/lock", middleware.IdempotencyMiddleware(idemRepo), periodH.LockPeriod)
	periodsGroup.Post("/:id/unlock", middleware.IdempotencyMiddleware(idemRepo), periodH.UnlockPeriod)
	periodsGroup.Get("/:id/summary", periodH.GetPeriodSummary)
	periodsGroup.Get("/:id/transactions", txH.ListTransactions)

	// Transaction routes
	txGroup := api.Group("/transactions")
	txGroup.Post("/", middleware.IdempotencyMiddleware(idemRepo), txH.CreateTransaction)
	txGroup.Post("/:id/reverse", middleware.IdempotencyMiddleware(idemRepo), txH.ReverseTransaction)

	// Auth & User Security routes
	authGroup := api.Group("/auth")
	authGroup.Post("/security-question", authH.SetSecurityQuestion)
	authGroup.Get("/security-question", authH.GetSecurityQuestion)
	authGroup.Post("/reset-password", authH.ResetPassword)

	// Tenant Member routes
	if tenantSvc != nil {
		tenantH := NewTenantHandler(tenantSvc)
		tenantGroup := api.Group("/tenants")
		tenantGroup.Get("/members", tenantH.ListMembers)
		tenantGroup.Post("/members", tenantH.AddMember)
		tenantGroup.Patch("/members/:user_id/role", tenantH.UpdateMemberRole)
		tenantGroup.Delete("/members/:user_id", tenantH.RemoveMember)
	}
}
