package handler

import (
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
	tenantSvc ...domain.TenantService,
) {
	app.Use(recover.New())
	app.Use(logger.New(logger.Config{
		Format: "[${time}] ${status} - ${latency} ${method} ${path}\n",
	}))

	// Dynamic CORS configuration accepting any frontend origin with credentials
	app.Use(cors.New(cors.Config{
		AllowOriginsFunc: func(origin string) bool {
			return true
		},
		AllowHeaders:     "Origin, Content-Type, Accept, Authorization, Idempotency-Key, X-Tenant-ID, X-User-ID, X-User-Role",
		AllowMethods:     "GET, POST, HEAD, PUT, DELETE, PATCH, OPTIONS",
		AllowCredentials: true,
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

	secRepo := repository.NewMockUserSecurityRepository()
	authSvc := service.NewAuthService(secRepo)
	authH := NewAuthHandler(authSvc)

	api := app.Group("/api/v1")

	// Context middleware for extracting X-Tenant-ID, X-User-ID, X-User-Role
	api.Use(middleware.ContextMiddleware())

	// Period routes
	periodsGroup := api.Group("/periods")
	periodsGroup.Get("/", periodH.ListPeriods)
	periodsGroup.Get("/history", periodH.GetPeriodHistory)
	periodsGroup.Get("/template/csv", exportH.DownloadSampleCSVTemplate)
	periodsGroup.Get("/:id/export/csv", exportH.ExportTransactionsCSV)
	periodsGroup.Post("/:id/import/csv", importH.ImportTransactionsCSV)
	periodsGroup.Post("/open", middleware.IdempotencyMiddleware(idemRepo), periodH.OpenNextPeriod)
	periodsGroup.Post("/:id/lock", middleware.IdempotencyMiddleware(idemRepo), periodH.LockPeriod)
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
	if len(tenantSvc) > 0 && tenantSvc[0] != nil {
		tenantH := NewTenantHandler(tenantSvc[0])
		tenantGroup := api.Group("/tenants")
		tenantGroup.Get("/members", tenantH.ListMembers)
		tenantGroup.Post("/members", tenantH.AddMember)
		tenantGroup.Patch("/members/:user_id/role", tenantH.UpdateMemberRole)
		tenantGroup.Delete("/members/:user_id", tenantH.RemoveMember)
	}
}
