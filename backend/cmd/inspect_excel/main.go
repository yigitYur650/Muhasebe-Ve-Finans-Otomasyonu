package main

import (
	"fmt"
	"os"

	"github.com/xuri/excelize/v2"
)

func main() {
	f1 := `C:\Users\yigit\OneDrive\Desktop\Yeni klasör (3)\defter-2026-08-aktif-kayitlar (1).xlsx`
	f2 := `C:\Users\yigit\OneDrive\Desktop\Yeni klasör (3)\KASA DEFTERİM 2026.xlsx`

	fmt.Println("==========================================================================")
	fmt.Println("📊 DOSYA 1 İNCELEMESİ (defter-2026-08-aktif-kayitlar)")
	fmt.Println("==========================================================================")
	if info, err := os.Stat(f1); err != nil {
		fmt.Printf("❌ Dosya 1 bulunamadı: %v\n", err)
	} else {
		fmt.Printf("Dosya 1 Boyutu: %d KB\n", info.Size()/1024)
		xl1, err := excelize.OpenFile(f1)
		if err != nil {
			fmt.Printf("❌ Dosya 1 açılamadı: %v\n", err)
		} else {
			defer xl1.Close()
			sheets := xl1.GetSheetList()
			fmt.Printf("Sayfalar (%d adet): %v\n", len(sheets), sheets)
			for _, s := range sheets {
				rows, _ := xl1.GetRows(s)
				fmt.Printf("  📄 Sayfa [%s] -> Toplam Satır: %d\n", s, len(rows))
				for i := 0; i < len(rows) && i < 15; i++ {
					fmt.Printf("     Satır %d: %v\n", i+1, rows[i])
				}
			}
		}
	}

	fmt.Println("\n==========================================================================")
	fmt.Println("📊 DOSYA 2 İNCELEMESİ (KASA DEFTERİM 2026)")
	fmt.Println("==========================================================================")
	if info, err := os.Stat(f2); err != nil {
		fmt.Printf("❌ Dosya 2 bulunamadı: %v\n", err)
	} else {
		fmt.Printf("Boyut: %d KB\n", info.Size()/1024)
		xl2, err := excelize.OpenFile(f2)
		if err != nil {
			fmt.Printf("❌ Dosya 2 açılamadı: %v\n", err)
		} else {
			defer xl2.Close()
			sheets := xl2.GetSheetList()
			fmt.Printf("Sayfalar (%d adet): %v\n", len(sheets), sheets)
			for _, s := range sheets {
				rows, _ := xl2.GetRows(s)
				fmt.Printf("  📄 Sayfa [%s] -> Toplam Satır: %d\n", s, len(rows))
				if len(rows) > 0 {
					fmt.Printf("     Başlık (Satır 1): %v\n", rows[0])
				}
				if len(rows) > 1 {
					fmt.Printf("     Örnek (Satır 2): %v\n", rows[1])
				}
			}
		}
	}
}
