"use client";

import { useState, useRef, useEffect, KeyboardEvent } from "react";
import { useTranslations } from "next-intl";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { parseDecimalAmount } from "@/lib/decimal";
import { ArrowUpRight, ArrowDownRight, Zap, CornerDownLeft, X, Sparkles } from "lucide-react";

interface QuickEntryRowProps {
  isPeriodLocked: boolean;
  onSubmitTransaction: (data: {
    direction: "in" | "out";
    channel: string;
    amount: string;
    description: string;
    idempotencyKey: string;
  }) => Promise<void>;
}

export function QuickEntryRow({ isPeriodLocked, onSubmitTransaction }: QuickEntryRowProps) {
  const t = useTranslations("quick_entry");
  const tCh = useTranslations("transaction.channels");

  const [direction, setDirection] = useState<"in" | "out">("in");
  const [channel, setChannel] = useState<string>("eft");
  const [amount, setAmount] = useState<string>("");
  const [description, setDescription] = useState<string>("");
  const [amountError, setAmountError] = useState<string | null>(null);
  const [isSubmitting, setIsSubmitting] = useState<boolean>(false);

  // Refs for keyboard focus management
  const amountRef = useRef<HTMLInputElement>(null);
  const descRef = useRef<HTMLInputElement>(null);

  if (isPeriodLocked) {
    return null; // Don't render quick entry bar when period is locked (Read-Only Mode)
  }

  const handleReset = () => {
    setAmount("");
    setDescription("");
    setAmountError(null);
    setDirection("in");
    setChannel("eft");
    amountRef.current?.focus();
  };

  const handleCustomKeyDown = (e: KeyboardEvent<HTMLInputElement>) => {
    // Escape: Clear row
    if (e.key === "Escape") {
      e.preventDefault();
      handleReset();
      return;
    }

    // Hotkey 'G' or '+' -> Income
    if ((e.key === "g" || e.key === "G" || e.key === "+") && document.activeElement !== descRef.current) {
      e.preventDefault();
      setDirection("in");
      return;
    }

    // Hotkey 'C' or '-' -> Expense
    if ((e.key === "c" || e.key === "C" || e.key === "-") && document.activeElement !== descRef.current) {
      e.preventDefault();
      setDirection("out");
      return;
    }

    // Enter: Submit
    if (e.key === "Enter") {
      e.preventDefault();
      handleSubmit();
    }
  };

  const handleSubmit = async () => {
    setAmountError(null);

    const parsedDecimal = parseDecimalAmount(amount);
    if (!parsedDecimal || parsedDecimal.lte(0)) {
      setAmountError(t("invalidAmount"));
      amountRef.current?.focus();
      return;
    }

    setIsSubmitting(true);
    try {
      const idempotencyKey = crypto.randomUUID();
      await onSubmitTransaction({
        direction,
        channel,
        amount: parsedDecimal.toFixed(2),
        description: description.trim() || "-",
        idempotencyKey,
      });

      // Continuous rapid entry: clear fields and refocus amount input
      setAmount("");
      setDescription("");
      setAmountError(null);
      amountRef.current?.focus();
    } catch (err: any) {
      setAmountError(err.message);
    } finally {
      setIsSubmitting(false);
    }
  };

  return (
    <div className="bg-gradient-to-r from-indigo-900 via-slate-900 to-slate-800 text-white p-4 rounded-xl shadow-md border border-indigo-700/50 mb-6">
      {/* Header bar with Excel UX badge & shortcut guide */}
      <div className="flex flex-wrap items-center justify-between gap-2 mb-3 pb-2 border-b border-indigo-500/20">
        <div className="flex items-center gap-2">
          <div className="p-1 rounded bg-indigo-500/30 text-indigo-300">
            <Zap className="w-4 h-4" />
          </div>
          <span className="text-xs font-bold tracking-wide uppercase text-indigo-200">
            {t("title")}
          </span>
          <span className="inline-flex items-center gap-1 text-[10px] bg-emerald-500/20 text-emerald-300 font-semibold px-2 py-0.5 rounded border border-emerald-500/30">
            <Sparkles className="w-3 h-3" />
            Excel Mode
          </span>
        </div>

        <div className="text-[11px] text-slate-400 font-mono">
          {t("shortcutHint")}
        </div>
      </div>

      {/* Main Quick Entry Form Bar */}
      <div className="grid grid-cols-1 sm:grid-cols-12 gap-3 items-start">
        {/* Direction Switcher (In / Out) */}
        <div className="sm:col-span-2 flex gap-1 bg-slate-950/60 p-1 rounded-lg border border-slate-700/60 h-9">
          <button
            type="button"
            onClick={() => setDirection("in")}
            className={`flex-1 flex items-center justify-center gap-1 text-xs font-bold rounded transition-colors ${
              direction === "in"
                ? "bg-emerald-600 text-white shadow-sm"
                : "text-slate-400 hover:text-slate-200"
            }`}
          >
            <ArrowUpRight className="w-3.5 h-3.5" />
            {t("directionIn")}
          </button>
          <button
            type="button"
            onClick={() => setDirection("out")}
            className={`flex-1 flex items-center justify-center gap-1 text-xs font-bold rounded transition-colors ${
              direction === "out"
                ? "bg-rose-600 text-white shadow-sm"
                : "text-slate-400 hover:text-slate-200"
            }`}
          >
            <ArrowDownRight className="w-3.5 h-3.5" />
            {t("directionOut")}
          </button>
        </div>

        {/* Channel Selector */}
        <div className="sm:col-span-2">
          <Select value={channel} onValueChange={setChannel}>
            <SelectTrigger className="h-9 text-xs bg-slate-950/60 border-slate-700/60 text-slate-100 font-medium">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="eft" className="text-xs">{tCh("eft")}</SelectItem>
              <SelectItem value="pos" className="text-xs">{tCh("pos")}</SelectItem>
              <SelectItem value="nakit" className="text-xs">{tCh("nakit")}</SelectItem>
              <SelectItem value="kredi" className="text-xs">{tCh("kredi")}</SelectItem>
              <SelectItem value="kira" className="text-xs">{tCh("kira")}</SelectItem>
              <SelectItem value="maas_banka" className="text-xs">{tCh("maas_banka")}</SelectItem>
              <SelectItem value="kredi_karti" className="text-xs">{tCh("kredi_karti")}</SelectItem>
              <SelectItem value="yakit" className="text-xs">{tCh("yakit")}</SelectItem>
              <SelectItem value="yemek" className="text-xs">{tCh("yemek")}</SelectItem>
              <SelectItem value="diger" className="text-xs">{tCh("diger")}</SelectItem>
            </SelectContent>
          </Select>
        </div>

        {/* Amount Input with Inline Decimal Validation */}
        <div className="sm:col-span-3">
          <Input
            ref={amountRef}
            type="text"
            inputMode="decimal"
            placeholder={t("amountPlaceholder")}
            value={amount}
            onChange={(e) => {
              setAmount(e.target.value);
              if (amountError) setAmountError(null);
            }}
            onKeyDown={handleCustomKeyDown}
            className={`h-9 text-xs font-semibold bg-slate-950/60 border-slate-700/60 text-white placeholder:text-slate-500 focus:border-indigo-400 ${
              amountError ? "border-rose-500 focus:ring-rose-500" : ""
            }`}
          />
          {amountError && (
            <p className="text-[11px] text-rose-400 mt-1 font-medium">{amountError}</p>
          )}
        </div>

        {/* Description Input */}
        <div className="sm:col-span-3">
          <Input
            ref={descRef}
            type="text"
            placeholder={t("descPlaceholder")}
            value={description}
            onChange={(e) => setDescription(e.target.value)}
            onKeyDown={handleCustomKeyDown}
            className="h-9 text-xs bg-slate-950/60 border-slate-700/60 text-white placeholder:text-slate-500 focus:border-indigo-400"
          />
        </div>

        {/* Actions (Submit / Reset) */}
        <div className="sm:col-span-2 flex gap-1.5 justify-end">
          <Button
            type="button"
            disabled={isSubmitting}
            onClick={handleSubmit}
            size="sm"
            className="h-9 gap-1 text-xs bg-emerald-500 hover:bg-emerald-600 text-white font-bold px-3 shadow"
          >
            <CornerDownLeft className="w-3.5 h-3.5" />
            {t("submitBtn")}
          </Button>

          <Button
            type="button"
            variant="ghost"
            onClick={handleReset}
            size="sm"
            className="h-9 w-9 p-0 text-slate-400 hover:text-white hover:bg-slate-800"
            title={t("resetBtn")}
          >
            <X className="w-4 h-4" />
          </Button>
        </div>
      </div>
    </div>
  );
}
