"use client";

import { useState, useMemo, useEffect, useCallback } from "react";
import { apiFetch } from "@/lib/api";
import { addDecimal, subDecimal } from "@/lib/decimal";
import { TransactionItem } from "@/components/ledger/TransactionTable";
import { PeriodOption } from "./usePeriods";

export interface PeriodSummaryData {
  period_id: string;
  starting_balance: string;
  total_in: string;
  total_out: string;
  closing_balance: string;
}

function isValidUuid(str: string): boolean {
  return /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i.test(str);
}

export function useTransactions(
  selectedPeriod: PeriodOption | undefined,
  currentUserLabel: string
) {
  const [transactions, setTransactions] = useState<TransactionItem[]>([]);
  const [liveSummary, setLiveSummary] = useState<PeriodSummaryData | null>(null);
  const [loadingSummary, setLoadingSummary] = useState<boolean>(false);

  // Modal dialog states
  const [createModalOpen, setCreateModalOpen] = useState<boolean>(false);
  const [reverseModalOpen, setReverseModalOpen] = useState<boolean>(false);
  const [importModalOpen, setImportModalOpen] = useState<boolean>(false);
  const [targetTxForReverse, setTargetTxForReverse] = useState<TransactionItem | null>(null);

  const selectedPeriodId = selectedPeriod?.id;
  const startingBalance = selectedPeriod?.startingBalance || "0.00";

  // Fetch Live Summary & Transactions from Backend API
  const fetchLiveData = useCallback(async () => {
    if (!selectedPeriodId || !isValidUuid(selectedPeriodId)) return;

    setLoadingSummary(true);
    try {
      const res = await apiFetch<PeriodSummaryData>(`/periods/${selectedPeriodId}/summary`);
      if (res.success && res.data) {
        setLiveSummary(res.data);
      }

      const txRes = await apiFetch<any[]>(`/periods/${selectedPeriodId}/transactions`);
      if (txRes.success && Array.isArray(txRes.data)) {
        const idMap = new Map<string, any>();
        txRes.data.forEach((tx: any) => idMap.set(tx.id, tx));

        const isOriginalReversed = new Set<string>();
        txRes.data.forEach((tx: any) => {
          if (tx.reversed_by) {
            isOriginalReversed.add(tx.reversed_by);
          }
        });

        const mapped: TransactionItem[] = txRes.data.map((tx: any) => {
          const isReversal = Boolean(
            tx.reversed_by ||
            tx.description?.startsWith("[İPTAL/TERS KAYIT]")
          );
          const wasReversed = isOriginalReversed.has(tx.id);

          let creatorDisplay = tx.created_by_name;
          if (!creatorDisplay) {
            const raw = String(tx.created_by || "");
            if (/^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i.test(raw)) {
              creatorDisplay = "Admin";
            } else {
              creatorDisplay = raw || "Admin";
            }
          }

          return {
            id: tx.id,
            periodId: tx.period_id,
            direction: tx.direction,
            channel: tx.channel,
            amount: tx.amount,
            description: tx.description,
            createdBy: creatorDisplay,
            createdAt: tx.created_at ? tx.created_at.slice(0, 16).replace("T", " ") : "",
            reversedBy: wasReversed ? "reversed" : null,
            isReversalEntry: isReversal,
          };
        });

        // Ensure newest transactions are always at the top
        mapped.sort((a, b) => (b.createdAt > a.createdAt ? 1 : b.createdAt < a.createdAt ? -1 : 0));

        setTransactions(mapped);
      }
    } catch (err) {
      console.error("Failed to fetch live transactions:", err);
    } finally {
      setLoadingSummary(false);
    }
  }, [selectedPeriodId]);

  useEffect(() => {
    fetchLiveData();
  }, [fetchLiveData]);

  // Dynamic KPI calculation fallback using decimal.js
  const localCalculatedSummary = useMemo(() => {
    if (!selectedPeriod) {
      return {
        period_id: "",
        starting_balance: "0.00",
        total_in: "0.00",
        total_out: "0.00",
        closing_balance: "0.00",
      };
    }

    let sumIn = "0";
    let sumOut = "0";

    const filtered = transactions.filter(
      (tx) => tx.periodId === selectedPeriod.id && !tx.reversedBy && !tx.isReversalEntry
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
  }, [transactions, selectedPeriod, startingBalance]);

  const kpiSummaryData: PeriodSummaryData = liveSummary || localCalculatedSummary;

  // Handler: Create Transaction
  const handleCreateTransaction = async (data: {
    direction: "in" | "out";
    channel: string;
    amount: string;
    description: string;
    idempotencyKey: string;
  }) => {
    if (!selectedPeriod || !isValidUuid(selectedPeriod.id)) {
      alert("Lütfen önce geçerli bir mali dönem seçin.");
      return;
    }

    if (selectedPeriod.status === "locked") {
      alert("Kilitli döneme yeni işlem eklenemez.");
      return;
    }

    const currentPeriodId = selectedPeriod.id;
    const tempUuid = crypto.randomUUID();

    const newTx: TransactionItem = {
      id: tempUuid,
      periodId: currentPeriodId,
      direction: data.direction,
      channel: data.channel,
      amount: data.amount,
      description: data.description,
      createdAt: new Date().toISOString().slice(0, 16).replace("T", " "),
      createdBy: currentUserLabel,
      reversedBy: null,
    };

    setTransactions((prev) => [newTx, ...prev]);

    try {
      const res = await apiFetch<any>(`/transactions`, {
        method: "POST",
        headers: { "Idempotency-Key": data.idempotencyKey },
        body: JSON.stringify({
          period_id: currentPeriodId,
          direction: data.direction,
          channel: data.channel,
          amount: data.amount,
          description: data.description,
        }),
      });

      if (res.success && res.data?.id) {
        const realId = res.data.id;
        setTransactions((prev) =>
          prev.map((t) => (t.id === tempUuid ? { ...t, id: realId } : t))
        );
        fetchLiveData();
      } else {
        // Rollback optimistic state if backend rejected
        setTransactions((prev) => prev.filter((t) => t.id !== tempUuid));
        alert(res.error?.message || "İşlem kaydedilemedi.");
      }
    } catch (err) {
      console.error("Failed to post transaction to backend:", err);
      setTransactions((prev) => prev.filter((t) => t.id !== tempUuid));
      alert("Sunucuya bağlanılamadı. İşlem kaydedilemedi.");
    }
  };

  // Handler: Reverse Transaction
  const handleReverseTransaction = async (targetTxId: string, reason: string, idempotencyKey: string) => {
    const origTx = transactions.find((t) => t.id === targetTxId);
    if (!origTx) return;

    const reversalTxId = crypto.randomUUID();

    const reversalTx: TransactionItem = {
      id: reversalTxId,
      periodId: origTx.periodId,
      direction: origTx.direction === "in" ? "out" : "in",
      channel: origTx.channel,
      amount: origTx.amount,
      description: `[İPTAL/TERS KAYIT] ${reason}`,
      createdAt: new Date().toISOString().slice(0, 16).replace("T", " "),
      createdBy: currentUserLabel,
      reversedBy: null,
      isReversalEntry: true,
    };

    setTransactions((prev) =>
      prev.map((t) => (t.id === targetTxId ? { ...t, reversedBy: reversalTxId } : t)).concat(reversalTx)
    );

    try {
      const res = await apiFetch<any>(`/transactions/${targetTxId}/reverse`, {
        method: "POST",
        headers: { "Idempotency-Key": idempotencyKey },
        body: JSON.stringify({ reason }),
      });
      if (res.success && res.data?.id) {
        fetchLiveData();
      } else {
        // Rollback optimistic reversal state if backend rejected
        setTransactions((prev) =>
          prev
            .filter((t) => t.id !== reversalTxId)
            .map((t) => (t.id === targetTxId ? { ...t, reversedBy: null } : t))
        );
        alert(res.error?.message || "İptal/Ters kayıt işlemi gerçekleştirilemedi.");
      }
    } catch (err) {
      setTransactions((prev) =>
        prev
          .filter((t) => t.id !== reversalTxId)
          .map((t) => (t.id === targetTxId ? { ...t, reversedBy: null } : t))
      );
      alert("Sunucuya bağlanılamadı. İptal işlemi tamamlanamadı.");
    }
  };

  return {
    transactions,
    setTransactions,
    liveSummary,
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
  };
}
