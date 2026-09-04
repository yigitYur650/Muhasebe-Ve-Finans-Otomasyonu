"use client";

import { useState } from "react";
import { useTranslations } from "next-intl";
import { useRouter, usePathname } from "next/navigation";
import { createClient } from "@/lib/supabase/client";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Building2, Globe, ShieldCheck, LogOut, Flame, KeyRound } from "lucide-react";
import { ChangePasswordDialog } from "@/components/auth/ChangePasswordDialog";

interface HeaderProps {
  tenantName?: string;
  userRole?: string;
  locale: string;
}

export function Header({ tenantName = "Öncü Otogaz Ana Şube", userRole = "admin", locale }: HeaderProps) {
  const t = useTranslations("common");
  const tAuth = useTranslations("auth");
  const router = useRouter();
  const pathname = usePathname();

  const [isChangePasswordOpen, setIsChangePasswordOpen] = useState(false);

  const toggleLanguage = () => {
    const nextLocale = locale === "tr" ? "en" : "tr";
    const newPath = pathname.replace(`/${locale}`, `/${nextLocale}`);
    router.push(newPath || `/${nextLocale}`);
  };

  const handleLogout = async () => {
    try {
      // Clear session cookie for Route Guard
      document.cookie = "defter_session=; path=/; expires=Thu, 01 Jan 1970 00:00:00 GMT";
      const supabase = createClient();
      await supabase.auth.signOut();
    } catch {
      // Ignore errors
    } finally {
      router.push(`/${locale}/login`);
    }
  };

  const getRoleLabel = (role: string) => {
    switch (role) {
      case "admin":
        return tAuth("roleAdmin");
      case "muhasebeci":
        return tAuth("roleMuhasebeci");
      default:
        return tAuth("roleStandart");
    }
  };

  return (
    <>
      <header className="sticky top-0 z-40 w-full border-b border-zinc-800 bg-zinc-950 text-white backdrop-blur shadow-md">
        <div className="container flex h-16 items-center justify-between px-4 mx-auto max-w-7xl">
          <div className="flex items-center gap-3">
            <div className="flex items-center justify-center w-10 h-10 rounded-xl bg-amber-500 text-zinc-950 font-black text-lg shadow-lg shadow-amber-500/20 border border-amber-400">
              <Flame className="w-6 h-6 fill-zinc-950 text-zinc-950" />
            </div>
            <div>
              <h1 className="text-base md:text-lg font-black tracking-tight text-amber-400">{t("appName")}</h1>
              <div className="flex items-center gap-1.5 text-xs text-zinc-400">
                <Building2 className="w-3.5 h-3.5 text-amber-400/80" />
                <span>{tenantName}</span>
              </div>
            </div>
          </div>

          <div className="flex items-center gap-2 md:gap-3">
            <Badge variant="outline" className="gap-1 border-amber-500/40 bg-amber-950/40 text-amber-300 font-medium">
              <ShieldCheck className="w-3.5 h-3.5 text-amber-400" />
              {getRoleLabel(userRole)}
            </Badge>

            <Button
              variant="outline"
              size="sm"
              onClick={() => setIsChangePasswordOpen(true)}
              className="gap-1.5 h-8 text-xs font-semibold border-zinc-800 bg-zinc-900 text-zinc-300 hover:text-amber-400 hover:border-amber-500/50"
            >
              <KeyRound className="w-3.5 h-3.5 text-amber-400" />
              <span className="hidden md:inline">{tAuth("changePassword")}</span>
            </Button>

            <Button
              variant="ghost"
              size="sm"
              onClick={toggleLanguage}
              className="gap-1.5 text-xs font-semibold text-zinc-300 hover:text-white hover:bg-zinc-800"
            >
              <Globe className="w-4 h-4 text-amber-400" />
              {locale === "tr" ? t("english") : t("turkish")}
            </Button>

            <Button
              variant="outline"
              size="sm"
              onClick={handleLogout}
              className="gap-1.5 h-8 text-xs font-semibold border-zinc-800 bg-zinc-900 hover:bg-rose-950/60 text-zinc-300 hover:text-rose-300 hover:border-rose-800 transition-colors"
            >
              <LogOut className="w-3.5 h-3.5 text-rose-400" />
              <span className="hidden sm:inline">{tAuth("logoutAction")}</span>
            </Button>
          </div>
        </div>
      </header>

      <ChangePasswordDialog open={isChangePasswordOpen} onOpenChange={setIsChangePasswordOpen} />
    </>
  );
}

export default Header;
