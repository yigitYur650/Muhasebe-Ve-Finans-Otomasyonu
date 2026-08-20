"use client";

import { useState } from "react";
import { useTranslations } from "next-intl";
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription, DialogFooter } from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { toDecimal } from "@/lib/decimal";
import { PlusCircle, Loader2 } from "lucide-react";

interface CreateTransactionDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  isPeriodLocked: boolean;
  onSubmitTransaction: (data: {
    direction: "in" | "out";
    channel: string;
    amount: string;
    description: string;
    idempotencyKey: string;
  }) => Promise<void>;
}

export function CreateTransactionDialog({
  open,
  onOpenChange,
  isPeriodLocked,
  onSubmitTransaction,
}: CreateTransactionDialogProps) {
  const t = useTranslations("common");
  const tTx = useTranslations("transaction");
  const tErr = useTranslations("errors");

  const [direction, setDirection] = useState<"in" | "out">("in");
  const [channel, setChannel] = useState<string>("eft");
  const [amountStr, setAmountStr] = useState<string>("");
  const [description, setDescription] = useState<string>("");
  const [errorMsg, setErrorMsg] = useState<string | null>(null);
  const [isSubmitting, setIsSubmitting] = useState<boolean>(false);

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

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setErrorMsg(null);

    if (isPeriodLocked) {
      setErrorMsg(tErr("PERIOD_LOCKED"));
      return;
    }

    const dAmount = toDecimal(amountStr);
    if (dAmount.lte(0)) {
      setErrorMsg(tErr("INVALID_AMOUNT"));
      return;
    }

    setIsSubmitting(true);
    try {
      // Client-side UUID generation for Idempotency-Key header
      const idempotencyKey = crypto.randomUUID();

      await onSubmitTransaction({
        direction,
        channel,
        amount: dAmount.toFixed(2),
        description,
        idempotencyKey,
      });

      // Reset form
      setAmountStr("");
      setDescription("");
      onOpenChange(false);
    } catch (err: any) {
      setErrorMsg(err?.message || tErr("generic"));
    } finally {
      setIsSubmitting(false);
    }
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2 text-slate-900">
            <PlusCircle className="w-5 h-5 text-emerald-600" />
            {tTx("createTitle")}
          </DialogTitle>
          <DialogDescription className="text-xs text-slate-500">
            {tTx("createDescription")}
          </DialogDescription>
        </DialogHeader>

        <form onSubmit={handleSubmit} className="space-y-4 py-2">
          {errorMsg && (
            <div className="p-3 text-xs text-rose-700 bg-rose-50 rounded-md border border-rose-200">
              {errorMsg}
            </div>
          )}

          {/* Direction Selection */}
          <div className="space-y-1.5">
            <label className="text-xs font-semibold text-slate-700">{tTx("direction")}</label>
            <div className="grid grid-cols-2 gap-2">
              <Button
                type="button"
                variant={direction === "in" ? "default" : "outline"}
                className={direction === "in" ? "bg-emerald-600 hover:bg-emerald-700 text-white" : ""}
                onClick={() => setDirection("in")}
              >
                {tTx("directionIn")}
              </Button>
              <Button
                type="button"
                variant={direction === "out" ? "default" : "outline"}
                className={direction === "out" ? "bg-rose-600 hover:bg-rose-700 text-white" : ""}
                onClick={() => setDirection("out")}
              >
                {tTx("directionOut")}
              </Button>
            </div>
          </div>

          {/* Channel Selection */}
          <div className="space-y-1.5">
            <label className="text-xs font-semibold text-slate-700">{tTx("channel")}</label>
            <Select value={channel} onValueChange={setChannel}>
              <SelectTrigger className="text-xs">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                {channelsList.map((ch) => (
                  <SelectItem key={ch} value={ch}>
                    {tTx(`channels.${ch}`)}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>

          {/* Amount Input */}
          <div className="space-y-1.5">
            <label className="text-xs font-semibold text-slate-700">{tTx("amount")}</label>
            <Input
              type="number"
              step="0.01"
              placeholder="0.00"
              value={amountStr}
              onChange={(e) => setAmountStr(e.target.value)}
              className="font-extrabold text-base text-slate-900"
              required
            />
          </div>

          {/* Description Input */}
          <div className="space-y-1.5">
            <label className="text-xs font-semibold text-slate-700">{tTx("description")}</label>
            <Input
              type="text"
              placeholder="ör. Kira ödemesi, EFT girişi..."
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              className="text-xs"
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
              disabled={isSubmitting || isPeriodLocked}
              className="bg-emerald-600 hover:bg-emerald-700 text-white gap-1.5"
            >
              {isSubmitting ? (
                <>
                  <Loader2 className="w-4 h-4 animate-spin" />
                  {t("loading")}
                </>
              ) : (
                t("save")
              )}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
