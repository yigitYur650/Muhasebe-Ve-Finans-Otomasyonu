package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"deftersystem/backend/internal/repository"
)

func main() {
	envPaths := []string{".env", "../.env"}
	for _, envPath := range envPaths {
		if data, err := os.ReadFile(envPath); err == nil {
			for _, line := range strings.Split(string(data), "\n") {
				parts := strings.SplitN(strings.TrimSpace(line), "=", 2)
				if len(parts) == 2 && os.Getenv(parts[0]) == "" {
					os.Setenv(parts[0], strings.Trim(parts[1], "'\""))
				}
			}
		}
	}
	pool, err := repository.NewPostgresPool(os.Getenv("DATABASE_URL"))
	if err != nil { panic(err) }
	defer pool.Close()

	rows, err := pool.Query(context.Background(), `
		SELECT st.id, st.tx_date, st.invoice_no, st.customer_name, st.amount, st.direction, st.description, st.reversed_by
		FROM public.supplier_transactions st
		JOIN public.suppliers s ON s.id = st.supplier_id
		WHERE s.name ILIKE '%ASİL%'
		ORDER BY st.tx_date DESC, st.created_at DESC;
	`)
	if err != nil { panic(err) }
	defer rows.Close()

	var totalPurchases float64
	var totalPurchaseReturns float64
	var totalPayments float64
	var totalPaymentReturns float64
	var count int

	fmt.Println("--- ASİL GRUP TÜM SATIRLARI ---")
	for rows.Next() {
		count++
		var id, dir, desc, cust string
		var inv, revBy *string
		var amt float64
		var txDate string
		rows.Scan(&id, &txDate, &inv, &cust, &amt, &dir, &desc, &revBy)
		in := "-"
		if inv != nil && *inv != "" { in = *inv }
		rv := "-"
		if revBy != nil && *revBy != "" { rv = *revBy }

		switch dir {
		case "purchase":
			totalPurchases += amt
		case "purchase_return":
			totalPurchaseReturns += amt
		case "payment":
			totalPayments += amt
		case "payment_return":
			totalPaymentReturns += amt
		}

		fmt.Printf("%02d | %s | %-16s | %-15s | %-15s | %10.2f TL | revBy: %-5t | desc: %s\n",
			count, txDate[:10], in, cust, dir, amt, rv != "-", desc)
	}

	fmt.Println("------------------------------------------------------------------------------")
	fmt.Printf("Toplam Satır Sayısı: %d\n", count)
	fmt.Printf("purchase (Alınan Mal +)       : ₺%.2f\n", totalPurchases)
	fmt.Printf("purchase_return (Alış İadesi -): ₺%.2f\n", totalPurchaseReturns)
	fmt.Printf("Net Alış (purchase - return)  : ₺%.2f\n", totalPurchases - totalPurchaseReturns)
	fmt.Printf("payment (Geçilen Ödeme -)     : ₺%.2f\n", totalPayments)
	fmt.Printf("payment_return (Ödeme İadesi +): ₺%.2f\n", totalPaymentReturns)
	fmt.Printf("Net Ödeme (payment - return)  : ₺%.2f\n", totalPayments - totalPaymentReturns)
	fmt.Printf("Kalan Bakiye                  : ₺%.2f\n", (totalPurchases - totalPurchaseReturns) - (totalPayments - totalPaymentReturns))
}
