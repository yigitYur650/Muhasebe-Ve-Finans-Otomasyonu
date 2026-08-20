"use client";

import { useState } from "react";
import { useTranslations } from "next-intl";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogTrigger,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Badge } from "@/components/ui/badge";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Users, UserPlus, Trash2, ShieldAlert } from "lucide-react";

export interface MemberItem {
  userId: string;
  email: string;
  role: "admin" | "muhasebeci" | "standart";
}

interface MemberManagementDialogProps {
  userRole: "admin" | "muhasebeci" | "standart";
  members: MemberItem[];
  onAddMember: (email: string, role: "admin" | "muhasebeci" | "standart") => Promise<void>;
  onUpdateRole: (userId: string, newRole: "admin" | "muhasebeci" | "standart") => Promise<void>;
  onRemoveMember: (userId: string) => Promise<void>;
}

export function MemberManagementDialog({
  userRole,
  members,
  onAddMember,
  onUpdateRole,
  onRemoveMember,
}: MemberManagementDialogProps) {
  const t = useTranslations("members");
  const tAuth = useTranslations("auth");
  const tCommon = useTranslations("common");

  const [open, setOpen] = useState(false);
  const [newEmail, setNewEmail] = useState("");
  const [newRole, setNewRole] = useState<"admin" | "muhasebeci" | "standart">("standart");
  const [errorMsg, setErrorMsg] = useState<string | null>(null);
  const [isSubmitting, setIsSubmitting] = useState(false);

  if (userRole !== "admin") {
    return null; // Strict RBAC Guard: non-admins cannot even see this trigger button
  }

  const handleAdd = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!newEmail.trim()) return;
    setErrorMsg(null);
    setIsSubmitting(true);
    try {
      await onAddMember(newEmail, newRole);
      setNewEmail("");
    } catch (err: any) {
      setErrorMsg(err.message || tCommon("noResults"));
    } finally {
      setIsSubmitting(false);
    }
  };

  const handleRoleChange = async (userId: string, role: "admin" | "muhasebeci" | "standart") => {
    setErrorMsg(null);
    try {
      await onUpdateRole(userId, role);
    } catch (err: any) {
      if (err.message?.includes("CANNOT_REMOVE_LAST_ADMIN")) {
        setErrorMsg(t("lastAdminError"));
      } else {
        setErrorMsg(err.message);
      }
    }
  };

  const handleRemove = async (userId: string) => {
    setErrorMsg(null);
    try {
      await onRemoveMember(userId);
    } catch (err: any) {
      if (err.message?.includes("CANNOT_REMOVE_LAST_ADMIN")) {
        setErrorMsg(t("lastAdminError"));
      } else {
        setErrorMsg(err.message);
      }
    }
  };

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger asChild>
        <Button variant="outline" size="sm" className="h-8 gap-1.5 text-xs">
          <Users className="w-3.5 h-3.5" />
          {t("title")}
        </Button>
      </DialogTrigger>

      <DialogContent className="sm:max-w-xl">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2 text-base">
            <Users className="w-4 h-4 text-indigo-600" />
            {t("title")}
          </DialogTitle>
          <DialogDescription className="text-xs text-slate-500">
            {t("description")}
          </DialogDescription>
        </DialogHeader>

        {errorMsg && (
          <div className="flex items-center gap-2 p-2.5 bg-rose-50 border border-rose-200 text-rose-700 text-xs rounded-md">
            <ShieldAlert className="w-4 h-4 shrink-0 text-rose-600" />
            <span>{errorMsg}</span>
          </div>
        )}

        {/* Add Member Form */}
        <form onSubmit={handleAdd} className="flex gap-2 items-end pt-1">
          <div className="flex-1 space-y-1">
            <label className="text-xs font-medium text-slate-700">{t("addMember")}</label>
            <Input
              type="email"
              placeholder={t("emailPlaceholder")}
              value={newEmail}
              onChange={(e) => setNewEmail(e.target.value)}
              className="h-8 text-xs"
              required
            />
          </div>

          <div className="w-[140px] space-y-1">
            <label className="text-xs font-medium text-slate-700">{t("role")}</label>
            <Select value={newRole} onValueChange={(val: any) => setNewRole(val)}>
              <SelectTrigger className="h-8 text-xs bg-white">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="admin" className="text-xs">{tAuth("roleAdmin")}</SelectItem>
                <SelectItem value="muhasebeci" className="text-xs">{tAuth("roleMuhasebeci")}</SelectItem>
                <SelectItem value="standart" className="text-xs">{tAuth("roleStandart")}</SelectItem>
              </SelectContent>
            </Select>
          </div>

          <Button type="submit" disabled={isSubmitting} size="sm" className="h-8 gap-1 text-xs">
            <UserPlus className="w-3.5 h-3.5" />
            {tCommon("save")}
          </Button>
        </form>

        {/* Members Table */}
        <div className="border border-slate-200 rounded-md overflow-hidden mt-2">
          <table className="w-full text-xs text-left border-collapse">
            <thead>
              <tr className="bg-slate-100 border-b border-slate-200 text-slate-600 font-semibold">
                <th className="p-2.5">Kullanıcı</th>
                <th className="p-2.5">{t("role")}</th>
                <th className="p-2.5 text-right">{tCommon("actions")}</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-slate-200 bg-white">
              {members.map((m) => (
                <tr key={m.userId} className="hover:bg-slate-50">
                  <td className="p-2.5 font-medium text-slate-800">
                    {m.email}
                  </td>
                  <td className="p-2.5">
                    <Select
                      value={m.role}
                      onValueChange={(val: any) => handleRoleChange(m.userId, val)}
                    >
                      <SelectTrigger className="h-7 text-xs w-[130px] bg-white">
                        <SelectValue />
                      </SelectTrigger>
                      <SelectContent>
                        <SelectItem value="admin" className="text-xs">{tAuth("roleAdmin")}</SelectItem>
                        <SelectItem value="muhasebeci" className="text-xs">{tAuth("roleMuhasebeci")}</SelectItem>
                        <SelectItem value="standart" className="text-xs">{tAuth("roleStandart")}</SelectItem>
                      </SelectContent>
                    </Select>
                  </td>
                  <td className="p-2.5 text-right">
                    <Button
                      variant="ghost"
                      size="sm"
                      onClick={() => handleRemove(m.userId)}
                      className="h-7 w-7 p-0 text-slate-400 hover:text-rose-600"
                    >
                      <Trash2 className="w-3.5 h-3.5" />
                    </Button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </DialogContent>
    </Dialog>
  );
}
