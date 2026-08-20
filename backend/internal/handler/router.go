package handler

import (
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/recover"

	"deftersystem/backend/internal/domain"
	"deftersystem/backend/internal/handler/middleware"
)

// SetupRouter registers middleware, error handler, and API routes on the Fiber instance.
func SetupRouter(
	app *fiber.App,
	periodSvc domain.PeriodService,
	txSvc domain.TransactionService,
	txRepo domain.TransactionRepository,
	idemRepo domain.IdempotencyRepository,
	tenantSvc ...domain.TenantService,
) {
	app.Use(recover.New())

	app.Get("/health", func(c *fiber.Ctx) error {
		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"status":  "ok",
			"service": "deftersystem-backend",
		})
	})

	periodH := NewPeriodHandler(periodSvc)
	txH := NewTransactionHandler(txSvc, txRepo)

	api := app.Group("/api/v1")

	// Context middleware for extracting X-Tenant-ID, X-User-ID, X-User-Role
	api.Use(middleware.ContextMiddleware())

	// Period routes
	periodsGroup := api.Group("/periods")
	periodsGroup.Get("/", periodH.ListPeriods)
	periodsGroup.Post("/open", middleware.IdempotencyMiddleware(idemRepo), periodH.OpenNextPeriod)
	periodsGroup.Post("/:id/lock", middleware.IdempotencyMiddleware(idemRepo), periodH.LockPeriod)
	periodsGroup.Get("/:id/summary", periodH.GetPeriodSummary)
	periodsGroup.Get("/:id/transactions", txH.ListTransactions)

	// Transaction routes
	txGroup := api.Group("/transactions")
	txGroup.Post("/", middleware.IdempotencyMiddleware(idemRepo), txH.CreateTransaction)
	txGroup.Post("/:id/reverse", middleware.IdempotencyMiddleware(idemRepo), txH.ReverseTransaction)

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
