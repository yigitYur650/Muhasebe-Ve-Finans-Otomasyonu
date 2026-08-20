import createMiddleware from 'next-intl/middleware';
import { locales, defaultLocale } from './i18n/request';

export default createMiddleware({
  locales,
  defaultLocale,
  localeDetection: true,
  localePrefix: 'as-needed'
});

export const config = {
  matcher: ['/', '/(tr|en)/:path*', '/((?!_next|_vercel|.*\\..*).*)']
};
