"use client";

import { useTranslations } from "next-intl";
import { useRouter, usePathname } from "next/navigation";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Building2, Globe, ShieldCheck } from "lucide-react";

interface HeaderProps {
  tenantName?: string;
  userRole?: string;
  locale: string;
}

export function Header({ tenantName = "Ana Şube", userRole = "admin", locale }: HeaderProps) {
  const t = useTranslations("common");
  const tAuth = useTranslations("auth");
  const router = useRouter();
  const pathname = usePathname();

  const toggleLanguage = () => {
    const nextLocale = locale === "tr" ? "en" : "tr";
    const newPath = pathname.replace(`/${locale}`, `/${nextLocale}`);
    router.push(newPath || `/${nextLocale}`);
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
    <header className="sticky top-0 z-40 w-full border-b bg-white/95 backdrop-blur shadow-sm">
      <div className="container flex h-16 items-center justify-between px-4 mx-auto">
        <div className="flex items-center gap-3">
          <div className="flex items-center justify-center w-10 h-10 rounded-lg bg-primary text-white font-bold text-lg shadow-md">
            K
          </div>
          <div>
            <h1 className="text-lg font-bold tracking-tight text-slate-900">{t("appName")}</h1>
            <div className="flex items-center gap-2 text-xs text-slate-500">
              <Building2 className="w-3.5 h-3.5 text-slate-400" />
              <span>{tenantName}</span>
            </div>
          </div>
        </div>

        <div className="flex items-center gap-3">
          <Badge variant="outline" className="gap-1 border-slate-300 bg-slate-50 text-slate-700">
            <ShieldCheck className="w-3.5 h-3.5 text-primary" />
            {getRoleLabel(userRole)}
          </Badge>

          <Button
            variant="ghost"
            size="sm"
            onClick={toggleLanguage}
            className="gap-1.5 text-xs font-semibold text-slate-600 hover:bg-slate-100"
          >
            <Globe className="w-4 h-4 text-primary" />
            {locale === "tr" ? t("english") : t("turkish")}
          </Button>
        </div>
      </div>
    </header>
  );
}
