import createMiddleware from 'next-intl/middleware';
import { NextRequest, NextResponse } from 'next/server';

const intlMiddleware = createMiddleware({
  locales: ['tr', 'en'],
  defaultLocale: 'tr',
  localePrefix: 'always',
});

export async function middleware(request: NextRequest) {
  try {
    // 1. Execute next-intl locale routing safely inside try/catch
    const response = intlMiddleware(request);
    return response;
  } catch {
    return NextResponse.next();
  }
}

export const config = {
  // Exclude static files, favicon, _next, and API routes from middleware execution
  matcher: ['/((?!api|_next|_vercel|.*\\..*).*)'],
};
