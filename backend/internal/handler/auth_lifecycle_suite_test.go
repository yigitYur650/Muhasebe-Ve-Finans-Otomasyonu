package handler_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"deftersystem/backend/internal/domain"
	"deftersystem/backend/internal/handler"
	"deftersystem/backend/internal/repository"
	"deftersystem/backend/internal/service"
)

func generateSignedJWT(secret string, userID uuid.UUID, tenantID *uuid.UUID, role domain.Role, exp int64) string {
	claims := jwt.MapClaims{
		"sub": userID.String(),
		"aud": "authenticated",
		"exp": exp,
		"iat": time.Now().Unix(),
	}
	if tenantID != nil {
		claims["app_metadata"] = map[string]interface{}{
			"tenant_id": tenantID.String(),
			"role":      string(role),
		}
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, _ := token.SignedString([]byte(secret))
	return tokenStr
}

func setupTestAppWithRepositories(t *testing.T) (*fiber.App, *repository.MockTenantRepo, *repository.MockPeriodRepo, *repository.MockTransactionRepo, *repository.MockUserSecurityRepository, string) {
	jwtSecret := "test-jwt-secret-for-lifecycle-scenarios-32bytes"
	t.Setenv("SUPABASE_JWT_SECRET", jwtSecret)
	t.Setenv("ENVIRONMENT", "development")

	tenantRepo := repository.NewMockTenantRepo()
	periodRepo := repository.NewMockPeriodRepo()
	txRepo := repository.NewMockTransactionRepo()
	idemRepo := repository.NewMockIdemRepo()
	secRepo := repository.NewMockUserSecurityRepository()
	supplierRepo := repository.NewMockSupplierRepository()

	periodSvc := service.NewPeriodService(periodRepo, tenantRepo, txRepo)
	txSvc := service.NewTransactionService(txRepo, periodRepo)
	tenantSvc := service.NewTenantService(tenantRepo)
	supplierSvc := service.NewSupplierService(supplierRepo, periodRepo)

	app := fiber.New(fiber.Config{
		ErrorHandler: handler.CustomErrorHandler,
	})

	handler.SetupRouter(app, periodSvc, txSvc, periodRepo, txRepo, idemRepo, tenantSvc, tenantRepo, secRepo, supplierSvc, supplierRepo)

	return app, tenantRepo, periodRepo, txRepo, secRepo, jwtSecret
}

// -----------------------------------------------------------------------------
// T1: İkinci Kullanıcı (Çalışan) Kaydı Doğru Tenant'a Bağlanıyor mu?
// -----------------------------------------------------------------------------
func TestLifecycle_T1_SecondUserConnectsToSameTenant(t *testing.T) {
	app, tenantRepo, periodRepo, _, _, secret := setupTestAppWithRepositories(t)

	// Patron ve Ana İşletme Kurulumu
	mainTenant, err := tenantRepo.GetFirstTenant(context.Background())
	require.NoError(t, err)
	bossUserID := uuid.New()
	require.NoError(t, tenantRepo.AddMember(context.Background(), &domain.TenantMember{
		ID:        uuid.New(),
		TenantID:  mainTenant.ID,
		UserID:    bossUserID,
		Role:      domain.RoleAdmin,
		CreatedAt: time.Now(),
	}))

	// Patronun Mevcut Bir Dönemi ve Bakiyesi Var
	periodID := uuid.New()
	periodRepo.Create(context.Background(), &domain.Period{
		ID:              periodID,
		TenantID:        mainTenant.ID,
		Label:           "2026-09",
		StartingBalance: decimal.NewFromInt(10000),
		Status:          domain.PeriodStatusOpen,
		OpenedAt:        time.Now(),
	})

	// İkinci Kullanıcı (Çalışan / Muhasebeci) Kayıt Olup İstek Atıyor
	employeeUserID := uuid.New()
	employeeJWT := generateSignedJWT(secret, employeeUserID, nil, domain.RoleStandart, time.Now().Add(time.Hour).Unix())

	req := httptest.NewRequest(http.MethodGet, "/api/v1/periods/", nil)
	req.Header.Set("Authorization", "Bearer "+employeeJWT)

	res, err := app.Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, res.StatusCode, "Yeni kullanıcı isteği başarılı olmalıdır")

	var env handler.ResponseEnvelope
	require.NoError(t, json.NewDecoder(res.Body).Decode(&env))
	assert.True(t, env.Success)

	// Çalışanın patron ile aynı tenant'a bağlandığı doğrulanır
	members, err := tenantRepo.GetMembersByUserID(context.Background(), employeeUserID)
	require.NoError(t, err)
	require.NotEmpty(t, members, "Çalışan için tenant_members kaydı oluşmalıdır")
	assert.Equal(t, mainTenant.ID, members[0].TenantID, "Çalışan ana işletmeye bağlı olmalıdır")
}

// -----------------------------------------------------------------------------
// T2: Otomatik Admin Rolünün Sonradan Düşürülebilmesi
// -----------------------------------------------------------------------------
func TestLifecycle_T2_RoleDowngradeEffectiveImmediately(t *testing.T) {
	app, tenantRepo, periodRepo, _, _, secret := setupTestAppWithRepositories(t)

	mainTenant, _ := tenantRepo.GetFirstTenant(context.Background())
	bossUserID := uuid.New()
	employeeUserID := uuid.New()

	_ = tenantRepo.AddMember(context.Background(), &domain.TenantMember{
		ID:        uuid.New(),
		TenantID:  mainTenant.ID,
		UserID:    bossUserID,
		Role:      domain.RoleAdmin,
		CreatedAt: time.Now(),
	})

	// Çalışan başlangıçta admin olarak eklendi
	_ = tenantRepo.AddMember(context.Background(), &domain.TenantMember{
		ID:        uuid.New(),
		TenantID:  mainTenant.ID,
		UserID:    employeeUserID,
		Role:      domain.RoleAdmin,
		CreatedAt: time.Now(),
	})

	// Patron çalışanın rolünü 'standart' rolüne düşürüyor
	tenantSvc := service.NewTenantService(tenantRepo)
	err := tenantSvc.UpdateMemberRole(context.Background(), mainTenant.ID, bossUserID, employeeUserID, domain.RoleStandart)
	require.NoError(t, err, "Patron çalışanın rolünü başarıyla değiştirebilmelidir")

	// Açık dönem oluşturalım
	pID := uuid.New()
	periodRepo.Create(context.Background(), &domain.Period{
		ID:              pID,
		TenantID:        mainTenant.ID,
		Label:           "2026-09",
		StartingBalance: decimal.Zero,
		Status:          domain.PeriodStatusOpen,
		OpenedAt:        time.Now(),
	})

	// Çalışan Admin yetkisi gerektiren Dönem Kilitleme işlemini dener
	employeeJWT := generateSignedJWT(secret, employeeUserID, nil, domain.RoleStandart, time.Now().Add(time.Hour).Unix())
	req := httptest.NewRequest(http.MethodPost, "/api/v1/periods/"+pID.String()+"/lock", nil)
	req.Header.Set("Authorization", "Bearer "+employeeJWT)

	res, err := app.Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, http.StatusForbidden, res.StatusCode, "Rolü düşürülen çalışan admin işlemlerinden anında men edilmelidir (HTTP 403)")
}

// -----------------------------------------------------------------------------
// T3: Son Admin Koruması
// -----------------------------------------------------------------------------
func TestLifecycle_T3_LastAdminProtection(t *testing.T) {
	_, tenantRepo, _, _, _, _ := setupTestAppWithRepositories(t)

	mainTenant, _ := tenantRepo.GetFirstTenant(context.Background())
	bossUserID := uuid.New()

	_ = tenantRepo.AddMember(context.Background(), &domain.TenantMember{
		ID:        uuid.New(),
		TenantID:  mainTenant.ID,
		UserID:    bossUserID,
		Role:      domain.RoleAdmin,
		CreatedAt: time.Now(),
	})

	tenantSvc := service.NewTenantService(tenantRepo)

	// Tek admin olan patron kendi rolünü standart yapmayı dener
	err := tenantSvc.UpdateMemberRole(context.Background(), mainTenant.ID, bossUserID, bossUserID, domain.RoleStandart)
	assert.ErrorIs(t, err, domain.ErrCannotRemoveLastAdmin, "Son admin rolünü düşüremez")

	// Tek admin olan patron kendini silmeyi dener
	err = tenantSvc.RemoveMember(context.Background(), mainTenant.ID, bossUserID, bossUserID)
	assert.ErrorIs(t, err, domain.ErrCannotRemoveLastAdmin, "Son admin silinemez")
}

// -----------------------------------------------------------------------------
// T4: Standart Başarılı Kayıt ve Boş Dönem Güvenliği
// -----------------------------------------------------------------------------
func TestLifecycle_T4_FirstUserRegistrationAndNoPeriodSafety(t *testing.T) {
	app, _, _, _, _, secret := setupTestAppWithRepositories(t)

	newBossID := uuid.New()
	validJWT := generateSignedJWT(secret, newBossID, nil, domain.RoleAdmin, time.Now().Add(time.Hour).Unix())

	// Henüz hiç dönem yokken dönem listesi çekilir
	req := httptest.NewRequest(http.MethodGet, "/api/v1/periods/", nil)
	req.Header.Set("Authorization", "Bearer "+validJWT)

	res, err := app.Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, res.StatusCode)

	var env handler.ResponseEnvelope
	require.NoError(t, json.NewDecoder(res.Body).Decode(&env))
	assert.True(t, env.Success)
}

// -----------------------------------------------------------------------------
// T5: Şifre Sıfırlama (Güvenlik Sorusu & Bcrypt)
// -----------------------------------------------------------------------------
func TestLifecycle_T5_SecurityQuestionPasswordReset(t *testing.T) {
	_, _, _, _, secRepo, _ := setupTestAppWithRepositories(t)
	authSvc := service.NewAuthService(secRepo)

	userID := uuid.New()
	userEmail := "patron@oncuotogaz.com"
	ctx := context.Background()

	// Güvenlik sorusu kaydetme
	err := authSvc.SetSecurityQuestion(ctx, userID, userEmail, "İlk evcil hayvanınızın adı nedir?", "Pamuk")
	require.NoError(t, err)

	// Yanlış cevap ile sıfırlama denemesi (Generic Hata / ErrInvalidSecurityAnswer)
	err = authSvc.ResetPasswordWithSecurityAnswer(ctx, userEmail, "Karabaş", "YeniGucluSifre123!")
	assert.ErrorIs(t, err, domain.ErrInvalidSecurityAnswer, "Yanlış cevap generic hata ile reddedilmelidir")

	// Doğru cevap ile sıfırlama denemesi (Büyük/Küçük harf toleransı)
	err = authSvc.ResetPasswordWithSecurityAnswer(ctx, userEmail, "pamuk", "YeniGucluSifre123!")
	assert.NoError(t, err, "Doğru cevap ile şifre sıfırlanmalıdır")
}

// -----------------------------------------------------------------------------
// T6: Zayıf Şifre ve Geçersiz Veri Doğrulaması
// -----------------------------------------------------------------------------
func TestLifecycle_T6_WeakPasswordAndInvalidInputHandling(t *testing.T) {
	_, _, _, _, secRepo, _ := setupTestAppWithRepositories(t)
	authSvc := service.NewAuthService(secRepo)

	userID := uuid.New()
	userEmail := "patron@oncuotogaz.com"
	ctx := context.Background()

	_ = authSvc.SetSecurityQuestion(ctx, userID, userEmail, "Soru", "Cevap")

	// 6 karakterden kısa zayıf şifre reddedilir
	err := authSvc.ResetPasswordWithSecurityAnswer(ctx, userEmail, "Cevap", "123")
	assert.Error(t, err, "Kısa/Zayıf şifre reddedilmelidir")
}

// -----------------------------------------------------------------------------
// T7: Kayıt Sırasında Gecikme / Ağ Kesintisi (Orphan User Auto-Provisioning)
// -----------------------------------------------------------------------------
func TestLifecycle_T7_OrphanUserAutoProvisioning(t *testing.T) {
	app, tenantRepo, _, _, _, secret := setupTestAppWithRepositories(t)

	// Veritabanında henüz kaydı olmayan (orphan) Supabase kullanıcısı
	orphanUserID := uuid.New()
	orphanJWT := generateSignedJWT(secret, orphanUserID, nil, domain.RoleAdmin, time.Now().Add(time.Hour).Unix())

	req := httptest.NewRequest(http.MethodGet, "/api/v1/suppliers", nil)
	req.Header.Set("Authorization", "Bearer "+orphanJWT)

	res, err := app.Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, res.StatusCode, "Orphan kullanıcı için otomatik provision yapılmalıdır")

	// DB kaydının oluştuğu teyit edilir
	members, err := tenantRepo.GetMembersByUserID(context.Background(), orphanUserID)
	require.NoError(t, err)
	assert.NotEmpty(t, members)
}

// -----------------------------------------------------------------------------
// T8: Çoklu Dil (tr/en) JSON Sözlük Eşitliği
// -----------------------------------------------------------------------------
func TestLifecycle_T8_LanguageDictionariesIntegrity(t *testing.T) {
	trData, err := os.ReadFile("../../frontend/src/messages/tr.json")
	if err != nil {
		trData, err = os.ReadFile("../../../frontend/src/messages/tr.json")
	}
	require.NoError(t, err, "tr.json okunabilmelidir")

	enData, err := os.ReadFile("../../frontend/src/messages/en.json")
	if err != nil {
		enData, err = os.ReadFile("../../../frontend/src/messages/en.json")
	}
	require.NoError(t, err, "en.json okunabilmelidir")

	var trMap, enMap map[string]interface{}
	require.NoError(t, json.Unmarshal(trData, &trMap))
	require.NoError(t, json.Unmarshal(enData, &enMap))

	// Temel namespace kontrolü
	for _, ns := range []string{"common", "auth", "period", "transaction", "suppliers", "navigation"} {
		assert.Contains(t, trMap, ns, "tr.json namespace içermeli: "+ns)
		assert.Contains(t, enMap, ns, "en.json namespace içermeli: "+ns)
	}
}

// -----------------------------------------------------------------------------
// T9: Oturum Süresi Dolduğunda Otomatik Red (Expired Token)
// -----------------------------------------------------------------------------
func TestLifecycle_T9_ExpiredTokenAutomaticRejection(t *testing.T) {
	app, _, _, _, _, secret := setupTestAppWithRepositories(t)

	expiredUserID := uuid.New()
	// 2 saat önce süresi dolmuş token
	expiredJWT := generateSignedJWT(secret, expiredUserID, nil, domain.RoleAdmin, time.Now().Add(-2*time.Hour).Unix())

	req := httptest.NewRequest(http.MethodGet, "/api/v1/periods/", nil)
	req.Header.Set("Authorization", "Bearer "+expiredJWT)

	res, err := app.Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, http.StatusForbidden, res.StatusCode, "Süresi dolmuş token kesinlikle reddedilmelidir (HTTP 403 UNAUTHORIZED)")
}
