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
		SELECT st.id, s.name, st.invoice_no, st.amount, st.direction, st.description, st.customer_name, st.reversed_by
		FROM public.supplier_transactions st
		JOIN public.suppliers s ON s.id = st.supplier_id
		WHERE st.invoice_no IN ('ASL2026000010430', 'ASL2026000010441', 'OOA2026000000011', 'ASL2026000010559')
		   OR s.name ILIKE '%İPTAL%' OR s.name ILIKE '%TERS%'
		ORDER BY st.invoice_no, st.tx_date;
	`)
	if err != nil { panic(err) }
	defer rows.Close()

	for rows.Next() {
		var id, name, dir, desc, cust string
		var inv *string
		var revBy *string
		var amt float64
		rows.Scan(&id, &name, &inv, &amt, &dir, &desc, &cust, &revBy)
		in := "-"
		if inv != nil { in = *inv }
		rv := "-"
		if revBy != nil { rv = *revBy }
		fmt.Printf("Firma: %-22s | Inv: %-16s | Amt: %10.2f | Dir: %-15s | Cust: %-15s | RevBy: %-10s | Desc: %s\n", 
			name, in, amt, dir, cust, rv, desc)
	}
}
