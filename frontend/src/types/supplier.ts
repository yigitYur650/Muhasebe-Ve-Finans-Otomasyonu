export type SupplierDirection = "purchase" | "payment" | "purchase_return" | "payment_return";

export interface Supplier {
  id: string;
  tenant_id: string;
  name: string;
  is_active: boolean;
  created_at: string;
  total_purchase: string; // Decimal string representation
  total_payment: string;
  balance: string; // total_purchase - total_payment
  transaction_count: number;
}

export interface SupplierTransaction {
  id: string;
  tenant_id: string;
  supplier_id: string;
  supplier_name?: string;
  period_id: string;
  period_label?: string;
  invoice_no: string;
  customer_name: string;
  document_status: string;
  tx_date: string;
  direction: SupplierDirection;
  amount: string; // Decimal string representation
  description: string;
  created_by?: string;
  created_at: string;
  reversed_by?: string | null;
  is_reversal_entry?: boolean;
  was_reversed?: boolean;
}

export interface SupplierSummary {
  total_purchases: string;
  total_payments: string;
  net_balance: string;
  active_supplier_count: number;
  total_transaction_rows: number;
}

export interface SupplierImportResult {
  total_rows: number;
  imported_count: number;
  skipped_count: number;
  total_purchase: string;
  total_payment: string;
  suppliers_created: string[];
  errors: string[];
}
