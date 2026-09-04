package middleware_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"deftersystem/backend/internal/domain"
	"deftersystem/backend/internal/handler/middleware"
)

type MockTenantRepo struct {
	members map[string]*domain.TenantMember
}

func NewMockTenantRepo() *MockTenantRepo {
	return &MockTenantRepo{members: make(map[string]*domain.TenantMember)}
}

func (m *MockTenantRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Tenant, error) {
	return nil, domain.ErrNotFound
}

func (m *MockTenantRepo) Create(ctx context.Context, tenant *domain.Tenant) error {
	return nil
}

func (m *MockTenantRepo) GetMember(ctx context.Context, tenantID, userID uuid.UUID) (*domain.TenantMember, error) {
	key := tenantID.String() + ":" + userID.String()
	if val, ok := m.members[key]; ok {
		return val, nil
	}
	return nil, domain.ErrNotFound
}

func (m *MockTenantRepo) GetMembersByTenantID(ctx context.Context, tenantID uuid.UUID) ([]domain.TenantMember, error) {
	return nil, nil
}

func generateTestJWT(secret string, userID uuid.UUID, aud string, exp int64) string {
	claims := jwt.MapClaims{
		"sub": userID.String(),
		"aud": aud,
		"exp": exp,
		"iat": time.Now().Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, _ := token.SignedString([]byte(secret))
	return tokenStr
}

func TestAuthMiddleware_ValidToken(t *testing.T) {
	secret := "super-secret-jwt-key"
	tenantID := uuid.New()
	userID := uuid.New()

	repo := NewMockTenantRepo()
	repo.members[tenantID.String()+":"+userID.String()] = &domain.TenantMember{
		TenantID: tenantID,
		UserID:   userID,
		Role:     domain.RoleAdmin,
	}

	app := fiber.New(fiber.Config{
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			return c.Status(http.StatusUnauthorized).JSON(fiber.Map{"error": err.Error()})
		},
	})

	app.Use(middleware.AuthMiddleware(secret, repo))
	app.Get("/protected", func(c *fiber.Ctx) error {
		uID := c.Locals(middleware.LocalUserID).(uuid.UUID)
		role := c.Locals(middleware.LocalRole).(domain.Role)
		return c.JSON(fiber.Map{"user_id": uID.String(), "role": string(role)})
	})

	validJWT := generateTestJWT(secret, userID, "authenticated", time.Now().Add(time.Hour).Unix())

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+validJWT)
	req.Header.Set(middleware.HeaderTenantID, tenantID.String())

	res, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, res.StatusCode)
}

func TestAuthMiddleware_ExpiredToken(t *testing.T) {
	secret := "super-secret-jwt-key"
	tenantID := uuid.New()
	userID := uuid.New()

	repo := NewMockTenantRepo()

	app := fiber.New(fiber.Config{
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			return c.Status(http.StatusUnauthorized).JSON(fiber.Map{"error": err.Error()})
		},
	})

	app.Use(middleware.AuthMiddleware(secret, repo))
	app.Get("/protected", func(c *fiber.Ctx) error {
		return c.SendString("OK")
	})

	// Expired 1 hour ago
	expiredJWT := generateTestJWT(secret, userID, "authenticated", time.Now().Add(-time.Hour).Unix())

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+expiredJWT)
	req.Header.Set(middleware.HeaderTenantID, tenantID.String())

	res, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, res.StatusCode)
}

func TestAuthMiddleware_TamperedSignature(t *testing.T) {
	correctSecret := "super-secret-jwt-key"
	wrongSecret := "hacker-wrong-secret"
	tenantID := uuid.New()
	userID := uuid.New()

	repo := NewMockTenantRepo()

	app := fiber.New(fiber.Config{
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			return c.Status(http.StatusUnauthorized).JSON(fiber.Map{"error": err.Error()})
		},
	})

	app.Use(middleware.AuthMiddleware(correctSecret, repo))
	app.Get("/protected", func(c *fiber.Ctx) error {
		return c.SendString("OK")
	})

	tamperedJWT := generateTestJWT(wrongSecret, userID, "authenticated", time.Now().Add(time.Hour).Unix())

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+tamperedJWT)
	req.Header.Set(middleware.HeaderTenantID, tenantID.String())

	res, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, res.StatusCode)
}

func TestAuthMiddleware_NonMemberTenantAccess(t *testing.T) {
	secret := "super-secret-jwt-key"
	tenantID := uuid.New()
	otherTenantID := uuid.New()
	userID := uuid.New()

	repo := NewMockTenantRepo()
	// Member of tenantID, NOT member of otherTenantID
	repo.members[tenantID.String()+":"+userID.String()] = &domain.TenantMember{
		TenantID: tenantID,
		UserID:   userID,
		Role:     domain.RoleStandart,
	}

	app := fiber.New(fiber.Config{
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			return c.Status(http.StatusUnauthorized).JSON(fiber.Map{"error": err.Error()})
		},
	})

	app.Use(middleware.AuthMiddleware(secret, repo))
	app.Get("/protected", func(c *fiber.Ctx) error {
		return c.SendString("OK")
	})

	validJWT := generateTestJWT(secret, userID, "authenticated", time.Now().Add(time.Hour).Unix())

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+validJWT)
	req.Header.Set(middleware.HeaderTenantID, otherTenantID.String()) // Target unauthorized tenant!

	res, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, res.StatusCode, "Non-member tenant access MUST be rejected")
}
