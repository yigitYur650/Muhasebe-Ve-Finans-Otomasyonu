"use client";

import { useState, useEffect } from "react";
import { useTranslations } from "next-intl";
import { useParams } from "next/navigation";
import { Header } from "@/components/shared/Header";
import { PeriodBadge } from "@/components/shared/PeriodBadge";
import { TransactionTable } from "@/components/ledger/TransactionTable";
import { CreateTransactionDialog } from "@/components/ledger/CreateTransactionDialog";
import { ReverseTransactionDialog } from "@/components/ledger/ReverseTransactionDialog";
import { PeriodActionDialog } from "@/components/ledger/PeriodActionDialog";
import { PeriodSelector } from "@/components/ledger/PeriodSelector";
import { KpiSummaryCards } from "@/components/ledger/KpiSummaryCards";
import { PeriodHistoryView } from "@/components/ledger/PeriodHistoryView";
import { ExportCsvButton } from "@/components/ledger/ExportCsvButton";
import { ImportCsvDialog } from "@/components/ledger/ImportCsvDialog";
import { SupplierSummaryCards } from "@/components/suppliers/SupplierSummaryCards";
import { SupplierLedgerTable } from "@/components/suppliers/SupplierLedgerTable";
import { ImportSupplierExcelDialog } from "@/components/suppliers/ImportSupplierExcelDialog";
import { CreateSupplierTransactionDialog } from "@/components/suppliers/CreateSupplierTransactionDialog";
import { ReverseSupplierTransactionDialog } from "@/components/suppliers/ReverseSupplierTransactionDialog";
import { SupplierTransaction } from "@/types/supplier";
import { Button } from "@/components/ui/button";
import { createClient } from "@/lib/supabase/client";
import { usePeriods } from "@/hooks/usePeriods";
import { useTransactions } from "@/hooks/useTransactions";
import { useSuppliers } from "@/hooks/useSuppliers";
import { apiFetch } from "@/lib/api";
import {
  PlusCircle,
  Lock,
  Unlock,
  Calendar,
  AlertTriangle,
  FileSpreadsheet,
  Archive,
  Upload,
  Wallet,
  Truck,
} from "lucide-react";

export default function HomePage() {
  const routeParams = useParams();
  const locale = (routeParams?.locale as string) || "tr";
  const tCommon = useTranslations("common");
  const tPeriod = useTranslations("period");
  const tTx = useTranslations("transaction");
  const tHistory = useTranslations("history");
  const tImport = useTranslations("import_export");
  const tNav = useTranslations("navigation");
  const tSuppliers = useTranslations("suppliers");

  // Mounted state guard to eliminate hydration mismatch (#418, #423)
  const [isMounted, setIsMounted] = useState<boolean>(false);
  const [userRole] = useState<"admin" | "muhasebeci" | "standart">("admin");
  const [currentUserLabel, setCurrentUserLabel] = useState<string>("Öncü Otogaz Yönetici");
  
  // Dual-Ledger Main Tab Switcher: "cash" | "suppliers"
  const [mainTab, setMainTab] = useState<"cash" | "suppliers">("cash");
  const [activeTab, setActiveTab] = useState<"ledger" | "history">("ledger");

  // Supplier Modals
  const [supplierExcelModalOpen, setSupplierExcelModalOpen] = useState<boolean>(false);
  const [createSupplierTxModalOpen, setCreateSupplierTxModalOpen] = useState<boolean>(false);
  const [reverseSupplierModalOpen, setReverseSupplierModalOpen] = useState<boolean>(false);
  const [targetSupplierTxForReverse, setTargetSupplierTxForReverse] = useState<SupplierTransaction | null>(null);

  // Modular custom hooks for Cash Ledger
  const {
    periods,
    selectedPeriodId,
    setSelectedPeriodId,
    selectedPeriod,
    periodStatus,
    periodLabel,
    historyItems,
    periodModalMode,
    setPeriodModalMode,
    fetchPeriods,
    fetchPeriodHistory,
    handleLockPeriod,
    handleUnlockPeriod,
  } = usePeriods(activeTab);

  const {
    transactions,
    loadingSummary,
    kpiSummaryData,
    createModalOpen,
    setCreateModalOpen,
    reverseModalOpen,
    setReverseModalOpen,
    importModalOpen,
    setImportModalOpen,
    targetTxForReverse,
    setTargetTxForReverse,
    fetchLiveData,
    handleCreateTransaction,
    handleReverseTransaction,
  } = useTransactions(selectedPeriod, currentUserLabel);

  // Supplier Hook (Dual-Ledger)
  const {
    suppliers,
    summary: supplierSummary,
    selectedSupplierId,
    setSelectedSupplierId,
    transactions: supplierTransactions,
    totalCount: supplierTotalCount,
    page: supplierPage,
    setPage: setSupplierPage,
    pageSize: supplierPageSize,
    setPageSize: setSupplierPageSize,
    loadingSuppliers,
    loadingTransactions: loadingSupplierTxs,
    searchQuery: supplierSearch,
    setSearchQuery: setSupplierSearch,
    directionFilter: supplierDirectionFilter,
    setDirectionFilter: setSupplierDirectionFilter,
    handleReverseSupplierTransaction,
    refreshAll: refreshSuppliers,
  } = useSuppliers(selectedPeriod?.id);

  useEffect(() => {
    setIsMounted(true);
    async function loadUserData() {
      try {
        const supabase = createClient();
        const { data } = await supabase.auth.getSession();
        if (data?.session?.user) {
          const u = data.session.user;
          const name =
            (u.user_metadata?.full_name as string) ||
            (u.user_metadata?.name as string) ||
            u.email?.split("@")[0] ||
            "Öncü Otogaz";
          setCurrentUserLabel(name);
        }
      } catch {
        // Fallback
      }
    }
    loadUserData();
  }, []);

  // Handler: Open Next Period
  const handleOpenNextPeriod = async (label: string, idempotencyKey: string) => {
    try {
      const res = await apiFetch<any>(`/periods/open-next`, {
        method: "POST",
        headers: { "Idempotency-Key": idempotencyKey },
        body: JSON.stringify({ label }),
      });

      if (res.success && res.data?.id) {
        setSelectedPeriodId(res.data.id);
        fetchPeriods();
        fetchPeriodHistory();
      } else {
        alert(res.error?.message || "Yeni dönem açılamadı.");
      }
    } catch (err: any) {
      console.error("Dönem açma API hatası:", err);
      alert(err?.message || "Sunucu bağlantı hatası: Yeni dönem açılamadı.");
    }
  };

  if (!isMounted) return null;

  const currentPeriodTxs = transactions.filter((tx) => tx.periodId === selectedPeriod?.id);

  return (
    <div className="min-h-screen bg-slate-50">
      <Header tenantName={tCommon("tenantName")} userRole={userRole} locale={locale} />

      <main className="container mx-auto px-4 py-8 max-w-7xl">
        {/* Dual-Ledger Upper Navigation Tabs */}
        <div className="flex items-center justify-between border-b border-slate-200 pb-3 mb-6">
          <div className="flex items-center gap-2">
            <button
              onClick={() => setMainTab("cash")}
              className={`flex items-center gap-2 px-4 py-2 rounded-xl text-xs font-bold transition-all shadow-sm ${
                mainTab === "cash"
                  ? "bg-emerald-600 text-white shadow-emerald-200"
                  : "bg-white text-slate-600 hover:bg-slate-100 border border-slate-200"
              }`}
            >
              <Wallet className="w-4 h-4" />
              {tNav("cashLedger")}
            </button>
            <button
              onClick={() => setMainTab("suppliers")}
              className={`flex items-center gap-2 px-4 py-2 rounded-xl text-xs font-bold transition-all shadow-sm ${
                mainTab === "suppliers"
                  ? "bg-indigo-600 text-white shadow-indigo-200"
                  : "bg-white text-slate-600 hover:bg-slate-100 border border-slate-200"
              }`}
            >
              <Truck className="w-4 h-4" />
              {tNav("suppliers")}
            </button>
          </div>

          <div className="text-xs text-slate-500 font-semibold hidden sm:block">
            {mainTab === "cash" ? "💰 Nakit Kasa Çetelesi" : "🚚 Toptancı & Parçacı Cari Çetelesi"}
          </div>
        </div>

        {/* Period Selector & Action Controls Header */}
        <div className="flex flex-col md:flex-row md:items-center justify-between gap-4 mb-6 bg-white p-6 rounded-xl border border-slate-200 shadow-sm">
          <div className="flex items-center gap-4">
            <div>
              <div className="flex items-center gap-3 mb-1">
                <h2 className="text-xl font-bold text-slate-900">
                  {mainTab === "cash" ? tPeriod("title") : tSuppliers("title")}
                </h2>
                <PeriodBadge status={periodStatus} label={periodLabel} />
              </div>
              <p className="text-sm text-slate-500">
                {tPeriod("currentPeriod")}: <strong className="text-slate-800">{periodLabel}</strong>
              </p>
            </div>

            {/* Period Selector Component */}
            <div className="ml-4 pl-4 border-l border-slate-200">
              <PeriodSelector
                periods={periods}
                selectedPeriodId={selectedPeriodId}
                onSelectPeriod={(p) => setSelectedPeriodId(p.id)}
              />
            </div>
          </div>

          <div className="flex flex-wrap items-center gap-3">
            {mainTab === "cash" ? (
              <>
                {/* View Mode Switcher */}
                <div className="flex items-center p-1 bg-slate-100 rounded-lg border border-slate-200">
                  <Button
                    size="sm"
                    variant={activeTab === "ledger" ? "default" : "ghost"}
                    onClick={() => setActiveTab("ledger")}
                    className="h-8 text-xs font-semibold gap-1.5"
                  >
                    <FileSpreadsheet className="w-3.5 h-3.5" />
                    {tPeriod("ledgerView")}
                  </Button>
                  <Button
                    size="sm"
                    variant={activeTab === "history" ? "default" : "ghost"}
                    onClick={() => setActiveTab("history")}
                    className="h-8 text-xs font-semibold gap-1.5"
                  >
                    <Archive className="w-3.5 h-3.5" />
                    {tHistory("title")}
                  </Button>
                </div>

                {/* Export CSV / Excel Button */}
                <ExportCsvButton periodId={selectedPeriod?.id || ""} periodLabel={periodLabel} />

                {/* Yeni Dönem Aç Butonu */}
                <Button
                  variant="outline"
                  onClick={() => setPeriodModalMode("open")}
                  className="gap-2 border-slate-300 h-9 text-xs font-semibold"
                >
                  <Calendar className="w-4 h-4 text-primary" />
                  {tPeriod("openNextPeriod")}
                </Button>

                {periodStatus === "open" ? (
                  <>
                    <Button
                      variant="outline"
                      onClick={() => setImportModalOpen(true)}
                      className="gap-2 border-slate-300 h-9 text-xs font-semibold text-slate-700 hover:bg-slate-50"
                    >
                      <Upload className="w-4 h-4 text-emerald-600" />
                      {tImport("importCsv")}
                    </Button>

                    <Button
                      variant="destructive"
                      onClick={() => setPeriodModalMode("lock")}
                      className="gap-2 h-9 text-xs font-semibold"
                    >
                      <Lock className="w-4 h-4" />
                      {tPeriod("lockPeriod")}
                    </Button>

                    <Button
                      onClick={() => setCreateModalOpen(true)}
                      className="gap-2 h-9 text-xs bg-emerald-600 hover:bg-emerald-700 text-white font-semibold shadow-sm"
                    >
                      <PlusCircle className="w-4 h-4" />
                      {tTx("newTransaction")}
                    </Button>
                  </>
                ) : (
                  <Button
                    variant="outline"
                    onClick={() => handleUnlockPeriod(crypto.randomUUID())}
                    className="gap-2 h-9 text-xs font-semibold border-amber-300 text-amber-800 bg-amber-50 hover:bg-amber-100"
                  >
                    <Unlock className="w-4 h-4 text-amber-600" />
                    {tPeriod("unlockPeriod")}
                  </Button>
                )}
              </>
            ) : (
              /* Supplier Tab Actions */
              <>
                <Button
                  variant="outline"
                  onClick={() => setSupplierExcelModalOpen(true)}
                  disabled={periodStatus === "locked"}
                  className="gap-2 border-indigo-300 bg-indigo-50/50 hover:bg-indigo-100 text-indigo-800 h-9 text-xs font-semibold"
                >
                  <FileSpreadsheet className="w-4 h-4 text-indigo-600" />
                  {tSuppliers("importExcel")}
                </Button>

                <Button
                  onClick={() => setCreateSupplierTxModalOpen(true)}
                  disabled={periodStatus === "locked"}
                  className="gap-2 h-9 text-xs bg-indigo-600 hover:bg-indigo-700 text-white font-semibold shadow-sm"
                >
                  <PlusCircle className="w-4 h-4" />
                  {tSuppliers("newTransaction")}
                </Button>
              </>
            )}
          </div>
        </div>

        {/* Read-Only Archive Mode Warning Banner */}
        {periodStatus === "locked" && (
          <div className="flex items-center gap-3 p-4 mb-6 bg-amber-50 border border-amber-200 text-amber-900 rounded-xl text-xs font-medium shadow-sm">
            <AlertTriangle className="w-4 h-4 text-amber-600 shrink-0" />
            <span>{tPeriod("archiveBanner")}</span>
          </div>
        )}

        {/* MAIN TAB CONTENT */}
        {mainTab === "cash" ? (
          <>
            {/* Modular Live KPI Cards */}
            <KpiSummaryCards summary={kpiSummaryData} loading={loadingSummary} />

            {activeTab === "ledger" ? (
              <div className="border border-slate-200 rounded-xl bg-white shadow-sm overflow-hidden mt-4">
                <div className="p-4 border-b border-slate-200 bg-slate-50/50 flex items-center justify-between">
                  <h3 className="font-bold text-slate-900">{tTx("title")}</h3>
                  <span className="text-xs text-slate-500 font-semibold">
                    {currentPeriodTxs.length} {tPeriod("recordsCount")}
                  </span>
                </div>

                <TransactionTable
                  transactions={currentPeriodTxs}
                  isPeriodLocked={periodStatus === "locked"}
                  onReverse={(tx) => {
                    setTargetTxForReverse(tx);
                    setReverseModalOpen(true);
                  }}
                />
              </div>
            ) : (
              <PeriodHistoryView
                history={historyItems}
                onSelectPeriod={(pId) => {
                  setSelectedPeriodId(pId);
                  setActiveTab("ledger");
                }}
              />
            )}
          </>
        ) : (
          /* SUPPLIER & VENDOR VIEW */
          <div>
            <SupplierSummaryCards
              suppliers={suppliers}
              summary={supplierSummary}
              selectedSupplierId={selectedSupplierId}
              onSelectSupplier={(id) => setSelectedSupplierId(id)}
              loading={loadingSuppliers}
            />

            <SupplierLedgerTable
              transactions={supplierTransactions}
              searchQuery={supplierSearch}
              onSearchChange={setSupplierSearch}
              directionFilter={supplierDirectionFilter}
              onDirectionFilterChange={setSupplierDirectionFilter}
              isPeriodLocked={periodStatus === "locked"}
              totalCount={supplierTotalCount}
              page={supplierPage}
              pageSize={supplierPageSize}
              onPageChange={setSupplierPage}
              onPageSizeChange={setSupplierPageSize}
              onReverse={(tx) => {
                setTargetSupplierTxForReverse(tx);
                setReverseSupplierModalOpen(true);
              }}
              loading={loadingSupplierTxs}
            />
          </div>
        )}
      </main>

      {/* Cash Ledger Interactive Modal Dialogs */}
      <CreateTransactionDialog
        open={createModalOpen}
        onOpenChange={setCreateModalOpen}
        isPeriodLocked={periodStatus === "locked"}
        onSubmitTransaction={handleCreateTransaction}
      />

      <ReverseTransactionDialog
        open={reverseModalOpen}
        onOpenChange={setReverseModalOpen}
        transaction={targetTxForReverse}
        onSubmitReversal={handleReverseTransaction}
      />

      <ImportCsvDialog
        open={importModalOpen}
        onOpenChange={setImportModalOpen}
        periodId={selectedPeriod?.id || ""}
        isPeriodLocked={periodStatus === "locked"}
        onImportSuccess={fetchLiveData}
      />

      <PeriodActionDialog
        mode={periodModalMode}
        open={periodModalMode !== null}
        onOpenChange={(open) => {
          if (!open) setPeriodModalMode(null);
        }}
        userRole={userRole}
        onLockPeriod={handleLockPeriod}
        onOpenNextPeriod={handleOpenNextPeriod}
        existingLabels={periods.map((p) => p.label)}
      />

      {/* Supplier Modal Dialogs */}
      <ImportSupplierExcelDialog
        open={supplierExcelModalOpen}
        onOpenChange={setSupplierExcelModalOpen}
        periodId={selectedPeriod?.id || ""}
        isPeriodLocked={periodStatus === "locked"}
        onImportSuccess={refreshSuppliers}
      />

      <CreateSupplierTransactionDialog
        open={createSupplierTxModalOpen}
        onOpenChange={setCreateSupplierTxModalOpen}
        periodId={selectedPeriod?.id || ""}
        suppliers={suppliers}
        isPeriodLocked={periodStatus === "locked"}
        onSuccess={refreshSuppliers}
      />

      <ReverseSupplierTransactionDialog
        open={reverseSupplierModalOpen}
        onOpenChange={setReverseSupplierModalOpen}
        transaction={targetSupplierTxForReverse}
        onSubmitReversal={handleReverseSupplierTransaction}
      />
    </div>
  );
}
