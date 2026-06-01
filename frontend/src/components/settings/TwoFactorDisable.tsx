'use client'

import { useState } from 'react'
import { X, Shield, AlertTriangle, Loader2, Eye, EyeOff } from 'lucide-react'
import { useMutation } from '@tanstack/react-query'
import { api, ApiError } from '@/lib/api'
import { useToastStore } from '@/stores'

interface TwoFactorDisableProps {
  open: boolean
  onClose: () => void
  onDisable: () => void
}

export function TwoFactorDisable({ open, onClose, onDisable }: TwoFactorDisableProps) {
  const { addToast } = useToastStore()
  const [password, setPassword] = useState('')
  const [totpCode, setTotpCode] = useState('')
  const [showPassword, setShowPassword] = useState(false)
  const [step, setStep] = useState<'password' | 'totp'>('password')

  const disableMutation = useMutation({
    mutationFn: (data: { password: string; code: string }) => 
      api.auth.login('disable-2fa', 'verification'), // Mock - would be actual 2FA disable
    onSuccess: () => {
      addToast({ type: 'success', title: '2FA Disabled', message: 'Two-factor authentication has been disabled.' })
      onDisable()
      handleClose()
    },
    onError: (error: ApiError) => {
      addToast({ type: 'error', title: 'Verification failed', message: error.message })
    },
  })

  const handleClose = () => {
    setPassword('')
    setTotpCode('')
    setStep('password')
    setShowPassword(false)
    onClose()
  }

  const handleContinue = () => {
    if (!password) return
    setStep('totp')
  }

  const handleDisable = () => {
    if (totpCode.length !== 6) return
    disableMutation.mutate({ password, code: totpCode })
  }

  if (!open) return null

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center">
      {/* Backdrop */}
      <div className="absolute inset-0 bg-black/50" onClick={handleClose} />

      {/* Modal */}
      <div className="relative bg-slate-800 border border-slate-700 rounded-lg shadow-xl w-full max-w-md mx-4">
        {/* Header */}
        <div className="flex items-center justify-between px-6 py-4 border-b border-slate-700">
          <div className="flex items-center gap-3">
            <div className="w-10 h-10 rounded-full bg-danger-500/10 flex items-center justify-center">
              <Shield className="w-5 h-5 text-danger-500" />
            </div>
            <div>
              <h2 className="text-lg font-semibold text-white">Disable Two-Factor Authentication</h2>
              <p className="text-sm text-slate-400">This will make your account less secure</p>
            </div>
          </div>
          <button
            onClick={handleClose}
            className="p-1 text-slate-400 hover:text-white hover:bg-slate-700 rounded transition-colors"
          >
            <X className="w-5 h-5" />
          </button>
        </div>

        {/* Content */}
        <div className="p-6">
          {/* Warning */}
          <div className="flex items-start gap-3 p-4 bg-amber-500/10 border border-amber-500/20 rounded-lg mb-6">
            <AlertTriangle className="w-5 h-5 text-amber-500 flex-shrink-0 mt-0.5" />
            <div>
              <h3 className="text-sm font-medium text-amber-200">Security Warning</h3>
              <p className="text-sm text-amber-100/80 mt-1">
                Disabling 2FA will remove an important layer of security from your account. 
                Make sure you understand the risks before continuing.
              </p>
            </div>
          </div>

          {step === 'password' && (
            <>
              <div className="mb-6">
                <label className="block text-sm font-medium text-slate-300 mb-2">
                  Enter your password to continue
                </label>
                <div className="relative">
                  <input
                    type={showPassword ? 'text' : 'password'}
                    value={password}
                    onChange={(e) => setPassword(e.target.value)}
                    placeholder="Your current password"
                    className="w-full px-3 py-2 pr-10 bg-slate-900 border border-slate-700 rounded-md text-white placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-primary-500"
                  />
                  <button
                    type="button"
                    onClick={() => setShowPassword(!showPassword)}
                    className="absolute right-3 top-1/2 -translate-y-1/2 text-slate-400 hover:text-white"
                  >
                    {showPassword ? <EyeOff className="w-5 h-5" /> : <Eye className="w-5 h-5" />}
                  </button>
                </div>
              </div>

              <div className="flex items-center justify-end gap-3">
                <button
                  type="button"
                  onClick={handleClose}
                  className="px-4 py-2 text-sm font-medium text-slate-300 hover:text-white hover:bg-slate-700 rounded-md transition-colors"
                >
                  Cancel
                </button>
                <button
                  onClick={handleContinue}
                  disabled={!password}
                  className="px-4 py-2 bg-primary-600 hover:bg-primary-700 disabled:bg-primary-500/50 text-white rounded-md text-sm font-medium transition-colors"
                >
                  Continue
                </button>
              </div>
            </>
          )}

          {step === 'totp' && (
            <>
              <div className="mb-6">
                <label className="block text-sm font-medium text-slate-300 mb-2">
                  Enter the 6-digit code from your authenticator app
                </label>
                <input
                  type="text"
                  value={totpCode}
                  onChange={(e) => setTotpCode(e.target.value.replace(/\D/g, '').slice(0, 6))}
                  placeholder="000000"
                  maxLength={6}
                  className="w-full px-4 py-3 bg-slate-900 border border-slate-700 rounded-lg text-center text-2xl tracking-widest text-white font-mono placeholder-slate-500 focus:outline-none focus:ring-2 focus:ring-primary-500"
                  autoFocus
                />
              </div>

              <div className="flex items-center justify-end gap-3">
                <button
                  type="button"
                  onClick={() => setStep('password')}
                  className="px-4 py-2 text-sm font-medium text-slate-300 hover:text-white hover:bg-slate-700 rounded-md transition-colors"
                >
                  Back
                </button>
                <button
                  onClick={handleDisable}
                  disabled={totpCode.length !== 6 || disableMutation.isPending}
                  className="flex items-center gap-2 px-4 py-2 bg-danger-600 hover:bg-danger-700 disabled:bg-danger-500/50 text-white rounded-md text-sm font-medium transition-colors"
                >
                  {disableMutation.isPending ? (
                    <>
                      <Loader2 className="w-4 h-4 animate-spin" />
                      Disabling...
                    </>
                  ) : (
                    <>
                      <Shield className="w-4 h-4" />
                      Disable 2FA
                    </>
                  )}
                </button>
              </div>
            </>
          )}
        </div>
      </div>
    </div>
  )
}