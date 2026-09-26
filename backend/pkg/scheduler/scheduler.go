package scheduler

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"deftersystem/backend/pkg/backup"
	"deftersystem/backend/pkg/gdrive"
	"deftersystem/backend/pkg/telegram"
)

// BackupOrchestrator manages database snapshots, Drive uploads, and Telegram alerts.
type BackupOrchestrator struct {
	pool        *pgxpool.Pool
	exporter    *backup.Exporter
	gdriveClient *gdrive.Client
	tgClient    *telegram.Client
}

// NewBackupOrchestrator initializes the backup orchestrator.
func NewBackupOrchestrator(pool *pgxpool.Pool) *BackupOrchestrator {
	exporter := backup.NewExporter(pool, "backups/daily")

	ctx := context.Background()
	gdriveEnabled := strings.ToLower(os.Getenv("GDRIVE_BACKUP_ENABLED")) == "true"
	credsFile := os.Getenv("GDRIVE_CREDENTIALS_FILE")
	if credsFile == "" {
		credsFile = "credentials.json"
	}
	folderID := os.Getenv("GDRIVE_BACKUP_FOLDER_ID")
	retentionStr := os.Getenv("GDRIVE_RETENTION_DAYS")
	retentionDays := 30
	if r, err := strconv.Atoi(retentionStr); err == nil && r > 0 {
		retentionDays = r
	}

	gdriveClient, err := gdrive.NewClient(ctx, gdrive.Config{
		CredentialsFile: credsFile,
		FolderID:        folderID,
		Enabled:         gdriveEnabled,
		RetentionDays:   retentionDays,
	})
	if err != nil {
		log.Printf("⚠️ [BACKUP] Google Drive client init notice: %v", err)
	}

	return &BackupOrchestrator{
		pool:         pool,
		exporter:     exporter,
		gdriveClient: gdriveClient,
		tgClient:     telegram.Global(),
	}
}

// RunBackup executes the complete 3-2-1 backup pipeline:
// 1. Snapshot to local disk (.sql.gz)
// 2. Send file directly to Telegram chat
// 3. Upload to Google Drive (if enabled) & clean old backups
// 4. Send detailed Telegram report
func (o *BackupOrchestrator) RunBackup(ctx context.Context) (*backup.SnapshotSummary, error) {
	log.Println("📦 [BACKUP] Starting automated 3-2-1 backup workflow...")
	start := time.Now()

	// 1. Export local snapshot
	summary, err := o.exporter.CreateSnapshot(ctx)
	if err != nil {
		errMsg := fmt.Sprintf("Veritabanı snapshot oluşturulamadı: %v", err)
		log.Printf("❌ [BACKUP] %s", errMsg)
		o.tgClient.SendBackupReport(false, "snapshot_error", 0, time.Since(start), errMsg)
		return nil, err
	}

	log.Printf("✅ [BACKUP] Local snapshot created: %s (%.2f KB, %d records)",
		summary.Filename, float64(summary.SizeBytes)/1024, summary.RecordCount)

	// 2. Send .sql.gz document directly to Telegram
	docCaption := fmt.Sprintf("📦 <b>[ÖNCÜ OTOGAZ GÜNLÜK VERİTABANI YEDEĞİ]</b>\n\n"+
		"📁 <b>Dosya:</b> <code>%s</code>\n"+
		"📊 <b>Kayıt Sayısı:</b> <code>%d</code>\n"+
		"⏰ <b>Tarih:</b> <code>%s</code>",
		summary.Filename, summary.RecordCount, time.Now().Format("2006-01-02 15:04:05 MST"),
	)
	if err := o.tgClient.SendDocument(summary.FilePath, docCaption); err != nil {
		log.Printf("⚠️ [BACKUP] Telegram document send notice: %v", err)
	}

	// 3. Upload to Google Drive if configured
	if o.gdriveClient != nil && o.gdriveClient.IsEnabled() {
		log.Println("☁️ [BACKUP] Uploading snapshot to Google Drive...")
		_, err := o.gdriveClient.UploadBackup(ctx, summary.FilePath, summary.Filename)
		if err != nil {
			log.Printf("⚠️ [BACKUP] Google Drive upload failed: %v", err)
		} else {
			log.Println("✅ [BACKUP] Snapshot successfully uploaded to Google Drive!")
			deleted, err := o.gdriveClient.CleanOldBackups(ctx)
			if err == nil && deleted > 0 {
				log.Printf("🧹 [BACKUP] Cleaned %d obsolete backups from Google Drive", deleted)
			}
		}
	}

	// 4. Send Telegram completion report
	o.tgClient.SendBackupReport(true, summary.Filename, summary.SizeBytes, time.Since(start), "")

	return summary, nil
}

// StartDailyCron starts a background scheduler that runs backup every day at the designated hour (default 03:00).
func (o *BackupOrchestrator) StartDailyCron() {
	go func() {
		for {
			now := time.Now()
			// Next 03:00 AM target
			nextRun := time.Date(now.Year(), now.Month(), now.Day(), 3, 0, 0, 0, now.Location())
			if now.After(nextRun) {
				nextRun = nextRun.Add(24 * time.Hour)
			}

			durationUntilNext := time.Until(nextRun)
			log.Printf("⏰ [SCHEDULER] Daily backup scheduled for: %s (in %v)",
				nextRun.Format("2006-01-02 15:04:05"), durationUntilNext.Round(time.Minute))

			time.Sleep(durationUntilNext)

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
			_, _ = o.RunBackup(ctx)
			cancel()
		}
	}()
}
