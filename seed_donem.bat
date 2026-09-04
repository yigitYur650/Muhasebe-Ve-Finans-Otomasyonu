@echo off
chcp 65001 > nul
echo ==========================================================================
echo  [MUHASEBE OTOMASYONU] AKILLI DÖNEM TEST VERİSİ YÜKLEME ARACI
echo ==========================================================================
cd backend
go run cmd/run_yaml_tests/main.go %1
pause
