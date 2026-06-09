'use client'

import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { useState, useEffect } from 'react'
import { ToastProvider } from '@/components/ui/toast'
import { CommandPalette } from '@/components/layout/CommandPalette'
import { ConnectionLost } from '@/components/ui/ConnectionLost'
import { useWebSocket } from '@/lib/websocket'
import { useToastStore, useAuthStore } from '@/stores'
import { useRouter, usePathname } from 'next/navigation'

function WebSocketConnectionListener({ children }: { children: React.ReactNode }) {
  const [showConnectionLost, setShowConnectionLost] = useState(false)
  const [disconnectTimer, setDisconnectTimer] = useState<NodeJS.Timeout | null>(null)
  const { isConnected, reconnect } = useWebSocket()
  const { addToast } = useToastStore()

  useEffect(() => {
    if (isConnected) {
      if (disconnectTimer) {
        clearTimeout(disconnectTimer)
        setDisconnectTimer(null)
      }
      setShowConnectionLost(false)
    } else {
      if (!showConnectionLost) {
        const timer = setTimeout(() => {
          setShowConnectionLost(true)
        }, 10000)
        setDisconnectTimer(timer)
      }
    }

    return () => {
      if (disconnectTimer) {
        clearTimeout(disconnectTimer)
      }
    }
  }, [isConnected, showConnectionLost, disconnectTimer])

  const handleRetry = () => {
    reconnect()
    setShowConnectionLost(false)
  }

  const handleCancel = () => {
    setShowConnectionLost(false)
  }

  return (
    <>
      {children}
      {showConnectionLost && !isConnected && (
        <ConnectionLost onRetry={handleRetry} onCancel={handleCancel} />
      )}
    </>
  )
}

// Routes that require authentication
const protectedRoutes = ['/', '/apps', '/apps/new', '/apps', '/server', '/settings', '/settings/backup', '/team', '/backups', '/cron', '/files', '/terminal', '/activity']

// Routes that should redirect to dashboard if authenticated
const authRoutes = ['/login', '/setup']

function AuthGuard({ children }: { children: React.ReactNode }) {
  const router = useRouter()
  const pathname = usePathname()
  const { isAuthenticated, checkUsersExist, usersExist, _hasHydrated } = useAuthStore()
  const [isLoading, setIsLoading] = useState(true)

  const normalizedPathname = pathname === '/' ? '/' : pathname.replace(/\/$/, '')

  useEffect(() => {
    if (!_hasHydrated) return

    async function checkAuth() {
      if (usersExist === null) {
        await checkUsersExist()
      }
      setIsLoading(false)
    }
    checkAuth()
  }, [_hasHydrated, checkUsersExist, usersExist])

  useEffect(() => {
    if (isLoading || !_hasHydrated) return

    const isProtectedRoute = protectedRoutes.some(route => normalizedPathname === route || normalizedPathname.startsWith(route + '/'))
    const isAuthRoute = authRoutes.includes(normalizedPathname)

    if (usersExist === false && !isAuthRoute && pathname !== '/setup') {
      router.replace('/setup')
      return
    }

    if (usersExist === true && !isAuthenticated && isProtectedRoute) {
      router.replace('/login')
      return
    }

    if (isAuthenticated && isAuthRoute) {
      router.replace('/')
      return
    }
  }, [isLoading, _hasHydrated, usersExist, isAuthenticated])

  if (isLoading) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-slate-900">
        <div className="text-center">
          <div className="w-8 h-8 border-2 border-primary-500 border-t-transparent rounded-full animate-spin mx-auto mb-4" />
          <p className="text-slate-400">Loading...</p>
        </div>
      </div>
    )
  }

  return <>{children}</>
}

export function Providers({ children }: { children: React.ReactNode }) {
  const [queryClient] = useState(
    () =>
      new QueryClient({
        defaultOptions: {
          queries: {
            staleTime: 60 * 1000,
            retry: 1,
          },
        },
      })
  )

  return (
    <QueryClientProvider client={queryClient}>
      <AuthGuard>
        <WebSocketConnectionListener>
          {children}
          <ToastProvider />
          <CommandPalette />
        </WebSocketConnectionListener>
      </AuthGuard>
    </QueryClientProvider>
  )
}