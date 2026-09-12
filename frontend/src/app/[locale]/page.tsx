"use client";

import { use, useState, useEffect } from "react";
import { useTranslations } from "next-intl";
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
import { Button } from "@/components/ui/button";
import { createClient } from "@/lib/supabase/client";
import { usePeriods } from "@/hooks/usePeriods";
import { useTransactions } from "@/hooks/useTransactions";
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
} from "lucide-react";

export default function HomePage({ params }: { params: Promise<{ locale: string }> }) {
  const { locale } = use(params);
  const tCommon = useTranslations("common");
  const tPeriod = useTranslations("period");
  const tTx = useTranslations("transaction");
  const tHistory = useTranslations("history");
  const tImport = useTranslations("import_export");

  // Mounted state guard to eliminate hydration mismatch (#418, #423)
  const [isMounted, setIsMounted] = useState<boolean>(false);
  const [userRole] = useState<"admin" | "muhasebeci" | "standart">("admin");
  const [currentUserLabel, setCurrentUserLabel] = useState<string>("Öncü Otogaz Yönetici");
  const [activeTab, setActiveTab] = useState<"ledger" | "history">("ledger");

  // Modular custom hooks
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
        {/* Period Selector & Action Controls Header */}
        <div className="flex flex-col md:flex-row md:items-center justify-between gap-4 mb-6 bg-white p-6 rounded-xl border border-slate-200 shadow-sm">
          <div className="flex items-center gap-4">
            <div>
              <div className="flex items-center gap-3 mb-1">
                <h2 className="text-xl font-bold text-slate-900">{tPeriod("title")}</h2>
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
            <ExportCsvButton periodId={selectedPeriod.id} periodLabel={periodLabel} />

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
          </div>
        </div>

        {/* Read-Only Archive Mode Warning Banner */}
        {periodStatus === "locked" && (
          <div className="flex items-center gap-3 p-4 mb-6 bg-amber-50 border border-amber-200 text-amber-900 rounded-xl text-xs font-medium shadow-sm">
            <AlertTriangle className="w-4 h-4 text-amber-600 shrink-0" />
            <span>{tPeriod("archiveBanner")}</span>
          </div>
        )}

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
      </main>

      {/* Interactive Modal Dialogs */}
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
        periodId={selectedPeriod.id}
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
    </div>
  );
}
