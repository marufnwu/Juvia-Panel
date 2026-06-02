'use client'

import { useState } from 'react'
import { useParams, useRouter } from 'next/navigation'

import Link from 'next/link'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import {
  ArrowLeft,
  RefreshCw,
  HardDrive,
  Loader2,
  ExternalLink,
  Database,
  Plus,
  Trash2,
  Copy,
  Check,
  AlertTriangle,
  Play,
  Square,
  MoreHorizontal,
  Eye,
  EyeOff
} from 'lucide-react'
import { api, ApiError } from '@/lib/api'
import { useToastStore } from '@/stores'

type ServiceStatus = 'running' | 'stopped' | 'starting' | 'failed'

interface ServiceDetail {
  id: string
  name: string
  type: string
  version?: string
  status: ServiceStatus
  port?: number
  internal_host?: string
  container_id?: string
  data_size_mb?: number
  resource_usage?: {
    cpu_percent?: number
    memory_mb?: number
    memory_limit_mb?: number
    connections_active?: number
    connections_max?: number
  }
  credentials?: {
    host?: string
    port?: number
    database?: string
    username?: string
    password?: string
    connection_string?: string
  }
  backup_schedule?: {
    enabled?: boolean
    frequency?: string
    time?: string
    timezone?: string
    retention_days?: number
    destination?: string
  }
  connected_apps?: Array<{
    id: string
    name: string
  }>
  created_at: string
  updated_at: string
}

interface Backup {
  id: string
  name?: string
  status: 'completed' | 'failed' | 'in_progress'
  size_mb?: number
  destination?: string
  started_at: string
  completed_at?: string
  triggered_by?: string
}

const statusColors: Record<ServiceStatus, { bg: string; text: string; dot: string }> = {
  running: { bg: 'bg-green-500/10', text: 'text-green-500', dot: 'bg-green-500' },
  stopped: { bg: 'bg-slate-500/10', text: 'text-slate-400', dot: 'bg-slate-400' },
  starting: { bg: 'bg-amber-500/10', text: 'text-amber-500', dot: 'bg-amber-500' },
  failed: { bg: 'bg-red-500/10', text: 'text-red-500', dot: 'bg-red-500' },
}

function formatBytes(mb?: number): string {
  if (!mb) return '—'
  if (mb < 1024) return `${mb} MB`
  return `${(mb / 1024).toFixed(1)} GB`
}

function formatRelativeTime(dateString?: string): string {
  if (!dateString) return '—'
  const date = new Date(dateString)
  const now = new Date()
  const diffMs = now.getTime() - date.getTime()
  const diffMins = Math.floor(diffMs / 60000)
  const diffHours = Math.floor(diffMins / 60)
  const diffDays = Math.floor(diffHours / 24)

  if (diffMins < 1) return 'Just now'
  if (diffMins < 60) return `${diffMins}m ago`
  if (diffHours < 24) return `${diffHours}h ago`
  if (diffDays < 30) return `${diffDays}d ago`
  return date.toLocaleDateString()
}

export function ServiceDetailClient() {
  const params = useParams()
  const router = useRouter()
  const { addToast } = useToastStore()
  const queryClient = useQueryClient()

  const serviceId = (params.slug as string[])?.[0] as string
  const [activeTab, setActiveTab] = useState('overview')
  const [showPassword, setShowPassword] = useState(false)
  const [showDeleteConfirm, setShowDeleteConfirm] = useState(false)
  const [deleteConfirmName, setDeleteConfirmName] = useState('')

  // Fetch service details
  const { data: service, isLoading: serviceLoading, error: serviceError } = useQuery<ServiceDetail, ApiError>({
    queryKey: ['service', serviceId],
    queryFn: () => api.services.get(serviceId) as unknown as Promise<ServiceDetail>,
  })

  // Fetch backups
  const { data: backups, isLoading: backupsLoading } = useQuery<Backup[]>({
    queryKey: ['service-backups', serviceId],
    queryFn: () => api.services.getBackups(serviceId) as unknown as Promise<Backup[]>,
    enabled: activeTab === 'backups',
  })

  // Restart mutation
  const restartMutation = useMutation({
    mutationFn: () => api.services.restart(serviceId),
    onSuccess: () => {
      addToast({ type: 'success', title: 'Service restarted', message: 'Service is restarting.' })
      queryClient.invalidateQueries({ queryKey: ['service', serviceId] })
    },
    onError: (error: ApiError) => {
      addToast({ type: 'error', title: 'Restart failed', message: error.message })
    },
  })

  // Backup mutation
  const backupMutation = useMutation({
    mutationFn: () => api.services.backup(serviceId),
    onSuccess: () => {
      addToast({ type: 'success', title: 'Backup started', message: 'Service backup is being created.' })
      queryClient.invalidateQueries({ queryKey: ['service-backups', serviceId] })
    },
    onError: (error: ApiError) => {
      addToast({ type: 'error', title: 'Backup failed', message: error.message })
    },
  })

  // Delete mutation
  const deleteMutation = useMutation({
    mutationFn: () => api.services.delete(serviceId),
    onSuccess: () => {
      addToast({ type: 'success', title: 'Service deleted', message: 'Service and all data have been removed.' })
      router.push('/services')
    },
    onError: (error: ApiError) => {
      addToast({ type: 'error', title: 'Delete failed', message: error.message })
      setShowDeleteConfirm(false)
    },
  })

  const copyToClipboard = (text: string) => {
    navigator.clipboard.writeText(text)
    addToast({ type: 'success', title: 'Copied', message: 'Copied to clipboard.' })
  }

  if (serviceLoading) {
    return (
      <div className="flex items-center justify-center h-64">
        <Loader2 className="w-8 h-8 text-primary-500 animate-spin" />
      </div>
    )
  }

  if (serviceError || !service) {
    return (
      <div className="p-6">
        <Link href="/services" className="flex items-center gap-2 text-slate-400 hover:text-white mb-4">
          <ArrowLeft className="w-4 h-4" />
          Back to Services
        </Link>
        <div className="bg-slate-800 border border-slate-700 rounded-lg p-6 text-center">
          <p className="text-slate-400">Failed to load service details</p>
        </div>
      </div>
    )
  }

  return (
    <div className="p-6">
      {/* Header */}
      <div className="flex items-center gap-4 mb-6">
        <Link href="/services" className="p-2 text-slate-400 hover:text-white hover:bg-slate-800 rounded-md transition-colors">
          <ArrowLeft className="w-5 h-5" />
        </Link>
        <div className="flex-1">
          <div className="flex items-center gap-3">
            <h1 className="text-2xl font-semibold text-white">{service.name}</h1>
            <span className={`flex items-center gap-1.5 px-2 py-0.5 rounded-full text-xs font-medium ${statusColors[service.status]?.bg} ${statusColors[service.status]?.text}`}>
              <span className={`w-1.5 h-1.5 rounded-full ${statusColors[service.status]?.dot}`} />
              {service.status}
            </span>
          </div>
          <p className="text-sm text-slate-400 mt-1">
            {service.type} {service.version && `• ${service.version}`}
            {service.port && ` • Port ${service.port}`}
          </p>
        </div>
        <div className="flex items-center gap-2">
          {service.status === 'running' ? (
            <button
              onClick={() => restartMutation.mutate()}
              className="flex items-center gap-2 px-4 py-2 bg-slate-700 hover:bg-slate-600 text-white rounded-md text-sm font-medium transition-colors"
            >
              <RefreshCw className={`w-4 h-4 ${restartMutation.isPending ? 'animate-spin' : ''}`} />
              Restart
            </button>
          ) : service.status === 'stopped' ? (
            <button
              className="flex items-center gap-2 px-4 py-2 bg-primary-600 hover:bg-primary-700 text-white rounded-md text-sm font-medium transition-colors"
            >
              <Play className="w-4 h-4" />
              Start
            </button>
          ) : null}
          <button
            onClick={() => backupMutation.mutate()}
            disabled={service.status !== 'running' || backupMutation.isPending}
            className="flex items-center gap-2 px-4 py-2 bg-slate-700 hover:bg-slate-600 text-white rounded-md text-sm font-medium transition-colors disabled:opacity-50"
          >
            <HardDrive className="w-4 h-4" />
            Backup
          </button>
          <button className="p-2 text-slate-400 hover:text-white hover:bg-slate-800 rounded-md transition-colors">
            <MoreHorizontal className="w-5 h-5" />
          </button>
        </div>
      </div>

      {/* Tabs */}
      <div className="flex gap-1 border-b border-slate-700 mb-6">
        {['overview', 'backups', 'logs', 'settings'].map((tab) => (
          <button
            key={tab}
            onClick={() => setActiveTab(tab)}
            className={`px-4 py-2 text-sm font-medium transition-colors capitalize ${
              activeTab === tab
                ? 'text-white border-b-2 border-primary-500'
                : 'text-slate-400 hover:text-white'
            }`}
          >
            {tab}
          </button>
        ))}
      </div>

      {/* Tab Content */}
      {activeTab === 'overview' && (
        <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
          {/* Service Info */}
          <div className="bg-slate-800 border border-slate-700 rounded-lg p-6">
            <h2 className="text-lg font-medium text-white mb-4">Service Info</h2>
            <dl className="space-y-3">
              <div className="flex justify-between">
                <dt className="text-slate-400">Type</dt>
                <dd className="text-white capitalize">{service.type}</dd>
              </div>
              <div className="flex justify-between">
                <dt className="text-slate-400">Version</dt>
                <dd className="text-white">{service.version || '—'}</dd>
              </div>
              <div className="flex justify-between">
                <dt className="text-slate-400">Status</dt>
                <dd className={`font-medium ${statusColors[service.status]?.text}`}>
                  {service.status}
                </dd>
              </div>
              <div className="flex justify-between">
                <dt className="text-slate-400">Internal Host</dt>
                <dd className="text-white font-mono text-sm">{service.internal_host || 'localhost'}</dd>
              </div>
              <div className="flex justify-between">
                <dt className="text-slate-400">Port</dt>
                <dd className="text-white">{service.port || '—'}</dd>
              </div>
              <div className="flex justify-between">
                <dt className="text-slate-400">Data Size</dt>
                <dd className="text-white flex items-center gap-1">
                  <HardDrive className="w-4 h-4 text-slate-400" />
                  {formatBytes(service.data_size_mb)}
                </dd>
              </div>
            </dl>
          </div>

          {/* Connection Details */}
          <div className="bg-slate-800 border border-slate-700 rounded-lg p-6">
            <h2 className="text-lg font-medium text-white mb-4">Connection Details</h2>
            {service.credentials ? (
              <dl className="space-y-3">
                <div className="flex justify-between">
                  <dt className="text-slate-400">Host</dt>
                  <dd className="text-white font-mono text-sm">{service.credentials.host || 'localhost'}</dd>
                </div>
                <div className="flex justify-between">
                  <dt className="text-slate-400">Port</dt>
                  <dd className="text-white">{service.credentials.port || service.port}</dd>
                </div>
                <div className="flex justify-between">
                  <dt className="text-slate-400">Database</dt>
                  <dd className="text-white">{service.credentials.database || service.name}</dd>
                </div>
                <div className="flex justify-between">
                  <dt className="text-slate-400">Username</dt>
                  <dd className="text-white">{service.credentials.username || '—'}</dd>
                </div>
                <div className="flex justify-between">
                  <dt className="text-slate-400">Password</dt>
                  <dd className="text-white flex items-center gap-2">
                    <span className="font-mono text-sm">
                      {showPassword ? service.credentials.password : '••••••••••••'}
                    </span>
                    <button
                      onClick={() => setShowPassword(!showPassword)}
                      className="p-1 text-slate-400 hover:text-white"
                    >
                      {showPassword ? <Eye /> : <Copy />}
                    </button>
                  </dd>
                </div>
                {service.credentials.connection_string && (
                  <div>
                    <dt className="text-slate-400 mb-1">Connection String</dt>
                    <dd className="bg-slate-700 p-2 rounded text-xs text-white font-mono break-all">
                      {service.credentials.connection_string}
                    </dd>
                  </div>
                )}
                <div className="flex gap-2 pt-2">
                  <button
                    onClick={() => copyToClipboard(service.credentials?.connection_string || '')}
                    className="flex items-center gap-1 px-3 py-1.5 bg-slate-700 hover:bg-slate-600 text-white rounded text-sm"
                  >
                    <Copy className="w-4 h-4" />
                    Copy
                  </button>
                </div>
              </dl>
            ) : (
              <p className="text-slate-400 text-sm">Connection details not available</p>
            )}
          </div>

          {/* Resource Usage */}
          <div className="bg-slate-800 border border-slate-700 rounded-lg p-6">
            <h2 className="text-lg font-medium text-white mb-4">Resource Usage</h2>
            {service.resource_usage ? (
              <dl className="space-y-3">
                <div className="flex justify-between">
                  <dt className="text-slate-400">CPU</dt>
                  <dd className="text-white">{service.resource_usage.cpu_percent?.toFixed(1) || 0}%</dd>
                </div>
                <div className="flex justify-between">
                  <dt className="text-slate-400">Memory</dt>
                  <dd className="text-white">
                    {service.resource_usage.memory_mb || 0} MB
                    {service.resource_usage.memory_limit_mb && ` / ${service.resource_usage.memory_limit_mb} MB`}
                  </dd>
                </div>
                {service.resource_usage.connections_active !== undefined && (
                  <div className="flex justify-between">
                    <dt className="text-slate-400">Connections</dt>
                    <dd className="text-white">
                      {service.resource_usage.connections_active}
                      {service.resource_usage.connections_max && ` / ${service.resource_usage.connections_max}`}
                    </dd>
                  </div>
                )}
              </dl>
            ) : (
              <p className="text-slate-400 text-sm">Resource usage not available</p>
            )}
          </div>

          {/* Connected Apps */}
          <div className="bg-slate-800 border border-slate-700 rounded-lg p-6">
            <div className="flex items-center justify-between mb-4">
              <h2 className="text-lg font-medium text-white">Connected Apps</h2>
              <button className="flex items-center gap-1 text-sm text-primary-400 hover:text-primary-300">
                <Plus className="w-4 h-4" />
                Connect
              </button>
            </div>
            {service.connected_apps && service.connected_apps.length > 0 ? (
              <div className="flex flex-wrap gap-2">
                {service.connected_apps.map((app) => (
                  <Link
                    key={app.id}
                    href={`/apps/${app.id}`}
                    className="flex items-center gap-2 px-3 py-1.5 bg-slate-700/50 rounded-full text-sm text-white hover:bg-slate-700"
                  >
                    <Database className="w-4 h-4 text-slate-400" />
                    {app.name}
                  </Link>
                ))}
              </div>
            ) : (
              <p className="text-slate-400 text-sm">No apps connected</p>
            )}
          </div>
        </div>
      )}

      {activeTab === 'backups' && (
        <div className="bg-slate-800 border border-slate-700 rounded-lg overflow-hidden">
          <div className="p-4 border-b border-slate-700 flex items-center justify-between">
            <div>
              <h2 className="text-lg font-medium text-white">Backups</h2>
              {service.backup_schedule && (
                <p className="text-sm text-slate-400 mt-1">
                  Schedule: {service.backup_schedule.frequency || 'Daily'} at {service.backup_schedule.time || '02:00'}
                  {service.backup_schedule.retention_days && ` • Retention: ${service.backup_schedule.retention_days} days`}
                </p>
              )}
            </div>
            <button
              onClick={() => backupMutation.mutate()}
              disabled={service.status !== 'running' || backupMutation.isPending}
              className="flex items-center gap-2 px-4 py-2 bg-primary-600 hover:bg-primary-700 text-white rounded text-sm disabled:opacity-50"
            >
              <HardDrive className="w-4 h-4" />
              Backup Now
            </button>
          </div>
          {backupsLoading ? (
            <div className="flex items-center justify-center h-32">
              <Loader2 className="w-6 h-6 text-primary-500 animate-spin" />
            </div>
          ) : backups && backups.length > 0 ? (
            <table className="w-full">
              <thead>
                <tr className="border-b border-slate-700">
                  <th className="px-4 py-3 text-left text-xs font-medium text-slate-400 uppercase">Status</th>
                  <th className="px-4 py-3 text-left text-xs font-medium text-slate-400 uppercase">Size</th>
                  <th className="px-4 py-3 text-left text-xs font-medium text-slate-400 uppercase">Location</th>
                  <th className="px-4 py-3 text-left text-xs font-medium text-slate-400 uppercase">Time</th>
                  <th className="px-4 py-3 text-right text-xs font-medium text-slate-400 uppercase">Actions</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-slate-700">
                {backups.map((backup) => (
                  <tr key={backup.id} className="hover:bg-slate-700/50">
                    <td className="px-4 py-4">
                      <span className={`inline-flex items-center gap-1.5 text-xs font-medium ${
                        backup.status === 'completed' ? 'text-green-400' :
                        backup.status === 'failed' ? 'text-red-400' :
                        'text-amber-400'
                      }`}>
                        <span className={`w-1.5 h-1.5 rounded-full ${
                          backup.status === 'completed' ? 'bg-green-400' :
                          backup.status === 'failed' ? 'bg-red-400' :
                          'bg-amber-400 animate-pulse'
                        }`} />
                        {backup.status}
                      </span>
                    </td>
                    <td className="px-4 py-4 text-slate-300 text-sm">
                      {formatBytes(backup.size_mb)}
                    </td>
                    <td className="px-4 py-4 text-slate-400 text-sm">
                      {backup.destination || '—'}
                    </td>
                    <td className="px-4 py-4 text-slate-400 text-sm">
                      {formatRelativeTime(backup.started_at)}
                    </td>
                    <td className="px-4 py-4 text-right">
                      {backup.status === 'completed' && (
                        <button className="px-3 py-1.5 bg-slate-700 hover:bg-slate-600 text-white rounded text-sm">
                          Restore
                        </button>
                      )}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          ) : (
            <div className="p-8 text-center text-slate-400">
              No backups yet
            </div>
          )}
        </div>
      )}

      {activeTab === 'logs' && (
        <div className="bg-slate-800 border border-slate-700 rounded-lg overflow-hidden">
          <div className="p-4 border-b border-slate-700 flex items-center justify-between">
            <h2 className="text-lg font-medium text-white">Logs</h2>
          </div>
          <div className="h-96 overflow-auto p-4 font-mono text-sm">
            <p className="text-slate-500">Logs streaming will be implemented with WebSocket</p>
          </div>
        </div>
      )}

      {activeTab === 'settings' && (
        <div className="space-y-6">
          {/* Backup Schedule */}
          <div className="bg-slate-800 border border-slate-700 rounded-lg p-6">
            <h2 className="text-lg font-medium text-white mb-4">Backup Schedule</h2>
            <dl className="space-y-3">
              <div className="flex justify-between">
                <dt className="text-slate-400">Enabled</dt>
                <dd className="text-white">{service.backup_schedule?.enabled ? 'Yes' : 'No'}</dd>
              </div>
              <div className="flex justify-between">
                <dt className="text-slate-400">Frequency</dt>
                <dd className="text-white capitalize">{service.backup_schedule?.frequency || '—'}</dd>
              </div>
              <div className="flex justify-between">
                <dt className="text-slate-400">Time</dt>
                <dd className="text-white">{service.backup_schedule?.time || '—'}</dd>
              </div>
              <div className="flex justify-between">
                <dt className="text-slate-400">Retention</dt>
                <dd className="text-white">{service.backup_schedule?.retention_days ? `${service.backup_schedule.retention_days} days` : '—'}</dd>
              </div>
              <div className="flex justify-between">
                <dt className="text-slate-400">Destination</dt>
                <dd className="text-white capitalize">{service.backup_schedule?.destination || '—'}</dd>
              </div>
            </dl>
          </div>

          {/* Danger Zone */}
          <div className="bg-slate-800 border border-red-900/50 rounded-lg p-6">
            <h2 className="text-lg font-medium text-red-400 mb-4">Danger Zone</h2>
            <div className="flex items-center justify-between">
              <div>
                <p className="text-white font-medium">Delete Service</p>
                <p className="text-sm text-slate-400">This will permanently delete the service and all data.</p>
              </div>
              {!showDeleteConfirm ? (
                <button
                  onClick={() => setShowDeleteConfirm(true)}
                  className="px-4 py-2 bg-red-600/20 hover:bg-red-600 text-red-400 hover:text-white rounded text-sm transition-colors"
                >
                  Delete Service
                </button>
              ) : (
                <div className="flex items-center gap-2">
                  <input
                    type="text"
                    placeholder={`Type "${service.name}" to confirm`}
                    value={deleteConfirmName}
                    onChange={(e) => setDeleteConfirmName(e.target.value)}
                    className="px-3 py-2 bg-slate-700 border border-slate-600 rounded text-sm text-white w-48"
                  />
                  <button
                    onClick={() => deleteMutation.mutate()}
                    disabled={deleteConfirmName !== service.name}
                    className="px-4 py-2 bg-red-600 hover:bg-red-700 text-white rounded text-sm disabled:opacity-50"
                  >
                    Confirm Delete
                  </button>
                  <button
                    onClick={() => {
                      setShowDeleteConfirm(false)
                      setDeleteConfirmName('')
                    }}
                    className="px-4 py-2 bg-slate-700 hover:bg-slate-600 text-white rounded text-sm"
                  >
                    Cancel
                  </button>
                </div>
              )}
            </div>
          </div>
        </div>
      )}
    </div>
  )
}
