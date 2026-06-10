import { NextResponse } from 'next/server'
import type { NextRequest } from 'next/server'

const PUBLIC_PATHS = ['/login', '/setup', '/_next']
const STATIC_EXTS = /\.(ico|png|svg|jpg|jpeg|gif|webp|css|js|woff2?|ttf|eot)$/

export function middleware(request: NextRequest) {
  const { pathname } = request.nextUrl

  if (STATIC_EXTS.test(pathname) || pathname.startsWith('/_next/')) {
    return NextResponse.next()
  }

  if (PUBLIC_PATHS.some(p => pathname === p || pathname.startsWith(p + '/'))) {
    return NextResponse.next()
  }

  const refreshToken = request.cookies.get('refresh_token')?.value
  const authHeader = request.headers.get('authorization')

  if (!refreshToken && !authHeader) {
    const loginUrl = new URL('/login', request.url)
    loginUrl.searchParams.set('redirect', pathname)
    return NextResponse.redirect(loginUrl)
  }

  return NextResponse.next()
}

export const config = {
  matcher: [
    '/((?!_next|api|health|favicon.ico).*)',
  ],
}
