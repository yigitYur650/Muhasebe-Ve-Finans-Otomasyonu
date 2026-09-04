import createMiddleware from 'next-intl/middleware';
import { NextRequest, NextResponse } from 'next/server';

const intlMiddleware = createMiddleware({
  locales: ['tr', 'en'],
  defaultLocale: 'tr',
  localePrefix: 'always',
});

export async function middleware(request: NextRequest) {
  try {
    const { pathname } = request.nextUrl;

    // 1. Execute next-intl locale routing
    const response = intlMiddleware(request);

    // 2. Extract current locale from path
    const pathnameSegments = pathname.split('/').filter(Boolean);
    const currentLocale = pathnameSegments[0] === 'en' ? 'en' : 'tr';
    const isLoginPage = pathname.includes('/login');

    // 3. Check for session cookie or Supabase auth token
    const hasSession =
      request.cookies.has('defter_session') ||
      request.cookies.has('sb-access-token') ||
      request.cookies.has('sb-refresh-token');

    // If NOT authenticated and attempting to access protected route -> Redirect to /login
    if (!hasSession && !isLoginPage) {
      return NextResponse.redirect(new URL(`/${currentLocale}/login`, request.url));
    }

    // If ALREADY authenticated and attempting to access /login -> Redirect to home page /${currentLocale}
    if (hasSession && isLoginPage) {
      return NextResponse.redirect(new URL(`/${currentLocale}`, request.url));
    }

    return response;
  } catch {
    return NextResponse.next();
  }
}

export const config = {
  // Exclude static assets, Next.js internal files, and API endpoints
  matcher: ['/((?!api|_next|_vercel|.*\\..*).*)'],
};
