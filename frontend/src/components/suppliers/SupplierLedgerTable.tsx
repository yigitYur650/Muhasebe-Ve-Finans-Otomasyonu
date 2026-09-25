"use client";

import React from "react";
import { useTranslations } from "next-intl";
import { Search, Filter, ShoppingBag, CreditCard, FileText, RotateCcw } from "lucide-react";
import { Input } from "@/components/ui/input";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { SupplierTransaction } from "@/types/supplier";
import { formatTL } from "@/lib/decimal";

interface SupplierLedgerTableProps {
  transactions: SupplierTransaction[];
  searchQuery: string;
  onSearchChange: (q: string) => void;
  directionFilter: string;
  onDirectionFilterChange: (d: string) => void;
  isPeriodLocked?: boolean;
  onReverse?: (tx: SupplierTransaction) => void;
  loading?: boolean;
}

export function SupplierLedgerTable({
  transactions,
  searchQuery,
  onSearchChange,
  directionFilter,
  onDirectionFilterChange,
  isPeriodLocked = false,
  onReverse,
  loading,
}: SupplierLedgerTableProps) {
  const t = useTranslations("suppliers");
  const [statusFilter, setStatusFilter] = React.useState<"active" | "reversed" | "all">("active");

  const formatDate = (dateStr: string) => {
    if (!dateStr) return "-";
    try {
      const d = new Date(dateStr);
      return d.toLocaleDateString("tr-TR", {
        day: "2-digit",
        month: "2-digit",
        year: "numeric",
      });
    } catch {
      return dateStr;
    }
  };

  const filteredTransactions = React.useMemo(() => {
    return transactions.filter((tx) => {
      const isReversedOrReversal = Boolean(tx.reversed_by || tx.is_reversal_entry || tx.was_reversed);
      if (statusFilter === "active" && isReversedOrReversal) {
        return false;
      }
      if (statusFilter === "reversed" && !isReversedOrReversal) {
        return false;
      }
      return true;
    });
  }, [transactions, statusFilter]);

  return (
    <div className="border border-slate-200 rounded-xl bg-white shadow-sm overflow-hidden mt-4">
      {/* Table Header Controls */}
      <div className="p-4 border-b border-slate-200 bg-slate-50/50 flex flex-col sm:flex-row items-center justify-between gap-3">
        <div className="flex flex-wrap items-center gap-3 w-full sm:w-auto">
          <div className="relative w-full sm:w-64">
            <Search className="w-4 h-4 text-slate-400 absolute left-3 top-1/2 -translate-y-1/2" />
            <Input
              type="text"
              placeholder={t("searchPlaceholder")}
              value={searchQuery}
              onChange={(e) => onSearchChange(e.target.value)}
              className="pl-9 h-9 text-xs bg-white"
            />
          </div>

          <div className="flex items-center gap-1.5 shrink-0">
            <Filter className="w-4 h-4 text-slate-400" />
            <select
              value={directionFilter}
              onChange={(e) => onDirectionFilterChange(e.target.value)}
              className="h-9 text-xs border border-slate-300 rounded-md px-2.5 bg-white text-slate-700 font-medium focus:outline-none focus:ring-1 focus:ring-indigo-500"
            >
              <option value="">{t("allDirections")}</option>
              <option value="purchase">{t("purchasesOnly")}</option>
              <option value="payment">{t("paymentsOnly")}</option>
            </select>
          </div>

          <div className="flex items-center gap-1.5 shrink-0">
            <select
              value={statusFilter}
              onChange={(e) => setStatusFilter(e.target.value as any)}
              className="h-9 text-xs border border-slate-300 rounded-md px-2.5 bg-white text-slate-700 font-medium focus:outline-none focus:ring-1 focus:ring-indigo-500"
            >
              <option value="active">Yalnızca Aktif Kayıtlar</option>
              <option value="reversed">İptal / Ters Kayıtlar</option>
              <option value="all">Tüm Kayıtlar (Geçmiş Dahil)</option>
            </select>
          </div>
        </div>

        <span className="text-xs text-slate-500 font-semibold self-end sm:self-center">
          {filteredTransactions.length} {t("recordsCount")}
        </span>
      </div>

      {/* Table View */}
      <div className="overflow-x-auto">
        <table className="w-full text-left border-collapse text-xs">
          <thead>
            <tr className="border-b border-slate-200 bg-slate-100/75 text-slate-600 font-bold uppercase tracking-wider text-[11px]">
              <th className="py-3 px-4">{t("date")}</th>
              <th className="py-3 px-4">{t("supplierName")}</th>
              <th className="py-3 px-4">{t("invoiceNo")}</th>
              <th className="py-3 px-4">{t("customerName")}</th>
              <th className="py-3 px-4">{t("documentStatus")}</th>
              <th className="py-3 px-4">{t("direction")}</th>
              <th className="py-3 px-4 text-right">{t("amount")}</th>
              <th className="py-3 px-4">{t("description")}</th>
              <th className="py-3 px-4 text-center">İşlem</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-slate-100 font-medium text-slate-700">
            {loading ? (
              <tr>
                <td colSpan={9} className="py-12 text-center text-slate-400">
                  <span className="inline-block animate-spin mr-2">⏳</span> Yükleniyor...
                </td>
              </tr>
            ) : filteredTransactions.length === 0 ? (
              <tr>
                <td colSpan={9} className="py-12 text-center text-slate-400">
                  <FileText className="w-8 h-8 text-slate-300 mx-auto mb-2" />
                  {t("noTransactions")}
                </td>
              </tr>
            ) : (
              filteredTransactions.map((tx) => {
                const isPurchase = tx.direction === "purchase";
                const isReversal = Boolean(tx.is_reversal_entry || tx.reversed_by || tx.description?.startsWith("[İPTAL/TERS KAYIT]"));
                const wasReversed = Boolean(tx.was_reversed);

                return (
                  <tr
                    key={tx.id}
                    className={`transition-colors ${
                      isReversal
                        ? "bg-amber-50/40 text-slate-500 italic hover:bg-amber-50/70"
                        : wasReversed
                        ? "bg-slate-50/80 text-slate-400 line-through hover:bg-slate-100/80"
                        : "hover:bg-slate-50/75"
                    }`}
                  >
                    <td className="py-3 px-4 whitespace-nowrap font-semibold">
                      {formatDate(tx.tx_date)}
                    </td>
                    <td className="py-3 px-4 font-bold whitespace-nowrap">
                      {tx.supplier_name || "-"}
                      {isReversal && (
                        <span className="ml-2 inline-flex items-center px-1.5 py-0.5 rounded text-[9px] font-bold bg-amber-100 text-amber-800 border border-amber-300 not-italic">
                          TERS KAYIT
                        </span>
                      )}
                      {wasReversed && (
                        <span className="ml-2 inline-flex items-center px-1.5 py-0.5 rounded text-[9px] font-bold bg-slate-200 text-slate-700 border border-slate-300 no-underline inline-block">
                          İPTAL EDİLDİ
                        </span>
                      )}
                    </td>
                    <td className="py-3 px-4 font-mono text-[11px] whitespace-nowrap">
                      {tx.invoice_no || "-"}
                    </td>
                    <td className="py-3 px-4 font-semibold">
                      {tx.customer_name || "-"}
                    </td>
                    <td className="py-3 px-4 whitespace-nowrap">
                      {tx.document_status ? (
                        <span className="inline-flex items-center px-2 py-0.5 rounded text-[10px] font-semibold bg-slate-100 text-slate-700 border border-slate-200">
                          {tx.document_status}
                        </span>
                      ) : (
                        "-"
                      )}
                    </td>
                    <td className="py-3 px-4 whitespace-nowrap">
                      {isPurchase ? (
                        <Badge variant="outline" className={`gap-1 font-semibold ${isReversal ? "border-amber-200 bg-amber-50 text-amber-800" : "border-rose-200 bg-rose-50 text-rose-700"}`}>
                          <ShoppingBag className="w-3 h-3" />
                          Alınan Mal (+)
                        </Badge>
                      ) : (
                        <Badge variant="outline" className={`gap-1 font-semibold ${isReversal ? "border-amber-200 bg-amber-50 text-amber-800" : "border-emerald-200 bg-emerald-50 text-emerald-700"}`}>
                          <CreditCard className="w-3 h-3" />
                          Geçilen Ödeme (-)
                        </Badge>
                      )}
                    </td>
                    <td
                      className={`py-3 px-4 text-right font-mono font-bold whitespace-nowrap text-sm ${
                        isReversal ? "text-amber-700" : isPurchase ? "text-rose-600" : "text-emerald-600"
                      }`}
                    >
                      {formatTL(tx.amount)}
                    </td>
                    <td className="py-3 px-4 max-w-xs truncate" title={tx.description}>
                      {tx.description || "-"}
                    </td>
                    <td className="py-3 px-4 text-center whitespace-nowrap">
                      {!isReversal && !wasReversed && onReverse && (
                        <Button
                          variant="ghost"
                          size="sm"
                          disabled={isPeriodLocked}
                          onClick={() => onReverse(tx)}
                          className="h-7 px-2 text-[11px] text-rose-600 hover:text-rose-700 hover:bg-rose-50 gap-1 font-semibold"
                          title="Ters Kayıt Yap (İptal Et)"
                        >
                          <RotateCcw className="w-3 h-3" />
                          İptal / Ters Kayıt
                        </Button>
                      )}
                    </td>
                  </tr>
                );
              })
            )}
          </tbody>
        </table>
      </div>
    </div>
  );
}
