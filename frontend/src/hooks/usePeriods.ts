"use client";

import { useState, useMemo, useEffect, useCallback } from "react";
import { apiFetch } from "@/lib/api";

export interface PeriodOption {
  id: string;
  label: string;
  status: "open" | "locked";
  startingBalance: string;
}

export interface PeriodHistoryItem {
  period_id: string;
  label: string;
  status: "open" | "locked";
  starting_balance: string;
  total_in: string;
  total_out: string;
  closing_balance: string;
  opened_at: string;
  locked_at?: string;
}

export function usePeriods(activeTab: "ledger" | "history") {
  const [periods, setPeriods] = useState<PeriodOption[]>([
    { id: "00000000-0000-0000-0000-000000000001", label: "2026-08", status: "open", startingBalance: "0.00" },
  ]);
  const [selectedPeriodId, setSelectedPeriodId] = useState<string>("00000000-0000-0000-0000-000000000001");
  const [historyItems, setHistoryItems] = useState<PeriodHistoryItem[]>([]);
  const [periodModalMode, setPeriodModalMode] = useState<"lock" | "open" | null>(null);

  const selectedPeriod = useMemo(
    () => periods.find((p) => p.id === selectedPeriodId) || periods[0],
    [periods, selectedPeriodId]
  );

  const periodStatus: "open" | "locked" = (selectedPeriod?.status as "open" | "locked") || "open";
  const periodLabel = selectedPeriod?.label || "2026-08";
  const startingBalance = selectedPeriod?.startingBalance || "0.00";

  // Fetch all periods
  const fetchPeriods = useCallback(async () => {
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
        setSelectedPeriodId((prev) => {
          if (prev && mapped.some((p) => p.id === prev)) {
            return prev;
          }
          const openPeriod = mapped.find((p) => p.status === "open");
          return openPeriod?.id || mapped[0]?.id || prev;
        });
      }
    } catch {
      // Fallback to default state
    }
  }, []);

  // Fetch Period History
  const fetchPeriodHistory = useCallback(async () => {
    try {
      const res = await apiFetch<any[]>("/periods/history");
      if (res.success && Array.isArray(res.data)) {
        const mappedHistory: PeriodHistoryItem[] = res.data.map((item: any) => ({
          period_id: item.period_id,
          label: item.label,
          status: item.status,
          starting_balance: item.starting_balance || "0.00",
          total_in: item.total_in || "0.00",
          total_out: item.total_out || "0.00",
          closing_balance: item.closing_balance || "0.00",
          opened_at: item.opened_at,
          locked_at: item.locked_at,
        }));
        setHistoryItems(mappedHistory);
      }
    } catch (err) {
      console.error("Failed to fetch period history:", err);
    }
  }, []);

  useEffect(() => {
    fetchPeriods();
    fetchPeriodHistory();
  }, [fetchPeriods, fetchPeriodHistory]);

  useEffect(() => {
    if (activeTab === "history") {
      fetchPeriodHistory();
    }
  }, [activeTab, fetchPeriodHistory]);

  // Handler: Lock Period
  const handleLockPeriod = async (idempotencyKey: string) => {
    if (!selectedPeriod) return;
    try {
      const res = await apiFetch<any>(`/periods/${selectedPeriod.id}/lock`, {
        method: "POST",
        headers: { "Idempotency-Key": idempotencyKey },
      });
      if (res.success) {
        setPeriods((prev) =>
          prev.map((p) => (p.id === selectedPeriod.id ? { ...p, status: "locked" } : p))
        );
        fetchPeriodHistory();
      } else {
        alert(res.error?.message || "Dönem kilitlenemedi.");
      }
    } catch {
      alert("Sunucuya bağlanılamadı. Dönem kilitlenemedi.");
    }
  };

  // Handler: Unlock Period
  const handleUnlockPeriod = async (idempotencyKey: string) => {
    if (!selectedPeriod) return;
    try {
      const res = await apiFetch<any>(`/periods/${selectedPeriod.id}/unlock`, {
        method: "POST",
        headers: { "Idempotency-Key": idempotencyKey },
      });
      if (res.success) {
        setPeriods((prev) =>
          prev.map((p) => (p.id === selectedPeriod.id ? { ...p, status: "open" } : p))
        );
        fetchPeriodHistory();
      } else {
        alert(res.error?.message || "Dönem kilidi açılamadı.");
      }
    } catch {
      alert("Sunucuya bağlanılamadı. Dönem kilidi açılamadı.");
    }
  };

  return {
    periods,
    selectedPeriodId,
    setSelectedPeriodId,
    selectedPeriod,
    periodStatus,
    periodLabel,
    startingBalance,
    historyItems,
    periodModalMode,
    setPeriodModalMode,
    fetchPeriods,
    fetchPeriodHistory,
    handleLockPeriod,
    handleUnlockPeriod,
  };
}
