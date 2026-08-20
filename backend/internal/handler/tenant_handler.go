package handler

import (
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"deftersystem/backend/internal/domain"
	"deftersystem/backend/internal/handler/middleware"
)

type TenantHandler struct {
	tenantSvc domain.TenantService
}

func NewTenantHandler(tenantSvc domain.TenantService) *TenantHandler {
	return &TenantHandler{
		tenantSvc: tenantSvc,
	}
}

type AddMemberRequest struct {
	UserID string      `json:"user_id"`
	Role   domain.Role `json:"role"`
}

type UpdateRoleRequest struct {
	Role domain.Role `json:"role"`
}

func (h *TenantHandler) ListMembers(c *fiber.Ctx) error {
	tenantID, err := getTenantIDFromLocals(c)
	if err != nil {
		return domain.ErrUnauthorized
	}

	members, err := h.tenantSvc.ListMembers(c.Context(), tenantID)
	if err != nil {
		return err
	}

	return c.Status(fiber.StatusOK).JSON(ResponseEnvelope{
		Success: true,
		Data:    members,
	})
}

func (h *TenantHandler) AddMember(c *fiber.Ctx) error {
	tenantID, requestingUserID, err := getContextCredentials(c)
	if err != nil {
		return domain.ErrUnauthorized
	}

	var req AddMemberRequest
	if err := c.BodyParser(&req); err != nil {
		return domain.ErrInvalidRole
	}

	targetUserID, err := uuid.Parse(req.UserID)
	if err != nil {
		return domain.ErrNotFound
	}

	if err := h.tenantSvc.AddMember(c.Context(), tenantID, requestingUserID, targetUserID, req.Role); err != nil {
		return err
	}

	return c.Status(fiber.StatusCreated).JSON(ResponseEnvelope{
		Success: true,
		Data:    fiber.Map{"message": "Üye başarıyla eklendi"},
	})
}

func (h *TenantHandler) UpdateMemberRole(c *fiber.Ctx) error {
	tenantID, requestingUserID, err := getContextCredentials(c)
	if err != nil {
		return domain.ErrUnauthorized
	}

	targetUserStr := c.Params("user_id")
	targetUserID, err := uuid.Parse(targetUserStr)
	if err != nil {
		return domain.ErrNotFound
	}

	var req UpdateRoleRequest
	if err := c.BodyParser(&req); err != nil {
		return domain.ErrInvalidRole
	}

	if err := h.tenantSvc.UpdateMemberRole(c.Context(), tenantID, requestingUserID, targetUserID, req.Role); err != nil {
		return err
	}

	return c.Status(fiber.StatusOK).JSON(ResponseEnvelope{
		Success: true,
		Data:    fiber.Map{"message": "Üye rolü güncellendi"},
	})
}

func (h *TenantHandler) RemoveMember(c *fiber.Ctx) error {
	tenantID, requestingUserID, err := getContextCredentials(c)
	if err != nil {
		return domain.ErrUnauthorized
	}

	targetUserStr := c.Params("user_id")
	targetUserID, err := uuid.Parse(targetUserStr)
	if err != nil {
		return domain.ErrNotFound
	}

	if err := h.tenantSvc.RemoveMember(c.Context(), tenantID, requestingUserID, targetUserID); err != nil {
		return err
	}

	return c.Status(fiber.StatusOK).JSON(ResponseEnvelope{
		Success: true,
		Data:    fiber.Map{"message": "Üye tenant'tan çıkarıldı"},
	})
}

func getContextCredentials(c *fiber.Ctx) (uuid.UUID, uuid.UUID, error) {
	tenantID, errT := getTenantIDFromLocals(c)
	userID, errU := getUserIDFromLocals(c)
	if errT != nil || errU != nil {
		return uuid.Nil, uuid.Nil, domain.ErrUnauthorized
	}
	return tenantID, userID, nil
}

func getUserIDFromLocals(c *fiber.Ctx) (uuid.UUID, error) {
	if val := c.Locals(middleware.LocalUserID); val != nil {
		if id, ok := val.(uuid.UUID); ok {
			return id, nil
		}
	}
	userHeader := c.Get(middleware.HeaderUserID)
	if userHeader != "" {
		return uuid.Parse(userHeader)
	}
	return uuid.Nil, domain.ErrUnauthorized
}

func getTenantIDFromLocals(c *fiber.Ctx) (uuid.UUID, error) {
	if val := c.Locals(middleware.LocalTenantID); val != nil {
		if id, ok := val.(uuid.UUID); ok {
			return id, nil
		}
	}
	tenantHeader := c.Get(middleware.HeaderTenantID)
	if tenantHeader != "" {
		return uuid.Parse(tenantHeader)
	}
	return uuid.Nil, domain.ErrUnauthorized
}
