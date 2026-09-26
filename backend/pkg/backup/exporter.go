package backup

import (
	"compress/gzip"
	"context"
	"database/sql/driver"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"
)

// SnapshotSummary holds statistics about the exported backup.
type SnapshotSummary struct {
	FilePath    string
	Filename    string
	SizeBytes   int64
	Duration    time.Duration
	RecordCount int
	Tables      map[string]int
}

// Exporter handles extracting PostgreSQL database snapshot into compressed SQL gzip files.
type Exporter struct {
	pool      *pgxpool.Pool
	backupDir string
}

// NewExporter creates a new Exporter instance.
func NewExporter(pool *pgxpool.Pool, backupDir string) *Exporter {
	if backupDir == "" {
		backupDir = "backups/daily"
	}
	return &Exporter{
		pool:      pool,
		backupDir: backupDir,
	}
}

// SchemaDDL defines the standalone schema definitions to ensure the backup is 100% self-contained.
const SchemaDDL = `
-- ==============================================================================
-- STANDALONE DATABASE SCHEMA DEFINITIONS (DDL)
-- ==============================================================================

CREATE TABLE IF NOT EXISTS public.tenants (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS public.tenant_members (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES public.tenants(id) ON DELETE CASCADE,
    user_id UUID NOT NULL,
    role TEXT NOT NULL CHECK (role IN ('admin', 'muhasebeci', 'standart')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, user_id)
);

CREATE TABLE IF NOT EXISTS public.periods (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES public.tenants(id) ON DELETE CASCADE,
    label TEXT NOT NULL,
    starting_balance NUMERIC(15,2) NOT NULL DEFAULT 0,
    status TEXT NOT NULL DEFAULT 'open' CHECK (status IN ('open', 'locked')),
    opened_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    locked_at TIMESTAMPTZ,
    UNIQUE (tenant_id, label)
);

CREATE TABLE IF NOT EXISTS public.transactions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES public.tenants(id) ON DELETE CASCADE,
    period_id UUID NOT NULL REFERENCES public.periods(id),
    direction TEXT NOT NULL CHECK (direction IN ('in', 'out')),
    channel TEXT NOT NULL,
    amount NUMERIC(15,2) NOT NULL CHECK (amount > 0),
    description TEXT,
    created_by UUID NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    reversed_by UUID REFERENCES public.transactions(id)
);

CREATE TABLE IF NOT EXISTS public.suppliers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES public.tenants(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    category TEXT NOT NULL DEFAULT 'diger',
    phone TEXT,
    notes TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, name)
);

CREATE TABLE IF NOT EXISTS public.supplier_transactions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES public.tenants(id) ON DELETE CASCADE,
    supplier_id UUID NOT NULL REFERENCES public.suppliers(id) ON DELETE CASCADE,
    period_id UUID NOT NULL REFERENCES public.periods(id),
    direction TEXT NOT NULL CHECK (direction IN ('purchase', 'payment')),
    amount NUMERIC(15,2) NOT NULL CHECK (amount > 0),
    description TEXT,
    transaction_date DATE NOT NULL DEFAULT CURRENT_DATE,
    created_by UUID,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    reversed_by UUID REFERENCES public.supplier_transactions(id)
);

CREATE TABLE IF NOT EXISTS public.user_security (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL UNIQUE,
    email TEXT NOT NULL UNIQUE,
    security_question TEXT NOT NULL,
    security_answer_hash TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

`

// formatSQLValue formats generic driver values into valid PostgreSQL literal strings.
func formatSQLValue(v interface{}) string {
	if v == nil {
		return "NULL"
	}

	// Evaluate driver.Valuer (e.g. pgtype.Numeric, pgtype.Date, etc.)
	if valuer, ok := v.(driver.Valuer); ok {
		if val, err := valuer.Value(); err == nil {
			if val == nil {
				return "NULL"
			}
			v = val
		}
	}

	switch val := v.(type) {
	case string:
		return "'" + strings.ReplaceAll(val, "'", "''") + "'"

	case []byte:
		if len(val) == 16 {
			// Format as canonical UUID string (e.g. 'e1909259-bde7-43ca-b7dc-739ccf1acd03')
			return fmt.Sprintf("'%02x%02x%02x%02x-%02x%02x-%02x%02x-%02x%02x-%02x%02x%02x%02x%02x%02x'",
				val[0], val[1], val[2], val[3],
				val[4], val[5],
				val[6], val[7],
				val[8], val[9],
				val[10], val[11], val[12], val[13], val[14], val[15],
			)
		}
		return fmt.Sprintf("'\\x%x'", val)

	case [16]byte:
		return fmt.Sprintf("'%02x%02x%02x%02x-%02x%02x-%02x%02x-%02x%02x-%02x%02x%02x%02x%02x%02x'",
			val[0], val[1], val[2], val[3],
			val[4], val[5],
			val[6], val[7],
			val[8], val[9],
			val[10], val[11], val[12], val[13], val[14], val[15],
		)

	case time.Time:
		if val.IsZero() || val.Year() < 1970 {
			return "'2026-08-20 00:00:00+00'"
		}
		return "'" + val.Format("2006-01-02 15:04:05.999999-07") + "'"

	case bool:
		if val {
			return "TRUE"
		}
		return "FALSE"

	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		return fmt.Sprintf("%d", val)

	case float32, float64:
		return fmt.Sprintf("%.2f", val)

	case decimal.Decimal:
		return val.StringFixed(2)

	case pgtype.Numeric:
		if !val.Valid {
			return "NULL"
		}
		if val.NaN {
			return "'NaN'"
		}
		if val.Int != nil {
			d := decimal.NewFromBigInt(val.Int, val.Exp)
			return d.StringFixed(2)
		}
		return "0.00"

	case *big.Int:
		if val == nil {
			return "NULL"
		}
		return val.String()

	default:
		str := fmt.Sprintf("%v", val)
		// Check for struct string format like {2890000 -2 false finite true}
		if strings.HasPrefix(str, "{") && strings.HasSuffix(str, "}") {
			parts := strings.Fields(strings.Trim(str, "{}"))
			if len(parts) >= 2 {
				// Parse big int and exp
				if bi, ok := new(big.Int).SetString(parts[0], 10); ok {
					var exp int32
					if _, err := fmt.Sscanf(parts[1], "%d", &exp); err == nil {
						d := decimal.NewFromBigInt(bi, exp)
						return d.StringFixed(2)
					}
				}
			}
		}
		if stringer, ok := v.(fmt.Stringer); ok {
			return "'" + strings.ReplaceAll(stringer.String(), "'", "''") + "'"
		}
		return "'" + strings.ReplaceAll(str, "'", "''") + "'"
	}
}

// CreateSnapshot dumps all core financial tables into a compressed .sql.gz file.
func (e *Exporter) CreateSnapshot(ctx context.Context) (*SnapshotSummary, error) {
	start := time.Now()

	if err := os.MkdirAll(e.backupDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create backup directory: %w", err)
	}

	timestamp := time.Now().Format("2006_01_02_150405")
	filename := fmt.Sprintf("backup_oncu_otogaz_%s.sql.gz", timestamp)
	fullPath := filepath.Join(e.backupDir, filename)

	file, err := os.Create(fullPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create backup file: %w", err)
	}
	defer file.Close()

	gzWriter := gzip.NewWriter(file)
	defer gzWriter.Close()

	tables := []string{
		"tenants",
		"tenant_members",
		"periods",
		"transactions",
		"suppliers",
		"supplier_transactions",
		"user_security",
	}

	header := fmt.Sprintf("-- ==============================================================================\n"+
		"-- ÖNCÜ OTOGAZ — OTOMATİK VERİTABANI YEDEĞİ (DAILY BACKUP SNAPSHOT)\n"+
		"-- Tarih: %s\n"+
		"-- Format: PostgreSQL SQL Dump (Self-Contained DDL + Data, Gzip Compressed)\n"+
		"-- ==============================================================================\n\n"+
		"BEGIN;\n\n"+
		"%s\n"+
		"-- ==============================================================================\n"+
		"-- DATA RESTORE INSERTS\n"+
		"-- ==============================================================================\n\n",
		time.Now().Format("2006-01-02 15:04:05 MST"),
		SchemaDDL,
	)
	if _, err := gzWriter.Write([]byte(header)); err != nil {
		return nil, err
	}

	tableStats := make(map[string]int)
	totalRecords := 0

	for _, tbl := range tables {
		// Note: Use created_at / transaction_date ordering
		query := fmt.Sprintf("SELECT * FROM public.%s ORDER BY created_at ASC", tbl)
		if tbl == "supplier_transactions" {
			query = fmt.Sprintf("SELECT * FROM public.%s ORDER BY transaction_date ASC, created_at ASC", tbl)
		}

		rows, err := e.pool.Query(ctx, query)
		if err != nil {
			// Fallback to simple select if order column differs
			rows, err = e.pool.Query(ctx, fmt.Sprintf("SELECT * FROM public.%s", tbl))
			if err != nil {
				continue
			}
		}

		fieldDescs := rows.FieldDescriptions()
		colNames := make([]string, len(fieldDescs))
		for i, fd := range fieldDescs {
			colNames[i] = string(fd.Name)
		}

		count := 0
		var sqlRows strings.Builder
		sqlRows.WriteString(fmt.Sprintf("-- Tablo: public.%s\n", tbl))

		for rows.Next() {
			values, err := rows.Values()
			if err != nil {
				rows.Close()
				return nil, fmt.Errorf("error reading row from %s: %w", tbl, err)
			}

			valStrings := make([]string, len(values))
			for i, v := range values {
				valStrings[i] = formatSQLValue(v)
			}

			sqlRows.WriteString(fmt.Sprintf(
				"INSERT INTO public.%s (%s) VALUES (%s) ON CONFLICT DO NOTHING;\n",
				tbl,
				strings.Join(colNames, ", "),
				strings.Join(valStrings, ", "),
			))
			count++
		}
		rows.Close()

		sqlRows.WriteString("\n")
		if _, err := gzWriter.Write([]byte(sqlRows.String())); err != nil {
			return nil, err
		}

		tableStats[tbl] = count
		totalRecords += count
	}

	footer := "COMMIT;\n-- YEDEKLEME BAŞARIYLA TAMAMLANDI\n"
	if _, err := gzWriter.Write([]byte(footer)); err != nil {
		return nil, err
	}

	if err := gzWriter.Flush(); err != nil {
		return nil, err
	}
	_ = gzWriter.Close()
	_ = file.Close()

	fileInfo, err := os.Stat(fullPath)
	if err != nil {
		return nil, fmt.Errorf("failed to stat backup file: %w", err)
	}

	return &SnapshotSummary{
		FilePath:    fullPath,
		Filename:    filename,
		SizeBytes:   fileInfo.Size(),
		Duration:    time.Since(start),
		RecordCount: totalRecords,
		Tables:      tableStats,
	}, nil
}
