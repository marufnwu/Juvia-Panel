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
      // Clear any pending disconnect timer
      if (disconnectTimer) {
        clearTimeout(disconnectTimer)
        setDisconnectTimer(null)
      }
      // Hide connection lost modal if it was shown
      setShowConnectionLost(false)
    } else {
      // Start a 10-second timer before showing connection lost
      // Only show if not already shown
      if (!showConnectionLost) {
        const timer = setTimeout(() => {
          setShowConnectionLost(true)
        }, 10000) // 10 seconds
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
const protectedRoutes = ['/', '/apps', '/apps/new', '/server', '/settings', '/settings/backup', '/team', '/backups', '/cron', '/files', '/terminal', '/activity']

// Routes that should redirect to dashboard if authenticated
const authRoutes = ['/login', '/setup']

function AuthGuard({ children }: { children: React.ReactNode }) {
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

    const isProtectedRoute = protectedRoutes.some(route => pathname.startsWith(route))
    const isAuthRoute = authRoutes.includes(pathname)

    // If no users exist, redirect to setup (but not if already on setup)
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
    if (isAuthenticated && isAuthRoute) {
      router.push('/')
      return
    }
  }, [isLoading, usersExist, isAuthenticated, pathname, router])

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