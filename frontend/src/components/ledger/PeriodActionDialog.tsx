"use client";

import { useState } from "react";
import { useTranslations } from "next-intl";
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription, DialogFooter } from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Lock, Calendar, AlertTriangle, Loader2 } from "lucide-react";

interface PeriodActionDialogProps {
  mode: "lock" | "open" | null;
  open: boolean;
  onOpenChange: (open: boolean) => void;
  userRole: string;
  onLockPeriod: (idempotencyKey: string) => Promise<void>;
  onOpenNextPeriod: (label: string, idempotencyKey: string) => Promise<void>;
}

export function PeriodActionDialog({
  mode,
  open,
  onOpenChange,
  userRole,
  onLockPeriod,
  onOpenNextPeriod,
}: PeriodActionDialogProps) {
  const t = useTranslations("common");
  const tPeriod = useTranslations("period");
  const tErr = useTranslations("errors");

  const [nextLabel, setNextLabel] = useState<string>("2026-09");
  const [errorMsg, setErrorMsg] = useState<string | null>(null);
  const [isSubmitting, setIsSubmitting] = useState<boolean>(false);

  const canManagePeriod = userRole === "admin" || userRole === "muhasebeci";

  if (!mode) return null;

  const handleLockSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setErrorMsg(null);

    if (!canManagePeriod) {
      setErrorMsg(tErr("UNAUTHORIZED"));
      return;
    }

    setIsSubmitting(true);
    try {
      const idempotencyKey = crypto.randomUUID();
      await onLockPeriod(idempotencyKey);
      onOpenChange(false);
    } catch (err: any) {
      setErrorMsg(err?.message || tErr("generic"));
    } finally {
      setIsSubmitting(false);
    }
  };

  const handleOpenSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setErrorMsg(null);

    if (!nextLabel.trim()) {
      setErrorMsg("Dönem etiketi boş bırakılamaz.");
      return;
    }

    setIsSubmitting(true);
    try {
      const idempotencyKey = crypto.randomUUID();
      await onOpenNextPeriod(nextLabel.trim(), idempotencyKey);
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
        {mode === "lock" ? (
          <>
            <DialogHeader>
              <DialogTitle className="flex items-center gap-2 text-rose-700">
                <Lock className="w-5 h-5" />
                {tPeriod("lockConfirmTitle")}
              </DialogTitle>
              <DialogDescription className="text-xs text-slate-500">
                {tPeriod("lockConfirmDescription")}
              </DialogDescription>
            </DialogHeader>

            <form onSubmit={handleLockSubmit} className="space-y-4 py-2">
              {!canManagePeriod && (
                <div className="p-3 bg-rose-50 border border-rose-200 rounded-lg text-xs text-rose-700 flex items-center gap-2">
                  <AlertTriangle className="w-4 h-4 shrink-0 text-rose-600" />
                  <span>{tErr("UNAUTHORIZED")}</span>
                </div>
              )}

              {errorMsg && (
                <div className="p-3 text-xs text-rose-700 bg-rose-50 rounded-md border border-rose-200">
                  {errorMsg}
                </div>
              )}

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
                  disabled={isSubmitting || !canManagePeriod}
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
          </>
        ) : (
          <>
            <DialogHeader>
              <DialogTitle className="flex items-center gap-2 text-slate-900">
                <Calendar className="w-5 h-5 text-primary" />
                {tPeriod("openTitle")}
              </DialogTitle>
              <DialogDescription className="text-xs text-slate-500">
                {tPeriod("openDescription")}
              </DialogDescription>
            </DialogHeader>

            <form onSubmit={handleOpenSubmit} className="space-y-4 py-2">
              {errorMsg && (
                <div className="p-3 text-xs text-rose-700 bg-rose-50 rounded-md border border-rose-200">
                  {errorMsg}
                </div>
              )}

              <div className="space-y-1.5">
                <label className="text-xs font-semibold text-slate-700">{tPeriod("periodLabel")}</label>
                <Input
                  type="text"
                  placeholder="2026-09"
                  value={nextLabel}
                  onChange={(e) => setNextLabel(e.target.value)}
                  className="text-xs font-mono"
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
                  disabled={isSubmitting}
                  className="bg-primary text-primary-foreground gap-1.5"
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
          </>
        )}
      </DialogContent>
    </Dialog>
  );
}
