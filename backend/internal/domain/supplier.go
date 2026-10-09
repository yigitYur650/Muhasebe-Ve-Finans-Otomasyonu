package domain

import (
	"context"
	"io"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// Supplier directions (Muhasebe Usulü 4 Temel İşlem Türü)
const (
	SupplierDirectionPurchase       = "purchase"        // Toptancıdan alınan mal (Borcu artırır +)
	SupplierDirectionPayment        = "payment"         // Toptancıya yapılan ödeme (Borcu azaltır -)
	SupplierDirectionPurchaseReturn = "purchase_return" // Alış İadesi / Alış İptali (Alış tutarından düşer -)
	SupplierDirectionPaymentReturn  = "payment_return"  // Ödeme İadesi / Ödeme İptali (Ödeme tutarından düşer -)
)

// Supplier represents a vendor/supplier entity.
type Supplier struct {
	ID               uuid.UUID       `json:"id"`
	TenantID         uuid.UUID       `json:"tenant_id"`
	Name             string          `json:"name"`
	IsActive         bool            `json:"is_active"`
	CreatedAt        time.Time       `json:"created_at"`
	TotalPurchase    decimal.Decimal `json:"total_purchase"`
	TotalPayment     decimal.Decimal `json:"total_payment"`
	Balance          decimal.Decimal `json:"balance"` // TotalPurchase - TotalPayment
	TransactionCount int             `json:"transaction_count"`
}

// SupplierTransaction represents a ledger entry for a supplier.
type SupplierTransaction struct {
	ID             uuid.UUID       `json:"id"`
	TenantID       uuid.UUID       `json:"tenant_id"`
	SupplierID     uuid.UUID       `json:"supplier_id"`
	SupplierName   string          `json:"supplier_name,omitempty"`
	PeriodID       uuid.UUID       `json:"period_id"`
	PeriodLabel    string          `json:"period_label,omitempty"`
	InvoiceNo      string          `json:"invoice_no"`
	CustomerName   string          `json:"customer_name"`
	DocumentStatus string          `json:"document_status"`
	TxDate         time.Time       `json:"tx_date"`
	Direction      string          `json:"direction"` // 'purchase' | 'payment'
	Amount         decimal.Decimal `json:"amount"`
	Description    string          `json:"description"`
	CreatedBy      *uuid.UUID      `json:"created_by,omitempty"`
	CreatedAt      time.Time       `json:"created_at"`
	RunningBalance decimal.Decimal `json:"running_balance,omitempty"`
	ReversedBy     *uuid.UUID      `json:"reversed_by,omitempty"`
}

// SupplierTransactionFilter contains filtering options for querying supplier transactions.
type SupplierTransactionFilter struct {
	SupplierID *uuid.UUID
	PeriodID   *uuid.UUID
	Direction  string
	Search     string
	DateFrom   *time.Time
	DateTo     *time.Time
	Limit      int
	Offset     int
}

// SupplierSummary provides aggregate stats across suppliers.
type SupplierSummary struct {
	TotalPurchases       decimal.Decimal `json:"total_purchases"`
	TotalPayments        decimal.Decimal `json:"total_payments"`
	NetBalance           decimal.Decimal `json:"net_balance"` // TotalPurchases - TotalPayments
	ActiveSupplierCount  int             `json:"active_supplier_count"`
	TotalTransactionRows int             `json:"total_transaction_rows"`
}

// SupplierImportResult contains the execution summary of an Excel import.
type SupplierImportResult struct {
	TotalRows        int             `json:"total_rows"`
	ImportedCount    int             `json:"imported_count"`
	SkippedCount     int             `json:"skipped_count"`
	TotalPurchase    decimal.Decimal `json:"total_purchase"`
	TotalPayment     decimal.Decimal `json:"total_payment"`
	SuppliersCreated []string        `json:"suppliers_created"`
	Errors           []string        `json:"errors"`
}

// SupplierRepository defines database operations for suppliers and their transactions.
type SupplierRepository interface {
	GetSuppliersWithBalances(ctx context.Context, tenantID uuid.UUID, periodID *uuid.UUID) ([]Supplier, error)
	GetSupplierByID(ctx context.Context, tenantID, supplierID uuid.UUID) (*Supplier, error)
	FindOrCreateSupplier(ctx context.Context, tenantID uuid.UUID, name string) (*Supplier, error)
	CreateSupplier(ctx context.Context, s *Supplier) error
	CreateTransaction(ctx context.Context, tx *SupplierTransaction) error
	GetTransactionByID(ctx context.Context, tenantID, txID uuid.UUID) (*SupplierTransaction, error)
	ReverseTransaction(ctx context.Context, tenantID, origID uuid.UUID, revTx *SupplierTransaction) error
	GetTransactionsBySupplier(ctx context.Context, tenantID, supplierID uuid.UUID, filter SupplierTransactionFilter) ([]SupplierTransaction, int, error)
	GetAllTransactions(ctx context.Context, tenantID uuid.UUID, filter SupplierTransactionFilter) ([]SupplierTransaction, int, error)
	BatchInsertTransactions(ctx context.Context, tenantID uuid.UUID, transactions []SupplierTransaction) (int, error)
	GetSummary(ctx context.Context, tenantID uuid.UUID, periodID *uuid.UUID) (*SupplierSummary, error)
}

// SupplierService defines business logic for supplier operations.
type SupplierService interface {
	ListSuppliers(ctx context.Context, tenantID uuid.UUID, periodID *uuid.UUID) ([]Supplier, error)
	GetSupplier(ctx context.Context, tenantID, supplierID uuid.UUID) (*Supplier, error)
	CreateSupplier(ctx context.Context, tenantID uuid.UUID, name string) (*Supplier, error)
	CreateTransaction(ctx context.Context, tenantID uuid.UUID, tx *SupplierTransaction) (*SupplierTransaction, error)
	ReverseTransaction(ctx context.Context, tenantID, origID uuid.UUID, reason string, createdBy *uuid.UUID) (*SupplierTransaction, error)
	ListSupplierTransactions(ctx context.Context, tenantID, supplierID uuid.UUID, filter SupplierTransactionFilter) ([]SupplierTransaction, int, error)
	ListAllTransactions(ctx context.Context, tenantID uuid.UUID, filter SupplierTransactionFilter) ([]SupplierTransaction, int, error)
	GetSummary(ctx context.Context, tenantID uuid.UUID, periodID *uuid.UUID) (*SupplierSummary, error)
	ImportExcel(ctx context.Context, tenantID, periodID uuid.UUID, r io.Reader, sheetName string, userID *uuid.UUID) (*SupplierImportResult, error)
}
