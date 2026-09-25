"use client";

import { useState, useRef, ChangeEvent } from "react";
import { useTranslations } from "next-intl";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Upload, FileSpreadsheet, AlertCircle, CheckCircle2, Loader2, Truck } from "lucide-react";
import { getApiUrl } from "@/lib/api";
import { createClient } from "@/lib/supabase/client";

interface ImportSupplierExcelDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  periodId: string;
  isPeriodLocked: boolean;
  onImportSuccess: () => void;
}

export function ImportSupplierExcelDialog({
  open,
  onOpenChange,
  periodId,
  isPeriodLocked,
  onImportSuccess,
}: ImportSupplierExcelDialogProps) {
  const t = useTranslations("suppliers");
  const tErr = useTranslations("errors");
  const fileInputRef = useRef<HTMLInputElement>(null);

  const [selectedFile, setSelectedFile] = useState<File | null>(null);
  const [sheetName, setSheetName] = useState<string>("");
  const [isUploading, setIsUploading] = useState<boolean>(false);
  const [errorMsg, setErrorMsg] = useState<string | null>(null);
  const [successMsg, setSuccessMsg] = useState<string | null>(null);
  const [createdSuppliers, setCreatedSuppliers] = useState<string[]>([]);

  const handleFileChange = (e: ChangeEvent<HTMLInputElement>) => {
    if (e.target.files && e.target.files[0]) {
      const file = e.target.files[0];
      const name = file.name.toLowerCase();
      if (!name.endsWith(".xlsx") && !name.endsWith(".xlsm")) {
        setErrorMsg("Lütfen yalnızca Excel (.xlsx veya .xlsm) dosyası seçin.");
        setSelectedFile(null);
        return;
      }
      setSelectedFile(file);
      setErrorMsg(null);
      setSuccessMsg(null);
    }
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!selectedFile) return;

    if (isPeriodLocked) {
      setErrorMsg(tErr("PERIOD_LOCKED"));
      return;
    }

    setIsUploading(true);
    setErrorMsg(null);
    setSuccessMsg(null);

    try {
      const formData = new FormData();
      formData.append("file", selectedFile);
      formData.append("period_id", periodId);
      if (sheetName.trim()) {
        formData.append("sheet_name", sheetName.trim());
      }

      // Retrieve auth session headers
      let token = "";
      if (typeof window !== "undefined") {
        try {
          const supabase = createClient();
          const { data } = await supabase.auth.getSession();
          if (data?.session?.access_token) {
            token = data.session.access_token;
          }
        } catch {}
      }

      const headers: Record<string, string> = {};
      if (token) {
        headers["Authorization"] = `Bearer ${token}`;
      }

      const response = await fetch(getApiUrl("/suppliers/import/excel"), {
        method: "POST",
        headers,
        body: formData,
      });

      const data = await response.json();

      if (!response.ok || !data.success) {
        throw new Error(data.error?.message || "Excel içe aktarılırken hata oluştu");
      }

      const resData = data.data;
      const count = resData?.imported_count || 0;
      const purchase = resData?.total_purchase || "0.00";
      const payment = resData?.total_payment || "0.00";
      const created = resData?.suppliers_created || [];

      setSuccessMsg(
        t("importSuccessSummary", {
          count,
          purchase,
          payment,
        })
      );
      setCreatedSuppliers(created);
      setSelectedFile(null);
      setSheetName("");
      onImportSuccess();

      setTimeout(() => {
        onOpenChange(false);
        setSuccessMsg(null);
      }, 3500);
    } catch (err: any) {
      setErrorMsg(err.message || "İçe aktarım sırasında beklenmeyen bir hata oluştu.");
    } finally {
      setIsUploading(false);
    }
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-[540px]">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2 text-slate-900">
            <FileSpreadsheet className="w-5 h-5 text-indigo-600" />
            {t("importTitle")}
          </DialogTitle>
          <DialogDescription className="text-xs text-slate-500">
            {t("importDescription")}
          </DialogDescription>
        </DialogHeader>

        <form onSubmit={handleSubmit} className="space-y-4 pt-2">
          {/* File Selector Dropzone */}
          <div
            onClick={() => fileInputRef.current?.click()}
            className={`border-2 border-dashed rounded-xl p-6 text-center cursor-pointer transition-colors ${
              selectedFile
                ? "border-indigo-400 bg-indigo-50/50"
                : "border-slate-300 hover:border-indigo-400 bg-slate-50/50 hover:bg-indigo-50/20"
            }`}
          >
            <input
              type="file"
              ref={fileInputRef}
              onChange={handleFileChange}
              accept=".xlsx, .xlsm"
              className="hidden"
            />
            <div className="flex flex-col items-center justify-center gap-2">
              <div className="p-3 bg-white border border-slate-200 rounded-full shadow-sm">
                <Upload className="w-6 h-6 text-indigo-600" />
              </div>
              {selectedFile ? (
                <div>
                  <p className="text-xs font-bold text-indigo-900">{selectedFile.name}</p>
                  <p className="text-[11px] text-slate-500">
                    {(selectedFile.size / 1024).toFixed(1)} KB — Dosyayı değiştirmek için tıklayın
                  </p>
                </div>
              ) : (
                <div>
                  <p className="text-xs font-semibold text-slate-700">{t("selectExcelFile")}</p>
                  <p className="text-[11px] text-slate-400">
                    36 Sayfalık Kasa Defteri (.xlsx, .xlsm) formatı desteklenir
                  </p>
                </div>
              )}
            </div>
          </div>

          {/* Optional Sheet Name Input */}
          <div className="space-y-1">
            <label className="text-xs font-bold text-slate-700">{t("sheetName")}</label>
            <Input
              type="text"
              placeholder={t("sheetPlaceholder")}
              value={sheetName}
              onChange={(e) => setSheetName(e.target.value)}
              className="h-9 text-xs"
            />
            <p className="text-[11px] text-slate-400">
              Boş bırakırsanız mevcut dönemin etiketine uygun sayfa veya ilk sayfa taranır.
            </p>
          </div>

          {/* Error Message */}
          {errorMsg && (
            <div className="flex items-center gap-2 p-3 bg-rose-50 border border-rose-200 text-rose-800 rounded-lg text-xs font-medium">
              <AlertCircle className="w-4 h-4 text-rose-600 shrink-0" />
              <span>{errorMsg}</span>
            </div>
          )}

          {/* Success Message */}
          {successMsg && (
            <div className="space-y-2">
              <div className="flex items-center gap-2 p-3 bg-emerald-50 border border-emerald-200 text-emerald-800 rounded-lg text-xs font-medium">
                <CheckCircle2 className="w-4 h-4 text-emerald-600 shrink-0" />
                <span>{successMsg}</span>
              </div>
              {createdSuppliers.length > 0 && (
                <div className="p-2.5 bg-slate-50 border border-slate-200 rounded-lg text-[11px] text-slate-600 flex items-center gap-1.5 flex-wrap">
                  <Truck className="w-3.5 h-3.5 text-slate-500 shrink-0" />
                  <span className="font-semibold">İşlenen Firmalar:</span>
                  {createdSuppliers.map((c) => (
                    <span key={c} className="px-1.5 py-0.5 rounded bg-white border border-slate-200 font-bold text-slate-800">
                      {c}
                    </span>
                  ))}
                </div>
              )}
            </div>
          )}

          <DialogFooter className="pt-2 sm:justify-between items-center gap-2">
            <Button
              type="button"
              variant="outline"
              onClick={() => onOpenChange(false)}
              className="h-9 text-xs font-semibold"
              disabled={isUploading}
            >
              Kapat
            </Button>
            <Button
              type="submit"
              disabled={!selectedFile || isUploading || isPeriodLocked}
              className="h-9 text-xs bg-indigo-600 hover:bg-indigo-700 text-white font-semibold gap-2"
            >
              {isUploading ? (
                <>
                  <Loader2 className="w-4 h-4 animate-spin" />
                  İçe Aktarılıyor...
                </>
              ) : (
                <>
                  <FileSpreadsheet className="w-4 h-4" />
                  Excel'i İçe Aktar
                </>
              )}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
