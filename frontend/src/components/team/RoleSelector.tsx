'use client'

import { useState } from 'react'
import { X, Shield, AlertTriangle, Loader2, Check } from 'lucide-react'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { api, ApiError } from '@/lib/api'
import { useToastStore } from '@/stores'

interface RoleSelectorProps {
  open: boolean
  onClose: () => void
  member: {
    id: string
    name: string
    email: string
    role: 'owner' | 'admin' | 'developer' | 'viewer'
  }
}

type UserRole = 'admin' | 'developer' | 'viewer'

const roles: { value: UserRole; label: string; description: string; permissions: string[] }[] = [
  {
    value: 'developer',
    label: 'Developer',
    description: 'Can deploy apps, manage environment variables, and view logs',
    permissions: ['Deploy applications', 'Manage environment variables', 'View logs', 'Restart apps'],
  },
  {
    value: 'admin',
    label: 'Admin',
    description: 'Full access to manage apps, services, and team members',
    permissions: ['All Developer permissions', 'Create/delete apps', 'Manage services', 'Manage team members', 'Update server settings'],
  },
  {
    value: 'viewer',
    label: 'Viewer',
    description: 'Read-only access to view all resources',
    permissions: ['View apps and services', 'View logs (read-only)', 'View server metrics'],
  },
]

export function RoleSelector({ open, onClose, member }: RoleSelectorProps) {
  const queryClient = useQueryClient()
  const { addToast } = useToastStore()
  const [selectedRole, setSelectedRole] = useState<UserRole>(member.role as UserRole)
  const [confirmText, setConfirmText] = useState('')
  const [step, setStep] = useState<'select' | 'confirm'>('select')

  const updateRoleMutation = useMutation({
    mutationFn: (data: { user_id: string; role: UserRole }) => api.users.updateRole(data.user_id, data.role),
    onSuccess: () => {
      addToast({ type: 'success', title: 'Role updated', message: `${member.name}'s role has been updated.` })
      queryClient.invalidateQueries({ queryKey: ['team'] })
      handleClose()
    },
    onError: (error: ApiError) => {
      addToast({ type: 'error', title: 'Failed to update role', message: error.message })
    },
  })

  const handleClose = () => {
    setSelectedRole(member.role as UserRole)
    setConfirmText('')
    setStep('select')
    onClose()
  }

  const handleConfirm = () => {
    if (confirmText.toLowerCase() !== 'change role') {
      return
    }
    updateRoleMutation.mutate({ user_id: member.id, role: selectedRole })
  }

  const handleRoleSelect = (role: UserRole) => {
    setSelectedRole(role)
    if (role !== member.role) {
      setStep('confirm')
    }
  }

  if (!open) return null

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center">
      {/* Backdrop */}
      <div className="absolute inset-0 bg-black/50" onClick={handleClose} />

      {/* Modal */}
      <div className="relative bg-slate-800 border border-slate-700 rounded-lg shadow-xl w-full max-w-lg mx-4">
        {/* Header */}
        <div className="flex items-center justify-between px-6 py-4 border-b border-slate-700">
          <div className="flex items-center gap-3">
            <div className="w-10 h-10 rounded-full bg-primary-600 flex items-center justify-center">
              <span className="text-sm font-medium text-white">
                {member.name.charAt(0).toUpperCase()}
              </span>
            </div>
            <div>
              <h2 className="text-lg font-semibold text-white">Change Role</h2>
              <p className="text-sm text-slate-400">{member.name} ({member.email})</p>
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
          {step === 'select' ? (
            <>
              <p className="text-sm text-slate-400 mb-4">
                Select a new role for this team member. They will receive a notification of the change.
              </p>

              <div className="space-y-3 mb-6">
                {roles.map((role) => {
                  const isSelected = selectedRole === role.value
                  const isCurrentRole = member.role === role.value
                  return (
                    <button
                      key={role.value}
                      onClick={() => handleRoleSelect(role.value)}
                      disabled={isCurrentRole}
                      className={`
                        w-full text-left p-4 rounded-lg border transition-colors
                        ${isSelected
                          ? 'bg-primary-500/10 border-primary-500'
                          : isCurrentRole
                            ? 'bg-slate-700/50 border-slate-600 opacity-50 cursor-not-allowed'
                            : 'bg-slate-900 border-slate-700 hover:border-slate-600'
                        }
                      `}
                    >
                      <div className="flex items-start justify-between mb-2">
                        <div>
                          <h3 className="text-sm font-medium text-white flex items-center gap-2">
                            {role.label}
                            {isCurrentRole && (
                              <span className="text-xs text-slate-500">(current)</span>
                            )}
                          </h3>
                          <p className="text-xs text-slate-400 mt-1">{role.description}</p>
                        </div>
                        <div className={`w-5 h-5 rounded-full border-2 flex items-center justify-center ${
                          isSelected ? 'border-primary-500' : 'border-slate-500'
                        }`}>
                          {isSelected && <div className="w-2.5 h-2.5 rounded-full bg-primary-500" />}
                        </div>
                      </div>
                      {isSelected && (
                        <ul className="mt-3 space-y-1">
                          {role.permissions.map((perm, idx) => (
                            <li key={idx} className="flex items-center gap-2 text-xs text-slate-300">
                              <Check className="w-3 h-3 text-green-500" />
                              {perm}
                            </li>
                          ))}
                        </ul>
                      )}
                    </button>
                  )
                })}
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
                {selectedRole !== member.role && (
                  <button
                    onClick={() => setStep('confirm')}
                    className="px-4 py-2 bg-primary-600 hover:bg-primary-700 text-white rounded-md text-sm font-medium transition-colors"
                  >
                    Continue
                  </button>
                )}
              </div>
            </>
          ) : (
            <>
              {/* Confirmation Step */}
              <div className="mb-6">
                <div className="flex items-start gap-3 p-4 bg-amber-500/10 border border-amber-500/20 rounded-lg mb-4">
                  <AlertTriangle className="w-5 h-5 text-amber-500 flex-shrink-0 mt-0.5" />
                  <div>
                    <h3 className="text-sm font-medium text-amber-200">Confirm Role Change</h3>
                    <p className="text-sm text-amber-100/80 mt-1">
                      You are about to change {member.name}'s role from{' '}
                      <span className="font-medium">{member.role}</span> to{' '}
                      <span className="font-medium">{selectedRole}</span>.
                    </p>
                  </div>
                </div>

                <label className="block text-sm font-medium text-slate-300 mb-2">
                  Type "change role" to confirm
                </label>
                <input
                  type="text"
                  value={confirmText}
                  onChange={(e) => setConfirmText(e.target.value)}
                  placeholder="change role"
                  className="w-full px-3 py-2 bg-slate-900 border border-slate-700 rounded-md text-white placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-primary-500 focus:border-transparent"
                />
              </div>

              {/* Actions */}
              <div className="flex items-center justify-end gap-3">
                <button
                  type="button"
                  onClick={() => setStep('select')}
                  className="px-4 py-2 text-sm font-medium text-slate-300 hover:text-white hover:bg-slate-700 rounded-md transition-colors"
                >
                  Back
                </button>
                <button
                  onClick={handleConfirm}
                  disabled={confirmText.toLowerCase() !== 'change role' || updateRoleMutation.isPending}
                  className="flex items-center gap-2 px-4 py-2 bg-amber-600 hover:bg-amber-700 disabled:bg-amber-500/50 text-white rounded-md text-sm font-medium transition-colors"
                >
                  {updateRoleMutation.isPending ? (
                    <>
                      <Loader2 className="w-4 h-4 animate-spin" />
                      Updating...
                    </>
                  ) : (
                    <>
                      <Shield className="w-4 h-4" />
                      Change Role
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