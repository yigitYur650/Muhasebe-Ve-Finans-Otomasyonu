'use client';

import React from 'react';
import { useTranslations } from 'next-intl';
import { Archive, Lock, ExternalLink, Calendar } from 'lucide-react';
import { Card, CardHeader, CardTitle, CardDescription, CardContent } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { formatTL } from '@/lib/decimal';

export interface PeriodHistoryItem {
  period_id: string;
  label: string;
  status: 'open' | 'locked';
  starting_balance: string | number;
  total_in: string | number;
  total_out: string | number;
  closing_balance: string | number;
  opened_at: string;
  locked_at?: string | null;
}

interface PeriodHistoryViewProps {
  history: PeriodHistoryItem[];
  loading?: boolean;
  onSelectPeriod?: (periodId: string) => void;
}

export const PeriodHistoryView: React.FC<PeriodHistoryViewProps> = ({
  history,
  loading,
  onSelectPeriod,
}) => {
  const t = useTranslations('history');

  if (loading) {
    return (
      <Card className="border border-slate-800 bg-slate-900/60 backdrop-blur-md">
        <CardContent className="p-8 text-center text-slate-400 animate-pulse">
          {t('title')}...
        </CardContent>
      </Card>
    );
  }

  return (
    <Card className="border border-slate-800 bg-slate-900/80 backdrop-blur-md shadow-lg">
      <CardHeader>
        <div className="flex items-center gap-3">
          <div className="p-2.5 rounded-xl bg-purple-500/10 text-purple-400 border border-purple-500/20">
            <Archive className="h-5 w-5" />
          </div>
          <div>
            <CardTitle className="text-xl font-bold text-slate-100">{t('title')}</CardTitle>
            <CardDescription className="text-sm text-slate-400">{t('description')}</CardDescription>
          </div>
        </div>
      </CardHeader>
      <CardContent>
        {history.length === 0 ? (
          <div className="text-center py-12 border border-dashed border-slate-800 rounded-xl bg-slate-950/40">
            <Archive className="h-10 w-10 text-slate-600 mx-auto mb-3" />
            <p className="text-slate-400 text-sm">{t('no_history')}</p>
          </div>
        ) : (
          <div className="overflow-x-auto rounded-lg border border-slate-800">
            <table className="w-full text-left text-sm text-slate-300">
              <thead className="bg-slate-950/80 text-xs uppercase font-semibold text-slate-400 border-b border-slate-800">
                <tr>
                  <th className="px-4 py-3">{t('label')}</th>
                  <th className="px-4 py-3">{t('status')}</th>
                  <th className="px-4 py-3 text-right">{t('starting_balance')}</th>
                  <th className="px-4 py-3 text-right">{t('total_in')}</th>
                  <th className="px-4 py-3 text-right">{t('total_out')}</th>
                  <th className="px-4 py-3 text-right">{t('closing_balance')}</th>
                  <th className="px-4 py-3 text-center">{t('locked_at')}</th>
                  <th className="px-4 py-3 text-center">İşlem</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-slate-800/60 bg-slate-900/40">
                {history.map((item) => (
                  <tr
                    key={item.period_id}
                    className="hover:bg-slate-800/40 transition-colors"
                  >
                    <td className="px-4 py-3 font-semibold text-slate-100 flex items-center gap-2">
                      <Calendar className="h-4 w-4 text-purple-400" />
                      {item.label}
                    </td>
                    <td className="px-4 py-3">
                      {item.status === 'locked' ? (
                        <Badge variant="secondary" className="bg-amber-500/10 text-amber-400 border border-amber-500/30 flex items-center gap-1 w-fit">
                          <Lock className="h-3 w-3" />
                          {t('status_locked')}
                        </Badge>
                      ) : (
                        <Badge variant="default" className="bg-emerald-500/10 text-emerald-400 border border-emerald-500/30 w-fit">
                          {t('status_open')}
                        </Badge>
                      )}
                    </td>
                    <td className="px-4 py-3 text-right font-mono text-slate-300">
                      {formatTL(item.starting_balance)}
                    </td>
                    <td className="px-4 py-3 text-right font-mono text-emerald-400 font-medium">
                      +{formatTL(item.total_in)}
                    </td>
                    <td className="px-4 py-3 text-right font-mono text-rose-400 font-medium">
                      -{formatTL(item.total_out)}
                    </td>
                    <td className="px-4 py-3 text-right font-mono font-bold text-cyan-300">
                      {formatTL(item.closing_balance)}
                    </td>
                    <td className="px-4 py-3 text-center text-xs text-slate-400" suppressHydrationWarning>
                      {item.locked_at ? new Date(item.locked_at).toLocaleDateString("tr-TR") : '-'}
                    </td>
                    <td className="px-4 py-3 text-center">
                      <Button
                        size="sm"
                        variant="ghost"
                        className="h-8 px-3 text-xs text-purple-400 hover:text-purple-300 hover:bg-purple-500/10 border border-purple-500/20"
                        onClick={() => onSelectPeriod?.(item.period_id)}
                      >
                        <ExternalLink className="h-3.5 w-3.5 mr-1" />
                        {t('inspect_ledger')}
                      </Button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </CardContent>
    </Card>
  );
};
