'use client'

import { useState } from 'react'
import { X, Smartphone, Copy, Check, Loader2, Download } from 'lucide-react'
import { useMutation } from '@tanstack/react-query'
import { api, ApiError } from '@/lib/api'
import { useToastStore } from '@/stores'

interface TwoFactorSetupProps {
  open: boolean
  onClose: () => void
}

export function TwoFactorSetup({ open, onClose }: TwoFactorSetupProps) {
  const { addToast } = useToastStore()
  const [step, setStep] = useState<'qr' | 'verify' | 'backup'>('qr')
  const [verificationCode, setVerificationCode] = useState('')
  const [copied, setCopied] = useState(false)

  // Mock data - in production this would come from API
  const qrCodeUrl = 'https://api.qrserver.com/v1/create-qr-code/?size=200x200&data=otpauth://totp/ServerPanel:admin@example.com?secret=JBSWY3DPEHPK3PXP&issuer=ServerPanel'
  const secretKey = 'JBSWY3DPEHPK3PXP'
  const backupCodes = [
    'A1B2C3D4', 'E5F6G7H8', 'I9J0K1L2', 'M3N4O5P6',
    'Q7R8S9T0', 'U1V2W3X4', 'Y5Z6A7B8', 'C9D0E1F2'
  ]

  const verifyMutation = useMutation({
    mutationFn: (code: string) => api.auth.login('verification', code), // Mock - would be actual 2FA verification
    onSuccess: () => {
      addToast({ type: 'success', title: '2FA Enabled', message: 'Two-factor authentication has been enabled on your account.' })
      handleClose()
    },
    onError: (error: ApiError) => {
      addToast({ type: 'error', title: 'Verification failed', message: error.message })
    },
  })

  const handleClose = () => {
    setVerificationCode('')
    setStep('qr')
    setCopied(false)
    onClose()
  }

  const handleCopySecret = () => {
    navigator.clipboard.writeText(secretKey)
    setCopied(true)
    setTimeout(() => setCopied(false), 2000)
  }

  const handleCopyBackupCodes = () => {
    navigator.clipboard.writeText(backupCodes.join('\n'))
    addToast({ type: 'success', title: 'Copied', message: 'Backup codes copied to clipboard.' })
  }

  const handleVerify = () => {
    if (verificationCode.length === 6) {
      verifyMutation.mutate(verificationCode)
    }
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
            <div className="w-10 h-10 rounded-full bg-primary-500/10 flex items-center justify-center">
              <Smartphone className="w-5 h-5 text-primary-500" />
            </div>
            <div>
              <h2 className="text-lg font-semibold text-white">Enable Two-Factor Authentication</h2>
              <p className="text-sm text-slate-400">Add an extra layer of security</p>
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
          {step === 'qr' && (
            <>
              <div className="text-center mb-6">
                <div className="w-48 h-48 mx-auto bg-white rounded-lg flex items-center justify-center mb-4">
                  {/* QR Code - using placeholder since we can't generate actual QR */}
                  <div className="w-40 h-40 bg-slate-200 flex items-center justify-center">
                    <span className="text-slate-500 text-sm">QR Code</span>
                  </div>
                </div>
                <p className="text-sm text-slate-400 mb-4">
                  Scan this QR code with your authenticator app (Google Authenticator, Authy, etc.)
                </p>
              </div>

              {/* Secret Key */}
              <div className="mb-6">
                <label className="block text-sm font-medium text-slate-300 mb-2">
                  Or enter this code manually:
                </label>
                <div className="flex items-center gap-2">
                  <code className="flex-1 px-3 py-2 bg-slate-900 border border-slate-700 rounded text-sm text-white font-mono">
                    {secretKey}
                  </code>
                  <button
                    onClick={handleCopySecret}
                    className="p-2 bg-slate-700 hover:bg-slate-600 rounded transition-colors"
                  >
                    {copied ? (
                      <Check className="w-5 h-5 text-green-500" />
                    ) : (
                      <Copy className="w-5 h-5 text-slate-300" />
                    )}
                  </button>
                </div>
              </div>

              {/* Actions */}
              <div className="flex items-center justify-end gap-3">
                <button
                  type="button"
                  onClick={handleClose}
                  className="px-4 py-2 text-sm font-medium text-slate-300 hover:text-white hover:bg-slate-700 rounded-md transition-colors"
                >
                  Cancel
                </button>
                <button
                  onClick={() => setStep('verify')}
                  className="px-4 py-2 bg-primary-600 hover:bg-primary-700 text-white rounded-md text-sm font-medium transition-colors"
                >
                  Continue
                </button>
              </div>
            </>
          )}

          {step === 'verify' && (
            <>
              <div className="text-center mb-6">
                <div className="w-12 h-12 bg-primary-500/10 rounded-full flex items-center justify-center mx-auto mb-4">
                  <Smartphone className="w-6 h-6 text-primary-500" />
                </div>
                <h3 className="text-lg font-medium text-white mb-2">Verify Code</h3>
                <p className="text-sm text-slate-400">
                  Enter the 6-digit code from your authenticator app
                </p>
              </div>

              <div className="mb-6">
                <input
                  type="text"
                  value={verificationCode}
                  onChange={(e) => setVerificationCode(e.target.value.replace(/\D/g, '').slice(0, 6))}
                  placeholder="000000"
                  maxLength={6}
                  className="w-full px-4 py-3 bg-slate-900 border border-slate-700 rounded-lg text-center text-2xl tracking-widest text-white font-mono placeholder-slate-500 focus:outline-none focus:ring-2 focus:ring-primary-500"
                  autoFocus
                />
              </div>

              <div className="flex items-center justify-end gap-3">
                <button
                  type="button"
                  onClick={() => setStep('qr')}
                  className="px-4 py-2 text-sm font-medium text-slate-300 hover:text-white hover:bg-slate-700 rounded-md transition-colors"
                >
                  Back
                </button>
                <button
                  onClick={handleVerify}
                  disabled={verificationCode.length !== 6 || verifyMutation.isPending}
                  className="flex items-center gap-2 px-4 py-2 bg-primary-600 hover:bg-primary-700 disabled:bg-primary-500/50 text-white rounded-md text-sm font-medium transition-colors"
                >
                  {verifyMutation.isPending ? (
                    <>
                      <Loader2 className="w-4 h-4 animate-spin" />
                      Verifying...
                    </>
                  ) : (
                    'Verify & Enable'
                  )}
                </button>
              </div>
            </>
          )}

          {step === 'backup' && (
            <>
              <div className="text-center mb-6">
                <div className="w-12 h-12 bg-green-500/10 rounded-full flex items-center justify-center mx-auto mb-4">
                  <Check className="w-6 h-6 text-green-500" />
                </div>
                <h3 className="text-lg font-medium text-white mb-2">2FA Enabled!</h3>
                <p className="text-sm text-slate-400">
                  Save your backup codes in a safe place. You can use these codes to access your account if you lose your authenticator.
                </p>
              </div>

              {/* Backup Codes */}
              <div className="mb-6">
                <div className="flex items-center justify-between mb-2">
                  <label className="block text-sm font-medium text-slate-300">Backup Codes</label>
                  <button
                    onClick={handleCopyBackupCodes}
                    className="flex items-center gap-1 text-sm text-primary-400 hover:text-primary-300"
                  >
                    <Copy className="w-4 h-4" />
                    Copy all
                  </button>
                </div>
                <div className="grid grid-cols-2 gap-2">
                  {backupCodes.map((code, idx) => (
                    <div key={idx} className="px-3 py-2 bg-slate-900 border border-slate-700 rounded text-sm text-white font-mono">
                      {code}
                    </div>
                  ))}
                </div>
                <p className="mt-2 text-xs text-amber-500 flex items-center gap-1">
                  Store these codes securely. Each code can only be used once.
                </p>
              </div>

              <div className="flex items-center justify-center">
                <button
                  onClick={handleClose}
                  className="px-6 py-2 bg-primary-600 hover:bg-primary-700 text-white rounded-md text-sm font-medium transition-colors"
                >
                  Done
                </button>
              </div>
            </>
          )}
        </div>
      </div>
    </div>
  )
}