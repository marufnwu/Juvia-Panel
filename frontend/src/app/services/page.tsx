'use client'

import { useState } from 'react'
import Link from 'next/link'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import {
  Plus,
  Search,
  RefreshCw,
  MoreHorizontal,
  Database,
  HardDrive,
  Play,
  Square,
  FileText,
  Loader2,
  Trash2,
  Settings
} from 'lucide-react'
import { api, ApiError } from '@/lib/api'
import { useToastStore } from '@/stores'

type ServiceStatus = 'running' | 'stopped' | 'starting' | 'failed'

interface ServiceListItem {
  id: string
  name: string
  type: string
  version?: string
  status: ServiceStatus
  port?: number
  data_size_mb?: number
  connected_apps?: number
  resource_usage?: {
    cpu_percent?: number
    memory_mb?: number
  }
  last_backup_at?: string
  created_at: string
  updated_at: string
}

interface ServicesListResponse {
  data: ServiceListItem[]
  meta: {
    total: number
    page: number
    per_page: number
    total_pages: number
  }
}

const statusColors: Record<ServiceStatus, { bg: string; text: string; dot: string }> = {
  running: { bg: 'bg-green-500/10', text: 'text-green-500', dot: 'bg-green-500' },
  stopped: { bg: 'bg-slate-500/10', text: 'text-slate-400', dot: 'bg-slate-400' },
  starting: { bg: 'bg-amber-500/10', text: 'text-amber-500', dot: 'bg-amber-500' },
  failed: { bg: 'bg-red-500/10', text: 'text-red-500', dot: 'bg-red-500' },
}

const serviceTypeIcons: Record<string, string> = {
  postgresql: 'pg',
  mysql: 'my',
  redis: 'rd',
  mongodb: 'mg',
  minio: 'mn',
  custom: 'sv',
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

export default function ServicesPage() {
  const queryClient = useQueryClient()
  const { addToast } = useToastStore()
  
  // Filters and pagination state
  const [search, setSearch] = useState('')
  const [typeFilter, setTypeFilter] = useState<string>('all')
  const [statusFilter, setStatusFilter] = useState<string>('all')
  const [page, setPage] = useState(1)

  // Fetch services
  const { data, isLoading, error, refetch } = useQuery({
    queryKey: ['services', { page, type: typeFilter, status: statusFilter, search }],
    queryFn: async () => {
      const response = await api.services.list({
        page,
        limit: 20,
        type: typeFilter !== 'all' ? typeFilter : undefined,
        // Note: status filter would need API support
      })
      return response as unknown as ServicesListResponse
    },
    refetchInterval: 30000,
  })

  // Restart mutation
  const restartMutation = useMutation({
    mutationFn: (serviceId: string) => api.services.restart(serviceId),
    onSuccess: () => {
      addToast({ type: 'success', title: 'Service restarted', message: 'Service is restarting.' })
      queryClient.invalidateQueries({ queryKey: ['services'] })
    },
    onError: (error: ApiError) => {
      addToast({ type: 'error', title: 'Restart failed', message: error.message })
    },
  })

  // Backup mutation
  const backupMutation = useMutation({
    mutationFn: (serviceId: string) => api.services.backup(serviceId),
    onSuccess: () => {
      addToast({ type: 'success', title: 'Backup started', message: 'Service backup is being created.' })
    },
    onError: (error: ApiError) => {
      addToast({ type: 'error', title: 'Backup failed', message: error.message })
    },
  })

  const services = data?.data || []
  const meta = data?.meta || { total: 0, page: 1, per_page: 20, total_pages: 0 }

  const handleAction = (action: 'restart' | 'backup', serviceId: string) => {
    switch (action) {
      case 'restart':
        restartMutation.mutate(serviceId)
        break
      case 'backup':
        backupMutation.mutate(serviceId)
        break
    }
  }

  return (
    <div className="p-6">
      {/* Header */}
      <div className="flex items-center justify-between mb-6">
        <div>
          <h1 className="text-2xl font-semibold text-white">Services</h1>
          <p className="text-sm text-slate-400 mt-1">
            Manage databases, caches, and other backing services
          </p>
        </div>
        <Link
          href="/services/new"
          className="flex items-center gap-2 px-4 py-2 bg-primary-600 hover:bg-primary-700 text-white rounded-md text-sm font-medium transition-colors"
        >
          <Plus className="w-4 h-4" />
          New Service
        </Link>
      </div>

      {/* Filters */}
      <div className="flex flex-wrap items-center gap-4 mb-6">
        {/* Search */}
        <div className="relative flex-1 min-w-[240px] max-w-md">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-slate-400" />
          <input
            type="text"
            placeholder="Search services..."
            value={search}
            onChange={(e) => {
              setSearch(e.target.value)
              setPage(1)
            }}
            className="w-full pl-10 pr-4 py-2 bg-slate-800 border border-slate-700 rounded-md text-sm text-white placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-primary-500 focus:border-transparent"
          />
        </div>

        {/* Type Filter */}
        <select
          value={typeFilter}
          onChange={(e) => {
            setTypeFilter(e.target.value)
            setPage(1)
          }}
          className="px-3 py-2 bg-slate-800 border border-slate-700 rounded-md text-sm text-white focus:outline-none focus:ring-2 focus:ring-primary-500"
        >
          <option value="all">All Types</option>
          <option value="postgresql">PostgreSQL</option>
          <option value="mysql">MySQL</option>
          <option value="redis">Redis</option>
          <option value="mongodb">MongoDB</option>
          <option value="minio">MinIO</option>
          <option value="custom">Custom</option>
        </select>

        {/* Status Filter */}
        <select
          value={statusFilter}
          onChange={(e) => {
            setStatusFilter(e.target.value)
            setPage(1)
          }}
          className="px-3 py-2 bg-slate-800 border border-slate-700 rounded-md text-sm text-white focus:outline-none focus:ring-2 focus:ring-primary-500"
        >
          <option value="all">All Status</option>
          <option value="running">Running</option>
          <option value="stopped">Stopped</option>
          <option value="starting">Starting</option>
          <option value="failed">Failed</option>
        </select>

        {/* Refresh Button */}
        <button
          onClick={() => refetch()}
          className="p-2 text-slate-400 hover:text-white hover:bg-slate-800 rounded-md transition-colors"
          title="Refresh"
        >
          <RefreshCw className={`w-4 h-4 ${isLoading ? 'animate-spin' : ''}`} />
        </button>
      </div>

      {/* Table */}
      <div className="bg-slate-800 border border-slate-700 rounded-lg overflow-hidden">
        {isLoading ? (
          <div className="flex items-center justify-center h-64">
            <Loader2 className="w-8 h-8 text-primary-500 animate-spin" />
          </div>
        ) : error ? (
          <div className="flex flex-col items-center justify-center h-64 text-slate-400">
            <p className="mb-2">Failed to load services</p>
            <button
              onClick={() => refetch()}
              className="text-primary-500 hover:text-primary-400 text-sm"
            >
              Try again
            </button>
          </div>
        ) : services.length === 0 ? (
          /* Empty State */
          <div className="flex flex-col items-center justify-center py-16">
            <div className="w-16 h-16 bg-slate-700 rounded-full flex items-center justify-center mb-4">
              <Database className="w-8 h-8 text-slate-400" />
            </div>
            <h3 className="text-lg font-medium text-white mb-2">No services yet</h3>
            <p className="text-sm text-slate-400 mb-6 text-center max-w-md">
              Add a database or cache for your apps. One-click setup with automatic backups.
            </p>
            <Link
              href="/services/new"
              className="flex items-center gap-2 px-4 py-2 bg-primary-600 hover:bg-primary-700 text-white rounded-md text-sm font-medium transition-colors"
            >
              <Plus className="w-4 h-4" />
              Create Service
            </Link>
          </div>
        ) : (
          <>
            <table className="w-full">
              <thead>
                <tr className="border-b border-slate-700">
                  <th className="px-4 py-3 text-left text-xs font-medium text-slate-400 uppercase tracking-wider w-12">
                    Status
                  </th>
                  <th className="px-4 py-3 text-left text-xs font-medium text-slate-400 uppercase tracking-wider">
                    Name
                  </th>
                  <th className="px-4 py-3 text-left text-xs font-medium text-slate-400 uppercase tracking-wider">
                    Type
                  </th>
                  <th className="px-4 py-3 text-left text-xs font-medium text-slate-400 uppercase tracking-wider">
                    Apps Using
                  </th>
                  <th className="px-4 py-3 text-left text-xs font-medium text-slate-400 uppercase tracking-wider">
                    Size
                  </th>
                  <th className="px-4 py-3 text-left text-xs font-medium text-slate-400 uppercase tracking-wider">
                    Updated
                  </th>
                  <th className="px-4 py-3 text-right text-xs font-medium text-slate-400 uppercase tracking-wider w-48">
                    Actions
                  </th>
                </tr>
              </thead>
              <tbody className="divide-y divide-slate-700">
                {services.map((service) => (
                  <tr
                    key={service.id}
                    className="hover:bg-slate-700/50 transition-colors"
                  >
                    <td className="px-4 py-4">
                      <div className="flex items-center gap-2">
                        <span className={`w-2 h-2 rounded-full ${statusColors[service.status]?.dot || 'bg-slate-400'}`} />
                        {service.status === 'starting' && (
                          <RefreshCw className="w-3 h-3 text-amber-500 animate-spin" />
                        )}
                      </div>
                    </td>
                    <td className="px-4 py-4">
                      <Link
                        href={`/services/${service.id}`}
                        className="font-medium text-white hover:text-primary-400 transition-colors"
                      >
                        {service.name}
                      </Link>
                    </td>
                    <td className="px-4 py-4">
                      <div className="flex items-center gap-2">
                        <div className="w-6 h-6 bg-slate-700 rounded flex items-center justify-center text-xs text-slate-400 font-medium">
                          {serviceTypeIcons[service.type] || service.type.slice(0, 2).toUpperCase()}
                        </div>
                        <span className="text-sm text-slate-300">
                          {service.type}
                          {service.version && <span className="text-slate-500 ml-1">({service.version})</span>}
                        </span>
                      </div>
                    </td>
                    <td className="px-4 py-4">
                      <span className="text-sm text-slate-400">
                        {service.connected_apps !== undefined ? (
                          <span className="text-white">{service.connected_apps} app{service.connected_apps !== 1 ? 's' : ''}</span>
                        ) : (
                          '—'
                        )}
                      </span>
                    </td>
                    <td className="px-4 py-4">
                      <div className="flex items-center gap-2">
                        <HardDrive className="w-4 h-4 text-slate-500" />
                        <span className="text-sm text-slate-400">
                          {formatBytes(service.data_size_mb)}
                        </span>
                      </div>
                    </td>
                    <td className="px-4 py-4">
                      <span className="text-sm text-slate-400" title={service.updated_at}>
                        {formatRelativeTime(service.updated_at)}
                      </span>
                    </td>
                    <td className="px-4 py-4">
                      <div className="flex items-center justify-end gap-1">
                        {/* Backup */}
                        <button
                          onClick={() => handleAction('backup', service.id)}
                          disabled={service.status !== 'running'}
                          className="p-1.5 text-slate-400 hover:text-white hover:bg-slate-600 rounded transition-colors disabled:opacity-50"
                          title="Backup"
                        >
                          <HardDrive className="w-4 h-4" />
                        </button>

                        {/* Restart */}
                        <button
                          onClick={() => handleAction('restart', service.id)}
                          disabled={service.status !== 'running'}
                          className="p-1.5 text-slate-400 hover:text-white hover:bg-slate-600 rounded transition-colors disabled:opacity-50"
                          title="Restart"
                        >
                          <RefreshCw className="w-4 h-4" />
                        </button>

                        {/* Logs */}
                        <Link
                          href={`/services/${service.id}?tab=logs`}
                          className="p-1.5 text-slate-400 hover:text-white hover:bg-slate-600 rounded transition-colors"
                          title="Logs"
                        >
                          <FileText className="w-4 h-4" />
                        </Link>

                        {/* Settings */}
                        <Link
                          href={`/services/${service.id}`}
                          className="p-1.5 text-slate-400 hover:text-white hover:bg-slate-600 rounded transition-colors"
                          title="Settings"
                        >
                          <Settings className="w-4 h-4" />
                        </Link>
                      </div>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>

            {/* Pagination */}
            {meta.total_pages > 1 && (
              <div className="flex items-center justify-between px-4 py-3 border-t border-slate-700">
                <p className="text-sm text-slate-400">
                  Showing {((meta.page - 1) * meta.per_page) + 1} to {Math.min(meta.page * meta.per_page, meta.total)} of {meta.total} services
                </p>
                <div className="flex items-center gap-2">
                  <button
                    onClick={() => setPage(page - 1)}
                    disabled={page === 1}
                    className="p-1.5 text-slate-400 hover:text-white hover:bg-slate-700 rounded disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
                  >
                    <RefreshCw className="w-4 h-4 rotate-180" />
                  </button>
                  <span className="text-sm text-slate-300">
                    Page {meta.page} of {meta.total_pages}
                  </span>
                  <button
                    onClick={() => setPage(page + 1)}
                    disabled={page >= meta.total_pages}
                    className="p-1.5 text-slate-400 hover:text-white hover:bg-slate-700 rounded disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
                  >
                    <RefreshCw className="w-4 h-4" />
                  </button>
                </div>
              </div>
            )}
          </>
        )}
      </div>
    </div>
  )
}
