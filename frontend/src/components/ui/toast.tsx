'use client'

import { useEffect, useState } from 'react'
import { X, CheckCircle, AlertCircle, AlertTriangle, Info, Loader2, Undo2 } from 'lucide-react'
import { useToastStore, Toast } from '@/stores'

const icons = {
  success: CheckCircle,
  error: AlertCircle,
  warning: AlertTriangle,
  info: Info,
  loading: Loader2,
  progress: Loader2,
}

const colors = {
  success: 'bg-success-600 border-success-500',
  error: 'bg-danger-600 border-danger-500',
  warning: 'bg-warning-600 border-warning-500',
  info: 'bg-primary-600 border-primary-500',
  loading: 'bg-slate-700 border-slate-600',
  progress: 'bg-slate-700 border-slate-600',
}

const iconColors = {
  success: 'text-success-400',
  error: 'text-danger-400',
  warning: 'text-warning-400',
  info: 'text-primary-400',
  loading: 'text-slate-400',
  progress: 'text-primary-400',
}

interface ToastItemProps {
  toast: Toast
  onDismiss: () => void
}

function ToastItem({ toast, onDismiss }: ToastItemProps) {
  const [progress, setProgress] = useState(100)
  const Icon = icons[toast.type] || Info

  useEffect(() => {
    if (toast.duration === 0 || toast.progress !== undefined) {
      // Progress toast or infinite duration - no auto-dismiss
      if (toast.progress !== undefined) {
        setProgress(toast.progress)
      }
      return
    }

    const timer = setTimeout(onDismiss, toast.duration || 5000)
    
    // Progress update interval
    const startTime = Date.now()
    const interval = setInterval(() => {
      const elapsed = Date.now() - startTime
      const remaining = Math.max(0, 100 - (elapsed / (toast.duration || 5000)) * 100)
      setProgress(remaining)
    }, 100)

    return () => {
      clearTimeout(timer)
      clearInterval(interval)
    }
  }, [toast.duration, toast.progress, onDismiss])

  return (
    <div
      className={`
        relative flex items-start gap-3 p-4 rounded-lg border shadow-lg overflow-hidden
        bg-slate-800 ${colors[toast.type] || colors.info}
        animate-in slide-in-from-right duration-300
      `}
    >
      {/* Progress bar for progress toasts */}
      {toast.type === 'progress' && (
        <div 
          className="absolute bottom-0 left-0 h-1 bg-primary-500 transition-all duration-100"
          style={{ width: `${progress}%` }}
        />
      )}

      {/* Icon */}
      <Icon className={`w-5 h-5 flex-shrink-0 ${iconColors[toast.type] || iconColors.info} ${toast.type === 'loading' || toast.type === 'progress' ? 'animate-spin' : ''}`} />

      {/* Content */}
      <div className="flex-1 min-w-0">
        <p className="text-sm font-medium text-white">{toast.title}</p>
        {toast.message && (
          <p className="mt-1 text-sm text-slate-300">{toast.message}</p>
        )}
        
        {/* Action buttons */}
        {toast.action && (
          <button
            onClick={() => {
              toast.action?.onClick()
              onDismiss()
            }}
            className="mt-2 px-3 py-1 bg-slate-700 hover:bg-slate-600 rounded text-xs font-medium text-white transition-colors"
          >
            {toast.action.label}
          </button>
        )}

        {/* Undo action */}
        {toast.undoAction && (
          <button
            onClick={() => {
              toast.undoAction?.()
              onDismiss()
            }}
            className="mt-2 flex items-center gap-1 px-3 py-1 bg-slate-700 hover:bg-slate-600 rounded text-xs font-medium text-white transition-colors"
          >
            <Undo2 className="w-3 h-3" />
            Undo
          </button>
        )}
      </div>

      {/* Dismiss button */}
      <button
        onClick={onDismiss}
        className="flex-shrink-0 p-1 text-slate-400 hover:text-white transition-colors"
      >
        <X className="w-4 h-4" />
      </button>
    </div>
  )
}

export function ToastProvider() {
  const { toasts, dismiss } = useToastStore()

  return (
    <div className="fixed bottom-4 right-4 z-50 flex flex-col gap-2 w-full max-w-sm">
      {toasts.map((toast) => (
        <ToastItem
          key={toast.id}
          toast={toast}
          onDismiss={() => dismiss(toast.id)}
        />
      ))}
    </div>
  )
}

// Toast container for use outside of components
export function ToastContainer() {
  return <ToastProvider />
}