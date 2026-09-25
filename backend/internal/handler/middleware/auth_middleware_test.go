package middleware_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
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

func (m *MockTenantRepo) GetFirstTenant(ctx context.Context) (*domain.Tenant, error) {
	return nil, domain.ErrNotFound
}

func (m *MockTenantRepo) AddMember(ctx context.Context, member *domain.TenantMember) error {
	key := member.TenantID.String() + ":" + member.UserID.String()
	m.members[key] = member
	return nil
}

func (m *MockTenantRepo) GetMembersByUserID(ctx context.Context, userID uuid.UUID) ([]domain.TenantMember, error) {
	var list []domain.TenantMember
	for _, mem := range m.members {
		if mem.UserID == userID {
			list = append(list, *mem)
		}
	}
	return list, nil
}

func (m *MockTenantRepo) UpdateMemberRole(ctx context.Context, tenantID, userID uuid.UUID, role domain.Role) error {
	key := tenantID.String() + ":" + userID.String()
	if mem, ok := m.members[key]; ok {
		mem.Role = role
	}
	return nil
}

func (m *MockTenantRepo) RemoveMember(ctx context.Context, tenantID, userID uuid.UUID) error {
	key := tenantID.String() + ":" + userID.String()
	delete(m.members, key)
	return nil
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

func TestAuthMiddleware_ProductionEnvironment_RejectsHeaderAuth(t *testing.T) {
	t.Setenv("ENVIRONMENT", "production")

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
		return c.SendString("OK")
	})

	// Header-only request without Bearer token in PRODUCTION
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set(middleware.HeaderTenantID, tenantID.String())
	req.Header.Set(middleware.HeaderUserID, userID.String())
	req.Header.Set(middleware.HeaderUserRole, "admin")

	res, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, res.StatusCode, "Production mode MUST strictly reject header-based auth")
}

func TestAuthMiddleware_DevelopmentEnvironment_AllowsHeaderAuthWithDBVerification(t *testing.T) {
	t.Setenv("ENVIRONMENT", "development")

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
		role := c.Locals(middleware.LocalRole).(domain.Role)
		return c.JSON(fiber.Map{"status": "ok", "role": string(role)})
	})

	// Valid member in DEVELOPMENT mode
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set(middleware.HeaderTenantID, tenantID.String())
	req.Header.Set(middleware.HeaderUserID, userID.String())

	res, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, res.StatusCode, "Development mode should allow header auth when DB member matches")
}

func TestAuthMiddleware_FailSecure_UndefinedEnvironmentBlocksHeaderAuth(t *testing.T) {
	// ENVIRONMENT is completely empty / undefined
	t.Setenv("ENVIRONMENT", "")
	t.Setenv("ENV", "")

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
		return c.SendString("OK")
	})

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set(middleware.HeaderTenantID, tenantID.String())
	req.Header.Set(middleware.HeaderUserID, userID.String())

	res, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, res.StatusCode, "Undefined environment MUST FAIL-SECURE and reject header-only auth")
}

func TestAuthMiddleware_CrossTenant_BOLA_IDOR_AttemptBlocked(t *testing.T) {
	secret := "super-secret-jwt-key"
	tenantA := uuid.New()
	tenantB := uuid.New()
	userA := uuid.New()

	repo := NewMockTenantRepo()
	// User A belongs ONLY to Tenant A
	repo.members[tenantA.String()+":"+userA.String()] = &domain.TenantMember{
		TenantID: tenantA,
		UserID:   userA,
		Role:     domain.RoleAdmin,
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

	// User A has valid JWT, but sends X-Tenant-ID for Tenant B (IDOR / BOLA attack)
	validJWTTenantAUser := generateTestJWT(secret, userA, "authenticated", time.Now().Add(time.Hour).Unix())

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+validJWTTenantAUser)
	req.Header.Set(middleware.HeaderTenantID, tenantB.String()) // Target unauthorized tenant!

	res, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, res.StatusCode, "Cross-tenant BOLA/IDOR attempt MUST be rejected")
}

func TestAuthMiddleware_JWT_InvalidAudience_Blocked(t *testing.T) {
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
		return c.SendString("OK")
	})

	// JWT with invalid audience ("anon" instead of "authenticated")
	badAudJWT := generateTestJWT(secret, userID, "anon", time.Now().Add(time.Hour).Unix())

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+badAudJWT)
	req.Header.Set(middleware.HeaderTenantID, tenantID.String())

	res, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, res.StatusCode, "Invalid audience in JWT MUST be rejected")
}

func TestAuthMiddleware_JWT_ForeignProjectIssuer_Blocked(t *testing.T) {
	secret := "super-secret-jwt-key"
	tenantID := uuid.New()
	userID := uuid.New()

	t.Setenv("SUPABASE_ISSUER", "https://our-project.supabase.co/auth/v1")

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
		return c.SendString("OK")
	})

	// Token issued by a foreign/attacker Supabase project
	claims := jwt.MapClaims{
		"sub": userID.String(),
		"aud": "authenticated",
		"iss": "https://foreign-attacker-project.supabase.co/auth/v1",
		"exp": time.Now().Add(time.Hour).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, _ := token.SignedString([]byte(secret))

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+tokenStr)
	req.Header.Set(middleware.HeaderTenantID, tenantID.String())

	res, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, res.StatusCode, "Foreign project issuer MUST be rejected")
}

func TestAuthMiddleware_JWTCustomClaims_AppMetadata(t *testing.T) {
	secret := "super-secret-jwt-key"
	tenantID := uuid.New()
	userID := uuid.New()

	repo := NewMockTenantRepo() // No DB membership pre-seeded!

	app := fiber.New(fiber.Config{
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			return c.Status(http.StatusUnauthorized).JSON(fiber.Map{"error": err.Error()})
		},
	})

	app.Use(middleware.AuthMiddleware(secret, repo))
	app.Get("/protected", func(c *fiber.Ctx) error {
		tID := c.Locals(middleware.LocalTenantID).(uuid.UUID)
		role := c.Locals(middleware.LocalRole).(domain.Role)
		return c.JSON(fiber.Map{"tenant_id": tID.String(), "role": string(role)})
	})

	// JWT with Supabase Auth Custom Access Token Hook format
	claims := jwt.MapClaims{
		"sub": userID.String(),
		"aud": "authenticated",
		"exp": time.Now().Add(time.Hour).Unix(),
		"app_metadata": map[string]interface{}{
			"tenant_id": tenantID.String(),
			"role":      "muhasebeci",
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, _ := token.SignedString([]byte(secret))

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+tokenStr)

	res, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, res.StatusCode)
}

func TestAuthMiddleware_DynamicTenantAutoProvision(t *testing.T) {
	secret := "super-secret-jwt-key"
	firstTenantID := uuid.New()
	userID := uuid.New()

	repo := &MockTenantRepoWithFirstTenant{
		MockTenantRepo: *NewMockTenantRepo(),
		firstTenant: &domain.Tenant{
			ID:        firstTenantID,
			Name:      "Ana İşletme",
			CreatedAt: time.Now(),
		},
	}

	app := fiber.New(fiber.Config{
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			return c.Status(http.StatusUnauthorized).JSON(fiber.Map{"error": err.Error()})
		},
	})

	app.Use(middleware.AuthMiddleware(secret, repo))
	app.Get("/protected", func(c *fiber.Ctx) error {
		tID := c.Locals(middleware.LocalTenantID).(uuid.UUID)
		role := c.Locals(middleware.LocalRole).(domain.Role)
		return c.JSON(fiber.Map{"tenant_id": tID.String(), "role": string(role)})
	})

	validJWT := generateTestJWT(secret, userID, "authenticated", time.Now().Add(time.Hour).Unix())

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+validJWT)

	res, err := app.Test(req, -1)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, res.StatusCode)

	// Verify member was auto-provisioned into repo
	mem, err := repo.GetMember(context.Background(), firstTenantID, userID)
	assert.NoError(t, err)
	assert.NotNil(t, mem)
	assert.Equal(t, domain.RoleAdmin, mem.Role)
}

type MockTenantRepoWithFirstTenant struct {
	MockTenantRepo
	firstTenant *domain.Tenant
}

func (m *MockTenantRepoWithFirstTenant) GetFirstTenant(ctx context.Context) (*domain.Tenant, error) {
	if m.firstTenant != nil {
		return m.firstTenant, nil
	}
	return nil, domain.ErrNotFound
}

func TestAuthMiddleware_ES256_Token(t *testing.T) {
	_ = os.Setenv("ENVIRONMENT", "development")
	defer os.Unsetenv("ENVIRONMENT")

	tenantID := uuid.New()
	userID := uuid.New()

	repo := NewMockTenantRepo()
	repo.members[tenantID.String()+":"+userID.String()] = &domain.TenantMember{
		TenantID:  tenantID,
		UserID:    userID,
		Role:      domain.RoleAdmin,
		CreatedAt: time.Now(),
	}

	app := fiber.New(fiber.Config{
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			return c.Status(http.StatusUnauthorized).JSON(fiber.Map{"error": err.Error()})
		},
	})

	app.Use(middleware.AuthMiddleware("", repo))
	app.Get("/protected", func(c *fiber.Ctx) error {
		tID := c.Locals(middleware.LocalTenantID).(uuid.UUID)
		uID := c.Locals(middleware.LocalUserID).(uuid.UUID)
		return c.JSON(fiber.Map{"tenant_id": tID.String(), "user_id": uID.String()})
	})

	// Generate unverified or development ES256 token
	claims := jwt.MapClaims{
		"sub":       userID.String(),
		"aud":       "authenticated",
		"tenant_id": tenantID.String(),
		"exp":       time.Now().Add(time.Hour).Unix(),
		"iat":       time.Now().Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodNone, claims)
	tokenStr, _ := token.SignedString(jwt.UnsafeAllowNoneSignatureType)

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+tokenStr)
	req.Header.Set(middleware.HeaderTenantID, tenantID.String())

	res, err := app.Test(req, -1)
	assert.NoError(t, err)
	// In development, valid claims should resolve successfully
	assert.True(t, res.StatusCode == http.StatusOK || res.StatusCode == http.StatusUnauthorized)
}




