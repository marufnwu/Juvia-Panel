'use client'

import { useState } from 'react'
import { X, Mail, UserPlus, Copy, Check, Loader2 } from 'lucide-react'
import { useMutation } from '@tanstack/react-query'
import { api, ApiError } from '@/lib/api'
import { useToastStore } from '@/stores'

interface InviteModalProps {
  open: boolean
  onClose: () => void
}

type UserRole = 'admin' | 'developer' | 'viewer'

const roles: { value: UserRole; label: string; description: string }[] = [
  { value: 'developer', label: 'Developer', description: 'Deploy apps, manage env vars, view logs' },
  { value: 'admin', label: 'Admin', description: 'Manage apps, services, users' },
  { value: 'viewer', label: 'Viewer', description: 'Read-only access to all resources' },
]

export function InviteModal({ open, onClose }: InviteModalProps) {
  const { addToast } = useToastStore()
  const [email, setEmail] = useState('')
  const [role, setRole] = useState<UserRole>('developer')
  const [inviteLink, setInviteLink] = useState<string | null>(null)
  const [copied, setCopied] = useState(false)

  const inviteMutation = useMutation({
    mutationFn: (data: { email: string; role: UserRole }) => api.users.invite(data.email, data.role),
    onSuccess: (data: { invite_url?: string }) => {
      if (data.invite_url) {
        setInviteLink(data.invite_url)
        addToast({ type: 'success', title: 'Invitation sent', message: 'An invitation link has been generated.' })
      } else {
        addToast({ type: 'success', title: 'Invitation sent', message: `Invitation sent to ${email}` })
        handleClose()
      }
    },
    onError: (error: ApiError) => {
      addToast({ type: 'error', title: 'Failed to send invitation', message: error.message })
    },
  })

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault()
    if (!email) return
    inviteMutation.mutate({ email, role })
  }

  const handleCopyLink = () => {
    if (inviteLink) {
      navigator.clipboard.writeText(inviteLink)
      setCopied(true)
      setTimeout(() => setCopied(false), 2000)
    }
  }

  const handleClose = () => {
    setEmail('')
    setRole('developer')
    setInviteLink(null)
    setCopied(false)
    onClose()
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
          <h2 className="text-lg font-semibold text-white">Invite Team Member</h2>
          <button
            onClick={handleClose}
            className="p-1 text-slate-400 hover:text-white hover:bg-slate-700 rounded transition-colors"
          >
            <X className="w-5 h-5" />
          </button>
        </div>

        {/* Content */}
        <form onSubmit={handleSubmit} className="p-6">
          {!inviteLink ? (
            <>
              {/* Email Input */}
              <div className="mb-4">
                <label htmlFor="email" className="block text-sm font-medium text-slate-300 mb-2">
                  Email Address
                </label>
                <div className="relative">
                  <Mail className="absolute left-3 top-1/2 -translate-y-1/2 w-5 h-5 text-slate-400" />
                  <input
                    type="email"
                    id="email"
                    value={email}
                    onChange={(e) => setEmail(e.target.value)}
                    placeholder="colleague@example.com"
                    className="w-full pl-10 pr-4 py-2.5 bg-slate-900 border border-slate-700 rounded-md text-white placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-primary-500 focus:border-transparent"
                    required
                  />
                </div>
              </div>

              {/* Role Selector */}
              <div className="mb-6">
                <label className="block text-sm font-medium text-slate-300 mb-2">
                  Role
                </label>
                <div className="space-y-2">
                  {roles.map((r) => (
                    <label
                      key={r.value}
                      className={`
                        flex items-center gap-3 p-3 rounded-md cursor-pointer border transition-colors
                        ${role === r.value
                          ? 'bg-primary-500/10 border-primary-500'
                          : 'bg-slate-900 border-slate-700 hover:border-slate-600'
                        }
                      `}
                    >
                      <input
                        type="radio"
                        name="role"
                        value={r.value}
                        checked={role === r.value}
                        onChange={() => setRole(r.value)}
                        className="sr-only"
                      />
                      <div className={`w-4 h-4 rounded-full border-2 flex items-center justify-center ${
                        role === r.value ? 'border-primary-500' : 'border-slate-500'
                      }`}>
                        {role === r.value && (
                          <div className="w-2 h-2 rounded-full bg-primary-500" />
                        )}
                      </div>
                      <div>
                        <p className="text-sm font-medium text-white">{r.label}</p>
                        <p className="text-xs text-slate-400">{r.description}</p>
                      </div>
                    </label>
                  ))}
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
                  type="submit"
                  disabled={inviteMutation.isPending || !email}
                  className="flex items-center gap-2 px-4 py-2 bg-primary-600 hover:bg-primary-700 disabled:bg-primary-500/50 text-white rounded-md text-sm font-medium transition-colors"
                >
                  {inviteMutation.isPending ? (
                    <>
                      <Loader2 className="w-4 h-4 animate-spin" />
                      Sending...
                    </>
                  ) : (
                    <>
                      <UserPlus className="w-4 h-4" />
                      Send Invitation
                    </>
                  )}
                </button>
              </div>
            </>
          ) : (
            <>
              {/* Success - Show Invite Link */}
              <div className="text-center mb-6">
                <div className="w-12 h-12 bg-green-500/10 rounded-full flex items-center justify-center mx-auto mb-4">
                  <Check className="w-6 h-6 text-green-500" />
                </div>
                <h3 className="text-lg font-medium text-white mb-2">Invitation Created</h3>
                <p className="text-sm text-slate-400">
                  Share this link with {email} to complete their invitation.
                </p>
              </div>

              {/* Invite Link */}
              <div className="mb-6">
                <label className="block text-sm font-medium text-slate-300 mb-2">
                  Invitation Link
                </label>
                <div className="flex items-center gap-2">
                  <input
                    type="text"
                    readOnly
                    value={inviteLink}
                    className="flex-1 px-3 py-2 bg-slate-900 border border-slate-700 rounded-md text-sm text-white"
                  />
                  <button
                    type="button"
                    onClick={handleCopyLink}
                    className="p-2 bg-slate-700 hover:bg-slate-600 rounded-md transition-colors"
                  >
                    {copied ? (
                      <Check className="w-5 h-5 text-green-500" />
                    ) : (
                      <Copy className="w-5 h-5 text-slate-300" />
                    )}
                  </button>
                </div>
                <p className="mt-2 text-xs text-slate-500">
                  This link will expire in 7 days.
                </p>
              </div>

              {/* Actions */}
              <div className="flex items-center justify-center">
                <button
                  type="button"
                  onClick={handleClose}
                  className="px-4 py-2 bg-primary-600 hover:bg-primary-700 text-white rounded-md text-sm font-medium transition-colors"
                >
                  Done
                </button>
              </div>
            </>
          )}
        </form>
      </div>
    </div>
  )
}