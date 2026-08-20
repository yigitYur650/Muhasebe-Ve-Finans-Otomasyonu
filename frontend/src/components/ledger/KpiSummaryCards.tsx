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
      <Card className="border shadow-sm hover:shadow-md transition-shadow bg-gradient-to-br from-slate-900/50 to-slate-800/50 backdrop-blur-sm">
        <CardHeader className="flex flex-row items-center justify-between pb-2">
          <CardTitle className="text-xs font-semibold text-slate-400 uppercase tracking-wider">
            {t('starting_balance')}
          </CardTitle>
          <div className="p-2 rounded-lg bg-blue-500/10 text-blue-400">
            <Wallet className="h-4 w-4" />
          </div>
        </CardHeader>
        <CardContent>
          <div className="text-2xl font-bold tracking-tight text-slate-100">
            {loading ? (
              <span className="animate-pulse bg-slate-700 h-7 w-24 block rounded" />
            ) : (
              formatTL(startingBalance)
            )}
          </div>
        </CardContent>
      </Card>

      {/* Total Income */}
      <Card className="border shadow-sm hover:shadow-md transition-shadow bg-gradient-to-br from-emerald-950/40 to-emerald-900/30 backdrop-blur-sm border-emerald-800/40">
        <CardHeader className="flex flex-row items-center justify-between pb-2">
          <CardTitle className="text-xs font-semibold text-emerald-400 uppercase tracking-wider">
            {t('total_in')}
          </CardTitle>
          <div className="p-2 rounded-lg bg-emerald-500/10 text-emerald-400">
            <TrendingUp className="h-4 w-4" />
          </div>
        </CardHeader>
        <CardContent>
          <div className="text-2xl font-bold tracking-tight text-emerald-300">
            {loading ? (
              <span className="animate-pulse bg-emerald-900/50 h-7 w-24 block rounded" />
            ) : (
              formatTL(totalIn)
            )}
          </div>
        </CardContent>
      </Card>

      {/* Total Expense */}
      <Card className="border shadow-sm hover:shadow-md transition-shadow bg-gradient-to-br from-rose-950/40 to-rose-900/30 backdrop-blur-sm border-rose-800/40">
        <CardHeader className="flex flex-row items-center justify-between pb-2">
          <CardTitle className="text-xs font-semibold text-rose-400 uppercase tracking-wider">
            {t('total_out')}
          </CardTitle>
          <div className="p-2 rounded-lg bg-rose-500/10 text-rose-400">
            <TrendingDown className="h-4 w-4" />
          </div>
        </CardHeader>
        <CardContent>
          <div className="text-2xl font-bold tracking-tight text-rose-300">
            {loading ? (
              <span className="animate-pulse bg-rose-900/50 h-7 w-24 block rounded" />
            ) : (
              formatTL(totalOut)
            )}
          </div>
        </CardContent>
      </Card>

      {/* Net Cash / Closing Balance */}
      <Card className="border shadow-sm hover:shadow-md transition-shadow bg-gradient-to-br from-cyan-950/40 to-blue-900/30 backdrop-blur-sm border-cyan-800/40">
        <CardHeader className="flex flex-row items-center justify-between pb-2">
          <CardTitle className="text-xs font-semibold text-cyan-400 uppercase tracking-wider">
            {t('closing_balance')}
          </CardTitle>
          <div className="p-2 rounded-lg bg-cyan-500/10 text-cyan-400">
            <Scale className="h-4 w-4" />
          </div>
        </CardHeader>
        <CardContent>
          <div className="text-2xl font-bold tracking-tight text-cyan-200">
            {loading ? (
              <span className="animate-pulse bg-cyan-900/50 h-7 w-24 block rounded" />
            ) : (
              formatTL(closingBalance)
            )}
          </div>
        </CardContent>
      </Card>
    </div>
  );
};
