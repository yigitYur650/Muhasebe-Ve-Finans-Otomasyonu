"use client";

import { useTranslations } from "next-intl";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Lock, LockOpen } from "lucide-react";

export interface PeriodOption {
  id: string;
  label: string;
  status: "open" | "locked";
  startingBalance: string;
}

interface PeriodSelectorProps {
  periods: PeriodOption[];
  selectedPeriodId: string;
  onSelectPeriod: (period: PeriodOption) => void;
}

export function PeriodSelector({ periods, selectedPeriodId, onSelectPeriod }: PeriodSelectorProps) {
  const t = useTranslations("period");

  const selected = periods.find((p) => p.id === selectedPeriodId) || periods[0];

  return (
    <Select
      value={selectedPeriodId}
      onValueChange={(val) => {
        const found = periods.find((p) => p.id === val);
        if (found) onSelectPeriod(found);
      }}
    >
      <SelectTrigger className="w-[180px] h-9 text-xs bg-white border-slate-300 font-medium">
        <SelectValue placeholder={t("selectPeriod")} />
      </SelectTrigger>
      <SelectContent>
        {periods.map((p) => {
          const isLocked = p.status === "locked";
          return (
            <SelectItem key={p.id} value={p.id} className="text-xs">
              <div className="flex items-center gap-2">
                {isLocked ? (
                  <Lock className="w-3.5 h-3.5 text-rose-500" />
                ) : (
                  <LockOpen className="w-3.5 h-3.5 text-emerald-500" />
                )}
                <span>
                  {p.label} ({isLocked ? t("statusLocked") : t("statusOpen")})
                </span>
              </div>
            </SelectItem>
          );
        })}
      </SelectContent>
    </Select>
  );
}
