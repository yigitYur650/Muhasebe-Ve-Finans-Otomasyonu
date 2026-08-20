"use client";

import { useState } from "react";
import { useTranslations } from "next-intl";
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription, DialogFooter } from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { formatTL } from "@/lib/decimal";
import { TransactionItem } from "./TransactionTable";
import { RotateCcw, AlertTriangle, Loader2 } from "lucide-react";

interface ReverseTransactionDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  transaction: TransactionItem | null;
  onSubmitReversal: (targetTxId: string, reason: string, idempotencyKey: string) => Promise<void>;
}

export function ReverseTransactionDialog({
  open,
  onOpenChange,
  transaction,
  onSubmitReversal,
}: ReverseTransactionDialogProps) {
  const t = useTranslations("common");
  const tTx = useTranslations("transaction");
  const tErr = useTranslations("errors");

  const [reason, setReason] = useState<string>("");
  const [errorMsg, setErrorMsg] = useState<string | null>(null);
  const [isSubmitting, setIsSubmitting] = useState<boolean>(false);

  if (!transaction) return null;

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setErrorMsg(null);

    if (!reason.trim()) {
      setErrorMsg("Lütfen iptal gerekçesini belirtiniz.");
      return;
    }

    setIsSubmitting(true);
    try {
      const idempotencyKey = crypto.randomUUID();
      await onSubmitReversal(transaction.id, reason.trim(), idempotencyKey);
      setReason("");
      onOpenChange(false);
    } catch (err: any) {
      setErrorMsg(err?.message || tErr("generic"));
    } finally {
      setIsSubmitting(false);
    }
  };

  const isIn = transaction.direction === "in";

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2 text-rose-700">
            <RotateCcw className="w-5 h-5" />
            {tTx("reverseConfirmTitle")}
          </DialogTitle>
          <DialogDescription className="text-xs text-slate-500">
            {tTx("reverseAction")}
          </DialogDescription>
        </DialogHeader>

        {/* Audit Compliance Notice */}
        <div className="p-3 bg-amber-50 border border-amber-200 rounded-lg flex items-start gap-2.5 text-xs text-amber-800">
          <AlertTriangle className="w-4 h-4 text-amber-600 shrink-0 mt-0.5" />
          <span>{tTx("reverseAuditNotice")}</span>
        </div>

        {/* Transaction Target Summary Card */}
        <div className="p-3 bg-slate-100 rounded-lg text-xs space-y-1 text-slate-700 border">
          <div className="flex justify-between">
            <span>{tTx("channel")}:</span>
            <strong className="text-slate-900">{tTx(`channels.${transaction.channel}`)}</strong>
          </div>
          <div className="flex justify-between">
            <span>{tTx("direction")}:</span>
            <strong className={isIn ? "text-emerald-600" : "text-rose-600"}>
              {isIn ? tTx("directionIn") : tTx("directionOut")}
            </strong>
          </div>
          <div className="flex justify-between">
            <span>{tTx("amount")}:</span>
            <strong className="text-slate-900 font-extrabold">{formatTL(transaction.amount)}</strong>
          </div>
          {transaction.description && (
            <div className="flex justify-between pt-1 border-t border-slate-200">
              <span>{tTx("description")}:</span>
              <span className="text-slate-600 max-w-[200px] truncate">{transaction.description}</span>
            </div>
          )}
        </div>

        <form onSubmit={handleSubmit} className="space-y-4 py-1">
          {errorMsg && (
            <div className="p-3 text-xs text-rose-700 bg-rose-50 rounded-md border border-rose-200">
              {errorMsg}
            </div>
          )}

          <div className="space-y-1.5">
            <label className="text-xs font-semibold text-slate-700">{tTx("reverseReason")}</label>
            <Input
              type="text"
              placeholder="ör. Hatalı tutar girildi, müşteri iade talebi..."
              value={reason}
              onChange={(e) => setReason(e.target.value)}
              className="text-xs"
              required
            />
          </div>

          <DialogFooter className="pt-2">
            <Button
              type="button"
              variant="outline"
              onClick={() => onOpenChange(false)}
              disabled={isSubmitting}
            >
              {t("cancel")}
            </Button>

            <Button
              type="submit"
              variant="destructive"
              disabled={isSubmitting}
              className="gap-1.5"
            >
              {isSubmitting ? (
                <>
                  <Loader2 className="w-4 h-4 animate-spin" />
                  {t("loading")}
                </>
              ) : (
                t("confirm")
              )}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
