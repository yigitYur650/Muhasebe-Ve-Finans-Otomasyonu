"use client";

import { useState } from "react";
import { useTranslations } from "next-intl";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { PlusCircle, ShoppingBag, CreditCard, AlertCircle, RotateCcw } from "lucide-react";
import { Supplier, SupplierDirection } from "@/types/supplier";
import { apiFetch } from "@/lib/api";

interface CreateSupplierTransactionDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  periodId: string;
  suppliers: Supplier[];
  isPeriodLocked: boolean;
  onSuccess: () => void;
}

export function CreateSupplierTransactionDialog({
  open,
  onOpenChange,
  periodId,
  suppliers,
  isPeriodLocked,
  onSuccess,
}: CreateSupplierTransactionDialogProps) {
  const t = useTranslations("suppliers");
  const tErr = useTranslations("errors");

  const [supplierId, setSupplierId] = useState<string>(suppliers[0]?.id || "");
  const [newSupplierName, setNewSupplierName] = useState<string>("");
  const [isCreatingNewSupplier, setIsCreatingNewSupplier] = useState<boolean>(false);
  const [invoiceNo, setInvoiceNo] = useState<string>("");
  const [customerName, setCustomerName] = useState<string>("");
  const [documentStatus, setDocumentStatus] = useState<string>("Faturalı");
  const [txDate, setTxDate] = useState<string>(new Date().toISOString().split("T")[0]);
  const [direction, setDirection] = useState<SupplierDirection>("purchase");
  const [amountStr, setAmountStr] = useState<string>("");
  const [description, setDescription] = useState<string>("");
  const [isSubmitting, setIsSubmitting] = useState<boolean>(false);
  const [errorMsg, setErrorMsg] = useState<string | null>(null);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();

    if (isPeriodLocked) {
      setErrorMsg(tErr("PERIOD_LOCKED"));
      return;
    }

    let targetSupplierId = supplierId;

    if (isCreatingNewSupplier) {
      if (!newSupplierName.trim()) {
        setErrorMsg("Lütfen yeni tedarikçi adını girin.");
        return;
      }
      setIsSubmitting(true);
      try {
        const res = await apiFetch<Supplier>("/suppliers", {
          method: "POST",
          body: JSON.stringify({ name: newSupplierName.trim() }),
        });
        if (!res.success || !res.data) {
          throw new Error(res.error?.message || "Tedarikçi oluşturulamadı");
        }
        targetSupplierId = res.data.id;
      } catch (err: any) {
        setErrorMsg(err.message);
        setIsSubmitting(false);
        return;
      }
    }

    if (!targetSupplierId) {
      setErrorMsg("Lütfen bir tedarikçi seçin.");
      return;
    }

    const cleanAmount = amountStr.replace(",", ".");
    const numericAmount = parseFloat(cleanAmount);
    if (isNaN(numericAmount) || numericAmount <= 0) {
      setErrorMsg("Lütfen sıfırdan büyük geçerli bir tutar girin.");
      return;
    }

    setIsSubmitting(true);
    setErrorMsg(null);

    try {
      const res = await apiFetch("/suppliers/transactions", {
        method: "POST",
        body: JSON.stringify({
          supplier_id: targetSupplierId,
          period_id: periodId,
          invoice_no: invoiceNo.trim(),
          customer_name: customerName.trim(),
          document_status: documentStatus.trim(),
          tx_date: txDate,
          direction,
          amount: cleanAmount,
          description: description.trim(),
        }),
      });

      if (!res.success) {
        throw new Error(res.error?.message || "İşlem kaydedilemedi");
      }

      onSuccess();
      onOpenChange(false);
      // Reset
      setAmountStr("");
      setInvoiceNo("");
      setCustomerName("");
      setDescription("");
      setIsCreatingNewSupplier(false);
      setNewSupplierName("");
    } catch (err: any) {
      setErrorMsg(err.message || "İşlem kaydedilirken bir hata oluştu.");
    } finally {
      setIsSubmitting(false);
    }
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-[500px]">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2 text-slate-900">
            <PlusCircle className="w-5 h-5 text-indigo-600" />
            {t("newTransaction")}
          </DialogTitle>
          <DialogDescription className="text-xs text-slate-500">
            Tedarikçi cari defterine yeni mal alımı veya ödeme kaydı ekleyin.
          </DialogDescription>
        </DialogHeader>

        <form onSubmit={handleSubmit} className="space-y-3.5 pt-2">
          {/* Supplier Selector */}
          <div className="space-y-1">
            <div className="flex items-center justify-between">
              <label className="text-xs font-bold text-slate-700">{t("supplierName")}</label>
              <button
                type="button"
                onClick={() => setIsCreatingNewSupplier(!isCreatingNewSupplier)}
                className="text-[11px] text-indigo-600 font-semibold hover:underline"
              >
                {isCreatingNewSupplier ? "Mevcut Tedarikçilerden Seç" : "+ Yeni Firma Ekle"}
              </button>
            </div>

            {isCreatingNewSupplier ? (
              <Input
                type="text"
                placeholder="Örn: ATİKER, PRİNS, ASİL GRUP..."
                value={newSupplierName}
                onChange={(e) => setNewSupplierName(e.target.value)}
                className="h-9 text-xs"
                required
              />
            ) : (
              <select
                value={supplierId}
                onChange={(e) => setSupplierId(e.target.value)}
                className="w-full h-9 text-xs border border-slate-300 rounded-md px-2.5 bg-white text-slate-800 font-semibold focus:outline-none focus:ring-1 focus:ring-indigo-500"
                required
              >
                <option value="">Firma Seçiniz...</option>
                {suppliers.map((s) => (
                  <option key={s.id} value={s.id}>
                    {s.name}
                  </option>
                ))}
              </select>
            )}
          </div>

          {/* Direction Toggle (4 Muhasebe Türü) */}
          <div className="space-y-1">
            <label className="text-xs font-bold text-slate-700">{t("direction")}</label>
            <div className="grid grid-cols-2 gap-2">
              <button
                type="button"
                onClick={() => setDirection("purchase")}
                className={`flex items-center justify-center gap-1.5 p-2 rounded-lg border text-xs font-bold transition-all ${
                  direction === "purchase"
                    ? "border-rose-500 bg-rose-50 text-rose-700 ring-1 ring-rose-500"
                    : "border-slate-200 bg-white text-slate-600 hover:bg-slate-50"
                }`}
              >
                <ShoppingBag className="w-4 h-4 text-rose-600" />
                Alınan Mal (+)
              </button>
              <button
                type="button"
                onClick={() => setDirection("payment")}
                className={`flex items-center justify-center gap-1.5 p-2 rounded-lg border text-xs font-bold transition-all ${
                  direction === "payment"
                    ? "border-emerald-500 bg-emerald-50 text-emerald-700 ring-1 ring-emerald-500"
                    : "border-slate-200 bg-white text-slate-600 hover:bg-slate-50"
                }`}
              >
                <CreditCard className="w-4 h-4 text-emerald-600" />
                Geçilen Ödeme (-)
              </button>
              <button
                type="button"
                onClick={() => setDirection("purchase_return")}
                className={`flex items-center justify-center gap-1.5 p-2 rounded-lg border text-xs font-bold transition-all ${
                  direction === "purchase_return"
                    ? "border-amber-500 bg-amber-50 text-amber-700 ring-1 ring-amber-500"
                    : "border-slate-200 bg-white text-slate-600 hover:bg-slate-50"
                }`}
              >
                <RotateCcw className="w-4 h-4 text-amber-600" />
                Alış İadesi (-)
              </button>
              <button
                type="button"
                onClick={() => setDirection("payment_return")}
                className={`flex items-center justify-center gap-1.5 p-2 rounded-lg border text-xs font-bold transition-all ${
                  direction === "payment_return"
                    ? "border-blue-500 bg-blue-50 text-blue-700 ring-1 ring-blue-500"
                    : "border-slate-200 bg-white text-slate-600 hover:bg-slate-50"
                }`}
              >
                <RotateCcw className="w-4 h-4 text-blue-600" />
                Ödeme İadesi (+)
              </button>
            </div>
          </div>

          {/* Amount and Date */}
          <div className="grid grid-cols-2 gap-3">
            <div className="space-y-1">
              <label className="text-xs font-bold text-slate-700">{t("amount")}</label>
              <Input
                type="text"
                placeholder="0.00"
                value={amountStr}
                onChange={(e) => setAmountStr(e.target.value)}
                className="h-9 text-xs font-mono font-bold"
                required
              />
            </div>
            <div className="space-y-1">
              <label className="text-xs font-bold text-slate-700">{t("date")}</label>
              <Input
                type="date"
                value={txDate}
                onChange={(e) => setTxDate(e.target.value)}
                className="h-9 text-xs"
                required
              />
            </div>
          </div>

          {/* Invoice No and Customer Name */}
          <div className="grid grid-cols-2 gap-3">
            <div className="space-y-1">
              <label className="text-xs font-bold text-slate-700">{t("invoiceNo")}</label>
              <Input
                type="text"
                placeholder="Örn: FAT-2026-001"
                value={invoiceNo}
                onChange={(e) => setInvoiceNo(e.target.value)}
                className="h-9 text-xs"
              />
            </div>
            <div className="space-y-1">
              <label className="text-xs font-bold text-slate-700">{t("customerName")}</label>
              <Input
                type="text"
                placeholder="Örn: Ahmet Yılmaz"
                value={customerName}
                onChange={(e) => setCustomerName(e.target.value)}
                className="h-9 text-xs"
              />
            </div>
          </div>

          {/* Document Status and Description */}
          <div className="space-y-1">
            <label className="text-xs font-bold text-slate-700">{t("documentStatus")}</label>
            <select
              value={documentStatus}
              onChange={(e) => setDocumentStatus(e.target.value)}
              className="w-full h-9 text-xs border border-slate-300 rounded-md px-2.5 bg-white text-slate-800 font-medium focus:outline-none focus:ring-1 focus:ring-indigo-500"
            >
              <option value="Faturalı">Faturalı</option>
              <option value="İrsaliyeli">İrsaliyeli</option>
              <option value="Fiş / Makbuz">Fiş / Makbuz</option>
              <option value="Kredi Kartı Slip">Kredi Kartı Slip</option>
              <option value="Havale Dekontu">Havale Dekontu</option>
              <option value="Diğer">Diğer</option>
            </select>
          </div>

          <div className="space-y-1">
            <label className="text-xs font-bold text-slate-700">{t("description")}</label>
            <Input
              type="text"
              placeholder="İşleme ait not veya detay..."
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              className="h-9 text-xs"
            />
          </div>

          {errorMsg && (
            <div className="flex items-center gap-2 p-2.5 bg-rose-50 border border-rose-200 text-rose-800 rounded-lg text-xs font-medium">
              <AlertCircle className="w-4 h-4 text-rose-600 shrink-0" />
              <span>{errorMsg}</span>
            </div>
          )}

          <DialogFooter className="pt-2 sm:justify-between items-center gap-2">
            <Button
              type="button"
              variant="outline"
              onClick={() => onOpenChange(false)}
              className="h-9 text-xs font-semibold"
            >
              İptal
            </Button>
            <Button
              type="submit"
              disabled={isSubmitting || isPeriodLocked}
              className="h-9 text-xs bg-indigo-600 hover:bg-indigo-700 text-white font-semibold"
            >
              {isSubmitting ? "Kaydediliyor..." : "Kaydet"}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
