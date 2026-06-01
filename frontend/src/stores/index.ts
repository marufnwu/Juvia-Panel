// Global state store using Zustand
import { create } from 'zustand'
import { persist } from 'zustand/middleware'
import { api } from '@/lib/api'

// Toast interface (defined locally to avoid import conflict)
export interface Toast {
  id: string
  type: 'success' | 'error' | 'warning' | 'info' | 'loading' | 'progress'
  title: string
  message?: string
  duration?: number
  action?: {
    label: string
    onClick: () => void
  }
  undoAction?: () => void
  progress?: number
}

// Auth store
interface AuthState {
  isAuthenticated: boolean
  accessToken: string | null
  user: { id: string; email: string; name: string; role: string } | null
  usersExist: boolean | null
  setAuth: (token: string, user: AuthState['user']) => void
  clearAuth: () => void
  login: (username: string, password: string) => Promise<void>
  logout: () => Promise<void>
  refreshAccessToken: () => Promise<string | null>
  checkUsersExist: () => Promise<boolean>
  register: (email: string, username: string, password: string) => Promise<{ access_token: string; user?: { id: number; email: string; username: string; role: string } }>
}

export const useAuthStore = create<AuthState>((set) => ({
  isAuthenticated: false,
  accessToken: null,
  user: null,
  usersExist: null,
  setAuth: (token, user) => set({ isAuthenticated: true, accessToken: token, user }),
  clearAuth: () => set({ isAuthenticated: false, accessToken: null, user: null }),

  login: async (username: string, password: string) => {
    try {
      const data = await api.auth.login(username, password)
      set({
        isAuthenticated: true,
        accessToken: data.access_token,
        user: data.user ? {
          id: String(data.user.id),
          email: data.user.email || username,
          name: data.user.username || username,
          role: data.user.role || 'viewer',
        } : null,
      })
    } catch (error) {
      set({ isAuthenticated: false, accessToken: null, user: null })
      throw error
    }
  },

  logout: async () => {
    try {
      await api.auth.logout()
    } finally {
      set({ isAuthenticated: false, accessToken: null, user: null })
    }
  },

  refreshAccessToken: async () => {
    try {
      const data = await api.auth.refresh()
      set({ accessToken: data.access_token })
      return data.access_token
    } catch {
      set({ isAuthenticated: false, accessToken: null, user: null })
      return null
    }
  },

  checkUsersExist: async () => {
    try {
      const data = await api.auth.status()
      set({ usersExist: data.users_exist })
      return data.users_exist
    } catch {
      return false
    }
  },

  register: async (email: string, username: string, password: string) => {
    try {
      const data = await api.auth.register(email, username, password)
      set({
        isAuthenticated: true,
        accessToken: data.access_token,
        user: data.user ? {
          id: String(data.user.id),
          email: data.user.email,
          name: data.user.username,
          role: data.user.role,
        } : null,
      })
      return data
    } catch (error) {
      set({ isAuthenticated: false, accessToken: null, user: null })
      throw error
    }
  },
}))

// Theme store
interface ThemeState {
  theme: 'dark' | 'light' | 'system'
  setTheme: (theme: 'dark' | 'light' | 'system') => void
}

export const useThemeStore = create<ThemeState>()(
  persist(
    (set) => ({
      theme: 'dark',
      setTheme: (theme) => set({ theme }),
    }),
    { name: 'panel-theme' }
  )
)

// Toast notifications store
interface ToastState {
  toasts: Toast[]
  addToast: (toast: Omit<Toast, 'id'>) => string
  dismiss: (id: string) => void
  clearToasts: () => void
  // Helper methods for common toast patterns
  showDeploying: (appName: string) => string
  showBackupComplete: (backupName: string) => string
  showErrorWithRetry: (error: string, retryFn: () => void) => string
  updateProgress: (id: string, progress: number) => void
}

export const useToastStore = create<ToastState>((set, get) => ({
  toasts: [],
  
  addToast: (toast) => {
    const id = Math.random().toString(36).substr(2, 9)
    set((state) => ({
      toasts: [...state.toasts, { ...toast, id }],
    }))
    
    // Auto-remove after duration (unless duration is 0 for infinite)
    if (toast.duration !== 0) {
      const duration = toast.duration || 5000
      setTimeout(() => {
        get().dismiss(id)
      }, duration)
    }
    
    return id
  },
  
  dismiss: (id) => set((state) => ({
    toasts: state.toasts.filter((t) => t.id !== id),
  })),
  
  clearToasts: () => set({ toasts: [] }),
  
  // Show deploying toast for an app
  showDeploying: (appName: string) => {
    return get().addToast({
      type: 'progress',
      title: `Deploying ${appName}`,
      message: 'Building and deploying your application...',
      duration: 0, // Infinite until dismissed
      progress: 0,
    })
  },
  
  // Show backup complete toast
  showBackupComplete: (backupName: string) => {
    return get().addToast({
      type: 'success',
      title: 'Backup Complete',
      message: `${backupName} has been backed up successfully.`,
      duration: 5000,
    })
  },
  
  // Show error with retry action
  showErrorWithRetry: (error: string, retryFn: () => void) => {
    return get().addToast({
      type: 'error',
      title: 'Operation Failed',
      message: error,
      duration: 10000,
      action: {
        label: 'Retry',
        onClick: retryFn,
      },
    })
  },
  
  // Update progress for a toast
  updateProgress: (id: string, progress: number) => {
    set((state) => ({
      toasts: state.toasts.map((t) =>
        t.id === id ? { ...t, progress } : t
      ),
    }))
  },
}))

// Command palette store
interface CommandPaletteState {
  isOpen: boolean
  open: () => void
  close: () => void
  toggle: () => void
}

export const useCommandPaletteStore = create<CommandPaletteState>((set) => ({
  isOpen: false,
  open: () => set({ isOpen: true }),
  close: () => set({ isOpen: false }),
  toggle: () => set((state) => ({ isOpen: !state.isOpen })),
}))

// Sidebar store (for mobile)
interface SidebarState {
  isOpen: boolean
  open: () => void
  close: () => void
  toggle: () => void
}

export const useSidebarStore = create<SidebarState>((set) => ({
  isOpen: false,
  open: () => set({ isOpen: true }),
  close: () => set({ isOpen: false }),
  toggle: () => set((state) => ({ isOpen: !state.isOpen })),
}))

// Notification panel store
interface NotificationState {
  isOpen: boolean
  unreadCount: number
  open: () => void
  close: () => void
  toggle: () => void
  setUnreadCount: (count: number) => void
  clearUnread: () => void
}

export const useNotificationStore = create<NotificationState>((set) => ({
  isOpen: false,
  unreadCount: 0,
  open: () => set({ isOpen: true }),
  close: () => set({ isOpen: false }),
  toggle: () => set((state) => ({ isOpen: !state.isOpen })),
  setUnreadCount: (count) => set({ unreadCount: count }),
  clearUnread: () => set({ unreadCount: 0 }),
}))