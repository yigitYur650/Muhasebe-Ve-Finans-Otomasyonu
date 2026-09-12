"use client";

import { useState } from "react";
import { useTranslations } from "next-intl";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
} from "@/components/ui/dialog";
import { Download, Loader2, FileSpreadsheet, CheckCircle2, History } from "lucide-react";
import { getApiUrl } from "@/lib/api";
import { createClient } from "@/lib/supabase/client";

interface ExportCsvButtonProps {
  periodId: string;
  periodLabel: string;
}

export function ExportCsvButton({ periodId, periodLabel }: ExportCsvButtonProps) {
  const t = useTranslations("import_export");
  const [isOpen, setIsOpen] = useState<boolean>(false);
  const [isExporting, setIsExporting] = useState<boolean>(false);
  const [exportMode, setExportMode] = useState<"active" | "all" | "csv" | null>(null);

  const handleDownload = async (status: "active" | "all", format: "excel" | "csv" = "excel") => {
    setIsExporting(true);
    setExportMode(format === "csv" ? "csv" : status);
    try {
      let authToken: string | undefined;
      let userId: string | undefined;

      if (typeof window !== "undefined") {
        try {
          const supabase = createClient();
          const { data } = await supabase.auth.getSession();
          if (data?.session?.access_token) {
            authToken = data.session.access_token;
            userId = data.session.user?.id;
          }
        } catch {
          // fallback
        }
      }

      const headers: Record<string, string> = {
        "X-Tenant-ID": "00000000-0000-0000-0000-000000000001",
        "X-User-ID": userId || "149c91f0-0d03-4e3a-81d7-0bc5688c01b0",
        "X-User-Role": "admin",
      };

      if (authToken) {
        headers["Authorization"] = `Bearer ${authToken}`;
      }

      const endpoint = format === "excel" ? "export/excel" : "export/csv";
      const ext = format === "excel" ? "xlsx" : "csv";

      const response = await fetch(
        getApiUrl(`/periods/${periodId}/${endpoint}?status=${status}`),
        { headers }
      );

      if (!response.ok) {
        throw new Error("Dışa aktarım başarısız oldu.");
      }

      const blob = await response.blob();
      const url = window.URL.createObjectURL(blob);
      const link = document.createElement("a");
      link.href = url;
      const suffix = status === "active" ? "aktif-kayitlar" : "tum-kayitlar";
      link.setAttribute("download", `defter-${periodLabel}-${suffix}.${ext}`);
      document.body.appendChild(link);
      link.click();
      link.parentNode?.removeChild(link);
      window.URL.revokeObjectURL(url);
      setIsOpen(false);
    } catch (err: any) {
      alert(err?.message || "Dışa aktarılırken bir hata oluştu.");
    } finally {
      setIsExporting(false);
      setExportMode(null);
    }
  };

  return (
    <>
      <Button
        variant="outline"
        size="sm"
        onClick={() => setIsOpen(true)}
        className="gap-2 h-9 text-xs font-semibold border-slate-300 hover:bg-slate-50 text-slate-700 shadow-sm"
      >
        <FileSpreadsheet className="w-4 h-4 text-emerald-600" />
        Excel'e Aktar
      </Button>

      <Dialog open={isOpen} onOpenChange={setIsOpen}>
        <DialogContent className="sm:max-w-lg">
          <DialogHeader>
            <DialogTitle className="flex items-center gap-2 text-base font-bold text-slate-800">
              <FileSpreadsheet className="w-5 h-5 text-emerald-600" />
              Excel (.xlsx) Raporunu İndir
            </DialogTitle>
            <DialogDescription className="text-xs text-slate-500">
              {periodLabel} dönemine ait verileri sütun genişlikleri, başlıkları ve para birimleri otomatik ayarlanmış hazır Excel (.xlsx) formatında indirin.
            </DialogDescription>
          </DialogHeader>

          <div className="space-y-3 py-3">
            {/* Option 1: Aktif İşlemler (Önerilen) */}
            <div className="p-3.5 rounded-xl border border-emerald-200 bg-emerald-50/50 hover:bg-emerald-50 transition-colors">
              <div className="flex items-start justify-between gap-3">
                <div className="space-y-1">
                  <div className="flex items-center gap-1.5 font-semibold text-xs text-emerald-900">
                    <CheckCircle2 className="w-4 h-4 text-emerald-600 shrink-0" />
                    Aktif İşlemler — Excel (.xlsx) (Önerilen)
                  </div>
                  <p className="text-[11px] text-emerald-700 leading-relaxed">
                    Yalnızca geçerli muhasebe kayıtlarını içerir. Sütun boyutları (tarih, tutar, açıklama) tam sığacak şekilde otomatik ayarlanmıştır. Excel'de toplandığında ekrandaki ciro ve kasa bakiyesiyle kuruşu kuruşuna tam tutar.
                  </p>
                </div>
                <Button
                  size="sm"
                  onClick={() => handleDownload("active", "excel")}
                  disabled={isExporting}
                  className="shrink-0 h-8 text-xs font-medium bg-emerald-600 hover:bg-emerald-700 text-white shadow-sm"
                >
                  {isExporting && exportMode === "active" ? (
                    <Loader2 className="w-3.5 h-3.5 animate-spin" />
                  ) : (
                    <Download className="w-3.5 h-3.5 mr-1" />
                  )}
                  Excel İndir
                </Button>
              </div>
            </div>

            {/* Option 2: Tüm Defter Geçmişi */}
            <div className="p-3.5 rounded-xl border border-slate-200 bg-slate-50/70 hover:bg-slate-50 transition-colors">
              <div className="flex items-start justify-between gap-3">
                <div className="space-y-1">
                  <div className="flex items-center gap-1.5 font-semibold text-xs text-slate-800">
                    <History className="w-4 h-4 text-slate-500 shrink-0" />
                    Tüm Defter Geçmişi — Excel (.xlsx) (Denetim Kayıtları Dahil)
                  </div>
                  <p className="text-[11px] text-slate-600 leading-relaxed">
                    İptal edilen kayıtlar ve onları dengeleyen ters kayıtlar dahil defterdeki tüm satırları durum etiketleriyle ("İptal Edildi", "Ters Kayıt", "Aktif") renkli ve filtreli Excel olarak listeler.
                  </p>
                </div>
                <Button
                  variant="outline"
                  size="sm"
                  onClick={() => handleDownload("all", "excel")}
                  disabled={isExporting}
                  className="shrink-0 h-8 text-xs font-medium border-slate-300 hover:bg-slate-100 text-slate-700 shadow-sm"
                >
                  {isExporting && exportMode === "all" ? (
                    <Loader2 className="w-3.5 h-3.5 animate-spin" />
                  ) : (
                    <Download className="w-3.5 h-3.5 mr-1" />
                  )}
                  Excel İndir
                </Button>
              </div>
            </div>

            {/* Option 3: Sade CSV İndirme */}
            <div className="pt-2 flex justify-end">
              <button
                type="button"
                onClick={() => handleDownload("active", "csv")}
                disabled={isExporting}
                className="text-[11px] text-slate-400 hover:text-slate-600 underline"
              >
                Metin tabanlı düz CSV formatında indir
              </button>
            </div>
          </div>
        </DialogContent>
      </Dialog>
    </>
  );
}

export default ExportCsvButton;
