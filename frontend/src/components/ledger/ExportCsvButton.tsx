"use client";

import { useState } from "react";
import { useTranslations } from "next-intl";
import { Button } from "@/components/ui/button";
import { Download, Loader2 } from "lucide-react";
import { getApiUrl } from "@/lib/api";

interface ExportCsvButtonProps {
  periodId: string;
  periodLabel: string;
}

export function ExportCsvButton({ periodId, periodLabel }: ExportCsvButtonProps) {
  const t = useTranslations("import_export");
  const [isExporting, setIsExporting] = useState<boolean>(false);

  const handleExport = async () => {
    setIsExporting(true);
    try {
      const response = await fetch(getApiUrl(`/periods/${periodId}/export/csv`));
      if (!response.ok) {
        throw new Error("Export failed");
      }

      const blob = await response.blob();
      const url = window.URL.createObjectURL(blob);
      const link = document.createElement("a");
      link.href = url;
      link.setAttribute("download", `defter-islem-defteri-${periodLabel}.csv`);
      document.body.appendChild(link);
      link.click();
      link.parentNode?.removeChild(link);
      window.URL.revokeObjectURL(url);
    } catch {
      // Fallback client-side CSV trigger
      window.open(getApiUrl(`/periods/${periodId}/export/csv`), "_blank");
    } finally {
      setIsExporting(false);
    }
  };

  return (
    <Button
      variant="outline"
      size="sm"
      onClick={handleExport}
      disabled={isExporting}
      className="gap-2 h-9 text-xs font-semibold border-slate-300 hover:bg-slate-50 text-slate-700 shadow-sm"
    >
      {isExporting ? (
        <Loader2 className="w-4 h-4 animate-spin text-slate-600" />
      ) : (
        <Download className="w-4 h-4 text-emerald-600" />
      )}
      {t("exportCsv")}
    </Button>
  );
}

export default ExportCsvButton;
