'use client'

import { useState, useEffect, useCallback } from 'react'
import { WifiOff, X, RefreshCw } from 'lucide-react'

interface ConnectionLostProps {
  onRetry: () => void
  onCancel?: () => void
  autoRetrySeconds?: number
}

export function ConnectionLost({
  onRetry,
  onCancel,
  autoRetrySeconds = 5
}: ConnectionLostProps) {
  const [countdown, setCountdown] = useState(autoRetrySeconds)
  const [isRetrying, setIsRetrying] = useState(false)

  const handleRetry = useCallback(() => {
    setIsRetrying(true)
    onRetry()
    setTimeout(() => setIsRetrying(false), 2000)
  }, [onRetry])

  useEffect(() => {
    if (countdown <= 0) {
      handleRetry()
      return
    }

    const timer = setTimeout(() => {
      setCountdown(countdown - 1)
    }, 1000)

    return () => clearTimeout(timer)
  }, [countdown, handleRetry])

  const handleCancel = () => {
    onCancel?.()
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 backdrop-blur-sm">
      <div className="bg-slate-800 border border-slate-700 rounded-xl shadow-2xl w-full max-w-sm mx-4 p-6">
        {/* Icon */}
        <div className="flex items-center justify-center mb-4">
          <div className="w-16 h-16 bg-danger-500/10 rounded-full flex items-center justify-center">
            <WifiOff className="w-8 h-8 text-danger-500" />
          </div>
        </div>

        {/* Text */}
        <div className="text-center mb-6">
          <h2 className="text-xl font-semibold text-white mb-2">Connection lost</h2>
          <p className="text-sm text-slate-400">
            Real-time updates are unavailable. Check your network connection.
          </p>
        </div>

        {/* Auto-retry countdown */}
        <div className="text-center mb-6">
          <p className="text-sm text-slate-500 mb-2">Auto-retrying in</p>
          <div className="flex items-center justify-center gap-2">
            <span className="text-3xl font-bold text-white">{countdown}</span>
            <span className="text-sm text-slate-400">seconds</span>
          </div>
        </div>

        {/* Progress bar */}
        <div className="w-full h-1 bg-slate-700 rounded-full mb-6 overflow-hidden">
          <div 
            className="h-full bg-primary-500 transition-all duration-1000 ease-linear"
            style={{ width: `${(countdown / autoRetrySeconds) * 100}%` }}
          />
        </div>

        {/* Actions */}
        <div className="flex items-center gap-3">
          {onCancel && (
            <button
              onClick={handleCancel}
              className="flex-1 flex items-center justify-center gap-2 px-4 py-2.5 bg-slate-700 hover:bg-slate-600 text-white rounded-lg text-sm font-medium transition-colors"
            >
              <X className="w-4 h-4" />
              Cancel
            </button>
          )}
          <button
            onClick={handleRetry}
            disabled={isRetrying}
            className="flex-1 flex items-center justify-center gap-2 px-4 py-2.5 bg-primary-600 hover:bg-primary-700 disabled:bg-primary-500/50 text-white rounded-lg text-sm font-medium transition-colors"
          >
            {isRetrying ? (
              <>
                <RefreshCw className="w-4 h-4 animate-spin" />
                Retrying...
              </>
            ) : (
              <>
                <RefreshCw className="w-4 h-4" />
                Retry Now
              </>
            )}
          </button>
        </div>
      </div>
    </div>
  )
}

export default ConnectionLost