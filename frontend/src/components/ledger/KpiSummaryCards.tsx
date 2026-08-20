'use client';

import React from 'react';
import { useTranslations } from 'next-intl';
import { Wallet, TrendingUp, TrendingDown, Scale } from 'lucide-react';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { formatTL } from '@/lib/decimal';

export interface PeriodSummaryData {
  period_id: string;
  starting_balance: string | number;
  total_in: string | number;
  total_out: string | number;
  closing_balance: string | number;
}

interface KpiSummaryCardsProps {
  summary: PeriodSummaryData | null;
  loading?: boolean;
}

export const KpiSummaryCards: React.FC<KpiSummaryCardsProps> = ({ summary, loading }) => {
  const t = useTranslations('kpi');

  const startingBalance = summary?.starting_balance ?? '0.00';
  const totalIn = summary?.total_in ?? '0.00';
  const totalOut = summary?.total_out ?? '0.00';
  const closingBalance = summary?.closing_balance ?? '0.00';

  return (
    <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4 mb-6">
      {/* Starting / Rollover Balance */}
      <Card className="border border-slate-200 shadow-sm hover:shadow-md transition-shadow bg-white">
        <CardHeader className="flex flex-row items-center justify-between pb-2">
          <CardTitle className="text-xs font-bold text-slate-500 uppercase tracking-wider">
            {t('starting_balance')}
          </CardTitle>
          <div className="p-2.5 rounded-lg bg-slate-100 text-slate-700">
            <Wallet className="h-5 w-5" />
          </div>
        </CardHeader>
        <CardContent>
          <div className="text-2xl font-extrabold tracking-tight text-slate-900">
            {loading ? (
              <span className="animate-pulse bg-slate-200 h-7 w-28 block rounded" />
            ) : (
              formatTL(startingBalance)
            )}
          </div>
        </CardContent>
      </Card>

      {/* Total Income */}
      <Card className="border border-emerald-200 shadow-sm hover:shadow-md transition-shadow bg-emerald-50/60">
        <CardHeader className="flex flex-row items-center justify-between pb-2">
          <CardTitle className="text-xs font-bold text-emerald-700 uppercase tracking-wider">
            {t('total_in')}
          </CardTitle>
          <div className="p-2.5 rounded-lg bg-emerald-100 text-emerald-700">
            <TrendingUp className="h-5 w-5" />
          </div>
        </CardHeader>
        <CardContent>
          <div className="text-2xl font-extrabold tracking-tight text-emerald-700">
            {loading ? (
              <span className="animate-pulse bg-emerald-200 h-7 w-28 block rounded" />
            ) : (
              formatTL(totalIn)
            )}
          </div>
        </CardContent>
      </Card>

      {/* Total Expense */}
      <Card className="border border-rose-200 shadow-sm hover:shadow-md transition-shadow bg-rose-50/60">
        <CardHeader className="flex flex-row items-center justify-between pb-2">
          <CardTitle className="text-xs font-bold text-rose-700 uppercase tracking-wider">
            {t('total_out')}
          </CardTitle>
          <div className="p-2.5 rounded-lg bg-rose-100 text-rose-700">
            <TrendingDown className="h-5 w-5" />
          </div>
        </CardHeader>
        <CardContent>
          <div className="text-2xl font-extrabold tracking-tight text-rose-700">
            {loading ? (
              <span className="animate-pulse bg-rose-200 h-7 w-28 block rounded" />
            ) : (
              formatTL(totalOut)
            )}
          </div>
        </CardContent>
      </Card>

      {/* Net Cash / Closing Balance */}
      <Card className="border border-blue-200 shadow-sm hover:shadow-md transition-shadow bg-blue-50/60">
        <CardHeader className="flex flex-row items-center justify-between pb-2">
          <CardTitle className="text-xs font-bold text-blue-700 uppercase tracking-wider">
            {t('closing_balance')}
          </CardTitle>
          <div className="p-2.5 rounded-lg bg-blue-100 text-blue-700">
            <Scale className="h-5 w-5" />
          </div>
        </CardHeader>
        <CardContent>
          <div className="text-2xl font-extrabold tracking-tight text-blue-800">
            {loading ? (
              <span className="animate-pulse bg-blue-200 h-7 w-28 block rounded" />
            ) : (
              formatTL(closingBalance)
            )}
          </div>
        </CardContent>
      </Card>
    </div>
  );
};

export default KpiSummaryCards;
