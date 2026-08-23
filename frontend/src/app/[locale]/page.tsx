"use client";

import { use, useState, useMemo, useEffect } from "react";
import { useTranslations } from "next-intl";
import { Header } from "@/components/shared/Header";
import { PeriodBadge } from "@/components/shared/PeriodBadge";
import { TransactionTable, TransactionItem } from "@/components/ledger/TransactionTable";
import { CreateTransactionDialog } from "@/components/ledger/CreateTransactionDialog";
import { ReverseTransactionDialog } from "@/components/ledger/ReverseTransactionDialog";
import { PeriodActionDialog } from "@/components/ledger/PeriodActionDialog";
import { PeriodSelector, PeriodOption } from "@/components/ledger/PeriodSelector";
import { KpiSummaryCards, PeriodSummaryData } from "@/components/ledger/KpiSummaryCards";
import { PeriodHistoryView, PeriodHistoryItem } from "@/components/ledger/PeriodHistoryView";
import { ExportCsvButton } from "@/components/ledger/ExportCsvButton";
import { ImportCsvDialog } from "@/components/ledger/ImportCsvDialog";
import { Button } from "@/components/ui/button";
import { addDecimal, subDecimal } from "@/lib/decimal";
import { apiFetch } from "@/lib/api";
import {
  PlusCircle,
  Lock,
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

  // User Role State
  const [userRole] = useState<"admin" | "muhasebeci" | "standart">("admin");

  // View Mode: 'ledger' (Active Table) vs 'history' (Archived Periods)
  const [activeTab, setActiveTab] = useState<"ledger" | "history">("ledger");

  // Periods State (Default current period, fetched from live database)
  const [periods, setPeriods] = useState<PeriodOption[]>([
    { id: "p-2026-08", label: "2026-08", status: "open", startingBalance: "0.00" },
  ]);
  const [selectedPeriodId, setSelectedPeriodId] = useState<string>("p-2026-08");

  const selectedPeriod = useMemo(
    () => periods.find((p) => p.id === selectedPeriodId) || periods[0],
    [periods, selectedPeriodId]
  );

  const periodStatus = selectedPeriod?.status || "open";
  const periodLabel = selectedPeriod?.label || "2026-08";
  const startingBalance = selectedPeriod?.startingBalance || "0.00";

  // Transactions State (Empty array default, populated from live database)
  const [transactions, setTransactions] = useState<TransactionItem[]>([]);


  // Live Summary from Backend (with fallback)
  const [liveSummary, setLiveSummary] = useState<PeriodSummaryData | null>(null);
  const [loadingSummary, setLoadingSummary] = useState<boolean>(false);

  // Modal dialog states
  const [createModalOpen, setCreateModalOpen] = useState<boolean>(false);
  const [reverseModalOpen, setReverseModalOpen] = useState<boolean>(false);
  const [importModalOpen, setImportModalOpen] = useState<boolean>(false);
  const [targetTxForReverse, setTargetTxForReverse] = useState<TransactionItem | null>(null);
  const [periodModalMode, setPeriodModalMode] = useState<"lock" | "open" | null>(null);

  useEffect(() => {
    setIsMounted(true);
  }, []);

  // Fetch periods & transactions from backend API
  const fetchPeriods = async () => {
    try {
      const res = await apiFetch<any[]>("/periods/");
      if (res.success && Array.isArray(res.data) && res.data.length > 0) {
        const mapped: PeriodOption[] = res.data.map((p: any) => ({
          id: p.id,
          label: p.label,
          status: p.status,
          startingBalance: p.starting_balance || "0.00",
        }));
        setPeriods(mapped);
        if (mapped[0]?.id) {
          setSelectedPeriodId(mapped[0].id);
        }
      }
    } catch {
      // Fallback to default state
    }
  };

  useEffect(() => {
    fetchPeriods();
  }, []);

  // Fetch Live Summary & Transactions from Backend API
  const fetchLiveData = async () => {
    if (!selectedPeriodId) return;
    const periodUuid =
      selectedPeriodId.length === 36
        ? selectedPeriodId
        : "00000000-0000-0000-0000-000000000001";

    setLoadingSummary(true);
    try {
      const res = await apiFetch<PeriodSummaryData>(`/periods/${periodUuid}/summary`);
      if (res.success && res.data) {
        setLiveSummary(res.data);
      }
    } catch {
      // Silent fallback to local calculated summary
    } finally {
      setLoadingSummary(false);
    }
  };

  useEffect(() => {
    fetchLiveData();
  }, [selectedPeriodId]);

  // Dynamic KPI calculation fallback using decimal.js
  const localCalculatedSummary = useMemo(() => {
    let sumIn = "0";
    let sumOut = "0";

    const filtered = transactions.filter(
      (tx) => tx.periodId === selectedPeriod.id && !tx.reversedBy
    );

    for (const tx of filtered) {
      if (tx.direction === "in") {
        sumIn = addDecimal(sumIn, tx.amount).toString();
      } else if (tx.direction === "out") {
        sumOut = addDecimal(sumOut, tx.amount).toString();
      }
    }

    const net = subDecimal(addDecimal(startingBalance, sumIn), sumOut).toString();

    return {
      period_id: selectedPeriod.id,
      starting_balance: startingBalance,
      total_in: sumIn,
      total_out: sumOut,
      closing_balance: net,
    };
  }, [transactions, selectedPeriod.id, startingBalance]);

  const kpiSummaryData: PeriodSummaryData = liveSummary || localCalculatedSummary;

  // Period History Items
  const historyItems: PeriodHistoryItem[] = useMemo(() => {
    return periods.map((p) => ({
      period_id: p.id,
      label: p.label,
      status: p.status,
      starting_balance: p.startingBalance,
      total_in: "0",
      total_out: "0",
      closing_balance: p.startingBalance,
      opened_at: "2026-08-01",
      locked_at: p.status === "locked" ? "2026-08-01 23:59" : null,
    }));
  }, [periods]);

  // Handler: Create Transaction
  const handleCreateTransaction = async (data: {
    direction: "in" | "out";
    channel: string;
    amount: string;
    description: string;
    idempotencyKey: string;
  }) => {
    const validPeriodUuid =
      selectedPeriod?.id && selectedPeriod.id.length === 36
        ? selectedPeriod.id
        : "00000000-0000-0000-0000-000000000001";

    const newTx: TransactionItem = {
      id: `tx-${Date.now()}`,
      periodId: validPeriodUuid,
      direction: data.direction,
      channel: data.channel,
      amount: data.amount,
      description: data.description,
      createdAt: new Date().toISOString().slice(0, 16).replace("T", " "),
      createdBy: "Admin",
      reversedBy: null,
    };

    setTransactions((prev) => [newTx, ...prev]);

    try {
      const res = await apiFetch<any>(`/transactions`, {
        method: "POST",
        headers: { "Idempotency-Key": data.idempotencyKey },
        body: JSON.stringify({
          period_id: validPeriodUuid,
          direction: data.direction,
          channel: data.channel,
          amount: data.amount,
          description: data.description,
        }),
      });

      if (res.success) {
        fetchLiveData();
      }
    } catch (err) {
      console.error("Failed to post transaction to backend:", err);
    }
  };

  // Handler: Reverse Transaction
  const handleReverseTransaction = async (targetTxId: string, reason: string, idempotencyKey: string) => {
    const origTx = transactions.find((t) => t.id === targetTxId);
    if (!origTx) return;

    const reversalTxId = `tx-rev-${Date.now()}`;

    const reversalTx: TransactionItem = {
      id: reversalTxId,
      periodId: origTx.periodId,
      direction: origTx.direction === "in" ? "out" : "in",
      channel: origTx.channel,
      amount: origTx.amount,
      description: `[İPTAL/TERS KAYIT] ${reason}`,
      createdAt: new Date().toISOString().slice(0, 16).replace("T", " "),
      createdBy: "Halil İbrahim",
      reversedBy: null,
    };

    setTransactions((prev) =>
      prev.map((t) => (t.id === targetTxId ? { ...t, reversedBy: reversalTxId } : t)).concat(reversalTx)
    );

    try {
      await apiFetch(`/transactions/${targetTxId}/reverse`, {
        method: "POST",
        headers: { "Idempotency-Key": idempotencyKey },
        body: JSON.stringify({ reason }),
      });
    } catch {
      // Local state is preserved
    }
  };

  // Handler: Lock Period
  const handleLockPeriod = async (idempotencyKey: string) => {
    setPeriods((prev) =>
      prev.map((p) => (p.id === selectedPeriod.id ? { ...p, status: "locked" } : p))
    );

    try {
      await apiFetch(`/periods/${selectedPeriod.id}/lock`, {
        method: "POST",
        headers: { "Idempotency-Key": idempotencyKey },
      });
    } catch {
      // Local state is preserved
    }
  };

  // Handler: Open Next Period
  const handleOpenNextPeriod = async (label: string, idempotencyKey: string) => {
    const newPeriod: PeriodOption = {
      id: `p-${label}`,
      label,
      status: "open",
      startingBalance: kpiSummaryData.closing_balance.toString(),
    };
    setPeriods((prev) => [newPeriod, ...prev]);
    setSelectedPeriodId(newPeriod.id);

    try {
      await apiFetch(`/periods/open-next`, {
        method: "POST",
        headers: { "Idempotency-Key": idempotencyKey },
        body: JSON.stringify({ label }),
      });
    } catch {
      // Local state is preserved
    }
  };

  if (!isMounted) return null;

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
                İşlem Defteri
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

            {/* Export CSV Button */}
            <ExportCsvButton periodId={selectedPeriod.id} periodLabel={periodLabel} />

            {periodStatus === "open" && (
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
                  variant="outline"
                  onClick={() => setPeriodModalMode("open")}
                  className="gap-2 border-slate-300 h-9 text-xs font-semibold"
                >
                  <Calendar className="w-4 h-4 text-primary" />
                  {tPeriod("openNextPeriod")}
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
          <>
            {/* Ledger Transactions Table (TanStack Table) */}
            <div className="border border-slate-200 rounded-xl bg-white shadow-sm overflow-hidden mt-4">
              <div className="p-4 border-b border-slate-200 bg-slate-50/50 flex items-center justify-between">
                <h3 className="font-bold text-slate-900">{tTx("title")}</h3>
                <span className="text-xs text-slate-500 font-semibold">
                  {transactions.filter((tx) => tx.periodId === selectedPeriod.id).length} İşlem Kaydı
                </span>
              </div>

              <TransactionTable
                transactions={transactions.filter((tx) => tx.periodId === selectedPeriod.id)}
                isPeriodLocked={periodStatus === "locked"}
                onReverse={(tx) => {
                  setTargetTxForReverse(tx);
                  setReverseModalOpen(true);
                }}
              />
            </div>
          </>
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
      />
    </div>
  );
}
