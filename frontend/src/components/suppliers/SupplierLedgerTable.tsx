"use client";

import React from "react";
import { useTranslations } from "next-intl";
import { Search, Filter, ShoppingBag, CreditCard, FileText, RotateCcw, ChevronLeft, ChevronRight } from "lucide-react";
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
  totalCount?: number;
  page?: number;
  pageSize?: number;
  onPageChange?: (page: number) => void;
  onPageSizeChange?: (pageSize: number) => void;
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
  totalCount = 0,
  page = 1,
  pageSize = 50,
  onPageChange,
  onPageSizeChange,
}: SupplierLedgerTableProps) {
  const t = useTranslations("suppliers");
  const [statusFilter, setStatusFilter] = React.useState<"active" | "reversed" | "all">("all");

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

  const totalPages = Math.max(1, Math.ceil((totalCount || filteredTransactions.length) / pageSize));
  const startRecord = Math.min((page - 1) * pageSize + 1, totalCount || filteredTransactions.length);
  const endRecord = Math.min(page * pageSize, totalCount || filteredTransactions.length);

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
              <option value="purchase_return">Yalnızca Alış İadeleri (-)</option>
              <option value="payment_return">Yalnızca Ödeme İadeleri (+)</option>
            </select>
          </div>

          <div className="flex items-center gap-1.5 shrink-0">
            <select
              value={statusFilter}
              onChange={(e) => setStatusFilter(e.target.value as any)}
              className="h-9 text-xs border border-slate-300 rounded-md px-2.5 bg-white text-slate-700 font-medium focus:outline-none focus:ring-1 focus:ring-indigo-500"
            >
              <option value="all">Tüm Kayıtlar (Geçmiş Dahil)</option>
              <option value="active">Yalnızca Aktif Kayıtlar</option>
              <option value="reversed">İptal / Ters Kayıtlar</option>
            </select>
          </div>
        </div>

        <div className="flex items-center gap-3 self-end sm:self-center">
          <span className="text-xs text-slate-600 font-semibold bg-slate-100 px-2.5 py-1 rounded-md border border-slate-200">
            Toplam <strong className="text-slate-900 font-extrabold">{totalCount || filteredTransactions.length}</strong> Cari Hareket
          </span>
        </div>
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
                      {tx.direction === "purchase" ? (
                        <Badge variant="outline" className="gap-1 font-semibold border-rose-200 bg-rose-50 text-rose-700">
                          <ShoppingBag className="w-3 h-3" />
                          Alınan Mal (+)
                        </Badge>
                      ) : tx.direction === "purchase_return" ? (
                        <Badge variant="outline" className="gap-1 font-semibold border-amber-300 bg-amber-50 text-amber-800">
                          <RotateCcw className="w-3 h-3" />
                          Alış İadesi (-)
                        </Badge>
                      ) : tx.direction === "payment_return" ? (
                        <Badge variant="outline" className="gap-1 font-semibold border-blue-300 bg-blue-50 text-blue-800">
                          <RotateCcw className="w-3 h-3" />
                          Ödeme İadesi (+)
                        </Badge>
                      ) : (
                        <Badge variant="outline" className="gap-1 font-semibold border-emerald-200 bg-emerald-50 text-emerald-700">
                          <CreditCard className="w-3 h-3" />
                          Geçilen Ödeme (-)
                        </Badge>
                      )}
                    </td>
                    <td
                      className={`py-3 px-4 text-right font-mono font-bold whitespace-nowrap text-sm ${
                        tx.direction === "purchase_return"
                          ? "text-amber-700"
                          : tx.direction === "payment_return"
                          ? "text-blue-700"
                          : isPurchase
                          ? "text-rose-600"
                          : "text-emerald-600"
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

      {/* Pagination Bar */}
      {totalCount > 0 && onPageChange && (
        <div className="p-3 border-t border-slate-200 bg-slate-50/80 flex flex-col sm:flex-row items-center justify-between gap-3 text-xs text-slate-600 font-medium">
          <div className="flex items-center gap-2">
            <span>
              Gösterilen: <strong>{startRecord} - {endRecord}</strong> / Toplam <strong>{totalCount}</strong> kayıt
            </span>
            {onPageSizeChange && (
              <div className="flex items-center gap-1.5 ml-3">
                <span className="text-slate-400">|</span>
                <span>Sayfa Başı:</span>
                <select
                  value={pageSize}
                  onChange={(e) => onPageSizeChange(Number(e.target.value))}
                  className="h-7 text-xs border border-slate-300 rounded px-1.5 bg-white text-slate-700 font-semibold focus:outline-none"
                >
                  <option value={25}>25</option>
                  <option value={50}>50</option>
                  <option value={100}>100</option>
                  <option value={200}>200</option>
                </select>
              </div>
            )}
          </div>

          <div className="flex items-center gap-1.5">
            <Button
              variant="outline"
              size="sm"
              disabled={page <= 1 || loading}
              onClick={() => onPageChange(page - 1)}
              className="h-8 px-2.5 text-xs gap-1 font-medium bg-white"
            >
              <ChevronLeft className="w-3.5 h-3.5" />
              Önceki
            </Button>

            <span className="px-3 py-1 font-bold text-slate-800 bg-white border border-slate-200 rounded-md text-xs">
              {page} / {totalPages}
            </span>

            <Button
              variant="outline"
              size="sm"
              disabled={page >= totalPages || loading}
              onClick={() => onPageChange(page + 1)}
              className="h-8 px-2.5 text-xs gap-1 font-medium bg-white"
            >
              Sonraki
              <ChevronRight className="w-3.5 h-3.5" />
            </Button>
          </div>
        </div>
      )}
    </div>
  );
}
