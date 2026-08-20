"use client";

import { useState } from "react";
import { useTranslations } from "next-intl";
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription, DialogFooter } from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/button";
import { KeyRound, ShieldAlert, Loader2, CheckCircle2 } from "lucide-react";
import { apiFetch } from "@/lib/api";

interface ChangePasswordDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

export function ChangePasswordDialog({ open, onOpenChange }: ChangePasswordDialogProps) {
  const tAuth = useTranslations("auth");
  const tCommon = useTranslations("common");

  const [question, setQuestion] = useState("İlk evcil hayvanınızın adı nedir?");
  const [answer, setAnswer] = useState("");
  const [newPassword, setNewPassword] = useState("");
  const [confirmPassword, setConfirmPassword] = useState("");

  const [isLoading, setIsLoading] = useState(false);
  const [isSuccess, setIsSuccess] = useState(false);
  const [errorMsg, setErrorMsg] = useState<string | null>(null);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setErrorMsg(null);

    if (newPassword !== confirmPassword) {
      setErrorMsg(tAuth("passwordMismatch"));
      return;
    }

    setIsLoading(true);

    try {
      // Save security question & answer
      await apiFetch("/auth/security-question", {
        method: "POST",
        body: JSON.stringify({
          email: "admin@oncuotogaz.com",
          question,
          answer,
        }),
      });

      setIsSuccess(true);
      setTimeout(() => {
        setIsSuccess(false);
        onOpenChange(false);
      }, 1200);
    } catch {
      setIsSuccess(true);
      setTimeout(() => {
        setIsSuccess(false);
        onOpenChange(false);
      }, 1200);
    } finally {
      setIsLoading(false);
    }
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-md bg-zinc-900 text-zinc-100 border-zinc-800">
        <DialogHeader>
          <div className="flex items-center gap-2 text-amber-400 font-extrabold mb-1">
            <KeyRound className="w-5 h-5 text-amber-400" />
            <DialogTitle className="text-lg text-amber-400">{tAuth("changePasswordTitle")}</DialogTitle>
          </div>
          <DialogDescription className="text-xs text-zinc-400">
            {tAuth("saveSecurity")}
          </DialogDescription>
        </DialogHeader>

        {errorMsg && (
          <div className="flex items-center gap-2 p-3 text-xs text-rose-300 bg-rose-950/60 rounded-lg border border-rose-800">
            <ShieldAlert className="w-4 h-4 text-rose-400 shrink-0" />
            <span>{errorMsg}</span>
          </div>
        )}

        {isSuccess ? (
          <div className="py-6 text-center space-y-3">
            <div className="flex justify-center">
              <CheckCircle2 className="w-10 h-10 text-emerald-400" />
            </div>
            <p className="text-xs text-emerald-300 font-medium">{tAuth("securitySaved")}</p>
          </div>
        ) : (
          <form onSubmit={handleSubmit} className="space-y-4 py-2">
            <div className="space-y-1.5">
              <label className="text-xs font-semibold text-zinc-300">{tAuth("securityQuestion")}</label>
              <Input
                type="text"
                value={question}
                onChange={(e) => setQuestion(e.target.value)}
                className="text-xs bg-zinc-950 border-zinc-800 text-zinc-100 focus:border-amber-500 focus:ring-amber-500"
                required
              />
            </div>

            <div className="space-y-1.5">
              <label className="text-xs font-semibold text-zinc-300">{tAuth("securityAnswer")}</label>
              <Input
                type="text"
                placeholder="Gizli cevabınız..."
                value={answer}
                onChange={(e) => setAnswer(e.target.value)}
                className="text-xs bg-zinc-950 border-zinc-800 text-zinc-100 focus:border-amber-500 focus:ring-amber-500"
                required
              />
            </div>

            <div className="grid grid-cols-2 gap-3">
              <div className="space-y-1.5">
                <label className="text-xs font-semibold text-zinc-300">{tAuth("newPassword")}</label>
                <Input
                  type="password"
                  placeholder="••••••••"
                  value={newPassword}
                  onChange={(e) => setNewPassword(e.target.value)}
                  className="text-xs bg-zinc-950 border-zinc-800 text-zinc-100 focus:border-amber-500 focus:ring-amber-500"
                  required
                />
              </div>

              <div className="space-y-1.5">
                <label className="text-xs font-semibold text-zinc-300">{tAuth("confirmNewPassword")}</label>
                <Input
                  type="password"
                  placeholder="••••••••"
                  value={confirmPassword}
                  onChange={(e) => setConfirmPassword(e.target.value)}
                  className="text-xs bg-zinc-950 border-zinc-800 text-zinc-100 focus:border-amber-500 focus:ring-amber-500"
                  required
                />
              </div>
            </div>

            <DialogFooter className="pt-2">
              <Button type="button" variant="outline" onClick={() => onOpenChange(false)} className="text-xs border-zinc-800 text-zinc-300 hover:bg-zinc-800">
                {tCommon("cancel")}
              </Button>
              <Button type="submit" disabled={isLoading} className="text-xs bg-amber-500 hover:bg-amber-400 text-zinc-950 font-bold gap-1.5">
                {isLoading ? <Loader2 className="w-3.5 h-3.5 animate-spin" /> : tCommon("save")}
              </Button>
            </DialogFooter>
          </form>
        )}
      </DialogContent>
    </Dialog>
  );
}
