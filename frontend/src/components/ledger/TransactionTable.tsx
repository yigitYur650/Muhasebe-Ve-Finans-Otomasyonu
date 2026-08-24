"use client";

import { useState, useMemo } from "react";
import { useTranslations } from "next-intl";
import {
  useReactTable,
  getCoreRowModel,
  getFilteredRowModel,
  getSortedRowModel,
  flexRender,
  createColumnHelper,
} from "@tanstack/react-table";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { formatTL } from "@/lib/decimal";
import { ArrowDownLeft, ArrowUpRight, Search, RotateCcw, Ban } from "lucide-react";

export interface TransactionItem {
  id: string;
  periodId: string;
  direction: "in" | "out";
  channel: string;
  amount: string;
  description?: string;
  createdBy: string;
  createdAt: string;
  reversedBy?: string | null;
  isReversalEntry?: boolean;
}

interface TransactionTableProps {
  transactions: TransactionItem[];
  isPeriodLocked: boolean;
  onReverse: (tx: TransactionItem) => void;
}

const columnHelper = createColumnHelper<TransactionItem>();

export function TransactionTable({ transactions, isPeriodLocked, onReverse }: TransactionTableProps) {
  const t = useTranslations("common");
  const tTx = useTranslations("transaction");

  const [globalFilter, setGlobalFilter] = useState("");
  const [channelFilter, setChannelFilter] = useState<string>("all");
  const [directionFilter, setDirectionFilter] = useState<string>("all");
  const [statusFilter, setStatusFilter] = useState<string>("all");

  const filteredData = useMemo(() => {
    return transactions.filter((tx) => {
      // Status filter
      const isReversedOrReversal = !!tx.reversedBy || !!tx.isReversalEntry;
      if (statusFilter === "active" && isReversedOrReversal) {
        return false;
      }
      if (statusFilter === "reversed" && !isReversedOrReversal) {
        return false;
      }
      // Direction filter
      if (directionFilter !== "all" && tx.direction !== directionFilter) {
        return false;
      }
      // Channel filter
      if (channelFilter !== "all" && tx.channel !== channelFilter) {
        return false;
      }
      // Global search filter (description, channel, createdBy, amount)
      if (globalFilter.trim() !== "") {
        const query = globalFilter.toLowerCase();
        const descMatch = tx.description?.toLowerCase().includes(query);
        const userMatch = tx.createdBy.toLowerCase().includes(query);
        const amountMatch = tx.amount.includes(query);
        if (!descMatch && !userMatch && !amountMatch) {
          return false;
        }
      }
      return true;
    });
  }, [transactions, globalFilter, channelFilter, directionFilter, statusFilter]);

  const columns = useMemo(
    () => [
      columnHelper.accessor("direction", {
        header: () => tTx("direction"),
        cell: (info) => {
          const isIn = info.getValue() === "in";
          const isReversed = !!info.row.original.reversedBy;
          return (
            <Badge
              variant={isReversed ? "outline" : isIn ? "success" : "destructive"}
              className="gap-1 font-semibold text-xs"
            >
              {isIn ? (
                <>
                  <ArrowDownLeft className="w-3 h-3" />
                  {tTx("directionIn")}
                </>
              ) : (
                <>
                  <ArrowUpRight className="w-3 h-3" />
                  {tTx("directionOut")}
                </>
              )}
            </Badge>
          );
        },
      }),
      columnHelper.accessor("channel", {
        header: () => tTx("channel"),
        cell: (info) => (
          <span className="font-medium text-slate-800">
            {tTx(`channels.${info.getValue()}`)}
          </span>
        ),
      }),
      columnHelper.accessor("amount", {
        header: () => <div className="text-right">{tTx("amount")}</div>,
        cell: (info) => {
          const isIn = info.row.original.direction === "in";
          const isReversed = !!info.row.original.reversedBy;
          return (
            <div
              className={`text-right font-extrabold ${
                isReversed
                  ? "line-through text-slate-400 opacity-60"
                  : isIn
                  ? "text-emerald-600"
                  : "text-rose-600"
              }`}
            >
              {isIn ? "+" : "-"}{formatTL(info.getValue())}
            </div>
          );
        },
      }),
      columnHelper.accessor("description", {
        header: () => tTx("description"),
        cell: (info) => {
          const isReversed = !!info.row.original.reversedBy;
          return (
            <span className={`text-slate-600 max-w-[260px] truncate block ${isReversed ? "line-through opacity-60" : ""}`}>
              {info.getValue() || "-"}
            </span>
          );
        },
      }),
      columnHelper.accessor("createdAt", {
        header: () => tTx("createdAt"),
        cell: (info) => <span className="text-xs text-slate-500" suppressHydrationWarning>{info.getValue()}</span>,
      }),
      columnHelper.accessor("createdBy", {
        header: () => tTx("createdBy"),
        cell: (info) => <span className="text-xs text-slate-700 font-medium">{info.getValue()}</span>,
      }),
      columnHelper.display({
        id: "status",
        header: () => t("status"),
        cell: (info) => {
          const isReversed = !!info.row.original.reversedBy;
          const isReversalEntry = !!info.row.original.isReversalEntry;
          if (isReversed) {
            return (
              <Badge variant="outline" className="gap-1 text-rose-600 border-rose-200 bg-rose-50 text-[10px]">
                <Ban className="w-3 h-3" />
                {tTx("reversed")}
              </Badge>
            );
          }
          if (isReversalEntry) {
            return (
              <Badge variant="outline" className="gap-1 text-blue-700 border-blue-200 bg-blue-50 text-[10px]">
                <RotateCcw className="w-3 h-3 text-blue-600" />
                Denkleştirme
              </Badge>
            );
          }
          return (
            <Badge variant="secondary" className="text-[10px] text-emerald-700 bg-emerald-50">
              {t("open")}
            </Badge>
          );
        },
      }),
      columnHelper.display({
        id: "actions",
        header: () => <div className="text-right">{t("actions")}</div>,
        cell: (info) => {
          const tx = info.row.original;
          const isReversed = !!tx.reversedBy;
          const isReversalEntry = !!tx.isReversalEntry;
          const isDisabled = isPeriodLocked || isReversed || isReversalEntry;

          return (
            <div className="text-right">
              <Button
                variant="ghost"
                size="sm"
                disabled={isDisabled}
                onClick={() => onReverse(tx)}
                className="gap-1 text-xs text-rose-600 hover:text-rose-700 hover:bg-rose-50 disabled:opacity-40"
              >
                <RotateCcw className="w-3.5 h-3.5" />
                {tTx("reverseAction")}
              </Button>
            </div>
          );
        },
      }),
    ],
    [t, tTx, isPeriodLocked, onReverse]
  );

  const table = useReactTable({
    data: filteredData,
    columns,
    getCoreRowModel: getCoreRowModel(),
    getFilteredRowModel: getFilteredRowModel(),
    getSortedRowModel: getSortedRowModel(),
  });

  const channelsList = [
    "eft",
    "pos",
    "nakit",
    "kredi",
    "kira",
    "maas_banka",
    "maas_elden",
    "kredi_karti",
    "kartus",
    "yemek",
    "yakit",
    "diger",
  ];

  return (
    <div className="space-y-4">
      {/* Filter Toolbar */}
      <div className="flex flex-col sm:flex-row items-center justify-between gap-3 p-4 bg-white rounded-t-xl border-b">
        <div className="relative w-full sm:w-72">
          <Search className="absolute left-3 top-2.5 h-4 w-4 text-slate-400" />
          <Input
            placeholder={t("search")}
            value={globalFilter}
            onChange={(e) => setGlobalFilter(e.target.value)}
            className="pl-9 text-xs"
          />
        </div>

        <div className="flex flex-wrap items-center gap-3 w-full sm:w-auto">
          {/* Status Filter (Aktif vs İptal) */}
          <Select value={statusFilter} onValueChange={setStatusFilter}>
            <SelectTrigger className="w-[165px] text-xs font-medium">
              <SelectValue placeholder="Durum Süzgeci" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">Tüm İşlemler</SelectItem>
              <SelectItem value="active">Sadece Aktif Kayıtlar</SelectItem>
              <SelectItem value="reversed">İptal/Denkleştirme</SelectItem>
            </SelectContent>
          </Select>

          {/* Direction Filter */}
          <Select value={directionFilter} onValueChange={setDirectionFilter}>
            <SelectTrigger className="w-[140px] text-xs">
              <SelectValue placeholder={tTx("filterDirection")} />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">{t("all")}</SelectItem>
              <SelectItem value="in">{tTx("directionIn")}</SelectItem>
              <SelectItem value="out">{tTx("directionOut")}</SelectItem>
            </SelectContent>
          </Select>

          {/* Channel Filter */}
          <Select value={channelFilter} onValueChange={setChannelFilter}>
            <SelectTrigger className="w-[160px] text-xs">
              <SelectValue placeholder={tTx("filterChannel")} />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">{t("all")}</SelectItem>
              {channelsList.map((ch) => (
                <SelectItem key={ch} value={ch}>
                  {tTx(`channels.${ch}`)}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>
      </div>

      {/* TanStack Table */}
      <div className="bg-white rounded-b-xl overflow-hidden">
        <Table>
          <TableHeader className="bg-slate-100/70">
            {table.getHeaderGroups().map((headerGroup) => (
              <TableRow key={headerGroup.id}>
                {headerGroup.headers.map((header) => (
                  <TableHead key={header.id}>
                    {header.isPlaceholder
                      ? null
                      : flexRender(header.column.columnDef.header, header.getContext())}
                  </TableHead>
                ))}
              </TableRow>
            ))}
          </TableHeader>
          <TableBody>
            {table.getRowModel().rows.length > 0 ? (
              table.getRowModel().rows.map((row) => {
                const isReversed = !!row.original.reversedBy;
                return (
                  <TableRow
                    key={row.id}
                    className={`transition-colors ${
                      isReversed ? "bg-slate-50/70 opacity-75" : "hover:bg-slate-50/80"
                    }`}
                  >
                    {row.getVisibleCells().map((cell) => (
                      <TableCell key={cell.id}>
                        {flexRender(cell.column.columnDef.cell, cell.getContext())}
                      </TableCell>
                    ))}
                  </TableRow>
                );
              })
            ) : (
              <TableRow>
                <TableCell colSpan={columns.length} className="h-24 text-center text-slate-500 text-sm">
                  {t("noResults")}
                </TableCell>
              </TableRow>
            )}
          </TableBody>
        </Table>
      </div>
    </div>
  );
}

export default TransactionTable;

