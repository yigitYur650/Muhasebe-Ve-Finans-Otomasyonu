"use client";

import { useState, useCallback, useEffect } from "react";
import { Supplier, SupplierTransaction, SupplierSummary } from "@/types/supplier";
import { apiFetch } from "@/lib/api";

export function useSuppliers(selectedPeriodId?: string) {
  const [suppliers, setSuppliers] = useState<Supplier[]>([]);
  const [summary, setSummary] = useState<SupplierSummary | null>(null);
  const [selectedSupplierId, setSelectedSupplierId] = useState<string | "all">("all");
  const [transactions, setTransactions] = useState<SupplierTransaction[]>([]);
  const [loadingSuppliers, setLoadingSuppliers] = useState<boolean>(false);
  const [loadingTransactions, setLoadingTransactions] = useState<boolean>(false);
  const [searchQuery, setSearchQuery] = useState<string>("");
  const [directionFilter, setDirectionFilter] = useState<string>("");

  const fetchSuppliers = useCallback(async (periodId?: string) => {
    setLoadingSuppliers(true);
    try {
      const pId = periodId || selectedPeriodId;
      const endpoint = pId ? `/suppliers?period_id=${pId}` : "/suppliers";
      const res = await apiFetch<Supplier[]>(endpoint);
      if (res.success && res.data) {
        setSuppliers(res.data);
      }
    } catch (err) {
      console.error("Failed to fetch suppliers:", err);
    } finally {
      setLoadingSuppliers(false);
    }
  }, [selectedPeriodId]);

  const fetchSummary = useCallback(async (periodId?: string) => {
    try {
      const pId = periodId || selectedPeriodId;
      const endpoint = pId ? `/suppliers/summary?period_id=${pId}` : "/suppliers/summary";
      const res = await apiFetch<SupplierSummary>(endpoint);
      if (res.success && res.data) {
        setSummary(res.data);
      }
    } catch (err) {
      console.error("Failed to fetch supplier summary:", err);
    }
  }, [selectedPeriodId]);

  const fetchTransactions = useCallback(async (
    supplierId?: string,
    periodId?: string,
    search?: string,
    direction?: string
  ) => {
    setLoadingTransactions(true);
    try {
      const activeSupplier = supplierId !== undefined ? supplierId : selectedSupplierId;
      const activePeriod = periodId !== undefined ? periodId : selectedPeriodId;
      const activeSearch = search !== undefined ? search : searchQuery;
      const activeDirection = direction !== undefined ? direction : directionFilter;

      const params = new URLSearchParams();
      if (activePeriod) params.append("period_id", activePeriod);
      if (activeSearch) params.append("search", activeSearch);
      if (activeDirection) params.append("direction", activeDirection);
      params.append("limit", "100");

      let endpoint = "/suppliers/transactions";
      if (activeSupplier && activeSupplier !== "all") {
        endpoint = `/suppliers/${activeSupplier}/transactions`;
      }

      const queryString = params.toString();
      if (queryString) {
        endpoint += `?${queryString}`;
      }

      const res = await apiFetch<SupplierTransaction[]>(endpoint);
      if (res.success && res.data) {
        const isOriginalReversed = new Set<string>();
        res.data.forEach((tx) => {
          if (tx.reversed_by) {
            isOriginalReversed.add(tx.reversed_by);
          }
        });

        const mapped: SupplierTransaction[] = res.data.map((tx) => {
          const isReversal = Boolean(
            tx.reversed_by ||
            tx.description?.startsWith("[İPTAL/TERS KAYIT]")
          );
          const wasReversed = isOriginalReversed.has(tx.id);

          return {
            ...tx,
            is_reversal_entry: isReversal,
            was_reversed: wasReversed,
          };
        });

        setTransactions(mapped);
      }
    } catch (err) {
      console.error("Failed to fetch supplier transactions:", err);
    } finally {
      setLoadingTransactions(false);
    }
  }, [selectedSupplierId, selectedPeriodId, searchQuery, directionFilter]);

  const refreshAll = useCallback(() => {
    fetchSuppliers();
    fetchSummary();
    fetchTransactions();
  }, [fetchSuppliers, fetchSummary, fetchTransactions]);

  const handleReverseSupplierTransaction = useCallback(async (
    targetTxId: string,
    reason: string,
    idempotencyKey: string
  ) => {
    try {
      const res = await apiFetch<any>(`/suppliers/transactions/${targetTxId}/reverse`, {
        method: "POST",
        headers: { "Idempotency-Key": idempotencyKey },
        body: JSON.stringify({ reason }),
      });

      if (res.success) {
        refreshAll();
        return { success: true };
      } else {
        return { success: false, error: res.error?.message || "Ters kayıt işlemi başarısız oldu." };
      }
    } catch (err: any) {
      return { success: false, error: err?.message || "Sunucuya bağlanılamadı." };
    }
  }, [refreshAll]);

  useEffect(() => {
    fetchSuppliers();
    fetchSummary();
  }, [fetchSuppliers, fetchSummary]);

  useEffect(() => {
    fetchTransactions();
  }, [fetchTransactions]);

  return {
    suppliers,
    summary,
    selectedSupplierId,
    setSelectedSupplierId,
    transactions,
    loadingSuppliers,
    loadingTransactions,
    searchQuery,
    setSearchQuery,
    directionFilter,
    setDirectionFilter,
    fetchSuppliers,
    fetchSummary,
    fetchTransactions,
    handleReverseSupplierTransaction,
    refreshAll,
  };
}
