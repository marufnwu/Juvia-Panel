'use client'

import { useState, useMemo } from 'react'
import Link from 'next/link'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import {
  Plus,
  Search,
  RefreshCw,
  Play,
  Square,
  FileText,
  MoreHorizontal,
  Trash2,
  Settings,
  Eye,
  ArrowUpCircle,
  ChevronLeft,
  ChevronRight,
  Boxes,
  GitBranch,
  ExternalLink,
  Loader2
} from 'lucide-react'
import { api, ApiError, App } from '@/lib/api'
import { useToastStore } from '@/stores'

type AppStatus = 'running' | 'stopped' | 'deploying' | 'failed'
type Runtime = 'nodejs' | 'python' | 'go' | 'php' | 'ruby' | 'static' | 'docker'

interface AppListItem {
  id: string
  name: string
  status: AppStatus
  runtime: string
  runtime_version?: string
  primary_domain?: string
  last_deployed_at?: string
  updated_at: string
  source?: {
    type: string
    repo_url?: string
    branch?: string
    last_commit?: string
    last_commit_message?: string
  }
}

interface AppsListResponse {
  data: AppListItem[]
  meta: {
    total: number
    page: number
    per_page: number
    total_pages: number
  }
}

const statusColors: Record<AppStatus, { bg: string; text: string; dot: string }> = {
  running: { bg: 'bg-green-500/10', text: 'text-green-500', dot: 'bg-green-500' },
  stopped: { bg: 'bg-slate-500/10', text: 'text-slate-400', dot: 'bg-slate-400' },
  deploying: { bg: 'bg-amber-500/10', text: 'text-amber-500', dot: 'bg-amber-500' },
  failed: { bg: 'bg-red-500/10', text: 'text-red-500', dot: 'bg-red-500' },
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

export default function AppsPage() {
  const queryClient = useQueryClient()
  const { addToast } = useToastStore()
  
  // Filters and pagination state
  const [search, setSearch] = useState('')
  const [statusFilter, setStatusFilter] = useState<string>('all')
  const [runtimeFilter, setRuntimeFilter] = useState<string>('all')
  const [page, setPage] = useState(1)
  const [hoveredApp, setHoveredApp] = useState<string | null>(null)

  // Fetch apps
  const { data, isLoading, error, refetch } = useQuery({
    queryKey: ['apps', { page, status: statusFilter, runtime: runtimeFilter, search }],
    queryFn: async () => {
      const response = await api.apps.list({
        page,
        limit: 20,
        status: statusFilter !== 'all' ? statusFilter : undefined,
        search: search || undefined,
      })
      return response as unknown as AppsListResponse
    },
    refetchInterval: 30000, // Refetch every 30 seconds
  })

  // Deploy mutation
  const deployMutation = useMutation({
    mutationFn: (appId: string) => api.apps.deploy(appId),
    onSuccess: (_, appId) => {
      addToast({ type: 'success', title: 'Deployment started', message: 'Your app is being deployed.' })
      queryClient.invalidateQueries({ queryKey: ['apps'] })
    },
    onError: (error: ApiError) => {
      addToast({ type: 'error', title: 'Deployment failed', message: error.message })
    },
  })

  // Restart mutation
  const restartMutation = useMutation({
    mutationFn: (appId: string) => api.apps.restart(appId),
    onSuccess: () => {
      addToast({ type: 'success', title: 'App restarted', message: 'Your app is restarting.' })
      queryClient.invalidateQueries({ queryKey: ['apps'] })
    },
    onError: (error: ApiError) => {
      addToast({ type: 'error', title: 'Restart failed', message: error.message })
    },
  })

  // Stop mutation
  const stopMutation = useMutation({
    mutationFn: (appId: string) => api.apps.stop(appId),
    onSuccess: () => {
      addToast({ type: 'success', title: 'App stopped', message: 'Your app has been stopped.' })
      queryClient.invalidateQueries({ queryKey: ['apps'] })
    },
    onError: (error: ApiError) => {
      addToast({ type: 'error', title: 'Stop failed', message: error.message })
    },
  })

  const apps = data?.data || []
  const meta = data?.meta || { total: 0, page: 1, per_page: 20, total_pages: 0 }

  // Debounced search
  const debouncedSearch = useMemo(() => {
    const timeout = setTimeout(() => {
      if (search !== undefined) {
        setPage(1)
      }
    }, 200)
    return () => clearTimeout(timeout)
  }, [search])

  const handleAction = (action: 'deploy' | 'restart' | 'stop', appId: string) => {
    switch (action) {
      case 'deploy':
        deployMutation.mutate(appId)
        break
      case 'restart':
        restartMutation.mutate(appId)
        break
      case 'stop':
        stopMutation.mutate(appId)
        break
    }
  }

  return (
    <div className="p-6">
      {/* Header */}
      <div className="flex items-center justify-between mb-6">
        <div>
          <h1 className="text-2xl font-semibold text-white">Apps</h1>
          <p className="text-sm text-slate-400 mt-1">
            Manage your deployed applications
          </p>
        </div>
        <Link
          href="/apps/new"
          className="flex items-center gap-2 px-4 py-2 bg-primary-600 hover:bg-primary-700 text-white rounded-md text-sm font-medium transition-colors"
        >
          <Plus className="w-4 h-4" />
          New App
        </Link>
      </div>

      {/* Filters */}
      <div className="flex flex-wrap items-center gap-4 mb-6">
        {/* Search */}
        <div className="relative flex-1 min-w-[240px] max-w-md">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-slate-400" />
          <input
            type="text"
            placeholder="Search apps..."
            value={search}
            onChange={(e) => {
              setSearch(e.target.value)
              debouncedSearch
            }}
            className="w-full pl-10 pr-4 py-2 bg-slate-800 border border-slate-700 rounded-md text-sm text-white placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-primary-500 focus:border-transparent"
          />
        </div>

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
          <option value="deploying">Deploying</option>
          <option value="failed">Failed</option>
        </select>

        {/* Runtime Filter */}
        <select
          value={runtimeFilter}
          onChange={(e) => {
            setRuntimeFilter(e.target.value)
            setPage(1)
          }}
          className="px-3 py-2 bg-slate-800 border border-slate-700 rounded-md text-sm text-white focus:outline-none focus:ring-2 focus:ring-primary-500"
        >
          <option value="all">All Runtimes</option>
          <option value="nodejs">Node.js</option>
          <option value="python">Python</option>
          <option value="go">Go</option>
          <option value="php">PHP</option>
          <option value="ruby">Ruby</option>
          <option value="static">Static</option>
          <option value="docker">Docker</option>
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
            <p className="mb-2">Failed to load apps</p>
            <button
              onClick={() => refetch()}
              className="text-primary-500 hover:text-primary-400 text-sm"
            >
              Try again
            </button>
          </div>
        ) : apps.length === 0 ? (
          /* Empty State */
          <div className="flex flex-col items-center justify-center py-16">
            <div className="w-16 h-16 bg-slate-700 rounded-full flex items-center justify-center mb-4">
              <Boxes className="w-8 h-8 text-slate-400" />
            </div>
            <h3 className="text-lg font-medium text-white mb-2">No apps yet</h3>
            <p className="text-sm text-slate-400 mb-6 text-center max-w-md">
              Deploy your first application from GitHub, GitLab, or upload files directly.
            </p>
            <Link
              href="/apps/new"
              className="flex items-center gap-2 px-4 py-2 bg-primary-600 hover:bg-primary-700 text-white rounded-md text-sm font-medium transition-colors"
            >
              <Plus className="w-4 h-4" />
              Create App
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
                    Runtime
                  </th>
                  <th className="px-4 py-3 text-left text-xs font-medium text-slate-400 uppercase tracking-wider">
                    Domain
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
                {apps.map((app) => (
                  <tr
                    key={app.id}
                    className="hover:bg-slate-700/50 transition-colors"
                    onMouseEnter={() => setHoveredApp(app.id)}
                    onMouseLeave={() => setHoveredApp(null)}
                  >
                    <td className="px-4 py-4">
                      <div className="flex items-center gap-2">
                        <span className={`w-2 h-2 rounded-full ${statusColors[app.status]?.dot || 'bg-slate-400'}`} />
                        {app.status === 'deploying' && (
                          <RefreshCw className="w-3 h-3 text-amber-500 animate-spin" />
                        )}
                      </div>
                    </td>
                    <td className="px-4 py-4">
                      <Link
                        href={`/apps/${app.id}`}
                        className="flex flex-col gap-1 hover:text-primary-400 transition-colors"
                      >
                        <span className="font-medium text-white">{app.name}</span>
                        {app.source?.repo_url && (
                          <span className="flex items-center gap-1 text-xs text-slate-400">
                            <GitBranch className="w-3 h-3" />
                            {app.source.branch || 'main'}
                          </span>
                        )}
                      </Link>
                    </td>
                    <td className="px-4 py-4">
                      <span className="text-sm text-slate-300">
                        {app.runtime}
                        {app.runtime_version && (
                          <span className="text-slate-500 ml-1">({app.runtime_version})</span>
                        )}
                      </span>
                    </td>
                    <td className="px-4 py-4">
                      {app.primary_domain ? (
                        <a
                          href={`https://${app.primary_domain}`}
                          target="_blank"
                          rel="noopener noreferrer"
                          className="flex items-center gap-1 text-sm text-primary-400 hover:text-primary-300 transition-colors"
                        >
                          {app.primary_domain}
                          <ExternalLink className="w-3 h-3" />
                        </a>
                      ) : (
                        <span className="text-sm text-slate-500">—</span>
                      )}
                    </td>
                    <td className="px-4 py-4">
                      <span className="text-sm text-slate-400" title={app.updated_at}>
                        {formatRelativeTime(app.updated_at || app.last_deployed_at)}
                      </span>
                    </td>
                    <td className="px-4 py-4">
                      <div className="flex items-center justify-end gap-1">
                        {/* Deploy */}
                        <button
                          onClick={() => handleAction('deploy', app.id)}
                          disabled={app.status === 'deploying'}
                          className="p-1.5 text-slate-400 hover:text-white hover:bg-slate-600 rounded transition-colors disabled:opacity-50"
                          title="Deploy"
                        >
                          <ArrowUpCircle className="w-4 h-4" />
                        </button>

                        {/* Restart */}
                        <button
                          onClick={() => handleAction('restart', app.id)}
                          disabled={app.status !== 'running'}
                          className="p-1.5 text-slate-400 hover:text-white hover:bg-slate-600 rounded transition-colors disabled:opacity-50"
                          title="Restart"
                        >
                          <RefreshCw className="w-4 h-4" />
                        </button>

                        {/* Stop */}
                        <button
                          onClick={() => handleAction('stop', app.id)}
                          disabled={app.status !== 'running'}
                          className="p-1.5 text-slate-400 hover:text-white hover:bg-slate-600 rounded transition-colors disabled:opacity-50"
                          title="Stop"
                        >
                          <Square className="w-4 h-4" />
                        </button>

                        {/* Logs */}
                        <Link
                          href={`/apps/${app.id}?tab=logs`}
                          className="p-1.5 text-slate-400 hover:text-white hover:bg-slate-600 rounded transition-colors"
                          title="Logs"
                        >
                          <FileText className="w-4 h-4" />
                        </Link>

                        {/* More */}
                        <Link
                          href={`/apps/${app.id}`}
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
                  Showing {((meta.page - 1) * meta.per_page) + 1} to {Math.min(meta.page * meta.per_page, meta.total)} of {meta.total} apps
                </p>
                <div className="flex items-center gap-2">
                  <button
                    onClick={() => setPage(page - 1)}
                    disabled={page === 1}
                    className="p-1.5 text-slate-400 hover:text-white hover:bg-slate-700 rounded disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
                  >
                    <ChevronLeft className="w-4 h-4" />
                  </button>
                  <span className="text-sm text-slate-300">
                    Page {meta.page} of {meta.total_pages}
                  </span>
                  <button
                    onClick={() => setPage(page + 1)}
                    disabled={page >= meta.total_pages}
                    className="p-1.5 text-slate-400 hover:text-white hover:bg-slate-700 rounded disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
                  >
                    <ChevronRight className="w-4 h-4" />
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
