'use client'

import { useEffect, useState } from 'react'
import { useRouter, usePathname } from 'next/navigation'
import { useAuthStore } from '@/stores'
import { Loader2 } from 'lucide-react'

// Routes that require authentication
const protectedRoutes = ['/', '/apps', '/apps/new', '/apps/[id]', '/services', '/services/[id]', '/server', '/settings', '/settings/backup', '/team', '/backups', '/cron', '/cron/[id]', '/files', '/terminal', '/activity']

// Routes that should redirect to dashboard if authenticated
const authRoutes = ['/login', '/setup']

export function AuthProvider({ children }: { children: React.ReactNode }) {
  const router = useRouter()
  const pathname = usePathname()
  const { isAuthenticated, checkUsersExist, usersExist } = useAuthStore()
  const [isLoading, setIsLoading] = useState(true)

  useEffect(() => {
    async function checkAuth() {
      // Wait for usersExist to be determined
      if (usersExist === null) {
        await checkUsersExist()
      }
      setIsLoading(false)
    }
    checkAuth()
  }, [checkUsersExist, usersExist])

  useEffect(() => {
    if (isLoading) return

    const isProtectedRoute = protectedRoutes.some(route => 
      pathname === route || pathname.startsWith(route.replace('[id]', ''))
    )
    const isAuthRoute = authRoutes.includes(pathname)

    // If no users exist, redirect to setup
    if (usersExist === false && !isAuthRoute && pathname !== '/setup') {
      router.push('/setup')
      return
    }

    // If users exist but not authenticated, redirect to login
    if (usersExist === true && !isAuthenticated && isProtectedRoute) {
      router.push('/login')
      return
    }

    // If authenticated and trying to access login/setup, redirect to dashboard
    if (isAuthenticated && (pathname === '/login' || pathname === '/setup')) {
      router.push('/')
      return
    }
  }, [isLoading, usersExist, isAuthenticated, pathname, router])

  if (isLoading) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-slate-900">
        <div className="text-center">
          <Loader2 className="w-8 h-8 animate-spin text-primary-400 mx-auto mb-4" />
          <p className="text-slate-400">Loading...</p>
        </div>
      </div>
    )
  }

  return <>{children}</>
}