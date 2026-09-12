import createMiddleware from 'next-intl/middleware';
import { NextRequest, NextResponse } from 'next/server';
import { createServerClient } from '@supabase/ssr';

const intlMiddleware = createMiddleware({
  locales: ['tr', 'en'],
  defaultLocale: 'tr',
  localePrefix: 'always',
});

export async function middleware(request: NextRequest) {
  try {
    const { pathname } = request.nextUrl;

    // 1. Execute next-intl locale routing first
    let response = intlMiddleware(request);

    // 2. Extract current locale from path
    const pathnameSegments = pathname.split('/').filter(Boolean);
    const currentLocale = pathnameSegments[0] === 'en' ? 'en' : 'tr';
    const isLoginPage = pathname.includes('/login');

    // 3. Initialize Supabase SSR client inside Edge Middleware
    const supabaseUrl = process.env.NEXT_PUBLIC_SUPABASE_URL || 'https://placeholder-project.supabase.co';
    const supabaseAnonKey = process.env.NEXT_PUBLIC_SUPABASE_ANON_KEY || 'placeholder-anon-key';

    const supabase = createServerClient(supabaseUrl, supabaseAnonKey, {
      cookies: {
        getAll() {
          return request.cookies.getAll();
        },
        setAll(cookiesToSet: Array<{ name: string; value: string; options?: any }>) {
          cookiesToSet.forEach(({ name, value }) => request.cookies.set(name, value));
          response = NextResponse.next({
            request,
          });
          cookiesToSet.forEach(({ name, value, options }) =>
            response.cookies.set(name, value, options)
          );
        },
      },
    });

    // 4. Cryptographically verify genuine Supabase session
    const {
      data: { user },
    } = await supabase.auth.getUser();

    // If NOT authenticated and attempting to access protected route -> Redirect to /login
    if (!user && !isLoginPage) {
      return NextResponse.redirect(new URL(`/${currentLocale}/login`, request.url));
    }

    // If ALREADY authenticated and attempting to access /login -> Redirect to home page /${currentLocale}
    if (user && isLoginPage) {
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
