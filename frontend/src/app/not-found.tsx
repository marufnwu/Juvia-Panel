'use client'

import Link from 'next/link'
import { Home, Search } from 'lucide-react'
import { useCommandPaletteStore } from '@/stores'

export default function NotFound() {
  const { open } = useCommandPaletteStore()

  return (
    <div className="min-h-screen bg-slate-900 flex items-center justify-center p-4">
      <div className="text-center max-w-md">
        {/* 404 Illustration */}
        <div className="mb-8">
          <div className="w-32 h-32 mx-auto bg-slate-800 rounded-full flex items-center justify-center">
            <svg
              className="w-16 h-16 text-slate-500"
              fill="none"
              viewBox="0 0 24 24"
              stroke="currentColor"
            >
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth={1.5}
                d="M9.172 16.172a4 4 0 015.656 0M9 10h.01M15 10h.01M12 2a10 10 0 110 20 10 10 0 010-20z"
              />
            </svg>
          </div>
        </div>

        {/* Text */}
        <h1 className="text-4xl font-bold text-white mb-4">404</h1>
        <h2 className="text-xl font-semibold text-white mb-2">Page not found</h2>
        <p className="text-slate-400 mb-8">
          The page you&apos;re looking for doesn&apos;t exist or has been moved.
        </p>

        {/* Actions */}
        <div className="flex flex-col sm:flex-row items-center justify-center gap-4">
          <Link
            href="/"
            className="flex items-center gap-2 px-6 py-3 bg-primary-600 hover:bg-primary-700 text-white rounded-lg font-medium transition-colors"
          >
            <Home className="w-5 h-5" />
            Go to Dashboard
          </Link>
          <button
            onClick={open}
            className="flex items-center gap-2 px-6 py-3 bg-slate-800 hover:bg-slate-700 text-white rounded-lg font-medium transition-colors border border-slate-700"
          >
            <Search className="w-5 h-5" />
            Open Command Palette
          </button>
        </div>

        {/* Keyboard Hint */}
        <p className="mt-8 text-sm text-slate-500">
          Press <kbd className="px-2 py-1 bg-slate-800 rounded text-slate-400">Cmd+K</kbd> to search
        </p>
      </div>
    </div>
  )
}