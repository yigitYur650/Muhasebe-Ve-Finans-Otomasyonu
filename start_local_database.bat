@echo off
chcp 65001 > nul
echo ==============================================================================
echo 🚀 DEFTER-İ KEBİR - YEREL İZOLE VERİTABANI BAŞLATICISI
echo ==============================================================================
echo 📌 Bu modda çalışırken canlı Supabase veritabanına HİÇBİR veri yazılmaz.
echo 📌 Tüm değişiklikler yerel PostgreSQL veritabanında izole kalır.
echo.

echo 1. Docker servisi kontrol ediliyor...
docker info >nul 2>&1
if %errorlevel% neq 0 (
    echo ❌ Docker Desktop çalışmıyor! Lütfen Docker Desktop uygulamasını başlatın.
    pause
    exit /b 1
)

echo 2. Yerel PostgreSQL başlatılıyor ve Supabase verileri içe aktarılıyor...
docker compose -f docker-compose.local.yml up -d local-postgres

echo.
echo ✅ Yerel PostgreSQL Hazır! (Port: 5433)
echo 📊 Bağlantı URL: postgres://postgres:postgrespassword@localhost:5433/postgres?sslmode=disable
echo.
echo 3. Backend'i yerel veritabanı ile başlatmak için:
echo    cd backend
echo    go run cmd/api/main.go
echo.
pause
