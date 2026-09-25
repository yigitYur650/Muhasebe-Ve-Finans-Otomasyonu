package handler_test

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"deftersystem/backend/internal/domain"
	"deftersystem/backend/internal/service"
)

// -----------------------------------------------------------------------------
// Y.1 — Yıl Sonu Veri Hacmi Stres Testi (10.000 İşlem Kaydı & Excel Export)
// -----------------------------------------------------------------------------
func TestStress_Y1_YearEndVolumeBenchmark(t *testing.T) {
	app, tenantRepo, periodRepo, txRepo, _, secret := setupTestAppWithRepositories(t)

	mainTenant, err := tenantRepo.GetFirstTenant(context.Background())
	require.NoError(t, err)
	adminUserID := uuid.New()
	jwtToken := generateSignedJWT(secret, adminUserID, &mainTenant.ID, domain.RoleAdmin, time.Now().Add(time.Hour).Unix())

	periodID := uuid.New()
	periodRepo.Create(context.Background(), &domain.Period{
		ID:              periodID,
		TenantID:        mainTenant.ID,
		Label:           "2026-12",
		StartingBalance: decimal.NewFromFloat(100000.00),
		Status:          domain.PeriodStatusOpen,
		OpenedAt:        time.Now(),
	})

	// 10.000 adet gerçekçi işlem kaydı üretiliyor (12 aylık yoğun ticari kasa hacmi)
	totalTxCount := 10000
	var expectedIn, expectedOut decimal.Decimal
	startTime := time.Now()

	for i := 1; i <= totalTxCount; i++ {
		amount := decimal.NewFromFloat(float64(i%500) + 0.50)
		dir := domain.DirectionIn
		if i%2 == 0 {
			dir = domain.DirectionOut
			expectedOut = expectedOut.Add(amount)
		} else {
			expectedIn = expectedIn.Add(amount)
		}

		desc := fmt.Sprintf("Yıl sonu stres testi işlem kaydı #%d", i)
		tx := &domain.Transaction{
			ID:          uuid.New(),
			TenantID:    mainTenant.ID,
			PeriodID:    periodID,
			Direction:   dir,
			Channel:     domain.ChannelEft,
			Amount:      amount,
			Description: &desc,
			CreatedBy:   adminUserID,
			CreatedAt:   time.Now(),
		}
		_ = txRepo.Create(context.Background(), tx)
	}

	dataGenDuration := time.Since(startTime)
	t.Logf("⚡ 10.000 adet işlem kaydı belleğe yüklendi: %v", dataGenDuration)

	// 1. KPI Panel Özeti (Summary) Hızı
	summaryStart := time.Now()
	reqSummary := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/periods/%s/summary", periodID.String()), nil)
	reqSummary.Header.Set("Authorization", "Bearer "+jwtToken)
	resSummary, err := app.Test(reqSummary, -1)
	require.NoError(t, err)
	summaryLatency := time.Since(summaryStart)

	assert.Equal(t, http.StatusOK, resSummary.StatusCode)
	assert.Less(t, summaryLatency, 50*time.Millisecond, "10.000 kayıtta KPI özet hesaplama <50ms olmalıdır")
	t.Logf("📊 10.000 işlem için KPI Özet Yanıt Süresi: %v", summaryLatency)

	// 2. Sayfalama (Pagination: 25, 50, 100) Hızı
	for _, limit := range []int{25, 50, 100} {
		pageStart := time.Now()
		reqPage := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/periods/%s/transactions?limit=%d&offset=5000", periodID.String(), limit), nil)
		reqPage.Header.Set("Authorization", "Bearer "+jwtToken)
		resPage, err := app.Test(reqPage, -1)
		require.NoError(t, err)
		pageLatency := time.Since(pageStart)

		assert.Equal(t, http.StatusOK, resPage.StatusCode)
		assert.Less(t, pageLatency, 30*time.Millisecond, "Sayfalanmış sorgu <30ms olmalıdır")
		t.Logf("📄 Sayfalama (Limit %d, Offset 5000) Yanıt Süresi: %v", limit, pageLatency)
	}

	// 3. Excel Export Hacim Testi
	exportStart := time.Now()
	reqExport := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/periods/%s/export/excel", periodID.String()), nil)
	reqExport.Header.Set("Authorization", "Bearer "+jwtToken)
	resExport, err := app.Test(reqExport, -1)
	require.NoError(t, err)
	exportLatency := time.Since(exportStart)

	assert.Equal(t, http.StatusOK, resExport.StatusCode)
	assert.Equal(t, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", resExport.Header.Get("Content-Type"))
	assert.Less(t, exportLatency, 5*time.Second, "10.000 satırlık Excel üretimi <5s olmalıdır")
	t.Logf("📥 10.000 satır Excel Export Yanıt Süresi: %v (Header boyutu: %s)", exportLatency, resExport.Header.Get("Content-Length"))
}

// -----------------------------------------------------------------------------
// Y.9 — Excel Import Boyut Sınırına Yakın Dosya (io.LimitReader 10MB)
// -----------------------------------------------------------------------------
func TestOperational_Y9_ExcelImportSizeLimits(t *testing.T) {
	_, _, periodRepo, txRepo, _, _ := setupTestAppWithRepositories(t)
	importSvc := service.NewImportService(txRepo, periodRepo)

	tenantID := uuid.New()
	periodID := uuid.New()
	periodRepo.Create(context.Background(), &domain.Period{
		ID:              periodID,
		TenantID:        tenantID,
		Label:           "2026-09",
		StartingBalance: decimal.Zero,
		Status:          domain.PeriodStatusOpen,
		OpenedAt:        time.Now(),
	})

	// 10MB sınırını aşan sahte veri akışı
	largeFakeData := make([]byte, 11*1024*1024) // 11 MB
	r := bytes.NewReader(largeFakeData)

	_, err := importSvc.ImportTransactionsFromCSV(context.Background(), tenantID, periodID, r, uuid.New())
	assert.Error(t, err, "10MB üzeri veri akışı güvenlik sınırına takılmalıdır")
}

// -----------------------------------------------------------------------------
// Y.13 — Eşzamanlı İki Cihazdan Aynı Dönemi Kilitleme/Açma (Race Condition)
// -----------------------------------------------------------------------------
func TestOperational_Y13_ConcurrentPeriodLockRaceCondition(t *testing.T) {
	_, tenantRepo, periodRepo, txRepo, _, _ := setupTestAppWithRepositories(t)
	periodSvc := service.NewPeriodService(periodRepo, tenantRepo, txRepo)

	tenantID := uuid.New()
	periodID := uuid.New()
	adminUserID := uuid.New()

	_ = tenantRepo.AddMember(context.Background(), &domain.TenantMember{
		ID:        uuid.New(),
		TenantID:  tenantID,
		UserID:    adminUserID,
		Role:      domain.RoleAdmin,
		CreatedAt: time.Now(),
	})

	periodRepo.Create(context.Background(), &domain.Period{
		ID:              periodID,
		TenantID:        tenantID,
		Label:           "2026-09",
		StartingBalance: decimal.Zero,
		Status:          domain.PeriodStatusOpen,
		OpenedAt:        time.Now(),
	})

	// Eşzamanlı 5 goroutine aynı dönemi kilitlemeyi dener
	concurrency := 5
	errChan := make(chan error, concurrency)

	for i := 0; i < concurrency; i++ {
		go func() {
			errChan <- periodSvc.LockPeriod(context.Background(), periodID, adminUserID)
		}()
	}

	var successes, failures int
	for i := 0; i < concurrency; i++ {
		err := <-errChan
		if err == nil {
			successes++
		} else {
			failures++
		}
	}

	assert.Equal(t, 1, successes, "Sadece 1 kilitleme isteği başarılı olmalıdır")
	assert.Equal(t, concurrency-1, failures, "Diğer eşzamanlı istekler çakışma hatası almalıdır")

	// Dönem durumunun tek ve net biçimde 'locked' olduğu doğrulanır
	p, err := periodRepo.GetByID(context.Background(), periodID)
	require.NoError(t, err)
	assert.Equal(t, domain.PeriodStatusLocked, p.Status)
}

// -----------------------------------------------------------------------------
// Y.14 — Yanlış Tutar Girişi Sonrası Ters Kayıt (Reversal) ve Düzeltme Akışı
// -----------------------------------------------------------------------------
func TestOperational_Y14_CorrectionFlowAndReversal(t *testing.T) {
	_, _, periodRepo, txRepo, _, _ := setupTestAppWithRepositories(t)
	txSvc := service.NewTransactionService(txRepo, periodRepo)

	tenantID := uuid.New()
	periodID := uuid.New()
	adminUserID := uuid.New()

	periodRepo.Create(context.Background(), &domain.Period{
		ID:              periodID,
		TenantID:        tenantID,
		Label:           "2026-09",
		StartingBalance: decimal.Zero,
		Status:          domain.PeriodStatusOpen,
		OpenedAt:        time.Now(),
	})

	// 1. Adım: Kullanıcı sehven 1.000 TL yerine 10.000 TL girer
	descWrong := "Müşteri ödemesi (Hatalı Fazla Giriş)"
	wrongTx := &domain.Transaction{
		ID:          uuid.New(),
		TenantID:    tenantID,
		PeriodID:    periodID,
		Direction:   domain.DirectionIn,
		Channel:     domain.ChannelNakit,
		Amount:      decimal.NewFromFloat(10000.00),
		Description: &descWrong,
		CreatedBy:   adminUserID,
		CreatedAt:   time.Now(),
	}
	require.NoError(t, txSvc.CreateTransaction(context.Background(), wrongTx))

	// Hatalı bakiye kontrolü (10000)
	summaryWrong, err := txRepo.GetSummaryByPeriodID(context.Background(), periodID)
	require.NoError(t, err)
	assert.True(t, summaryWrong.ClosingBalance.Equal(decimal.NewFromFloat(10000.00)))

	// 2. Adım: Kullanıcı tek tıkla 'Ters Kayıt (İptal)' yapar
	revTx, err := txSvc.ReverseTransaction(context.Background(), wrongTx.ID, "Fazla sıfır girilmişti, iptal edildi", adminUserID)
	require.NoError(t, err)
	assert.Equal(t, domain.DirectionOut, revTx.Direction, "Ters kayıt giriş işlemini çıkış olarak nötrler")

	// İptal sonrası bakiye kontrolü (10.000 nötrlendi -> 0.00 net)
	summaryReversed, err := txRepo.GetSummaryByPeriodID(context.Background(), periodID)
	require.NoError(t, err)
	assert.True(t, summaryReversed.ClosingBalance.Equal(decimal.Zero), "İptal edilen kayıt sonrası net bakiye 0 olmalıdır")

	// 3. Adım: Kullanıcı doğru tutar olan 1.000 TL'yi girer
	descCorrect := "Müşteri ödemesi (Doğru Tutar)"
	correctTx := &domain.Transaction{
		ID:          uuid.New(),
		TenantID:    tenantID,
		PeriodID:    periodID,
		Direction:   domain.DirectionIn,
		Channel:     domain.ChannelNakit,
		Amount:      decimal.NewFromFloat(1000.00),
		Description: &descCorrect,
		CreatedBy:   adminUserID,
		CreatedAt:   time.Now(),
	}
	require.NoError(t, txSvc.CreateTransaction(context.Background(), correctTx))

	// Nihai net bakiye (1000.00) kuruşu kuruşuna doğrulanır
	summaryFinal, err := txRepo.GetSummaryByPeriodID(context.Background(), periodID)
	require.NoError(t, err)
	assert.True(t, summaryFinal.ClosingBalance.Equal(decimal.NewFromFloat(1000.00)), "Doğru tutar sonrası net bakiye 1000.00 olmalıdır")
}
