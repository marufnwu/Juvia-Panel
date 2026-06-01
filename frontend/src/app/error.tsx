'use client'

import Link from 'next/link'
import { RefreshCw, FileText, Home } from 'lucide-react'
import { useEffect, useState } from 'react'

interface ErrorPageProps {
  error: Error & { digest?: string }
  reset: () => void
}

export default function Error({ error, reset }: ErrorPageProps) {
  const [errorId, setErrorId] = useState('')

  useEffect(() => {
    // Generate a unique error ID for tracking
    const id = `err-${Date.now().toString(36)}-${Math.random().toString(36).substring(2, 8)}`
    setErrorId(id)
    console.error('Error:', error)
  }, [error])

  return (
    <div className="min-h-screen bg-slate-900 flex items-center justify-center p-4">
      <div className="text-center max-w-md">
        {/* Error Illustration */}
        <div className="mb-8">
          <div className="w-32 h-32 mx-auto bg-danger-500/10 rounded-full flex items-center justify-center">
            <svg
              className="w-16 h-16 text-danger-500"
              fill="none"
              viewBox="0 0 24 24"
              stroke="currentColor"
            >
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth={1.5}
                d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z"
              />
            </svg>
          </div>
        </div>

        {/* Text */}
        <h1 className="text-4xl font-bold text-white mb-4">Something went wrong</h1>
        <p className="text-slate-400 mb-4">
          The panel encountered an unexpected error. The error has been logged.
        </p>

        {/* Error ID */}
        {errorId && (
          <div className="mb-8">
            <p className="text-xs text-slate-500 mb-1">Error ID</p>
            <code className="px-3 py-1.5 bg-slate-800 rounded text-sm text-slate-400 font-mono">
              {errorId}
            </code>
          </div>
        )}

        {/* Actions */}
        <div className="flex flex-col sm:flex-row items-center justify-center gap-4">
          <button
            onClick={reset}
            className="flex items-center gap-2 px-6 py-3 bg-primary-600 hover:bg-primary-700 text-white rounded-lg font-medium transition-colors"
          >
            <RefreshCw className="w-5 h-5" />
            Reload Page
          </button>
          <Link
            href="/server"
            className="flex items-center gap-2 px-6 py-3 bg-slate-800 hover:bg-slate-700 text-white rounded-lg font-medium transition-colors border border-slate-700"
          >
            <FileText className="w-5 h-5" />
            View Logs
          </Link>
        </div>

        {/* Home Link */}
        <Link
          href="/"
          className="mt-8 inline-flex items-center gap-2 text-sm text-slate-500 hover:text-slate-400 transition-colors"
        >
          <Home className="w-4 h-4" />
          Go to Dashboard
        </Link>
      </div>
    </div>
  )
}