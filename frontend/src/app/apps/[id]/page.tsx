'use client'

import { useState } from 'react'
import { useParams, useRouter, useSearchParams } from 'next/navigation'

export const dynamic = 'force-dynamic'
export const dynamicParams = true

import Link from 'next/link'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import {
  ArrowLeft,
  RefreshCw,
  Play,
  Square,
  MoreHorizontal,
  Loader2,
  ExternalLink,
  GitBranch,
  Clock,
  Server as ServerIcon,
  Database,
  HardDrive,
  Plus,
  Trash2,
  Eye,
  EyeOff,
  Copy,
  Check,
  FileText,
  Settings,
  Globe,
  Shield,
  AlertTriangle
} from 'lucide-react'
import { api, ApiError } from '@/lib/api'
import { useToastStore } from '@/stores'

type AppStatus = 'running' | 'stopped' | 'deploying' | 'failed'

interface AppDetail {
  id: string
  name: string
  status: AppStatus
  health_status?: string
  runtime: string
  runtime_version?: string
  primary_domain?: string
  domains?: Array<{
    domain: string
    ssl_status: string
    ssl_expires_at?: string
    force_https: boolean
  }>
  source?: {
    type: string
    provider?: string
    repo_url?: string
    branch?: string
    auto_deploy?: boolean
    last_commit?: string
    last_commit_message?: string
    last_commit_author?: string
    last_commit_timestamp?: string
  }
  build?: {
    strategy: string
    build_command?: string
    start_command?: string
    pre_deploy_hook?: string
    post_deploy_hook?: string
    health_check?: {
      path: string
      interval: number
      timeout: number
      retries: number
    }
  }
  resources?: {
    cpu_limit: number
    memory_limit_mb: number
    memory_swap_mb?: number
  }
  volumes?: Array<{
    id: string
    host_path: string
    container_path: string
    size_mb?: number
  }>
  connected_services?: Array<{
    id: string
    name: string
    type: string
  }>
  created_at: string
  updated_at: string
}

interface EnvVariable {
  key: string
  value: string
  secret: boolean
  created_at?: string
  updated_at?: string
}

interface Deployment {
  id: string
  app_id: string
  status: 'pending' | 'building' | 'deploying' | 'success' | 'failed'
  commit?: string
  commit_message?: string
  commit_author?: string
  branch?: string
  build_duration_seconds?: number
  deploy_duration_seconds?: number
  started_at: string
  completed_at?: string
  triggered_by?: string
  triggered_by_user?: string
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

export default function AppDetailPage() {
  const params = useParams()
  const router = useRouter()
  const searchParams = useSearchParams()
  const { addToast } = useToastStore()
  const queryClient = useQueryClient()
  
  const appId = params.id as string
  const initialTab = searchParams.get('tab') || 'overview'
  
  const [activeTab, setActiveTab] = useState(initialTab)
  const [showRestartConfirm, setShowRestartConfirm] = useState(false)
  const [showDeleteConfirm, setShowDeleteConfirm] = useState(false)
  const [deleteConfirmName, setDeleteConfirmName] = useState('')
  const [visibleSecrets, setVisibleSecrets] = useState<Set<string>>(new Set())
  const [envVars, setEnvVars] = useState<EnvVariable[]>([])
  const [newEnvKey, setNewEnvKey] = useState('')
  const [newEnvValue, setNewEnvValue] = useState('')
  const [newEnvSecret, setNewEnvSecret] = useState(false)
  // Map secret to secret for API compatibility
  const envVarsWithSecret = envVars.map(v => ({ ...v, secret: v.secret || (v as unknown as { secret?: boolean }).secret }))
  const [expandedDeployment, setExpandedDeployment] = useState<string | null>(null)
  const [deploymentLogs, setDeploymentLogs] = useState<Record<string, string>>( {})

  // Fetch app details
  const { data: app, isLoading: appLoading, error: appError } = useQuery<AppDetail, ApiError>({
    queryKey: ['app', appId],
    queryFn: () => api.apps.get(appId) as Promise<AppDetail>,
  })

  // Fetch deployments
  const { data: deployments, isLoading: deploymentsLoading } = useQuery({
    queryKey: ['app-deployments', appId],
    queryFn: () => api.apps.getDeployments(appId) as unknown as Promise<Deployment[]>,
  })

  // Fetch environment variables
  const { data: envData } = useQuery({
    queryKey: ['app-env', appId],
    queryFn: async () => {
      const response = await api.apps.getEnv(appId)
      const data = response as unknown as { variables: EnvVariable[] }
      setEnvVars(data.variables || [])
      return data
    },
  })

  // Restart mutation
  const restartMutation = useMutation({
    mutationFn: () => api.apps.restart(appId),
    onSuccess: () => {
      addToast({ type: 'success', title: 'App restarted', message: 'Your app is restarting.' })
      queryClient.invalidateQueries({ queryKey: ['app', appId] })
      setShowRestartConfirm(false)
    },
    onError: (error: ApiError) => {
      addToast({ type: 'error', title: 'Restart failed', message: error.message })
    },
  })

  // Stop mutation
  const stopMutation = useMutation({
    mutationFn: () => api.apps.stop(appId),
    onSuccess: () => {
      addToast({ type: 'success', title: 'App stopped', message: 'Your app has been stopped.' })
      queryClient.invalidateQueries({ queryKey: ['app', appId] })
    },
    onError: (error: ApiError) => {
      addToast({ type: 'error', title: 'Stop failed', message: error.message })
    },
  })

  // Deploy mutation
  const deployMutation = useMutation({
    mutationFn: () => api.apps.deploy(appId),
    onSuccess: () => {
      addToast({ type: 'success', title: 'Deployment started', message: 'Your app is being deployed.' })
      queryClient.invalidateQueries({ queryKey: ['app', appId] })
      queryClient.invalidateQueries({ queryKey: ['app-deployments', appId] })
    },
    onError: (error: ApiError) => {
      addToast({ type: 'error', title: 'Deployment failed', message: error.message })
    },
  })

  // Update env mutation
  const updateEnvMutation = useMutation({
    mutationFn: (variables: EnvVariable[]) => api.apps.updateEnv(appId, variables),
    onSuccess: () => {
      addToast({ type: 'success', title: 'Environment updated', message: 'Restart app to apply changes.' })
      queryClient.invalidateQueries({ queryKey: ['app-env', appId] })
    },
    onError: (error: ApiError) => {
      addToast({ type: 'error', title: 'Update failed', message: error.message })
    },
  })

  // Delete mutation
  const deleteMutation = useMutation({
    mutationFn: () => api.apps.delete(appId),
    onSuccess: () => {
      addToast({ type: 'success', title: 'App deleted', message: 'App and all resources have been removed.' })
      router.push('/apps')
    },
    onError: (error: ApiError) => {
      addToast({ type: 'error', title: 'Delete failed', message: error.message })
      setShowDeleteConfirm(false)
    },
  })

  const toggleSecretVisibility = (key: string) => {
    const newSet = new Set(visibleSecrets)
    if (newSet.has(key)) {
      newSet.delete(key)
    } else {
      newSet.add(key)
    }
    setVisibleSecrets(newSet)
  }

  const copyToClipboard = (text: string) => {
    navigator.clipboard.writeText(text)
    addToast({ type: 'success', title: 'Copied', message: 'Copied to clipboard.' })
  }

  const handleAddEnvVar = () => {
    if (!newEnvKey.trim()) return
    const newVar: EnvVariable = {
      key: newEnvKey.trim(),
      value: newEnvValue,
      secret: newEnvSecret,
    }
    updateEnvMutation.mutate([...envVars, newVar])
    setNewEnvKey('')
    setNewEnvValue('')
    setNewEnvSecret(false)
  }

  const handleDeleteEnvVar = (key: string) => {
    const updated = envVars.filter(v => v.key !== key)
    updateEnvMutation.mutate(updated)
  }

  const fetchDeploymentLogs = async (deploymentId: string) => {
    if (deploymentLogs[deploymentId]) return
    try {
      const response = await api.apps.getDeploymentLogs(appId, deploymentId)
      const data = response as unknown as { logs: string }
      setDeploymentLogs(prev => ({ ...prev, [deploymentId]: data.logs || 'No logs available' }))
    } catch {
      setDeploymentLogs(prev => ({ ...prev, [deploymentId]: 'Failed to fetch logs' }))
    }
  }

  if (appLoading) {
    return (
      <div className="flex items-center justify-center h-64">
        <Loader2 className="w-8 h-8 text-primary-500 animate-spin" />
      </div>
    )
  }

  if (appError || !app) {
    return (
      <div className="p-6">
        <Link href="/apps" className="flex items-center gap-2 text-slate-400 hover:text-white mb-4">
          <ArrowLeft className="w-4 h-4" />
          Back to Apps
        </Link>
        <div className="bg-slate-800 border border-slate-700 rounded-lg p-6 text-center">
          <p className="text-slate-400">Failed to load app details</p>
        </div>
      </div>
    )
  }

  return (
    <div className="p-6">
      {/* Header */}
      <div className="flex items-center gap-4 mb-6">
        <Link href="/apps" className="p-2 text-slate-400 hover:text-white hover:bg-slate-800 rounded-md transition-colors">
          <ArrowLeft className="w-5 h-5" />
        </Link>
        <div className="flex-1">
          <div className="flex items-center gap-3">
            <h1 className="text-2xl font-semibold text-white">{app.name}</h1>
            <span className={`flex items-center gap-1.5 px-2 py-0.5 rounded-full text-xs font-medium ${statusColors[app.status]?.bg} ${statusColors[app.status]?.text}`}>
              <span className={`w-1.5 h-1.5 rounded-full ${statusColors[app.status]?.dot}`} />
              {app.status}
              {app.status === 'deploying' && <RefreshCw className="w-3 h-3 animate-spin ml-1" />}
            </span>
          </div>
          <p className="text-sm text-slate-400 mt-1">
            {app.runtime} {app.runtime_version && `• ${app.runtime_version}`}
            {app.primary_domain && ` • ${app.primary_domain}`}
          </p>
        </div>
        <div className="flex items-center gap-2">
          <button
            onClick={() => deployMutation.mutate()}
            disabled={app.status === 'deploying'}
            className="flex items-center gap-2 px-4 py-2 bg-primary-600 hover:bg-primary-700 text-white rounded-md text-sm font-medium transition-colors disabled:opacity-50"
          >
            <RefreshCw className={`w-4 h-4 ${app.status === 'deploying' ? 'animate-spin' : ''}`} />
            Deploy
          </button>
          {app.status === 'running' ? (
            <button
              onClick={() => setShowRestartConfirm(true)}
              className="p-2 text-slate-400 hover:text-white hover:bg-slate-800 rounded-md transition-colors"
              title="Restart"
            >
              <RefreshCw className="w-5 h-5" />
            </button>
          ) : app.status === 'stopped' ? (
            <button
              className="p-2 text-slate-400 hover:text-white hover:bg-slate-800 rounded-md transition-colors"
              title="Start"
            >
              <Play className="w-5 h-5" />
            </button>
          ) : null}
          {app.status === 'running' && (
            <button
              onClick={() => stopMutation.mutate()}
              className="p-2 text-slate-400 hover:text-white hover:bg-slate-800 rounded-md transition-colors"
              title="Stop"
            >
              <Square className="w-5 h-5" />
            </button>
          )}
          <button className="p-2 text-slate-400 hover:text-white hover:bg-slate-800 rounded-md transition-colors">
            <MoreHorizontal className="w-5 h-5" />
          </button>
        </div>
      </div>

      {/* Tabs */}
      <div className="flex gap-1 border-b border-slate-700 mb-6">
        {['overview', 'deployments', 'logs', 'environment', 'settings'].map((tab) => (
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
          {/* App Info */}
          <div className="bg-slate-800 border border-slate-700 rounded-lg p-6">
            <h2 className="text-lg font-medium text-white mb-4">App Info</h2>
            <dl className="space-y-3">
              <div className="flex justify-between">
                <dt className="text-slate-400">Status</dt>
                <dd className={`font-medium ${statusColors[app.status]?.text}`}>
                  {app.health_status || app.status}
                </dd>
              </div>
              <div className="flex justify-between">
                <dt className="text-slate-400">Runtime</dt>
                <dd className="text-white">{app.runtime} {app.runtime_version}</dd>
              </div>
              {app.source?.repo_url && (
                <>
                  <div className="flex justify-between">
                    <dt className="text-slate-400">Repository</dt>
                    <dd className="text-white truncate max-w-[200px]">
                      {app.source.repo_url.replace('https://github.com/', '')}
                    </dd>
                  </div>
                  <div className="flex justify-between">
                    <dt className="text-slate-400">Branch</dt>
                    <dd className="flex items-center gap-1 text-white">
                      <GitBranch className="w-3 h-3" />
                      {app.source.branch || 'main'}
                    </dd>
                  </div>
                  {app.source.last_commit && (
                    <div className="flex justify-between">
                      <dt className="text-slate-400">Last Commit</dt>
                      <dd className="text-white font-mono text-sm">{app.source.last_commit.slice(0, 7)}</dd>
                    </div>
                  )}
                </>
              )}
              <div className="flex justify-between">
                <dt className="text-slate-400">Build Strategy</dt>
                <dd className="text-white capitalize">{app.build?.strategy || 'auto'}</dd>
              </div>
            </dl>
          </div>

          {/* Domains */}
          <div className="bg-slate-800 border border-slate-700 rounded-lg p-6">
            <div className="flex items-center justify-between mb-4">
              <h2 className="text-lg font-medium text-white">Domains</h2>
              <button className="flex items-center gap-1 text-sm text-primary-400 hover:text-primary-300">
                <Plus className="w-4 h-4" />
                Add Domain
              </button>
            </div>
            {app.domains && app.domains.length > 0 ? (
              <div className="space-y-2">
                {app.domains.map((domain) => (
                  <div key={domain.domain} className="flex items-center justify-between p-3 bg-slate-700/50 rounded">
                    <div className="flex items-center gap-2">
                      <Globe className="w-4 h-4 text-slate-400" />
                      <span className="text-white">{domain.domain}</span>
                    </div>
                    <div className="flex items-center gap-2">
                      <span className={`text-xs px-2 py-0.5 rounded ${
                        domain.ssl_status === 'valid' ? 'bg-green-500/20 text-green-400' : 'bg-amber-500/20 text-amber-400'
                      }`}>
                        {domain.ssl_status}
                      </span>
                      <a
                        href={`https://${domain.domain}`}
                        target="_blank"
                        rel="noopener noreferrer"
                        className="p-1 text-slate-400 hover:text-white"
                      >
                        <ExternalLink className="w-4 h-4" />
                      </a>
                    </div>
                  </div>
                ))}
              </div>
            ) : (
              <p className="text-slate-400 text-sm">No domains configured</p>
            )}
          </div>

          {/* Connected Services */}
          <div className="bg-slate-800 border border-slate-700 rounded-lg p-6">
            <div className="flex items-center justify-between mb-4">
              <h2 className="text-lg font-medium text-white">Connected Services</h2>
              <button className="flex items-center gap-1 text-sm text-primary-400 hover:text-primary-300">
                <Plus className="w-4 h-4" />
                Connect
              </button>
            </div>
            {app.connected_services && app.connected_services.length > 0 ? (
              <div className="flex flex-wrap gap-2">
                {app.connected_services.map((service) => (
                  <Link
                    key={service.id}
                    href={`/services/${service.id}`}
                    className="flex items-center gap-2 px-3 py-1.5 bg-slate-700/50 rounded-full text-sm text-white hover:bg-slate-700"
                  >
                    <Database className="w-4 h-4 text-slate-400" />
                    {service.name}
                  </Link>
                ))}
              </div>
            ) : (
              <p className="text-slate-400 text-sm">No services connected</p>
            )}
          </div>

          {/* Volumes */}
          <div className="bg-slate-800 border border-slate-700 rounded-lg p-6">
            <div className="flex items-center justify-between mb-4">
              <h2 className="text-lg font-medium text-white">Persistent Volumes</h2>
              <button className="flex items-center gap-1 text-sm text-primary-400 hover:text-primary-300">
                <Plus className="w-4 h-4" />
                Add Volume
              </button>
            </div>
            {app.volumes && app.volumes.length > 0 ? (
              <div className="space-y-2">
                {app.volumes.map((volume) => (
                  <div key={volume.id} className="flex items-center justify-between p-3 bg-slate-700/50 rounded">
                    <div className="flex items-center gap-2">
                      <HardDrive className="w-4 h-4 text-slate-400" />
                      <span className="text-white text-sm">{volume.container_path}</span>
                    </div>
                    {volume.size_mb && (
                      <span className="text-slate-400 text-sm">
                        {(volume.size_mb / 1024).toFixed(1)} GB
                      </span>
                    )}
                  </div>
                ))}
              </div>
            ) : (
              <p className="text-slate-400 text-sm">No volumes attached</p>
            )}
          </div>
        </div>
      )}

      {activeTab === 'deployments' && (
        <div className="bg-slate-800 border border-slate-700 rounded-lg overflow-hidden">
          <div className="p-4 border-b border-slate-700 flex items-center justify-between">
            <h2 className="text-lg font-medium text-white">Deployments</h2>
            <button
              onClick={() => deployMutation.mutate()}
              className="flex items-center gap-2 px-3 py-1.5 bg-primary-600 hover:bg-primary-700 text-white rounded text-sm"
            >
              <RefreshCw className="w-4 h-4" />
              Deploy Now
            </button>
          </div>
          {deploymentsLoading ? (
            <div className="flex items-center justify-center h-32">
              <Loader2 className="w-6 h-6 text-primary-500 animate-spin" />
            </div>
          ) : deployments && deployments.length > 0 ? (
            <table className="w-full">
              <thead>
                <tr className="border-b border-slate-700">
                  <th className="px-4 py-3 text-left text-xs font-medium text-slate-400 uppercase">Status</th>
                  <th className="px-4 py-3 text-left text-xs font-medium text-slate-400 uppercase">Commit</th>
                  <th className="px-4 py-3 text-left text-xs font-medium text-slate-400 uppercase">Branch</th>
                  <th className="px-4 py-3 text-left text-xs font-medium text-slate-400 uppercase">Author</th>
                  <th className="px-4 py-3 text-left text-xs font-medium text-slate-400 uppercase">Duration</th>
                  <th className="px-4 py-3 text-left text-xs font-medium text-slate-400 uppercase">Time</th>
                  <th className="px-4 py-3 text-right text-xs font-medium text-slate-400 uppercase">Actions</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-slate-700">
                {deployments.map((deployment) => (
                  <tr key={deployment.id} className="hover:bg-slate-700/50">
                    <td className="px-4 py-4">
                      <span className={`inline-flex items-center gap-1.5 text-xs font-medium ${
                        deployment.status === 'success' ? 'text-green-400' :
                        deployment.status === 'failed' ? 'text-red-400' :
                        deployment.status === 'building' || deployment.status === 'deploying' ? 'text-amber-400' :
                        'text-slate-400'
                      }`}>
                        <span className={`w-1.5 h-1.5 rounded-full ${
                          deployment.status === 'success' ? 'bg-green-400' :
                          deployment.status === 'failed' ? 'bg-red-400' :
                          deployment.status === 'building' || deployment.status === 'deploying' ? 'bg-amber-400 animate-pulse' :
                          'bg-slate-400'
                        }`} />
                        {deployment.status}
                      </span>
                    </td>
                    <td className="px-4 py-4">
                      <span className="text-white font-mono text-sm">
                        {deployment.commit?.slice(0, 7) || '—'}
                      </span>
                    </td>
                    <td className="px-4 py-4 text-slate-300 text-sm">
                      <span className="flex items-center gap-1">
                        <GitBranch className="w-3 h-3" />
                        {deployment.branch || 'main'}
                      </span>
                    </td>
                    <td className="px-4 py-4 text-slate-300 text-sm">
                      {deployment.commit_author || deployment.triggered_by_user || '—'}
                    </td>
                    <td className="px-4 py-4 text-slate-400 text-sm">
                      {deployment.build_duration_seconds ? `${deployment.build_duration_seconds}s` : '—'}
                    </td>
                    <td className="px-4 py-4 text-slate-400 text-sm">
                      {formatRelativeTime(deployment.started_at)}
                    </td>
                    <td className="px-4 py-4 text-right">
                      <button
                        onClick={() => {
                          setExpandedDeployment(expandedDeployment === deployment.id ? null : deployment.id)
                          if (!deploymentLogs[deployment.id]) {
                            fetchDeploymentLogs(deployment.id)
                          }
                        }}
                        className="p-1.5 text-slate-400 hover:text-white hover:bg-slate-600 rounded transition-colors"
                      >
                        <FileText className="w-4 h-4" />
                      </button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          ) : (
            <div className="p-8 text-center text-slate-400">
              No deployments yet
            </div>
          )}
        </div>
      )}

      {activeTab === 'logs' && (
        <div className="bg-slate-800 border border-slate-700 rounded-lg overflow-hidden">
          <div className="p-4 border-b border-slate-700 flex items-center justify-between">
            <h2 className="text-lg font-medium text-white">Logs</h2>
            <div className="flex items-center gap-2">
              <button className="p-2 text-slate-400 hover:text-white hover:bg-slate-700 rounded transition-colors">
                <FileText className="w-4 h-4" />
              </button>
            </div>
          </div>
          <div className="h-96 overflow-auto p-4 font-mono text-sm">
            <p className="text-slate-500">Logs streaming will be implemented with WebSocket</p>
          </div>
        </div>
      )}

      {activeTab === 'environment' && (
        <div className="bg-slate-800 border border-slate-700 rounded-lg overflow-hidden">
          <div className="p-4 border-b border-slate-700 flex items-center justify-between">
            <h2 className="text-lg font-medium text-white">Environment Variables</h2>
            <button className="flex items-center gap-1 text-sm text-primary-400 hover:text-primary-300">
              <Plus className="w-4 h-4" />
              Import .env
            </button>
          </div>
          
          {/* Add new variable */}
          <div className="p-4 border-b border-slate-700 bg-slate-700/30">
            <div className="flex items-center gap-4">
              <input
                type="text"
                placeholder="KEY"
                value={newEnvKey}
                onChange={(e) => setNewEnvKey(e.target.value.toUpperCase())}
                className="w-48 px-3 py-2 bg-slate-700 border border-slate-600 rounded text-sm text-white placeholder-slate-400"
              />
              <input
                type="text"
                placeholder="VALUE"
                value={newEnvValue}
                onChange={(e) => setNewEnvValue(e.target.value)}
                className="flex-1 px-3 py-2 bg-slate-700 border border-slate-600 rounded text-sm text-white placeholder-slate-400"
              />
              <label className="flex items-center gap-2 text-sm text-slate-400">
                <input
                  type="checkbox"
                  checked={newEnvSecret}
                  onChange={(e) => setNewEnvSecret(e.target.checked)}
                  className="rounded border-slate-600 bg-slate-700"
                />
                Secret
              </label>
              <button
                onClick={handleAddEnvVar}
                disabled={!newEnvKey.trim() || updateEnvMutation.isPending}
                className="px-4 py-2 bg-primary-600 hover:bg-primary-700 text-white rounded text-sm disabled:opacity-50"
              >
                Add
              </button>
            </div>
          </div>

          {/* Variables table */}
          <div className="divide-y divide-slate-700">
            {envVars.map((envVar) => (
              <div key={envVar.key} className="flex items-center justify-between px-4 py-3 hover:bg-slate-700/30">
                <div className="flex items-center gap-4 flex-1">
                  <span className="text-primary-400 font-mono text-sm w-48">{envVar.key}</span>
                  <span className="text-slate-300 font-mono text-sm flex-1 truncate">
                    {envVar.secret && !visibleSecrets.has(envVar.key)
                      ? '••••••••••••'
                      : envVar.value}
                  </span>
                </div>
                <div className="flex items-center gap-1">
                  {envVar.secret && (
                    <button
                      onClick={() => toggleSecretVisibility(envVar.key)}
                      className="p-1.5 text-slate-400 hover:text-white hover:bg-slate-600 rounded transition-colors"
                    >
                      {visibleSecrets.has(envVar.key) ? <EyeOff className="w-4 h-4" /> : <Eye className="w-4 h-4" />}
                    </button>
                  )}
                  <button
                    onClick={() => copyToClipboard(envVar.value)}
                    className="p-1.5 text-slate-400 hover:text-white hover:bg-slate-600 rounded transition-colors"
                  >
                    <Copy className="w-4 h-4" />
                  </button>
                  <button
                    onClick={() => handleDeleteEnvVar(envVar.key)}
                    className="p-1.5 text-slate-400 hover:text-red-400 hover:bg-slate-600 rounded transition-colors"
                  >
                    <Trash2 className="w-4 h-4" />
                  </button>
                </div>
              </div>
            ))}
          </div>

          <div className="p-4 border-t border-slate-700 bg-amber-500/10">
            <p className="text-sm text-amber-400 flex items-center gap-2">
              <AlertTriangle className="w-4 h-4" />
              Changes require app restart to take effect.
            </p>
          </div>
        </div>
      )}

      {activeTab === 'settings' && (
        <div className="space-y-6">
          {/* Build & Deploy */}
          <div className="bg-slate-800 border border-slate-700 rounded-lg p-6">
            <h2 className="text-lg font-medium text-white mb-4">Build & Deploy</h2>
            <dl className="space-y-4">
              <div>
                <dt className="text-sm text-slate-400 mb-1">Git Repository</dt>
                <dd className="text-white">{app.source?.repo_url || '—'}</dd>
              </div>
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <dt className="text-sm text-slate-400 mb-1">Branch</dt>
                  <dd className="text-white">{app.source?.branch || 'main'}</dd>
                </div>
                <div>
                  <dt className="text-sm text-slate-400 mb-1">Build Strategy</dt>
                  <dd className="text-white capitalize">{app.build?.strategy || 'auto-detect'}</dd>
                </div>
              </div>
              {app.build?.build_command && (
                <div>
                  <dt className="text-sm text-slate-400 mb-1">Build Command</dt>
                  <dd className="text-white font-mono text-sm bg-slate-700 px-2 py-1 rounded inline-block">
                    {app.build.build_command}
                  </dd>
                </div>
              )}
              {app.build?.start_command && (
                <div>
                  <dt className="text-sm text-slate-400 mb-1">Start Command</dt>
                  <dd className="text-white font-mono text-sm bg-slate-700 px-2 py-1 rounded inline-block">
                    {app.build.start_command}
                  </dd>
                </div>
              )}
            </dl>
          </div>

          {/* Danger Zone */}
          <div className="bg-slate-800 border border-red-900/50 rounded-lg p-6">
            <h2 className="text-lg font-medium text-red-400 mb-4">Danger Zone</h2>
            <div className="flex items-center justify-between">
              <div>
                <p className="text-white font-medium">Delete App</p>
                <p className="text-sm text-slate-400">This will permanently delete the app and all associated resources.</p>
              </div>
              {!showDeleteConfirm ? (
                <button
                  onClick={() => setShowDeleteConfirm(true)}
                  className="px-4 py-2 bg-red-600/20 hover:bg-red-600 text-red-400 hover:text-white rounded text-sm transition-colors"
                >
                  Delete App
                </button>
              ) : (
                <div className="flex items-center gap-2">
                  <input
                    type="text"
                    placeholder={`Type "${app.name}" to confirm`}
                    value={deleteConfirmName}
                    onChange={(e) => setDeleteConfirmName(e.target.value)}
                    className="px-3 py-2 bg-slate-700 border border-slate-600 rounded text-sm text-white w-48"
                  />
                  <button
                    onClick={() => deleteMutation.mutate()}
                    disabled={deleteConfirmName !== app.name}
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

      {/* Restart Confirmation Modal */}
      {showRestartConfirm && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50">
          <div className="bg-slate-800 border border-slate-700 rounded-lg p-6 max-w-md w-full mx-4">
            <h3 className="text-lg font-medium text-white mb-2">Restart App?</h3>
            <p className="text-slate-400 mb-4">
              This will restart the {app.name} application. The app will be unavailable briefly.
            </p>
            <div className="flex justify-end gap-2">
              <button
                onClick={() => setShowRestartConfirm(false)}
                className="px-4 py-2 bg-slate-700 hover:bg-slate-600 text-white rounded text-sm"
              >
                Cancel
              </button>
              <button
                onClick={() => restartMutation.mutate()}
                disabled={restartMutation.isPending}
                className="px-4 py-2 bg-primary-600 hover:bg-primary-700 text-white rounded text-sm disabled:opacity-50"
              >
                {restartMutation.isPending ? 'Restarting...' : 'Restart'}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}
