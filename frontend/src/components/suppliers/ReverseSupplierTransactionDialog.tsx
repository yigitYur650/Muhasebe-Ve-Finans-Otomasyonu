"use client";

import { useState } from "react";
import { useTranslations } from "next-intl";
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription, DialogFooter } from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { formatTL } from "@/lib/decimal";
import { SupplierTransaction } from "@/types/supplier";
import { RotateCcw, AlertTriangle, Loader2 } from "lucide-react";

interface ReverseSupplierTransactionDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  transaction: SupplierTransaction | null;
  onSubmitReversal: (targetTxId: string, reason: string, idempotencyKey: string) => Promise<{ success: boolean; error?: string }>;
}

export function ReverseSupplierTransactionDialog({
  open,
  onOpenChange,
  transaction,
  onSubmitReversal,
}: ReverseSupplierTransactionDialogProps) {
  const t = useTranslations("common");
  const tSuppliers = useTranslations("suppliers");
  const tErr = useTranslations("errors");

  const [reason, setReason] = useState<string>("");
  const [errorMsg, setErrorMsg] = useState<string | null>(null);
  const [isSubmitting, setIsSubmitting] = useState<boolean>(false);

  if (!transaction) return null;

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setErrorMsg(null);

    if (!reason.trim()) {
      setErrorMsg("Lütfen iptal / ters kayıt gerekçesini belirtiniz.");
      return;
    }

    setIsSubmitting(true);
    try {
      const idempotencyKey = crypto.randomUUID();
      const res = await onSubmitReversal(transaction.id, reason.trim(), idempotencyKey);
      if (res.success) {
        setReason("");
        onOpenChange(false);
      } else {
        setErrorMsg(res.error || tErr("generic"));
      }
    } catch (err: any) {
      setErrorMsg(err?.message || tErr("generic"));
    } finally {
      setIsSubmitting(false);
    }
  };

  const isPurchase = transaction.direction === "purchase";

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2 text-rose-700">
            <RotateCcw className="w-5 h-5" />
            Tedarikçi Hareketini Ters Kayıt Yap (İptal Et)
          </DialogTitle>
          <DialogDescription className="text-xs text-slate-500">
            Bu işlem tedarikçi carisine zıt yönlü denkleştirici bir ters kayıt ekler.
          </DialogDescription>
        </DialogHeader>

        {/* Audit Compliance Notice */}
        <div className="p-3 bg-amber-50 border border-amber-200 rounded-lg flex items-start gap-2.5 text-xs text-amber-800">
          <AlertTriangle className="w-4 h-4 text-amber-600 shrink-0 mt-0.5" />
          <span>
            Muhasebe ilkeleri gereği geçmiş işlemler doğrudan silinemez. Orijinal tutarı sıfırlayan ters kayıt oluşturulur.
          </span>
        </div>

        {/* Transaction Target Summary Card */}
        <div className="p-3 bg-slate-100 rounded-lg text-xs space-y-1 text-slate-700 border">
          <div className="flex justify-between">
            <span>Tedarikçi / Firma:</span>
            <strong className="text-slate-900 font-bold">{transaction.supplier_name || "-"}</strong>
          </div>
          {transaction.invoice_no && (
            <div className="flex justify-between">
              <span>Fatura No:</span>
              <span className="font-mono text-slate-800 font-semibold">{transaction.invoice_no}</span>
            </div>
          )}
          {transaction.customer_name && (
            <div className="flex justify-between">
              <span>Müşteri / Açıklama:</span>
              <span className="text-slate-800">{transaction.customer_name}</span>
            </div>
          )}
          <div className="flex justify-between">
            <span>İşlem Yönü:</span>
            <strong className={isPurchase ? "text-rose-600" : "text-emerald-600"}>
              {isPurchase ? "Alınan Mal (Borç Artışı)" : "Geçilen Ödeme (Borç Düşüşü)"}
            </strong>
          </div>
          <div className="flex justify-between">
            <span>Tutar:</span>
            <strong className="text-slate-900 font-extrabold">{formatTL(transaction.amount)}</strong>
          </div>
        </div>

        <form onSubmit={handleSubmit} className="space-y-4">
          <div className="space-y-1.5">
            <label className="text-xs font-bold text-slate-700">
              Ters Kayıt / İptal Gerekçesi <span className="text-rose-600">*</span>
            </label>
            <Input
              value={reason}
              onChange={(e) => setReason(e.target.value)}
              placeholder="Örn: Yanlış fatura tutarı, mükerrer giriş vb."
              className="text-xs"
              autoFocus
            />
          </div>

          {errorMsg && (
            <div className="p-2.5 bg-rose-50 border border-rose-200 rounded text-xs text-rose-700">
              {errorMsg}
            </div>
          )}

          <DialogFooter className="gap-2 sm:gap-0">
            <Button
              type="button"
              variant="outline"
              onClick={() => onOpenChange(false)}
              disabled={isSubmitting}
              className="text-xs"
            >
              {t("cancel")}
            </Button>
            <Button
              type="submit"
              variant="destructive"
              disabled={isSubmitting}
              className="text-xs gap-1.5 font-bold"
            >
              {isSubmitting && <Loader2 className="w-3.5 h-3.5 animate-spin" />}
              Ters Kaydı Onayla
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
