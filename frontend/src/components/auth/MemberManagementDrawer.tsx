"use client";

import { useState, useEffect } from "react";
import { useTranslations } from "next-intl";
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription } from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Badge } from "@/components/ui/badge";
import { createClient } from "@/lib/supabase/client";
import { apiFetch } from "@/lib/api";
import { Users, UserPlus, Shield, ShieldCheck, Mail, Lock, Loader2, CheckCircle2, AlertCircle } from "lucide-react";

interface Member {
  id: string;
  tenant_id: string;
  user_id: string;
  role: "admin" | "muhasebeci" | "standart";
  created_at?: string;
  email?: string;
}

interface MemberManagementDrawerProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

export function MemberManagementDrawer({ open, onOpenChange }: MemberManagementDrawerProps) {
  const tAuth = useTranslations("auth");
  const tCommon = useTranslations("common");

  const [activeTab, setActiveTab] = useState<"list" | "add">("list");
  const [members, setMembers] = useState<Member[]>([]);
  const [isLoading, setIsLoading] = useState(false);

  // Form states
  const [newEmail, setNewEmail] = useState("");
  const [newPassword, setNewPassword] = useState("");
  const [newRole, setNewRole] = useState<"admin" | "muhasebeci" | "standart">("muhasebeci");
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [errorMsg, setErrorMsg] = useState<string | null>(null);
  const [successMsg, setSuccessMsg] = useState<string | null>(null);

  const fetchMembers = async () => {
    setIsLoading(true);
    try {
      const res = await apiFetch<Member[]>("/tenants/members");
      if (res.success && Array.isArray(res.data)) {
        setMembers(res.data);
      }
    } catch (err) {
      console.error("Üye listesi çekilemedi:", err);
    } finally {
      setIsLoading(false);
    }
  };

  useEffect(() => {
    if (open) {
      fetchMembers();
      setErrorMsg(null);
      setSuccessMsg(null);
    }
  }, [open]);

  const handleAddMember = async (e: React.FormEvent) => {
    e.preventDefault();
    setErrorMsg(null);
    setSuccessMsg(null);
    setIsSubmitting(true);

    try {
      const supabase = createClient();

      // 1. Create account via Supabase Auth
      const { data: authData, error: authError } = await supabase.auth.signUp({
        email: newEmail.trim(),
        password: newPassword,
      });

      if (authError) {
        setErrorMsg(authError.message);
        setIsSubmitting(false);
        return;
      }

      const createdUserId = authData?.user?.id;
      if (!createdUserId) {
        setErrorMsg("Kullanıcı kimliği oluşturulamadı.");
        setIsSubmitting(false);
        return;
      }

      // 2. Add member to current tenant via backend API
      const res = await apiFetch("/tenants/members", {
        method: "POST",
        body: JSON.stringify({
          user_id: createdUserId,
          role: newRole,
        }),
      });

      if (res.success) {
        setSuccessMsg(`${newEmail} kullanıcısı ${newRole} rolü ile şirkete başarıyla eklendi.`);
        setNewEmail("");
        setNewPassword("");
        fetchMembers();
        setTimeout(() => {
          setActiveTab("list");
          setSuccessMsg(null);
        }, 1500);
      } else {
        setErrorMsg(res.error?.message || "Üye şirkete bağlanamadı.");
      }
    } catch (err: any) {
      setErrorMsg(err.message || "Beklenmedik bir hata oluştu.");
    } finally {
      setIsSubmitting(false);
    }
  };

  const getRoleBadge = (role: string) => {
    switch (role) {
      case "admin":
        return <Badge className="bg-amber-500/20 text-amber-300 border-amber-500/40">Yönetici</Badge>;
      case "muhasebeci":
        return <Badge className="bg-sky-500/20 text-sky-300 border-sky-500/40">Muhasebeci</Badge>;
      default:
        return <Badge className="bg-zinc-800 text-zinc-300 border-zinc-700">Standart</Badge>;
    }
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-lg bg-zinc-950 border-zinc-800 text-zinc-100 shadow-2xl p-0 overflow-hidden">
        <DialogHeader className="p-6 pb-4 border-b border-zinc-800 bg-zinc-900/50">
          <div className="flex items-center gap-3">
            <div className="flex items-center justify-center w-10 h-10 rounded-xl bg-amber-500/10 text-amber-400 border border-amber-500/20">
              <Users className="w-5 h-5" />
            </div>
            <div>
              <DialogTitle className="text-base font-bold text-amber-400">
                Şirket Personel ve Üye Yönetimi
              </DialogTitle>
              <DialogDescription className="text-xs text-zinc-400">
                Şirket içi yetkili kullanıcıları görüntüleyin veya yeni personel kaydedin.
              </DialogDescription>
            </div>
          </div>

          <div className="flex gap-2 mt-4 pt-2">
            <Button
              variant={activeTab === "list" ? "default" : "outline"}
              size="sm"
              onClick={() => setActiveTab("list")}
              className={`text-xs font-semibold gap-1.5 h-8 ${
                activeTab === "list"
                  ? "bg-amber-500 hover:bg-amber-400 text-zinc-950 font-bold"
                  : "border-zinc-800 text-zinc-400 hover:text-zinc-100"
              }`}
            >
              <Users className="w-3.5 h-3.5" />
              Kayıtlı Üyeler ({members.length})
            </Button>
            <Button
              variant={activeTab === "add" ? "default" : "outline"}
              size="sm"
              onClick={() => setActiveTab("add")}
              className={`text-xs font-semibold gap-1.5 h-8 ${
                activeTab === "add"
                  ? "bg-amber-500 hover:bg-amber-400 text-zinc-950 font-bold"
                  : "border-zinc-800 text-zinc-400 hover:text-zinc-100"
              }`}
            >
              <UserPlus className="w-3.5 h-3.5" />
              Yeni Üye Kaydet
            </Button>
          </div>
        </DialogHeader>

        <div className="p-6 max-h-[60vh] overflow-y-auto">
          {activeTab === "list" ? (
            <div className="space-y-3">
              {isLoading ? (
                <div className="flex items-center justify-center py-8 text-xs text-zinc-500 gap-2">
                  <Loader2 className="w-4 h-4 animate-spin text-amber-500" />
                  Üye listesi yükleniyor...
                </div>
              ) : members.length === 0 ? (
                <div className="text-center py-8 text-xs text-zinc-500">
                  Henüz kayıtlı ek personel bulunmuyor.
                </div>
              ) : (
                members.map((member) => (
                  <div
                    key={member.id || member.user_id}
                    className="flex items-center justify-between p-3.5 rounded-xl bg-zinc-900 border border-zinc-800/80 hover:border-zinc-700 transition-colors"
                  >
                    <div className="flex items-center gap-3">
                      <div className="w-8 h-8 rounded-lg bg-zinc-800 flex items-center justify-center text-zinc-400">
                        <Shield className="w-4 h-4" />
                      </div>
                      <div>
                        <div className="text-xs font-semibold text-zinc-200">
                          {member.email || `Kullanıcı ID: ${member.user_id.slice(0, 8)}...`}
                        </div>
                        <div className="text-[10px] text-zinc-500 font-mono">
                          {member.user_id}
                        </div>
                      </div>
                    </div>
                    <div>{getRoleBadge(member.role)}</div>
                  </div>
                ))
              )}
            </div>
          ) : (
            <form onSubmit={handleAddMember} className="space-y-4">
              {errorMsg && (
                <div className="flex items-center gap-2 p-3 text-xs text-rose-300 bg-rose-950/60 rounded-lg border border-rose-800">
                  <AlertCircle className="w-4 h-4 shrink-0" />
                  <span>{errorMsg}</span>
                </div>
              )}

              {successMsg && (
                <div className="flex items-center gap-2 p-3 text-xs text-amber-300 bg-amber-950/60 rounded-lg border border-amber-800">
                  <CheckCircle2 className="w-4 h-4 shrink-0" />
                  <span>{successMsg}</span>
                </div>
              )}

              <div className="space-y-1.5">
                <label className="text-xs font-semibold text-zinc-300 flex items-center gap-1.5">
                  <Mail className="w-3.5 h-3.5 text-amber-400" />
                  Personel E-Posta Adresi
                </label>
                <Input
                  type="email"
                  placeholder="muhasebe@oncuotogaz.com"
                  value={newEmail}
                  onChange={(e) => setNewEmail(e.target.value)}
                  className="text-xs bg-zinc-900 border-zinc-800 text-zinc-100 placeholder:text-zinc-600 focus:border-amber-500"
                  required
                />
              </div>

              <div className="space-y-1.5">
                <label className="text-xs font-semibold text-zinc-300 flex items-center gap-1.5">
                  <Lock className="w-3.5 h-3.5 text-amber-400" />
                  Geçici Başlangıç Şifresi
                </label>
                <Input
                  type="password"
                  placeholder="En az 6 karakter"
                  value={newPassword}
                  onChange={(e) => setNewPassword(e.target.value)}
                  className="text-xs bg-zinc-900 border-zinc-800 text-zinc-100 placeholder:text-zinc-600 focus:border-amber-500"
                  required
                />
              </div>

              <div className="space-y-1.5">
                <label className="text-xs font-semibold text-zinc-300 flex items-center gap-1.5">
                  <ShieldCheck className="w-3.5 h-3.5 text-amber-400" />
                  Yetki / Rol Tanımı
                </label>
                <div className="grid grid-cols-3 gap-2 pt-1">
                  {[
                    { key: "muhasebeci", label: "Muhasebeci", desc: "Kayıt & Rapor" },
                    { key: "standart", label: "Standart", desc: "Sadece Görüntüleme" },
                    { key: "admin", label: "Yönetici", desc: "Tam Yetki" },
                  ].map((item) => (
                    <button
                      key={item.key}
                      type="button"
                      onClick={() => setNewRole(item.key as any)}
                      className={`p-2.5 rounded-xl border text-left transition-all ${
                        newRole === item.key
                          ? "border-amber-500 bg-amber-500/10 text-amber-300"
                          : "border-zinc-800 bg-zinc-900 text-zinc-400 hover:border-zinc-700"
                      }`}
                    >
                      <div className="text-xs font-bold">{item.label}</div>
                      <div className="text-[10px] text-zinc-500 mt-0.5">{item.desc}</div>
                    </button>
                  ))}
                </div>
              </div>

              <Button
                type="submit"
                disabled={isSubmitting}
                className="w-full mt-4 bg-amber-500 hover:bg-amber-400 text-zinc-950 font-extrabold gap-2 h-9 shadow-lg shadow-amber-500/20"
              >
                {isSubmitting ? (
                  <>
                    <Loader2 className="w-4 h-4 animate-spin" />
                    Personel Kaydediliyor...
                  </>
                ) : (
                  <>
                    <UserPlus className="w-4 h-4" />
                    Şirkete Üye Olarak Ekle
                  </>
                )}
              </Button>
            </form>
          )}
        </div>
      </DialogContent>
    </Dialog>
  );
}
