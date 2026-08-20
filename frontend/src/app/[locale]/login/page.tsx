"use client";

import { use, useState } from "react";
import { useTranslations } from "next-intl";
import { useRouter } from "next/navigation";
import { createClient } from "@/lib/supabase/client";
import { Card, CardHeader, CardTitle, CardDescription, CardContent, CardFooter } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/button";
import { Lock, Mail, Loader2, ShieldCheck } from "lucide-react";

export default function LoginPage({ params }: { params: Promise<{ locale: string }> }) {
  const { locale } = use(params);
  const tAuth = useTranslations("auth");
  const tCommon = useTranslations("common");
  const router = useRouter();

  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [errorMsg, setErrorMsg] = useState<string | null>(null);
  const [successMsg, setSuccessMsg] = useState<string | null>(null);
  const [isLoading, setIsLoading] = useState(false);

  const handleLogin = async (e: React.FormEvent) => {
    e.preventDefault();
    setErrorMsg(null);
    setSuccessMsg(null);
    setIsLoading(true);

    try {
      const supabase = createClient();
      const { error } = await supabase.auth.signInWithPassword({
        email,
        password,
      });

      if (error) {
        setErrorMsg(tAuth("invalidCredentials"));
      } else {
        setSuccessMsg(tAuth("loginSuccess"));
        setTimeout(() => {
          router.push(`/${locale}`);
        }, 1000);
      }
    } catch {
      setErrorMsg(tAuth("invalidCredentials"));
    } finally {
      setIsLoading(false);
    }
  };

  return (
    <div className="min-h-screen flex items-center justify-center bg-slate-100 px-4 py-12">
      <Card className="w-full max-w-md border-slate-200 shadow-xl bg-white">
        <CardHeader className="space-y-2 text-center pb-6 border-b border-slate-100">
          <div className="flex justify-center mb-2">
            <div className="flex items-center justify-center w-12 h-12 rounded-xl bg-primary text-white font-bold text-xl shadow-md">
              <ShieldCheck className="w-6 h-6" />
            </div>
          </div>
          <CardTitle className="text-xl font-extrabold text-slate-900">
            {tAuth("loginTitle")}
          </CardTitle>
          <CardDescription className="text-xs text-slate-500 max-w-xs mx-auto">
            {tAuth("loginDescription")}
          </CardDescription>
        </CardHeader>

        <form onSubmit={handleLogin}>
          <CardContent className="space-y-4 pt-6">
            {errorMsg && (
              <div className="p-3 text-xs text-rose-700 bg-rose-50 rounded-lg border border-rose-200">
                {errorMsg}
              </div>
            )}

            {successMsg && (
              <div className="p-3 text-xs text-emerald-700 bg-emerald-50 rounded-lg border border-emerald-200">
                {successMsg}
              </div>
            )}

            <div className="space-y-1.5">
              <label className="text-xs font-semibold text-slate-700 flex items-center gap-1.5">
                <Mail className="w-3.5 h-3.5 text-slate-400" />
                {tAuth("email")}
              </label>
              <Input
                type="email"
                placeholder="ornek@firma.com"
                value={email}
                onChange={(e) => setEmail(e.target.value)}
                className="text-xs"
                required
              />
            </div>

            <div className="space-y-1.5">
              <label className="text-xs font-semibold text-slate-700 flex items-center gap-1.5">
                <Lock className="w-3.5 h-3.5 text-slate-400" />
                {tAuth("password")}
              </label>
              <Input
                type="password"
                placeholder="••••••••"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                className="text-xs"
                required
              />
            </div>
          </CardContent>

          <CardFooter className="pt-2">
            <Button
              type="submit"
              disabled={isLoading}
              className="w-full bg-primary hover:bg-primary/90 text-white font-semibold gap-2"
            >
              {isLoading ? (
                <>
                  <Loader2 className="w-4 h-4 animate-spin" />
                  {tCommon("loading")}
                </>
              ) : (
                tAuth("loginAction")
              )}
            </Button>
          </CardFooter>
        </form>
      </Card>
    </div>
  );
}
