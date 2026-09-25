"use client";

import React from "react";
import { useTranslations } from "next-intl";
import { Truck, ShoppingCart, CreditCard, Layers, CheckCircle2 } from "lucide-react";
import { Card, CardContent } from "@/components/ui/card";
import { Supplier, SupplierSummary } from "@/types/supplier";
import { formatTL } from "@/lib/decimal";

interface SupplierSummaryCardsProps {
  suppliers: Supplier[];
  summary: SupplierSummary | null;
  selectedSupplierId: string | "all";
  onSelectSupplier: (id: string | "all") => void;
  loading?: boolean;
}

export function SupplierSummaryCards({
  suppliers,
  summary,
  selectedSupplierId,
  onSelectSupplier,
  loading,
}: SupplierSummaryCardsProps) {
  const t = useTranslations("suppliers");

  const totalDebt = summary?.net_balance ?? "0.00";
  const totalPurchases = summary?.total_purchases ?? "0.00";
  const totalPayments = summary?.total_payments ?? "0.00";

  return (
    <div className="space-y-4 mb-6">
      {/* 1. Global Macro KPIs */}
      <div className="grid grid-cols-1 sm:grid-cols-3 gap-4">
        {/* Total Debt */}
        <Card
          onClick={() => onSelectSupplier("all")}
          className={`border cursor-pointer transition-all ${
            selectedSupplierId === "all"
              ? "ring-2 ring-indigo-500 border-indigo-400 bg-indigo-50/40 shadow-sm"
              : "border-slate-200 bg-white hover:border-slate-300 shadow-sm"
          }`}
        >
          <CardContent className="p-4 flex items-center justify-between">
            <div>
              <div className="flex items-center gap-2">
                <span className="text-xs font-bold text-slate-500 uppercase tracking-wider">
                  {t("totalDebt")}
                </span>
                {selectedSupplierId === "all" && (
                  <span className="text-[10px] font-bold px-1.5 py-0.5 rounded bg-indigo-100 text-indigo-700">
                    Seçili
                  </span>
                )}
              </div>
              <div className="text-2xl font-extrabold text-rose-600 mt-1">
                {loading ? (
                  <span className="animate-pulse bg-slate-200 h-7 w-28 block rounded" />
                ) : (
                  formatTL(totalDebt)
                )}
              </div>
              <span className="text-xs text-slate-400">
                {suppliers.length} {t("activeCount")}
              </span>
            </div>
            <div className="p-3 rounded-xl bg-rose-50 text-rose-600">
              <Layers className="h-6 w-6" />
            </div>
          </CardContent>
        </Card>

        {/* Total Purchases */}
        <Card className="border border-slate-200 bg-white shadow-sm">
          <CardContent className="p-4 flex items-center justify-between">
            <div>
              <span className="text-xs font-bold text-slate-500 uppercase tracking-wider">
                {t("totalPurchases")}
              </span>
              <div className="text-2xl font-extrabold text-slate-900 mt-1">
                {loading ? (
                  <span className="animate-pulse bg-slate-200 h-7 w-28 block rounded" />
                ) : (
                  formatTL(totalPurchases)
                )}
              </div>
              <span className="text-xs text-slate-400">Alınan Mal Tutarı</span>
            </div>
            <div className="p-3 rounded-xl bg-blue-50 text-blue-600">
              <ShoppingCart className="h-6 w-6" />
            </div>
          </CardContent>
        </Card>

        {/* Total Payments */}
        <Card className="border border-slate-200 bg-white shadow-sm">
          <CardContent className="p-4 flex items-center justify-between">
            <div>
              <span className="text-xs font-bold text-slate-500 uppercase tracking-wider">
                {t("totalPayments")}
              </span>
              <div className="text-2xl font-extrabold text-emerald-600 mt-1">
                {loading ? (
                  <span className="animate-pulse bg-slate-200 h-7 w-28 block rounded" />
                ) : (
                  formatTL(totalPayments)
                )}
              </div>
              <span className="text-xs text-slate-400">Geçilen Ödemeler</span>
            </div>
            <div className="p-3 rounded-xl bg-emerald-50 text-emerald-600">
              <CreditCard className="h-6 w-6" />
            </div>
          </CardContent>
        </Card>
      </div>

      {/* 2. Individual Supplier Balance Cards (Horizontal Scroll or Grid) */}
      {suppliers.length > 0 && (
        <div>
          <div className="flex items-center justify-between mb-2">
            <h4 className="text-xs font-bold text-slate-600 uppercase tracking-wider">
              Firma Bazlı Bakiye Çetelesi (Filtrelemek İçin Tıklayın)
            </h4>
          </div>

          <div className="grid grid-cols-1 sm:grid-cols-2 md:grid-cols-3 lg:grid-cols-5 gap-3">
            {suppliers.map((s) => {
              const isSelected = selectedSupplierId === s.id;
              const hasDebt = parseFloat(s.balance) > 0;

              return (
                <div
                  key={s.id}
                  onClick={() => onSelectSupplier(isSelected ? "all" : s.id)}
                  className={`p-3.5 rounded-xl border cursor-pointer transition-all text-left relative overflow-hidden ${
                    isSelected
                      ? "ring-2 ring-indigo-600 border-indigo-500 bg-indigo-50/50 shadow-sm"
                      : "border-slate-200 bg-white hover:border-slate-300 hover:shadow-sm"
                  }`}
                >
                  <div className="flex items-start justify-between">
                    <div className="flex items-center gap-1.5 font-bold text-sm text-slate-900 truncate pr-2">
                      <Truck className="w-4 h-4 text-indigo-600 shrink-0" />
                      <span className="truncate">{s.name}</span>
                    </div>
                    {isSelected && (
                      <CheckCircle2 className="w-4 h-4 text-indigo-600 shrink-0" />
                    )}
                  </div>

                  <div className="mt-2.5">
                    <span className="text-[11px] font-semibold text-slate-500 block">
                      {t("balance")}
                    </span>
                    <div
                      className={`text-lg font-black tracking-tight ${
                        hasDebt ? "text-rose-600" : "text-emerald-700"
                      }`}
                    >
                      {formatTL(s.balance)}
                    </div>
                  </div>

                  <div className="mt-2 pt-2 border-t border-slate-100 flex items-center justify-between text-[11px] text-slate-500 font-medium">
                    <span title="Alınan Mal">Alış: {formatTL(s.total_purchase)}</span>
                    <span title="Geçilen Ödeme">Ödeme: {formatTL(s.total_payment)}</span>
                  </div>
                </div>
              );
            })}
          </div>
        </div>
      )}
    </div>
  );
}
