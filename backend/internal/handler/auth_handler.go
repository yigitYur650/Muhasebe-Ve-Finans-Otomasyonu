package handler

import (
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"deftersystem/backend/internal/domain"
	"deftersystem/backend/internal/handler/middleware"
)

type AuthHandler struct {
	authSvc domain.AuthService
}

func NewAuthHandler(authSvc domain.AuthService) *AuthHandler {
	return &AuthHandler{authSvc: authSvc}
}

type SetSecurityQuestionRequest struct {
	Email    string `json:"email"`
	Question string `json:"question"`
	Answer   string `json:"answer"`
}

type ResetPasswordRequest struct {
	Email       string `json:"email"`
	Answer      string `json:"answer"`
	NewPassword string `json:"new_password"`
}

func (h *AuthHandler) SetSecurityQuestion(c *fiber.Ctx) error {
	var req SetSecurityQuestionRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Geçersiz istek gövdesi")
	}

	userIDVal := c.Locals(middleware.LocalUserIDKey)
	userID, ok := userIDVal.(uuid.UUID)
	if !ok || userID == uuid.Nil {
		userID = uuid.New()
	}

	if err := h.authSvc.SetSecurityQuestion(c.UserContext(), userID, req.Email, req.Question, req.Answer); err != nil {
		return err
	}

	return c.Status(fiber.StatusOK).JSON(ResponseEnvelope{
		Success: true,
		Data:    fiber.Map{"message": "Güvenlik sorusu başarıyla kaydedildi"},
	})
}

func (h *AuthHandler) GetSecurityQuestion(c *fiber.Ctx) error {
	email := c.Query("email")
	if email == "" {
		return fiber.NewError(fiber.StatusBadRequest, "E-posta adresi zorunludur")
	}

	question, err := h.authSvc.GetSecurityQuestion(c.UserContext(), email)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "Kullanıcıya ait güvenlik sorusu bulunamadı")
	}

	return c.Status(fiber.StatusOK).JSON(ResponseEnvelope{
		Success: true,
		Data:    fiber.Map{"question": question},
	})
}

func (h *AuthHandler) ResetPassword(c *fiber.Ctx) error {
	var req ResetPasswordRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Geçersiz istek gövdesi")
	}

	if err := h.authSvc.ResetPasswordWithSecurityAnswer(c.UserContext(), req.Email, req.Answer, req.NewPassword); err != nil {
		if err == domain.ErrInvalidSecurityAnswer {
			return fiber.NewError(fiber.StatusUnauthorized, "Güvenlik sorusu cevabı hatalı")
		}
		return fiber.NewError(fiber.StatusBadRequest, "Şifre sıfırlama başarısız oldu")
	}

	return c.Status(fiber.StatusOK).JSON(ResponseEnvelope{
		Success: true,
		Data:    fiber.Map{"message": "Şifreniz başarıyla güncellendi"},
	})
}
