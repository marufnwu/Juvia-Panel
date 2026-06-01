'use client'

import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { Plus, Copy, Check, Trash2, Key, Loader2, Eye, EyeOff } from 'lucide-react'
import { api, ApiError } from '@/lib/api'
import { useToastStore } from '@/stores'

interface APIKey {
  id: string
  name: string
  scopes: string
  last_used_at: string | null
  created_at: string
  expires_at: string | null
  masked_token: string
}

interface CreateKeyModalProps {
  open: boolean
  onClose: () => void
  onCreated: () => void
}

function CreateKeyModal({ open, onClose, onCreated }: CreateKeyModalProps) {
  const { addToast } = useToastStore()
  const [name, setName] = useState('')
  const [scopes, setScopes] = useState<string[]>(['read'])
  const [newKey, setNewKey] = useState<string | null>(null)
  const [copied, setCopied] = useState(false)

  const availableScopes = [
    { value: 'read', label: 'Read', description: 'View apps, services, and server info' },
    { value: 'deploy', label: 'Deploy', description: 'Deploy apps and manage deployments' },
    { value: 'manage', label: 'Manage', description: 'Create and delete apps and services' },
    { value: 'admin', label: 'Admin', description: 'Full administrative access' },
  ]

  const createMutation = useMutation({
    mutationFn: (data: { name: string; scopes: string; expires_in?: number }) =>
      api.users.createApiKey(data),
    onSuccess: (result) => {
      setNewKey(result.token)
      addToast({ type: 'success', title: 'API Key Created', message: 'Store this key securely - it won\'t be shown again.' })
      onCreated()
    },
    onError: (error: ApiError) => {
      addToast({ type: 'error', title: 'Failed to create key', message: error.message })
    },
  })

  const handleClose = () => {
    setName('')
    setScopes(['read'])
    setNewKey(null)
    setCopied(false)
    onClose()
  }

  const handleCopy = () => {
    if (newKey) {
      navigator.clipboard.writeText(newKey)
      setCopied(true)
      setTimeout(() => setCopied(false), 2000)
    }
  }

  const handleDone = () => {
    handleClose()
  }

  const toggleScope = (scope: string) => {
    if (scopes.includes(scope)) {
      setScopes(scopes.filter(s => s !== scope))
    } else {
      setScopes([...scopes, scope])
    }
  }

  if (!open) return null

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center">
      <div className="absolute inset-0 bg-black/50" onClick={handleClose} />

      <div className="relative bg-slate-800 border border-slate-700 rounded-lg shadow-xl w-full max-w-md mx-4">
        <div className="flex items-center justify-between px-6 py-4 border-b border-slate-700">
          <h2 className="text-lg font-semibold text-white">Create API Key</h2>
          <button onClick={handleClose} className="p-1 text-slate-400 hover:text-white hover:bg-slate-700 rounded transition-colors">
            ×
          </button>
        </div>

        <div className="p-6">
          {!newKey ? (
            <>
              <div className="mb-4">
                <label className="block text-sm font-medium text-slate-300 mb-2">Name</label>
                <input
                  type="text"
                  value={name}
                  onChange={(e) => setName(e.target.value)}
                  placeholder="My API Key"
                  className="w-full px-3 py-2 bg-slate-900 border border-slate-700 rounded-md text-white placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-primary-500"
                />
              </div>

              <div className="mb-6">
                <label className="block text-sm font-medium text-slate-300 mb-2">Scopes</label>
                <div className="space-y-2">
                  {availableScopes.map((scope) => (
                    <label
                      key={scope.value}
                      className={`flex items-center gap-3 p-3 rounded-md cursor-pointer border transition-colors ${
                        scopes.includes(scope.value)
                          ? 'bg-primary-500/10 border-primary-500'
                          : 'bg-slate-900 border-slate-700 hover:border-slate-600'
                      }`}
                    >
                      <input
                        type="checkbox"
                        checked={scopes.includes(scope.value)}
                        onChange={() => toggleScope(scope.value)}
                        className="sr-only"
                      />
                      <div className={`w-4 h-4 rounded border-2 flex items-center justify-center ${
                        scopes.includes(scope.value) ? 'border-primary-500 bg-primary-500' : 'border-slate-500'
                      }`}>
                        {scopes.includes(scope.value) && (
                          <Check className="w-3 h-3 text-white" />
                        )}
                      </div>
                      <div>
                        <p className="text-sm font-medium text-white">{scope.label}</p>
                        <p className="text-xs text-slate-400">{scope.description}</p>
                      </div>
                    </label>
                  ))}
                </div>
              </div>

              <div className="flex items-center justify-end gap-3">
                <button
                  onClick={handleClose}
                  className="px-4 py-2 text-sm font-medium text-slate-300 hover:text-white hover:bg-slate-700 rounded-md transition-colors"
                >
                  Cancel
                </button>
                <button
                  onClick={() => createMutation.mutate({ name, scopes: scopes.join(' ') })}
                  disabled={!name || scopes.length === 0 || createMutation.isPending}
                  className="flex items-center gap-2 px-4 py-2 bg-primary-600 hover:bg-primary-700 disabled:bg-primary-500/50 text-white rounded-md text-sm font-medium transition-colors"
                >
                  {createMutation.isPending ? <Loader2 className="w-4 h-4 animate-spin" /> : <Key className="w-4 h-4" />}
                  Create Key
                </button>
              </div>
            </>
          ) : (
            <>
              <div className="text-center mb-6">
                <div className="w-12 h-12 bg-green-500/10 rounded-full flex items-center justify-center mx-auto mb-4">
                  <Check className="w-6 h-6 text-green-500" />
                </div>
                <h3 className="text-lg font-medium text-white mb-2">API Key Created</h3>
                <p className="text-sm text-slate-400">
                  Copy this key now. You won't be able to see it again.
                </p>
              </div>

              <div className="mb-6">
                <label className="block text-sm font-medium text-slate-300 mb-2">Your API Key</label>
                <div className="flex items-center gap-2">
                  <code className="flex-1 px-3 py-2 bg-slate-900 border border-slate-700 rounded text-sm text-white font-mono break-all">
                    {newKey}
                  </code>
                  <button
                    onClick={handleCopy}
                    className="p-2 bg-slate-700 hover:bg-slate-600 rounded transition-colors"
                  >
                    {copied ? <Check className="w-5 h-5 text-green-500" /> : <Copy className="w-5 h-5 text-slate-300" />}
                  </button>
                </div>
              </div>

              <div className="flex items-center justify-center">
                <button
                  onClick={handleDone}
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

export function APIKeyList() {
  const queryClient = useQueryClient()
  const { addToast } = useToastStore()
  const [createModalOpen, setCreateModalOpen] = useState(false)

  const { data, isLoading, error } = useQuery({
    queryKey: ['api-keys'],
    queryFn: () => api.users.listApiKeys(),
  })

  const keys: APIKey[] = data?.data || []

  const revokeMutation = useMutation({
    mutationFn: (keyId: string) => api.users.revokeApiKey(keyId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['api-keys'] })
      addToast({ type: 'success', title: 'Key revoked', message: 'The API key has been revoked.' })
    },
    onError: (error: ApiError) => {
      addToast({ type: 'error', title: 'Failed to revoke key', message: error.message })
    },
  })

  const handleRevoke = (key: APIKey) => {
    if (confirm(`Are you sure you want to revoke "${key.name}"? This action cannot be undone.`)) {
      revokeMutation.mutate(key.id)
    }
  }

  const formatDate = (dateString: string) => {
    return new Date(dateString).toLocaleDateString('en-US', {
      year: 'numeric',
      month: 'short',
      day: 'numeric',
    })
  }

  const formatRelativeTime = (dateString: string | null) => {
    if (!dateString) return 'Never'
    const date = new Date(dateString)
    const now = new Date()
    const diffMs = now.getTime() - date.getTime()
    const diffHours = Math.floor(diffMs / 3600000)
    const diffDays = Math.floor(diffHours / 24)

    if (diffHours < 1) return 'Just now'
    if (diffHours < 24) return `${diffHours}h ago`
    if (diffDays < 30) return `${diffDays}d ago`
    return formatDate(dateString)
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h3 className="text-lg font-medium text-white">API Keys</h3>
          <p className="text-sm text-slate-400">Manage API keys for programmatic access</p>
        </div>
        <button
          onClick={() => setCreateModalOpen(true)}
          className="flex items-center gap-2 px-4 py-2 bg-primary-600 hover:bg-primary-700 text-white rounded-md text-sm font-medium transition-colors"
        >
          <Plus className="w-4 h-4" />
          Create Key
        </button>
      </div>

      {isLoading ? (
        <div className="flex items-center justify-center py-12">
          <Loader2 className="w-6 h-6 text-primary-500 animate-spin" />
        </div>
      ) : error ? (
        <div className="text-center py-12">
          <p className="text-danger-400">Failed to load API keys</p>
        </div>
      ) : keys.length === 0 ? (
        <div className="text-center py-12">
          <div className="w-16 h-16 bg-slate-700 rounded-full flex items-center justify-center mx-auto mb-4">
            <Key className="w-8 h-8 text-slate-400" />
          </div>
          <h3 className="text-lg font-medium text-white mb-2">No API keys yet</h3>
          <p className="text-sm text-slate-400 mb-6">
            Create an API key to access the panel programmatically.
          </p>
          <button
            onClick={() => setCreateModalOpen(true)}
            className="inline-flex items-center gap-2 px-4 py-2 bg-primary-600 hover:bg-primary-700 text-white rounded-md text-sm font-medium transition-colors"
          >
            <Plus className="w-4 h-4" />
            Create Key
          </button>
        </div>
      ) : (
        <div className="bg-slate-900 border border-slate-700 rounded-lg overflow-hidden">
          <table className="w-full">
            <thead>
              <tr className="border-b border-slate-700">
                <th className="px-4 py-3 text-left text-xs font-medium text-slate-400 uppercase tracking-wider">Name</th>
                <th className="px-4 py-3 text-left text-xs font-medium text-slate-400 uppercase tracking-wider">Scopes</th>
                <th className="px-4 py-3 text-left text-xs font-medium text-slate-400 uppercase tracking-wider">Last Used</th>
                <th className="px-4 py-3 text-left text-xs font-medium text-slate-400 uppercase tracking-wider">Created</th>
                <th className="px-4 py-3 text-right text-xs font-medium text-slate-400 uppercase tracking-wider">Actions</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-slate-700">
              {keys.map((key) => (
                <tr key={key.id} className="hover:bg-slate-800/50">
                  <td className="px-4 py-4">
                    <div className="flex items-center gap-3">
                      <div className="w-8 h-8 bg-slate-700 rounded flex items-center justify-center">
                        <Key className="w-4 h-4 text-slate-400" />
                      </div>
                      <span className="text-sm font-medium text-white">{key.name}</span>
                    </div>
                  </td>
                  <td className="px-4 py-4">
                    <div className="flex items-center gap-1">
                      {key.scopes.split(' ').map((scope) => (
                        <span
                          key={scope}
                          className="px-2 py-0.5 bg-slate-700 rounded text-xs text-slate-300 capitalize"
                        >
                          {scope}
                        </span>
                      ))}
                    </div>
                  </td>
                  <td className="px-4 py-4">
                    <span className="text-sm text-slate-400">{formatRelativeTime(key.last_used_at)}</span>
                  </td>
                  <td className="px-4 py-4">
                    <span className="text-sm text-slate-400">{formatDate(key.created_at)}</span>
                  </td>
                  <td className="px-4 py-4">
                    <button
                      onClick={() => handleRevoke(key)}
                      disabled={revokeMutation.isPending}
                      className="p-1.5 text-slate-400 hover:text-danger-500 hover:bg-slate-700 rounded transition-colors"
                      title="Revoke key"
                    >
                      <Trash2 className="w-4 h-4" />
                    </button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      <CreateKeyModal
        open={createModalOpen}
        onClose={() => setCreateModalOpen(false)}
        onCreated={() => queryClient.invalidateQueries({ queryKey: ['api-keys'] })}
      />
    </div>
  )
}