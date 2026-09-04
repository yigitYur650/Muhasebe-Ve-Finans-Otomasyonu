"use client";

import { use, useState } from "react";
import { useTranslations } from "next-intl";
import { useRouter } from "next/navigation";
import { createClient } from "@/lib/supabase/client";
import { Card, CardHeader, CardTitle, CardDescription, CardContent, CardFooter } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/button";
import { Lock, Mail, Loader2, Flame, ShieldAlert, KeyRound } from "lucide-react";
import { ForgotPasswordDialog } from "@/components/auth/ForgotPasswordDialog";

export default function LoginPage({ params }: { params: Promise<{ locale: string }> }) {
  const { locale } = use(params);
  const tAuth = useTranslations("auth");
  const tCommon = useTranslations("common");
  const router = useRouter();

  const envAdminEmail = process.env.NEXT_PUBLIC_ADMIN_EMAIL || "admin@oncuotogaz.com";
  const envAdminPassword = process.env.NEXT_PUBLIC_ADMIN_PASSWORD || "oncu123456";

  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [errorMsg, setErrorMsg] = useState<string | null>(null);
  const [successMsg, setSuccessMsg] = useState<string | null>(null);
  const [isLoading, setIsLoading] = useState(false);
  const [isForgotOpen, setIsForgotOpen] = useState(false);

  const handleLogin = async (e: React.FormEvent) => {
    e.preventDefault();
    setErrorMsg(null);
    setSuccessMsg(null);
    setIsLoading(true);

    try {
      // 1. Env credentials match check
      const isEnvAdmin = email.trim() === envAdminEmail && password === envAdminPassword;

      if (isEnvAdmin) {
        document.cookie = "defter_session=active; path=/; max-age=86400";
        setSuccessMsg(tAuth("loginSuccess"));
        setTimeout(() => {
          router.push(`/${locale}`);
        }, 800);
        return;
      }

      // 2. Try Supabase Auth
      const supabase = createClient();
      const { error } = await supabase.auth.signInWithPassword({
        email,
        password,
      });

      if (!error) {
        document.cookie = "defter_session=active; path=/; max-age=86400";
        setSuccessMsg(tAuth("loginSuccess"));
        setTimeout(() => {
          router.push(`/${locale}`);
        }, 800);
      } else {
        setErrorMsg(tAuth("invalidCredentials"));
        document.cookie = "defter_session=; path=/; expires=Thu, 01 Jan 1970 00:00:00 GMT";
      }
    } catch {
      setErrorMsg(tAuth("invalidCredentials"));
    } finally {
      setIsLoading(false);
    }
  };

  return (
    <div className="min-h-screen flex items-center justify-center bg-zinc-950 px-4 py-12 relative overflow-hidden">
      {/* Corporate Yellow Accent Ambient Glow */}
      <div className="absolute top-1/4 left-1/2 -translate-x-1/2 -translate-y-1/2 w-96 h-96 bg-amber-500/10 rounded-full blur-3xl pointer-events-none" />

      <Card className="w-full max-w-md border-zinc-800 shadow-2xl bg-zinc-900 text-zinc-100 z-10 relative">
        <CardHeader className="space-y-3 text-center pb-6 border-b border-zinc-800">
          <div className="flex justify-center mb-1">
            <div className="flex items-center justify-center w-14 h-14 rounded-2xl bg-amber-500 text-zinc-950 font-black text-2xl shadow-lg shadow-amber-500/20 border border-amber-400">
              <Flame className="w-8 h-8 fill-zinc-950" />
            </div>
          </div>
          <CardTitle className="text-xl font-black tracking-tight text-amber-400">
            {tAuth("loginTitle")}
          </CardTitle>
          <CardDescription className="text-xs text-zinc-400 max-w-xs mx-auto">
            {tAuth("loginDescription")}
          </CardDescription>
        </CardHeader>

        <form onSubmit={handleLogin}>
          <CardContent className="space-y-4 pt-6">
            {errorMsg && (
              <div className="flex items-center gap-2 p-3 text-xs text-rose-300 bg-rose-950/60 rounded-lg border border-rose-800">
                <ShieldAlert className="w-4 h-4 text-rose-400 shrink-0" />
                <span>{errorMsg}</span>
              </div>
            )}

            {successMsg && (
              <div className="p-3 text-xs text-amber-300 bg-amber-950/60 rounded-lg border border-amber-800 font-medium">
                {successMsg}
              </div>
            )}

            <div className="space-y-1.5">
              <label className="text-xs font-semibold text-zinc-300 flex items-center gap-1.5">
                <Mail className="w-3.5 h-3.5 text-amber-400" />
                {tAuth("email")}
              </label>
              <Input
                type="email"
                placeholder="eposta@sirket.com"
                value={email}
                onChange={(e) => setEmail(e.target.value)}
                className="text-xs bg-zinc-950 border-zinc-800 text-zinc-100 placeholder:text-zinc-600 focus:border-amber-500 focus:ring-amber-500"
                required
              />
            </div>

            <div className="space-y-1.5">
              <div className="flex items-center justify-between">
                <label className="text-xs font-semibold text-zinc-300 flex items-center gap-1.5">
                  <Lock className="w-3.5 h-3.5 text-amber-400" />
                  {tAuth("password")}
                </label>
                <button
                  type="button"
                  onClick={() => setIsForgotOpen(true)}
                  className="text-[11px] font-medium text-amber-400 hover:text-amber-300 hover:underline flex items-center gap-1"
                >
                  <KeyRound className="w-3 h-3" />
                  {tAuth("forgotPassword")}
                </button>
              </div>
              <Input
                type="password"
                placeholder="••••••••"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                className="text-xs bg-zinc-950 border-zinc-800 text-zinc-100 placeholder:text-zinc-600 focus:border-amber-500 focus:ring-amber-500"
                required
              />
            </div>
          </CardContent>

          <CardFooter className="pt-2 pb-6 flex-col space-y-3">
            <Button
              type="submit"
              disabled={isLoading}
              className="w-full bg-amber-500 hover:bg-amber-400 text-zinc-950 font-extrabold gap-2 h-10 shadow-lg shadow-amber-500/20"
            >
              {isLoading ? (
                <>
                  <Loader2 className="w-4 h-4 animate-spin text-zinc-950" />
                  {tCommon("loading")}
                </>
              ) : (
                tAuth("loginAction")
              )}
            </Button>
          </CardFooter>
        </form>
      </Card>

      <ForgotPasswordDialog open={isForgotOpen} onOpenChange={setIsForgotOpen} />
    </div>
  );
}
