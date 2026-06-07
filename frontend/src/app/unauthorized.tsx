'use client'

import Link from 'next/link'
import { ArrowLeft, Lock } from 'lucide-react'

export default function Unauthorized() {
  return (
    <div className="min-h-screen bg-slate-900 flex items-center justify-center p-4">
      <div className="text-center max-w-md">
        {/* Lock Icon */}
        <div className="mb-8">
          <div className="w-32 h-32 mx-auto bg-slate-800 rounded-full flex items-center justify-center">
            <Lock className="w-16 h-16 text-slate-500" />
          </div>
        </div>

        {/* Text */}
        <h1 className="text-4xl font-bold text-white mb-4">Access denied</h1>
        <p className="text-slate-400 mb-8">
          You don&apos;t have permission to view this resource.
          <br />
          Contact your team administrator.
        </p>

        {/* Actions */}
        <div className="flex flex-col sm:flex-row items-center justify-center gap-4">
          <button
            onClick={() => window.history.back()}
            className="flex items-center gap-2 px-6 py-3 bg-slate-800 hover:bg-slate-700 text-white rounded-lg font-medium transition-colors border border-slate-700"
          >
            <ArrowLeft className="w-5 h-5" />
            Go Back
          </button>
          <Link
            href="/team"
            className="flex items-center gap-2 px-6 py-3 bg-primary-600 hover:bg-primary-700 text-white rounded-lg font-medium transition-colors"
          >
            Request Access
          </Link>
        </div>
      </div>
    </div>
  )
}