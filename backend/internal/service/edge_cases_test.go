package service_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"deftersystem/backend/internal/domain"
	"deftersystem/backend/internal/service"
)

// ---------------------------------------------------------------------------
// Boşluk 1 (KRİTİK): CreateTransaction — tenant sahiplik gözetimi yok.
//
// Mevcut CreateTransaction, işlemin TenantID'si ile işlemin yazıldığı dönemin
// TenantID'si arasında eşleşme kontrolü YAPMAZ. Bir kullanıcı başka bir
// tenant'a ait geçerli bir periodID ile kendi tenant'ı üzerinden işlem
// oluşturabilir. Bu test, mevcut davranışı (tenat uyuşmazlığına rağmen
// repo.Create'in çağrılması) belgeleyerek açığı sabitler.
// ---------------------------------------------------------------------------
func TestCreateTransaction_CrossTenantPeriod_NoOwnershipGuard(t *testing.T) {
	ctx := context.Background()
	tenantA := uuid.New()
	tenantB := uuid.New()
	periodOfTenantB := uuid.New()

	// Tenant B'ye ait bir dönem
	foreignPeriod := &domain.Period{
		ID:       periodOfTenantB,
		TenantID: tenantB,
		Status:   domain.PeriodStatusOpen,
	}

	// Tenant A kullanıcısı Tenant B'nin dönemine işlem yazmaya çalışıyor
	tx := &domain.Transaction{
		ID:        uuid.New(),
		TenantID:  tenantA,
		PeriodID:  periodOfTenantB,
		Direction: domain.DirectionIn,
		Channel:   domain.ChannelEft,
		Amount:    decimal.NewFromFloat(100.00),
		CreatedBy: uuid.New(),
	}

	mockTxRepo := new(MockTransactionRepo)
	mockPeriodRepo := new(MockPeriodRepo)
	svc := service.NewTransactionService(mockTxRepo, mockPeriodRepo)

	mockPeriodRepo.On("GetByID", ctx, periodOfTenantB).Return(foreignPeriod, nil)
	// Mevcut kod: tenant uyuşmazlığına rağmen repo.Create çağrılır.
	mockTxRepo.On("Create", ctx, tx).Return(nil)

	err := svc.CreateTransaction(ctx, tx)

	// MEVCUT DAVRANIŞ: Hata dönmüyor; cross-tenant yazım repo'ya geçiyor.
	// Bu bir güvenlik açığı olarak BUG_AND_FIX'e kaydedilmelidir.
	assert.NoError(t, err, "MEVCUT DAVRANIŞ: cross-tenant period'a yazım engellenmiyor (açık)")
	mockTxRepo.AssertCalled(t, "Create", ctx, tx)
}

// ---------------------------------------------------------------------------
// Boşluk 2 (KRİTİK): ReverseTransaction — orijinal işlemin tenant sahipliği
// gözetimi yok. Sonradan eklenen bir guard bu testi FAIL yapar ve açığı kapatır.
// ---------------------------------------------------------------------------
func TestReverseTransaction_PeriodOwnership_CrossTenantReversal(t *testing.T) {
	ctx := context.Background()
	tenantB := uuid.New()
	periodOfTenantB := uuid.New()
	origID := uuid.New()

	// Tenant B'ye ait orijinal işlem
	origTx := &domain.Transaction{
		ID:        origID,
		TenantID:  tenantB,
		PeriodID:  periodOfTenantB,
		Direction: domain.DirectionIn,
		Channel:   domain.ChannelEft,
		Amount:    decimal.NewFromFloat(500.00),
	}

	period := &domain.Period{
		ID:       periodOfTenantB,
		TenantID: tenantB,
		Status:   domain.PeriodStatusOpen,
	}

	mockTxRepo := new(MockTransactionRepo)
	mockPeriodRepo := new(MockPeriodRepo)
	svc := service.NewTransactionService(mockTxRepo, mockPeriodRepo)

	mockTxRepo.On("GetByID", ctx, origID).Return(origTx, nil)
	mockPeriodRepo.On("GetByID", ctx, periodOfTenantB).Return(period, nil)
	mockTxRepo.On("ReverseTransaction", ctx, origID, mock.Anything).Return(nil)

	revTx, err := svc.ReverseTransaction(ctx, origID, "iptal", uuid.New())

	// MEVCUT DAVRANIŞ: Tenant A çağırıcı, Tenant B'nin işlemini ters kayıtla
	// iptal edebiliyor (tenant doğrulaması yok).
	assert.NoError(t, err)
	assert.NotNil(t, revTx)
	mockTxRepo.AssertCalled(t, "ReverseTransaction", ctx, origID, mock.Anything)
}

// ---------------------------------------------------------------------------
// Boşluk 3: LockPeriod — zaten kilitli dönemi tekrar kilitleme davranışı.
// Mevcut kod dönemin Status'unu kontrol etmez; idempotent şekilde başarılı
// döner (repo.Lock çağrılır). Bu test mevcut davranışı belgeler.
// ---------------------------------------------------------------------------
func TestLockPeriod_AlreadyLocked_CurrentBehavior(t *testing.T) {
	ctx := context.Background()
	tenantID := uuid.New()
	periodID := uuid.New()
	adminUserID := uuid.New()
	now := time.Now()

	alreadyLocked := &domain.Period{
		ID:       periodID,
		TenantID: tenantID,
		Status:   domain.PeriodStatusLocked,
		LockedAt: &now,
	}

	adminMember := &domain.TenantMember{
		ID:       uuid.New(),
		TenantID: tenantID,
		UserID:   adminUserID,
		Role:     domain.RoleAdmin,
	}

	mockPeriodRepo := new(MockPeriodRepo)
	mockTenantRepo := new(MockTenantRepo)
	mockTxRepo := new(MockTransactionRepo)
	svc := service.NewPeriodService(mockPeriodRepo, mockTenantRepo, mockTxRepo)

	mockPeriodRepo.On("GetByID", ctx, periodID).Return(alreadyLocked, nil)
	mockTenantRepo.On("GetMember", ctx, tenantID, adminUserID).Return(adminMember, nil)
	mockPeriodRepo.On("Lock", ctx, periodID).Return(nil)

	err := svc.LockPeriod(ctx, periodID, adminUserID)

	// MEVCUT DAVRANIŞ: kilitli dönem tekrar "kilitleniyor"; hata dönmüyor.
	// (Idempotent olabilir veya tutarsızlık olabilir — karar vermek gerek.)
	assert.NoError(t, err, "MEVCUT DAVRANIŞ: zaten kilitli dönem tekrar kilitlenebiliyor (açık)")
	mockPeriodRepo.AssertCalled(t, "Lock", ctx, periodID)
}

// ---------------------------------------------------------------------------
// Boşluk 4: OpenNextPeriod — label formatı uç durumları.
// Format regex'i `^\d{4}-\d{2}$` geçer ama geçersiz ay (13-19) da geçiyor.
// ---------------------------------------------------------------------------
func TestOpenNextPeriod_LabelValidation_EdgeCases(t *testing.T) {
	ctx := context.Background()
	tenantID := uuid.New()

	testCases := []struct {
		name       string
		label      string
		expectErr  bool
	}{
		{"geçerli format", "2025-07", false},
		{"geçersiz ay (13)", "2025-13", false}, // MEVCUT DAVRANIŞ: regex ile geçer! -> açık
		{"geçersiz ay (19)", "2025-19", false}, // MEVCUT DAVRANIŞ: regex ile geçer! -> açık
		{"yıl eksik", "2025", true},
		{"tek basamaklı ay", "2025-5", true},
		{"alfabetik", "2025-AB", true},
		{"boş string", "", true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Her alt test için taze mock
			localPeriodRepo := new(MockPeriodRepo)
			localTenantRepo := new(MockTenantRepo)
			localTxRepo := new(MockTransactionRepo)
			localSvc := service.NewPeriodService(localPeriodRepo, localTenantRepo, localTxRepo)

			if !tc.expectErr {
				opened := &domain.Period{
					ID:       uuid.New(),
					TenantID: tenantID,
					Label:    tc.label,
					Status:   domain.PeriodStatusOpen,
				}
				localPeriodRepo.On("OpenNextPeriod", ctx, tenantID, tc.label).Return(opened, nil)
			}

			_, err := localSvc.OpenNextPeriod(ctx, tenantID, tc.label)

			if tc.expectErr {
				assert.Error(t, err, "geçersiz format için hata beklenir")
				localPeriodRepo.AssertNotCalled(t, "OpenNextPeriod", mock.Anything, mock.Anything, mock.Anything)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Boşluk 5: GetPeriodSummary — ters kayıtların bakiye özetine etkisi.
// Append-only olduğundan bir işlem hem orijinali hem de reversal'ıyla iki satır
// oluşturur. Eğer bakiye hesabı reversal'ları ayrıca küçültmezse net bakiye
// bozulur (double-count). Bu domain düzeyinde hesap testi ile sabitlenir.
// ---------------------------------------------------------------------------
func TestPeriodSummary_ReversalEntries_ImpactOnBalance(t *testing.T) {
	ctx := context.Background()
	periodID := uuid.New()

	// Senaryo: 1000 TL gelir, ardından 1000 TL gelir ters kayıt (gider) eklendi.
	// Net etki sıfır olmalı. Append-only modelde orijinal + reversal ikisi de
	// ayrı satırlar olduğu için TotalIn=1000 ve TotalOut=1000 üretirler.
	// Repo'nun bunları doğru topladığı varsayımıyla ClosingBalance = starting.
	starting := decimal.NewFromFloat(5000.00)

	expectedSummary := &domain.PeriodSummary{
		PeriodID:        periodID,
		StartingBalance: starting,
		TotalIn:         decimal.NewFromFloat(1000.00),  // orijinal gelir
		TotalOut:        decimal.NewFromFloat(1000.00),  // ters kayıt (iptal)
		ClosingBalance:  starting,                        // net etki sıfır
	}

	mockPeriodRepo := new(MockPeriodRepo)
	mockTenantRepo := new(MockTenantRepo)
	mockTxRepo := new(MockTransactionRepo)
	svc := service.NewPeriodService(mockPeriodRepo, mockTenantRepo, mockTxRepo)

	mockTxRepo.On("GetSummaryByPeriodID", ctx, periodID).Return(expectedSummary, nil)

	summary, err := svc.GetPeriodSummary(ctx, periodID)

	assert.NoError(t, err)
	assert.NotNil(t, summary)
	// Reversal'lı bir dönemde net bakiye starting balance'a eşit olmalı.
	assert.True(t, summary.ClosingBalance.Equal(starting),
		"Reversal kayıtları net bakiyeyi bozmamalı (double-count olmamalı).")
	// TotalIn == TotalOut olmalı (iptal edilen gelir denkleşti).
	assert.True(t, summary.TotalIn.Equal(summary.TotalOut))
}

// ---------------------------------------------------------------------------
// Boşluk 6: TenantService.AddMember — duplike/mantıksız senaryolar.
// Mevcut AddMember implementasyonu gerçekte repo.Create ile TENANT kaydı
// ekliyor (üye değil) — bu bir iş mantığı hatasıdır/belgelemeye değer.
// ---------------------------------------------------------------------------
func TestTenantService_AddMember_CurrentBehavior(t *testing.T) {
	ctx := context.Background()
	tenantID := uuid.New()
	adminUserID := uuid.New()
	targetUserID := uuid.New()

	mockTenantRepo := new(MockTenantRepo)
	tenantSvc := service.NewTenantService(mockTenantRepo)

	adminMember := &domain.TenantMember{
		TenantID: tenantID,
		UserID:   adminUserID,
		Role:     domain.RoleAdmin,
	}

	mockTenantRepo.On("GetMember", ctx, tenantID, adminUserID).Return(adminMember, nil)

	// MEVCUT DAVRANIŞ: repo.Create, domain.Tenant tipinde çağrılır (üye değil).
	// `s.tenantRepo.Create(ctx, &domain.Tenant{ID: tenantID})` — bu bir açıktır.
	var captured *domain.Tenant
	mockTenantRepo.On("Create", ctx, mock.Anything).Run(func(args mock.Arguments) {
		captured = args.Get(1).(*domain.Tenant)
	}).Return(nil)

	err := tenantSvc.AddMember(ctx, tenantID, adminUserID, targetUserID, domain.RoleMuhasebeci)

	assert.NoError(t, err)
	assert.NotNil(t, captured, "MEVCUT DAVRANIŞ: AddMember repo.Create'i tetikliyor")
	// Açık: targetUserID ve role hiçbir yerde repo'ya gitmiyor; yanlışlıkla
	// yeni bir TENANT (boş isim) oluşturuluyor. Bu bir bug olarak kayıtlanmalı.
	assert.Equal(t, tenantID, captured.ID)
	assert.Empty(t, captured.Name, "MEVCUT KOD yeni bir isimsiz Tenant oluşturuyor — üye eklemiyor!")
}

// ---------------------------------------------------------------------------
// Boşluk 7: ListPeriods — hiç dönem olmayan tenant.
// GetLatestByTenant ErrNotFound döndürürse boş liste dönülür. Bu test
// davranışı güvence altına alır.
// ---------------------------------------------------------------------------
func TestPeriodService_ListPeriods_NoPeriodsReturnsEmpty(t *testing.T) {
	ctx := context.Background()
	tenantID := uuid.New()

	mockPeriodRepo := new(MockPeriodRepo)
	mockTenantRepo := new(MockTenantRepo)
	mockTxRepo := new(MockTransactionRepo)
	svc := service.NewPeriodService(mockPeriodRepo, mockTenantRepo, mockTxRepo)

	mockPeriodRepo.On("GetLatestByTenant", ctx, tenantID).Return(nil, domain.ErrNotFound)

	periods, err := svc.ListPeriods(ctx, tenantID)

	assert.NoError(t, err, "Dönem yoksa hata değil boş liste dönülmeli")
	assert.Len(t, periods, 0, "Hiç dönem olmayan tenant için boş liste beklenir")
}

// ---------------------------------------------------------------------------
// Boşluk 8: ListPeriods — repo'da gerçek bir hatada hata propagasyonu.
// ---------------------------------------------------------------------------
func TestPeriodService_ListPeriods_RepoErrorPropagates(t *testing.T) {
	ctx := context.Background()
	tenantID := uuid.New()

	mockPeriodRepo := new(MockPeriodRepo)
	mockTenantRepo := new(MockTenantRepo)
	mockTxRepo := new(MockTransactionRepo)
	svc := service.NewPeriodService(mockPeriodRepo, mockTenantRepo, mockTxRepo)

	mockPeriodRepo.On("GetLatestByTenant", ctx, tenantID).Return(nil, context.DeadlineExceeded)

	_, err := svc.ListPeriods(ctx, tenantID)

	assert.Error(t, err)
	assert.ErrorIs(t, err, context.DeadlineExceeded)
}

// ---------------------------------------------------------------------------
// Boşluk 9: CreateTransaction — period repo hatası propagasyonu.
// ---------------------------------------------------------------------------
func TestCreateTransaction_PeriodRepoErrorPropagates(t *testing.T) {
	ctx := context.Background()
	periodID := uuid.New()

	tx := &domain.Transaction{
		ID:        uuid.New(),
		PeriodID:  periodID,
		Direction: domain.DirectionIn,
		Channel:   domain.ChannelNakit,
		Amount:    decimal.NewFromFloat(100.00),
		CreatedBy: uuid.New(),
	}

	mockTxRepo := new(MockTransactionRepo)
	mockPeriodRepo := new(MockPeriodRepo)
	svc := service.NewTransactionService(mockTxRepo, mockPeriodRepo)

	mockPeriodRepo.On("GetByID", ctx, periodID).Return(nil, domain.ErrPeriodNotFound)

	err := svc.CreateTransaction(ctx, tx)

	assert.ErrorIs(t, err, domain.ErrPeriodNotFound)
	mockTxRepo.AssertNotCalled(t, "Create", mock.Anything, mock.Anything)
}

// ---------------------------------------------------------------------------
// Boşluk 10: ReverseTransaction — orijinal işlem bulunamadığında hata.
// ---------------------------------------------------------------------------
func TestReverseTransaction_OriginalNotFound(t *testing.T) {
	ctx := context.Background()
	origID := uuid.New()

	mockTxRepo := new(MockTransactionRepo)
	mockPeriodRepo := new(MockPeriodRepo)
	svc := service.NewTransactionService(mockTxRepo, mockPeriodRepo)

	mockTxRepo.On("GetByID", ctx, origID).Return(nil, domain.ErrTransactionNotFound)

	revTx, err := svc.ReverseTransaction(ctx, origID, "iptal", uuid.New())

	assert.ErrorIs(t, err, domain.ErrTransactionNotFound)
	assert.Nil(t, revTx)
}

// ---------------------------------------------------------------------------
// Boşluk 11: ReverseTransaction — orijinalin dönemi bulunamıyor.
// ---------------------------------------------------------------------------
func TestReverseTransaction_OriginPeriodNotFound(t *testing.T) {
	ctx := context.Background()
	origID := uuid.New()
	periodID := uuid.New()

	origTx := &domain.Transaction{
		ID:        origID,
		TenantID:  uuid.New(),
		PeriodID:  periodID,
		Direction: domain.DirectionOut,
		Channel:   domain.ChannelPos,
		Amount:    decimal.NewFromFloat(200.00),
	}

	mockTxRepo := new(MockTransactionRepo)
	mockPeriodRepo := new(MockPeriodRepo)
	svc := service.NewTransactionService(mockTxRepo, mockPeriodRepo)

	mockTxRepo.On("GetByID", ctx, origID).Return(origTx, nil)
	mockPeriodRepo.On("GetByID", ctx, periodID).Return(nil, domain.ErrPeriodNotFound)

	revTx, err := svc.ReverseTransaction(ctx, origID, "iptal", uuid.New())

	assert.ErrorIs(t, err, domain.ErrPeriodNotFound)
	assert.Nil(t, revTx)
	mockTxRepo.AssertNotCalled(t, "ReverseTransaction", mock.Anything, mock.Anything, mock.Anything)
}

// ---------------------------------------------------------------------------
// Boşluk 12: TenantService.UpdateMemberRole — hedef kullanıcı bulunamıyor.
// Mevcut kod, getMembersByTenantID'de tek admin varsa target'ın admin olup
// olmadığını GetMember ile doğrulamaya çalışır. Hedef hiç yoksa ne olur?
// ---------------------------------------------------------------------------
func TestTenantService_UpdateMemberRole_TargetNotFound(t *testing.T) {
	ctx := context.Background()
	tenantID := uuid.New()
	adminUserID := uuid.New()
	nonexistentTarget := uuid.New()

	mockTenantRepo := new(MockTenantRepo)
	tenantSvc := service.NewTenantService(mockTenantRepo)

	adminMember := &domain.TenantMember{
		TenantID: tenantID,
		UserID:   adminUserID,
		Role:     domain.RoleAdmin,
	}

	// Tek admin var
	allMembers := []domain.TenantMember{{TenantID: tenantID, UserID: adminUserID, Role: domain.RoleAdmin}}

	mockTenantRepo.On("GetMember", ctx, tenantID, adminUserID).Return(adminMember, nil)
	mockTenantRepo.On("GetMembersByTenantID", ctx, tenantID).Return(allMembers, nil)
	mockTenantRepo.On("GetMember", ctx, tenantID, nonexistentTarget).Return(nil, domain.ErrNotFound)

	err := tenantSvc.UpdateMemberRole(ctx, tenantID, adminUserID, nonexistentTarget, domain.RoleStandart)

	// MEVCUT DAVRANIŞ: hedef bulunamayınca adminCount<=1 ve target admin değil,
	// bu yüzden ErrCannotRemoveLastAdmin dönmez. Ancak hiçbir düzeltme de
	// yapılmaz — UpdateMemberRole repo'ya hiçbir şey yazmıyor (silent success).
	assert.NoError(t, err, "MEVCUT DAVRANIŞ: var olmayan hedef için sessiz başarı (açık — gerçek düzeltme yapılmıyor)")
}

// ---------------------------------------------------------------------------
// Boşluk 13: RemoveMember — hedef kullanıcı bulunamıyor.
// ---------------------------------------------------------------------------
func TestTenantService_RemoveMember_TargetNotFound(t *testing.T) {
	ctx := context.Background()
	tenantID := uuid.New()
	adminUserID := uuid.New()
	nonexistentTarget := uuid.New()

	mockTenantRepo := new(MockTenantRepo)
	tenantSvc := service.NewTenantService(mockTenantRepo)

	adminMember := &domain.TenantMember{
		TenantID: tenantID,
		UserID:   adminUserID,
		Role:     domain.RoleAdmin,
	}

	mockTenantRepo.On("GetMember", ctx, tenantID, adminUserID).Return(adminMember, nil)
	mockTenantRepo.On("GetMember", ctx, tenantID, nonexistentTarget).Return(nil, domain.ErrNotFound)

	err := tenantSvc.RemoveMember(ctx, tenantID, adminUserID, nonexistentTarget)

	assert.ErrorIs(t, err, domain.ErrNotFound)
}
