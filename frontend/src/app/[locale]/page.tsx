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
import { MemberManagementDialog, MemberItem } from "@/components/admin/MemberManagementDialog";
import { QuickEntryRow } from "@/components/ledger/QuickEntryRow";
import { KpiSummaryCards } from "@/components/ledger/KpiSummaryCards";
import { PeriodHistoryView, PeriodHistoryItem } from "@/components/ledger/PeriodHistoryView";
import { Button } from "@/components/ui/button";
import { addDecimal, subDecimal } from "@/lib/decimal";
import {
  PlusCircle,
  Lock,
  Calendar,
  AlertTriangle,
  FileSpreadsheet,
  Archive,
} from "lucide-react";

export default function HomePage({ params }: { params: Promise<{ locale: string }> }) {
  const { locale } = use(params);
  const tPeriod = useTranslations("period");
  const tTx = useTranslations("transaction");
  const tHistory = useTranslations("history");

  // Mounted state guard to eliminate hydration mismatch (#418, #423)
  const [isMounted, setIsMounted] = useState<boolean>(false);

  // User Role State (Mocked as admin for interactive demonstration)
  const [userRole, setUserRole] = useState<"admin" | "muhasebeci" | "standart">("admin");

  // View Mode: 'ledger' (Active Table) vs 'history' (Archived Periods)
  const [activeTab, setActiveTab] = useState<"ledger" | "history">("ledger");

  // Periods State
  const [periods, setPeriods] = useState<PeriodOption[]>([
    { id: "p-2026-08", label: "2026-08", status: "open", startingBalance: "15000.50" },
    { id: "p-2026-07", label: "2026-07", status: "locked", startingBalance: "10000.00" },
    { id: "p-2026-06", label: "2026-06", status: "locked", startingBalance: "8500.00" },
  ]);
  const [selectedPeriodId, setSelectedPeriodId] = useState<string>("p-2026-08");

  const selectedPeriod = useMemo(
    () => periods.find((p) => p.id === selectedPeriodId) || periods[0],
    [periods, selectedPeriodId]
  );

  const periodStatus = selectedPeriod.status;
  const periodLabel = selectedPeriod.label;
  const startingBalance = selectedPeriod.startingBalance;

  // Tenant Members State
  const [members, setMembers] = useState<MemberItem[]>([
    { userId: "usr-1", email: "admin@deftersystem.com", role: "admin" },
    { userId: "usr-2", email: "muhasebe@deftersystem.com", role: "muhasebeci" },
    { userId: "usr-3", email: "satis@deftersystem.com", role: "standart" },
  ]);

  // Transactions State
  const [transactions, setTransactions] = useState<TransactionItem[]>([
    {
      id: "tx-101",
      periodId: "p-2026-08",
      direction: "in",
      channel: "eft",
      amount: "5450.75",
      description: "Müşteri Ahmet Yılmaz Ödeme",
      createdAt: "2026-08-20 10:15",
      createdBy: "Halil İbrahim",
      reversedBy: null,
    },
    {
      id: "tx-102",
      periodId: "p-2026-08",
      direction: "out",
      channel: "kira",
      amount: "3200.25",
      description: "Ağustos Ayı Ofis Kirası",
      createdAt: "2026-08-20 11:30",
      createdBy: "Ahmet Muhasebe",
      reversedBy: null,
    },
    {
      id: "tx-103",
      periodId: "p-2026-08",
      direction: "in",
      channel: "pos",
      amount: "3000.00",
      description: "Gün Sonu POS Çekimi",
      createdAt: "2026-08-20 17:00",
      createdBy: "Halil İbrahim",
      reversedBy: null,
    },
  ]);

  // Modal dialog states
  const [createModalOpen, setCreateModalOpen] = useState<boolean>(false);
  const [reverseModalOpen, setReverseModalOpen] = useState<boolean>(false);
  const [targetTxForReverse, setTargetTxForReverse] = useState<TransactionItem | null>(null);
  const [periodModalMode, setPeriodModalMode] = useState<"lock" | "open" | null>(null);

  useEffect(() => {
    setIsMounted(true);
  }, []);

  // Dynamic KPI calculation using decimal.js (strictly excluding reversed entries)
  const { totalIn, totalOut, closingBalance } = useMemo(() => {
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
      totalIn: sumIn,
      totalOut: sumOut,
      closingBalance: net,
    };
  }, [transactions, selectedPeriod.id, startingBalance]);

  // Summary object for KpiSummaryCards
  const kpiSummaryData = useMemo(() => {
    return {
      period_id: selectedPeriod.id,
      starting_balance: startingBalance,
      total_in: totalIn,
      total_out: totalOut,
      closing_balance: closingBalance,
    };
  }, [selectedPeriod.id, startingBalance, totalIn, totalOut, closingBalance]);

  // Period History Items
  const historyItems: PeriodHistoryItem[] = useMemo(() => {
    return periods.map((p) => ({
      period_id: p.id,
      label: p.label,
      status: p.status,
      starting_balance: p.startingBalance,
      total_in: "5450.75",
      total_out: "3200.25",
      closing_balance: "17250.50",
      opened_at: "2026-08-01",
      locked_at: p.status === "locked" ? "2026-08-01 23:59" : null,
    }));
  }, [periods]);

  // Handlers for Member Management
  const handleAddMember = async (email: string, role: "admin" | "muhasebeci" | "standart") => {
    const newM: MemberItem = {
      userId: `usr-${Date.now()}`,
      email,
      role,
    };
    setMembers((prev) => [...prev, newM]);
  };

  const handleUpdateMemberRole = async (userId: string, newRole: "admin" | "muhasebeci" | "standart") => {
    if (newRole !== "admin") {
      const adminCount = members.filter((m) => m.role === "admin").length;
      const targetM = members.find((m) => m.userId === userId);
      if (adminCount <= 1 && targetM?.role === "admin") {
        throw new Error("CANNOT_REMOVE_LAST_ADMIN");
      }
    }
    setMembers((prev) => prev.map((m) => (m.userId === userId ? { ...m, role: newRole } : m)));
  };

  const handleRemoveMember = async (userId: string) => {
    const targetM = members.find((m) => m.userId === userId);
    if (targetM?.role === "admin") {
      const adminCount = members.filter((m) => m.role === "admin").length;
      if (adminCount <= 1) {
        throw new Error("CANNOT_REMOVE_LAST_ADMIN");
      }
    }
    setMembers((prev) => prev.filter((m) => m.userId !== userId));
  };

  // Handler: Create Transaction
  const handleCreateTransaction = async (data: {
    direction: "in" | "out";
    channel: string;
    amount: string;
    description: string;
    idempotencyKey: string;
  }) => {
    const newTx: TransactionItem = {
      id: `tx-${Date.now()}`,
      periodId: selectedPeriod.id,
      direction: data.direction,
      channel: data.channel,
      amount: data.amount,
      description: data.description,
      createdAt: new Date().toISOString().slice(0, 16).replace("T", " "),
      createdBy: "Halil İbrahim",
      reversedBy: null,
    };

    setTransactions((prev) => [newTx, ...prev]);
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
  };

  // Handler: Lock Period
  const handleLockPeriod = async (idempotencyKey: string) => {
    setPeriods((prev) =>
      prev.map((p) => (p.id === selectedPeriod.id ? { ...p, status: "locked" } : p))
    );
  };

  // Handler: Open Next Period
  const handleOpenNextPeriod = async (label: string, idempotencyKey: string) => {
    const newPeriod: PeriodOption = {
      id: `p-${label}`,
      label,
      status: "open",
      startingBalance: closingBalance,
    };
    setPeriods((prev) => [newPeriod, ...prev]);
    setSelectedPeriodId(newPeriod.id);
  };

  return (
    <div className="min-h-screen bg-slate-50">
      <Header tenantName="Deftersystem Ana Şube" userRole={userRole} locale={locale} />

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

            {/* Admin Member Management Modal */}
            <MemberManagementDialog
              userRole={userRole}
              members={members}
              onAddMember={handleAddMember}
              onUpdateRole={handleUpdateMemberRole}
              onRemoveMember={handleRemoveMember}
            />

            {periodStatus === "open" && (
              <>
                <Button
                  variant="outline"
                  onClick={() => setPeriodModalMode("open")}
                  className="gap-2 border-slate-300 h-9 text-xs"
                >
                  <Calendar className="w-4 h-4 text-primary" />
                  {tPeriod("openNextPeriod")}
                </Button>

                <Button
                  variant="destructive"
                  onClick={() => setPeriodModalMode("lock")}
                  className="gap-2 h-9 text-xs"
                >
                  <Lock className="w-4 h-4" />
                  {tPeriod("lockPeriod")}
                </Button>

                <Button
                  onClick={() => setCreateModalOpen(true)}
                  className="gap-2 h-9 text-xs bg-emerald-600 hover:bg-emerald-700 text-white shadow-sm"
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
        <KpiSummaryCards summary={kpiSummaryData} />

        {activeTab === "ledger" ? (
          <>
            {/* Quick Entry Excel-like Keyboard Bar */}
            <QuickEntryRow
              isPeriodLocked={periodStatus === "locked"}
              onSubmitTransaction={handleCreateTransaction}
            />

            {/* Ledger Transactions Table (TanStack Table) */}
            <div className="border rounded-xl bg-white shadow-sm overflow-hidden mt-6">
              <div className="p-4 border-b bg-slate-50/50 flex items-center justify-between">
                <h3 className="font-bold text-slate-900">{tTx("title")}</h3>
                <span className="text-xs text-slate-500 font-medium">
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
