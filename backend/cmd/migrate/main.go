package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"deftersystem/backend/internal/repository"

	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	statusOnly := flag.Bool("status", false, "Yalnızca migration durumunu gösterir, çalıştırmaz")
	flag.Parse()

	dbURL := getDatabaseURL()
	if dbURL == "" {
		log.Fatalf("❌ DATABASE_URL bulunamadı. Lütfen .env dosyasını kontrol edin.")
	}

	fmt.Println("==========================================================================")
	fmt.Println("🚀 DEFTER-İ KEBİR — AKILLI MİGRATİON VE SÜRÜM TAKİP YÖNETİCİSİ")
	fmt.Println("==========================================================================")

	pool, err := repository.NewPostgresPool(dbURL)
	if err != nil {
		log.Fatalf("❌ Veritabanı bağlantı hatası: %v", err)
	}
	defer pool.Close()

	ctx := context.Background()

	// 1. schema_migrations tablosunun varlığını garantiye al
	err = ensureMigrationTable(ctx, pool)
	if err != nil {
		log.Fatalf("❌ schema_migrations tablosu hazırlanamadı: %v", err)
	}

	// 2. migrations/ dizinini bul
	migrationsDir := findMigrationsDir()
	if migrationsDir == "" {
		log.Fatalf("❌ migrations/ dizini bulunamadı!")
	}
	fmt.Printf("📁 Migration Klasörü: %s\n", migrationsDir)

	// 3. Sıralı migration dosyalarını topla (00_... ile 98_... arasındaki standart migrationlar)
	files, err := getMigrationFiles(migrationsDir)
	if err != nil {
		log.Fatalf("❌ Migration dosyaları okunamadı: %v", err)
	}

	// 4. Halihazırda uygulanmış olan migrationları veritabanından çek
	appliedMap, err := getAppliedMigrations(ctx, pool)
	if err != nil {
		log.Fatalf("❌ Uygulanmış migrationlar sorgulanamadı: %v", err)
	}

	// 5. Akıllı Baseline Tespiti:
	// Eğer schema_migrations tablosu boşsa ama veritabanında 'tenants' tablosu zaten varsa
	// (yani veritabanı daha önce combined schema ile kurulmuşsa),
	// mevcut 00-13 arası migration'ları 'baseline' olarak kaydet ki tabloları tekrar yaratmaya çalışıp çakışmasın.
	if len(appliedMap) == 0 && isDatabaseAlreadyInitialized(ctx, pool) {
		fmt.Println("ℹ️  Mevcut veritabanı şeması tespit edildi. Başlangıç migrationları (00-13) baseline olarak işaretleniyor...")
		for _, f := range files {
			// 00 ile 13 arasındaki dosyaları baseline olarak işaretle
			if strings.HasPrefix(f, "0") || strings.HasPrefix(f, "1") {
				if err := recordMigration(ctx, pool, f); err != nil {
					log.Fatalf("❌ Baseline kaydı başarısız (%s): %v", f, err)
				}
				appliedMap[f] = time.Now()
				fmt.Printf("   📌 [BASELINE]: %s başarıyla işaretlendi\n", f)
			}
		}
	}

	// 6. Durum Tablosunu Ekrana Yazdır
	fmt.Println("\n--------------------------------------------------------------------------")
	fmt.Printf("%-40s | %-12s | %s\n", "MİGRATİON DOSYASI", "DURUM", "UYGULANMA TARİHİ")
	fmt.Println("--------------------------------------------------------------------------")

	var pendingFiles []string
	for _, f := range files {
		appliedAt, exists := appliedMap[f]
		if exists {
			fmt.Printf("%-40s | \033[32m%-12s\033[0s | %s\n", f, "UYGULANDI", appliedAt.Format("2006-01-02 15:04:05"))
		} else {
			fmt.Printf("%-40s | \033[33m%-12s\033[0s | -\n", f, "BEKLİYOR")
			pendingFiles = append(pendingFiles, f)
		}
	}
	fmt.Println("--------------------------------------------------------------------------")

	if *statusOnly {
		fmt.Printf("\nDurum özeti: Toplam %d dosya, %d uygulandı, %d bekliyor.\n", len(files), len(appliedMap), len(pendingFiles))
		return
	}

	// 7. Bekleyen Migrationları Sırayla Çalıştır
	if len(pendingFiles) == 0 {
		fmt.Println("\n✅ Veritabanı tamamen güncel! Çalıştırılacak yeni migration bulunmuyor.")
		return
	}

	fmt.Printf("\n⚙️  %d adet bekleyen migration çalıştırılıyor...\n\n", len(pendingFiles))
	for _, f := range pendingFiles {
		filePath := filepath.Join(migrationsDir, f)
		content, err := os.ReadFile(filePath)
		if err != nil {
			log.Fatalf("❌ Dosya okuma hatası (%s): %v", f, err)
		}

		sqlContent := string(content)
		fmt.Printf("▶️  Çalıştırılıyor: %s ... ", f)

		tx, err := pool.Begin(ctx)
		if err != nil {
			log.Fatalf("\n❌ Transaction başlatılamadı (%s): %v", f, err)
		}

		if _, err := tx.Exec(ctx, sqlContent); err != nil {
			_ = tx.Rollback(ctx)
			log.Fatalf("\n❌ Migration çalıştırma hatası (%s): %v\nİşlem geri alındı (ROLLBACK).", f, err)
		}

		// schema_migrations tablosuna kaydet
		_, err = tx.Exec(ctx, `INSERT INTO public.schema_migrations (version, applied_at) VALUES ($1, now())`, f)
		if err != nil {
			_ = tx.Rollback(ctx)
			log.Fatalf("\n❌ Migration kayıt hatası (%s): %v", f, err)
		}

		if err := tx.Commit(ctx); err != nil {
			log.Fatalf("\n❌ Transaction commit hatası (%s): %v", f, err)
		}

		fmt.Println("\033[32m[BAŞARILI]\033[0s")
	}

	fmt.Println("\n==========================================================================")
	fmt.Println("🎉 TÜM BEKLEYEN MİGRATİONLAR BAŞARIYLA TAMAMLANDI!")
	fmt.Println("==========================================================================")
}

func ensureMigrationTable(ctx context.Context, pool *pgxpool.Pool) error {
	query := `
		CREATE TABLE IF NOT EXISTS public.schema_migrations (
			version VARCHAR(255) PRIMARY KEY,
			applied_at TIMESTAMPTZ NOT NULL DEFAULT now()
		);
	`
	_, err := pool.Exec(ctx, query)
	return err
}

func getAppliedMigrations(ctx context.Context, pool *pgxpool.Pool) (map[string]time.Time, error) {
	rows, err := pool.Query(ctx, `SELECT version, applied_at FROM public.schema_migrations ORDER BY applied_at ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	applied := make(map[string]time.Time)
	for rows.Next() {
		var ver string
		var t time.Time
		if err := rows.Scan(&ver, &t); err != nil {
			return nil, err
		}
		applied[ver] = t
	}
	return applied, nil
}

func recordMigration(ctx context.Context, pool *pgxpool.Pool, version string) error {
	_, err := pool.Exec(ctx, `
		INSERT INTO public.schema_migrations (version, applied_at) 
		VALUES ($1, now()) 
		ON CONFLICT (version) DO NOTHING
	`, version)
	return err
}

func isDatabaseAlreadyInitialized(ctx context.Context, pool *pgxpool.Pool) bool {
	var exists bool
	query := `
		SELECT EXISTS (
			SELECT FROM information_schema.tables 
			WHERE table_schema = 'public' 
			AND table_name = 'tenants'
		);
	`
	_ = pool.QueryRow(ctx, query).Scan(&exists)
	return exists
}

func findMigrationsDir() string {
	candidates := []string{
		"migrations",
		"../migrations",
		"../../migrations",
	}
	for _, c := range candidates {
		info, err := os.Stat(c)
		if err == nil && info.IsDir() {
			return c
		}
	}
	return ""
}

func getMigrationFiles(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var files []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		// Yalnızca standart migration formatındaki .sql dosyalarını al (Örn: 01_..., 12_...)
		// 99_reset_and_seed.sql, supabase_combined_schema.sql veya test_scenarios.sql otomatik çalıştırılmaz!
		if strings.HasSuffix(name, ".sql") &&
			len(name) >= 3 &&
			name[0] >= '0' && name[0] <= '8' && // 0x ile 8x arasındaki sıralı dosyalar
			name[1] >= '0' && name[1] <= '9' &&
			name[2] == '_' {
			files = append(files, name)
		}
	}
	sort.Strings(files)
	return files, nil
}

func getDatabaseURL() string {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL != "" {
		return dbURL
	}
	envPaths := []string{".env", "../.env", "backend/.env"}
	for _, p := range envPaths {
		data, err := os.ReadFile(p)
		if err == nil {
			for _, line := range strings.Split(string(data), "\n") {
				line = strings.TrimSpace(line)
				if strings.HasPrefix(line, "DATABASE_URL=") {
					dbURL = strings.TrimPrefix(line, "DATABASE_URL=")
					return strings.Trim(dbURL, `"'`)
				}
			}
		}
	}
	return ""
}
