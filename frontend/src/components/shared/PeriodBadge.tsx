"use client";

import { useTranslations } from "next-intl";
import { Badge } from "@/components/ui/badge";
import { Lock, LockOpen } from "lucide-react";

interface PeriodBadgeProps {
  status: "open" | "locked";
  label?: string;
}

export function PeriodBadge({ status, label }: PeriodBadgeProps) {
  const t = useTranslations("period");

  const isLocked = status === "locked";

  return (
    <div className="inline-flex items-center gap-2">
      {label && <span className="font-semibold text-slate-700">{label}</span>}
      <Badge variant={isLocked ? "destructive" : "success"} className="gap-1 px-2.5 py-1 text-xs">
        {isLocked ? (
          <>
            <Lock className="w-3 h-3" />
            {t("statusLocked")}
          </>
        ) : (
          <>
            <LockOpen className="w-3 h-3" />
            {t("statusOpen")}
          </>
        )}
      </Badge>
    </div>
  );
}

export default PeriodBadge;

