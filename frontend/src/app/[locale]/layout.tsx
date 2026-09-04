import { NextIntlClientProvider } from 'next-intl';
import { getMessages } from 'next-intl/server';
import { notFound } from 'next/navigation';
import { locales } from '@/i18n/request';
import trMessages from '@/messages/tr.json';
import enMessages from '@/messages/en.json';
import '../globals.css';

export const metadata = {
  title: 'Öncü Otogaz — Kasa ve Defter-i Kebir Platformu',
  description: 'Excel hızında, devir garantili yeni nesil kasa ve defter-i kebir platformu.',
};

export function generateStaticParams() {
  return locales.map((locale) => ({ locale }));
}

export default async function LocaleLayout({
  children,
  params,
}: {
  children: React.ReactNode;
  params: Promise<{ locale: string }>;
}) {
  const { locale } = await params;

  if (!locales.includes(locale as any)) {
    notFound();
  }

  let messages;
  try {
    messages = await getMessages();
  } catch {
    messages = locale === 'en' ? enMessages : trMessages;
  }

  if (!messages || Object.keys(messages).length === 0) {
    messages = locale === 'en' ? enMessages : trMessages;
  }

  return (
    <html lang={locale}>
      <body className="min-h-screen bg-slate-50 text-slate-900 font-sans antialiased">
        <NextIntlClientProvider messages={messages} locale={locale}>
          {children}
        </NextIntlClientProvider>
      </body>
    </html>
  );
}
