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
import { Upload, FileSpreadsheet, Download, AlertCircle, CheckCircle2, Loader2 } from "lucide-react";
import { apiFetch } from "@/lib/api";

interface ImportCsvDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  periodId: string;
  isPeriodLocked: boolean;
  onImportSuccess: () => void;
}

export function ImportCsvDialog({
  open,
  onOpenChange,
  periodId,
  isPeriodLocked,
  onImportSuccess,
}: ImportCsvDialogProps) {
  const t = useTranslations("import_export");
  const tErr = useTranslations("errors");
  const fileInputRef = useRef<HTMLInputElement>(null);

  const [selectedFile, setSelectedFile] = useState<File | null>(null);
  const [isUploading, setIsUploading] = useState<boolean>(false);
  const [errorMsg, setErrorMsg] = useState<string | null>(null);
  const [successMsg, setSuccessMsg] = useState<string | null>(null);

  const handleFileChange = (e: ChangeEvent<HTMLInputElement>) => {
    if (e.target.files && e.target.files[0]) {
      setSelectedFile(e.target.files[0]);
      setErrorMsg(null);
      setSuccessMsg(null);
    }
  };

  const handleDownloadTemplate = () => {
    window.open(`/api/v1/periods/template/csv`, "_blank");
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

      const response = await fetch(`/api/v1/periods/${periodId}/import/csv`, {
        method: "POST",
        body: formData,
      });

      const data = await response.json();

      if (!response.ok || !data.success) {
        throw new Error(data.error?.message || t("importError"));
      }

      const count = data.data?.imported_count || 0;
      const amount = data.data?.total_amount || "0.00";

      setSuccessMsg(t("successSummary", { count, amount }));
      setSelectedFile(null);
      onImportSuccess();

      setTimeout(() => {
        onOpenChange(false);
        setSuccessMsg(null);
      }, 2000);
    } catch (err: any) {
      setErrorMsg(err.message || t("importError"));
    } finally {
      setIsUploading(false);
    }
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2 text-slate-900">
            <Upload className="w-5 h-5 text-emerald-600" />
            {t("importTitle")}
          </DialogTitle>
          <DialogDescription className="text-xs text-slate-500">
            {t("importDescription")}
          </DialogDescription>
        </DialogHeader>

        <form onSubmit={handleSubmit} className="space-y-4 py-2">
          {errorMsg && (
            <div className="flex items-start gap-2 p-3 text-xs text-rose-700 bg-rose-50 rounded-lg border border-rose-200">
              <AlertCircle className="w-4 h-4 text-rose-600 shrink-0 mt-0.5" />
              <span>{errorMsg}</span>
            </div>
          )}

          {successMsg && (
            <div className="flex items-start gap-2 p-3 text-xs text-emerald-700 bg-emerald-50 rounded-lg border border-emerald-200">
              <CheckCircle2 className="w-4 h-4 text-emerald-600 shrink-0 mt-0.5" />
              <span>{successMsg}</span>
            </div>
          )}

          {/* Sample Template Link */}
          <div className="flex items-center justify-between p-3 bg-slate-50 rounded-lg border border-slate-200 text-xs">
            <span className="text-slate-600 font-medium">{t("downloadTemplate")}</span>
            <Button
              type="button"
              variant="ghost"
              size="sm"
              onClick={handleDownloadTemplate}
              className="h-7 text-xs font-semibold text-emerald-600 hover:text-emerald-700 hover:bg-emerald-50 gap-1.5"
            >
              <Download className="w-3.5 h-3.5" />
              Şablon (.CSV)
            </Button>
          </div>

          {/* Upload Area */}
          <div
            onClick={() => fileInputRef.current?.click()}
            className="border-2 border-dashed border-slate-300 hover:border-emerald-500 bg-slate-50 hover:bg-emerald-50/30 rounded-xl p-6 flex flex-col items-center justify-center text-center cursor-pointer transition-colors"
          >
            <input
              ref={fileInputRef}
              type="file"
              accept=".csv,text/csv"
              onChange={handleFileChange}
              className="hidden"
            />
            <FileSpreadsheet className="w-10 h-10 text-slate-400 mb-2" />
            <span className="text-xs font-semibold text-slate-700">
              {selectedFile ? selectedFile.name : t("selectFile")}
            </span>
            <span className="text-[11px] text-slate-500 mt-1">
              {selectedFile ? `${(selectedFile.size / 1024).toFixed(1)} KB` : t("dragDropHint")}
            </span>
          </div>

          <DialogFooter className="pt-2">
            <Button
              type="button"
              variant="outline"
              onClick={() => onOpenChange(false)}
              disabled={isUploading}
            >
              İptal
            </Button>

            <Button
              type="submit"
              disabled={!selectedFile || isUploading || isPeriodLocked}
              className="bg-emerald-600 hover:bg-emerald-700 text-white gap-1.5 font-semibold"
            >
              {isUploading ? (
                <>
                  <Loader2 className="w-4 h-4 animate-spin" />
                  {t("uploading")}
                </>
              ) : (
                t("startImport")
              )}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}

export default ImportCsvDialog;
