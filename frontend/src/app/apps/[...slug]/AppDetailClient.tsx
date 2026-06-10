'use client'

import { useState, useEffect, useRef } from 'react'
import { useRouter, useSearchParams, useParams } from 'next/navigation'

import Link from 'next/link'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import {
  ArrowLeft,
  RefreshCw,
  Play,
  Square,
  Loader2,
  ExternalLink,
  GitBranch,
  Globe,
  Database,
  HardDrive,
  Plus,
  Trash2,
  Eye,
  EyeOff,
  Copy,
  FileText,
  Upload,
  History,
  X,
  AlertTriangle
} from 'lucide-react'
import { api, ApiError } from '@/lib/api'
import { useToastStore } from '@/stores'
import { getWebSocket } from '@/lib/websocket'

type AppStatus = 'running' | 'stopped' | 'deploying' | 'failed' | 'restarting'

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
  status: 'queued' | 'in_progress' | 'success' | 'failed' | 'cancelled'
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

interface ServiceItem {
  id: string
  name: string
  type: string
  status: string
}

const statusColors: Record<AppStatus, { bg: string; text: string; dot: string }> = {
  running: { bg: 'bg-green-500/10', text: 'text-green-500', dot: 'bg-green-500' },
  stopped: { bg: 'bg-slate-500/10', text: 'text-slate-400', dot: 'bg-slate-400' },
  deploying: { bg: 'bg-amber-500/10', text: 'text-amber-500', dot: 'bg-amber-500' },
  failed: { bg: 'bg-red-500/10', text: 'text-red-500', dot: 'bg-red-500' },
  restarting: { bg: 'bg-blue-500/10', text: 'text-blue-500', dot: 'bg-blue-500' },
}

function formatRelativeTime(dateString?: string): string {
  if (!dateString) return '\u2014'
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

export function AppDetailClient() {
  const router = useRouter()
  const searchParams = useSearchParams()
  const { addToast } = useToastStore()
  const queryClient = useQueryClient()

  const params = useParams()
  const slug = (params?.slug as string[])?.[0] || ''
  const appId = slug && slug !== 'new' && slug !== '_' ? slug : ''
  const initialTab = searchParams.get('tab') || 'overview'

  const isInvalidAppId = !appId || appId === 'new' || appId === '_'

  const [activeTab, setActiveTab] = useState(initialTab)
  const [showRestartConfirm, setShowRestartConfirm] = useState(false)
  const [showDeleteConfirm, setShowDeleteConfirm] = useState(false)
  const [deleteConfirmName, setDeleteConfirmName] = useState('')
  const [visibleSecrets, setVisibleSecrets] = useState<Set<string>>(new Set())
  const [envVars, setEnvVars] = useState<EnvVariable[]>([])
  const [deletedEnvKeys, setDeletedEnvKeys] = useState<string[]>([])
  const [newEnvKey, setNewEnvKey] = useState('')
  const [newEnvValue, setNewEnvValue] = useState('')
  const [newEnvSecret, setNewEnvSecret] = useState(false)
  const [expandedDeployment, setExpandedDeployment] = useState<string | null>(null)
  const [deploymentLogs, setDeploymentLogs] = useState<Record<string, string>>({})

  // Modal states
  const [showAddDomain, setShowAddDomain] = useState(false)
  const [newDomain, setNewDomain] = useState('')
  const [forceHttps, setForceHttps] = useState(true)
  const [showAddVolume, setShowAddVolume] = useState(false)
  const [newVolumePath, setNewVolumePath] = useState('')
  const [newVolumeName, setNewVolumeName] = useState('')
  const [showConnectService, setShowConnectService] = useState(false)
  const [showImportEnv, setShowImportEnv] = useState(false)
  const [importEnvContent, setImportEnvContent] = useState('')
  const [liveLogs, setLiveLogs] = useState<string[]>([])
  const logsEndRef = useRef<HTMLDivElement>(null)
  const [autoScroll, setAutoScroll] = useState(true)

  // Fetch app details - poll while deploying/restarting so status updates
  const { data: app, isLoading: appLoading, error: appError } = useQuery<AppDetail, ApiError>({
    queryKey: ['app', appId],
    queryFn: () => api.apps.get(appId) as Promise<AppDetail>,
    enabled: !!appId && !isInvalidAppId,
    refetchInterval: (query) => {
      const status = query?.state?.data?.status
      return status === 'deploying' || status === 'restarting' ? 3000 : false
    },
  })

  // Fetch deployments
  const { data: deployments, isLoading: deploymentsLoading } = useQuery({
    queryKey: ['app-deployments', appId],
    queryFn: async () => {
      const response = await api.apps.getDeployments(appId)
      return (response as unknown as { data: Deployment[] }).data || []
    },
    enabled: !!appId && !isInvalidAppId,
    refetchInterval: 5000,
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
    enabled: !!appId && !isInvalidAppId,
  })

  // Fetch available services for connect modal
  const { data: servicesData } = useQuery({
    queryKey: ['services-all'],
    queryFn: async () => {
      const response = await api.services.list({ per_page: 100 })
      return (response as unknown as { data: ServiceItem[] }).data || []
    },
    enabled: showConnectService,
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

  // Start mutation
  const startMutation = useMutation({
    mutationFn: () => api.apps.start(appId),
    onSuccess: () => {
      addToast({ type: 'success', title: 'App started', message: 'Your app is starting.' })
      queryClient.invalidateQueries({ queryKey: ['app', appId] })
    },
    onError: (error: ApiError) => {
      addToast({ type: 'error', title: 'Start failed', message: error.message })
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
    mutationFn: ({ variables, deleteKeys }: { variables: EnvVariable[]; deleteKeys: string[] }) =>
      api.apps.updateEnv(appId, variables, deleteKeys),
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

  // Add domain mutation
  const addDomainMutation = useMutation({
    mutationFn: (data: { domain: string; force_https: boolean }) =>
      api.apps.addDomain(appId, data),
    onSuccess: () => {
      addToast({ type: 'success', title: 'Domain added', message: 'SSL certificate will be provisioned automatically.' })
      queryClient.invalidateQueries({ queryKey: ['app', appId] })
      setShowAddDomain(false)
      setNewDomain('')
    },
    onError: (error: ApiError) => {
      addToast({ type: 'error', title: 'Failed to add domain', message: error.message })
    },
  })

  // Remove domain
  const removeDomainMutation = useMutation({
    mutationFn: (domain: string) => api.apps.removeDomain(appId, domain),
    onSuccess: () => {
      addToast({ type: 'success', title: 'Domain removed', message: 'The domain has been removed.' })
      queryClient.invalidateQueries({ queryKey: ['app', appId] })
    },
    onError: (error: ApiError) => {
      addToast({ type: 'error', title: 'Failed to remove domain', message: error.message })
    },
  })

  // Create volume mutation
  const createVolumeMutation = useMutation({
    mutationFn: (data: { container_path: string; name?: string }) =>
      api.apps.createVolume(appId, data),
    onSuccess: () => {
      addToast({ type: 'success', title: 'Volume added', message: 'The volume has been created. Restart the app to apply.' })
      queryClient.invalidateQueries({ queryKey: ['app', appId] })
      setShowAddVolume(false)
      setNewVolumePath('')
      setNewVolumeName('')
    },
    onError: (error: ApiError) => {
      addToast({ type: 'error', title: 'Failed to add volume', message: error.message })
    },
  })

  // Delete volume
  const deleteVolumeMutation = useMutation({
    mutationFn: (volumeId: string) => api.apps.deleteVolume(appId, volumeId),
    onSuccess: () => {
      addToast({ type: 'success', title: 'Volume removed', message: 'The volume has been removed from the app configuration.' })
      queryClient.invalidateQueries({ queryKey: ['app', appId] })
    },
    onError: (error: ApiError) => {
      addToast({ type: 'error', title: 'Failed to remove volume', message: error.message })
    },
  })

  // Connect service
  const connectServiceMutation = useMutation({
    mutationFn: (serviceId: string) => api.services.connectApp(serviceId, appId),
    onSuccess: () => {
      addToast({ type: 'success', title: 'Service connected', message: 'The service has been connected to this app.' })
      queryClient.invalidateQueries({ queryKey: ['app', appId] })
      setShowConnectService(false)
    },
    onError: (error: ApiError) => {
      addToast({ type: 'error', title: 'Failed to connect service', message: error.message })
    },
  })

  // Disconnect service
  const disconnectServiceMutation = useMutation({
    mutationFn: (serviceId: string) => api.services.disconnectApp(serviceId, appId),
    onSuccess: () => {
      addToast({ type: 'success', title: 'Service disconnected', message: 'The service has been disconnected.' })
      queryClient.invalidateQueries({ queryKey: ['app', appId] })
    },
    onError: (error: ApiError) => {
      addToast({ type: 'error', title: 'Failed to disconnect service', message: error.message })
    },
  })

  // Import env
  const importEnvMutation = useMutation({
    mutationFn: (content: string) => api.apps.importEnv(appId, content),
    onSuccess: (data) => {
      const imported = (data as unknown as { imported?: number; message?: string }).imported
      addToast({ type: 'success', title: 'Environment imported', message: data.message || `Imported ${imported ?? 0} variables.` })
      queryClient.invalidateQueries({ queryKey: ['app-env', appId] })
      setShowImportEnv(false)
      setImportEnvContent('')
    },
    onError: (error: ApiError) => {
      addToast({ type: 'error', title: 'Failed to import environment', message: error.message })
    },
  })

  // Rollback to a previous deployment
  const rollbackMutation = useMutation({
    mutationFn: (deploymentId: string) => api.apps.rollback(appId, deploymentId),
    onSuccess: () => {
      addToast({ type: 'success', title: 'Rollback started', message: 'Rolling back to that deployment.' })
      queryClient.invalidateQueries({ queryKey: ['app', appId] })
      queryClient.invalidateQueries({ queryKey: ['app-deployments', appId] })
    },
    onError: (error: ApiError) => {
      addToast({ type: 'error', title: 'Rollback failed', message: error.message })
    },
  })

  // Real-time logs via WebSocket (with polling fallback)
  useEffect(() => {
    if (activeTab !== 'logs' || isInvalidAppId || !appId) return
    setLiveLogs([])

    const fetchRecentLogs = async () => {
      try {
        const res = await api.apps.getLogs(appId, { tail: 200 })
        const lines = (res as unknown as { lines: Array<{ message: string; timestamp: string }> }).lines || []
        const text = lines.map(l => `[${new Date(l.timestamp).toLocaleTimeString()}] ${l.message}`).join('\n')
        setLiveLogs(text ? text.split('\n') : [])
      } catch {
        setLiveLogs(['No logs available'])
      }
    }
    fetchRecentLogs()

    // Subscribe to WebSocket for live updates
    const ws = getWebSocket()
    ws.connect()
    const unsubscribe = ws.onAppLogs(appId, (message: string, stream: 'stdout' | 'stderr') => {
      setLiveLogs(prev => {
        const next = [...prev, `[${new Date().toLocaleTimeString()}] [${stream}] ${message}`]
        return next.slice(-500)
      })
    })

    // Fallback: poll every 5 seconds
    const pollInterval = setInterval(async () => {
      try {
        const res = await api.apps.getLogs(appId, { tail: 100 })
        const lines = (res as unknown as { lines: Array<{ message: string; timestamp: string }> }).lines || []
        if (lines.length > 0) {
          const text = lines.map((l: { message: string; timestamp: string }) => `[${new Date(l.timestamp).toLocaleTimeString()}] ${l.message}`).join('\n')
          setLiveLogs(prev => {
            const lastNew = text.split('\n').slice(-1)[0]
            if (prev.length === 0 || prev[prev.length - 1] !== lastNew) {
              return text.split('\n')
            }
            return prev
          })
        }
      } catch {
        // Ignore
      }
    }, 5000)

    return () => {
      unsubscribe()
      clearInterval(pollInterval)
    }
  }, [activeTab, appId, isInvalidAppId])

  // Auto-scroll logs to bottom
  useEffect(() => {
    if (autoScroll && logsEndRef.current) {
      logsEndRef.current.scrollIntoView({ behavior: 'smooth' })
    }
  }, [liveLogs, autoScroll])

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
    const updated = [...envVars, newVar]
    setEnvVars(updated)
    updateEnvMutation.mutate({ variables: updated, deleteKeys: deletedEnvKeys })
    setNewEnvKey('')
    setNewEnvValue('')
    setNewEnvSecret(false)
  }

  const handleDeleteEnvVar = (key: string) => {
    const updated = envVars.filter(v => v.key !== key)
    setEnvVars(updated)
    const newDeletedKeys = [...deletedEnvKeys, key]
    setDeletedEnvKeys(newDeletedKeys)
    updateEnvMutation.mutate({ variables: updated, deleteKeys: newDeletedKeys }, {
      onSuccess: () => {
        setDeletedEnvKeys([])
      }
    })
  }

  const handleAddDomain = () => {
    if (!newDomain.trim()) return
    addDomainMutation.mutate({ domain: newDomain.trim(), force_https: forceHttps })
  }

  const handleAddVolume = () => {
    if (!newVolumePath.trim()) return
    createVolumeMutation.mutate({
      container_path: newVolumePath.trim(),
      name: newVolumeName.trim() || undefined,
    })
  }

  const handleImportEnv = () => {
    if (!importEnvContent.trim()) return
    importEnvMutation.mutate(importEnvContent)
  }

  const handleFileUpload = (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0]
    if (!file) return
    const reader = new FileReader()
    reader.onload = (ev) => {
      const text = ev.target?.result as string
      setImportEnvContent(text)
    }
    reader.readAsText(file)
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

  const availableServices = (servicesData || []).filter(
    s => !app?.connected_services?.some(cs => cs.id === s.id)
  )

  if (isInvalidAppId) {
    return (
      <div className="flex items-center justify-center h-64">
        <Loader2 className="w-8 h-8 text-primary-500 animate-spin" />
      </div>
    )
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
              {(app.status === 'deploying' || app.status === 'restarting') && <RefreshCw className="w-3 h-3 animate-spin ml-1" />}
            </span>
          </div>
          <p className="text-sm text-slate-400 mt-1">
            {app.runtime} {app.runtime_version && `\u2022 ${app.runtime_version}`}
            {app.primary_domain && ` \u2022 ${app.primary_domain}`}
          </p>
        </div>
        <div className="flex items-center gap-2">
          <button
            onClick={() => deployMutation.mutate()}
            disabled={app.status === 'deploying' || app.status === 'restarting'}
            className="flex items-center gap-2 px-4 py-2 bg-primary-600 hover:bg-primary-700 text-white rounded-md text-sm font-medium transition-colors disabled:opacity-50"
          >
            <RefreshCw className={`w-4 h-4 ${(app.status === 'deploying' || app.status === 'restarting') ? 'animate-spin' : ''}`} />
            Deploy
          </button>
          {app.status === 'running' && (
            <button
              onClick={() => setShowRestartConfirm(true)}
              className="p-2 text-slate-400 hover:text-white hover:bg-slate-800 rounded-md transition-colors"
              title="Restart"
            >
              <RefreshCw className="w-5 h-5" />
            </button>
          )}
          {app.status === 'stopped' && (
            <button
              onClick={() => startMutation.mutate()}
              className="p-2 text-slate-400 hover:text-white hover:bg-slate-800 rounded-md transition-colors"
              title="Start"
            >
              <Play className="w-5 h-5" />
            </button>
          )}
          {app.status === 'running' && (
            <button
              onClick={() => stopMutation.mutate()}
              className="p-2 text-slate-400 hover:text-white hover:bg-slate-800 rounded-md transition-colors"
              title="Stop"
            >
              <Square className="w-5 h-5" />
            </button>
          )}
        </div>
      </div>

      {/* Tabs */}
      <div className="flex gap-1 border-b border-slate-700 mb-6 overflow-x-auto">
        {['overview', 'deployments', 'logs', 'environment', 'settings'].map((tab) => (
          <button
            key={tab}
            onClick={() => setActiveTab(tab)}
            className={`px-4 py-2 text-sm font-medium transition-colors capitalize whitespace-nowrap ${
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
              <button
                onClick={() => setShowAddDomain(true)}
                className="flex items-center gap-1 text-sm text-primary-400 hover:text-primary-300"
              >
                <Plus className="w-4 h-4" />
                Add Domain
              </button>
            </div>
            {app.domains && app.domains.length > 0 ? (
              <div className="space-y-2">
                {app.domains.map((domain) => (
                  <div key={domain.domain} className="flex items-center justify-between p-3 bg-slate-700/50 rounded">
                    <div className="flex items-center gap-2 min-w-0">
                      <Globe className="w-4 h-4 text-slate-400 flex-shrink-0" />
                      <span className="text-white truncate">{domain.domain}</span>
                    </div>
                    <div className="flex items-center gap-2 flex-shrink-0">
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
                      <button
                        onClick={() => {
                          if (confirm(`Remove domain "${domain.domain}"?`)) {
                            removeDomainMutation.mutate(domain.domain)
                          }
                        }}
                        className="p-1 text-slate-400 hover:text-red-400"
                        title="Remove domain"
                      >
                        <Trash2 className="w-4 h-4" />
                      </button>
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
              <button
                onClick={() => setShowConnectService(true)}
                className="flex items-center gap-1 text-sm text-primary-400 hover:text-primary-300"
              >
                <Plus className="w-4 h-4" />
                Connect
              </button>
            </div>
            {app.connected_services && app.connected_services.length > 0 ? (
              <div className="flex flex-wrap gap-2">
                {app.connected_services.map((service) => (
                  <div
                    key={service.id}
                    className="flex items-center gap-2 px-3 py-1.5 bg-slate-700/50 rounded-full text-sm text-white"
                  >
                    <Database className="w-4 h-4 text-slate-400" />
                    <Link href={`/services/${service.id}`} className="hover:text-primary-400">
                      {service.name}
                    </Link>
                    <button
                      onClick={() => {
                        if (confirm(`Disconnect from "${service.name}"?`)) {
                          disconnectServiceMutation.mutate(service.id)
                        }
                      }}
                      className="text-slate-400 hover:text-red-400 ml-1"
                    >
                      <X className="w-3 h-3" />
                    </button>
                  </div>
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
              <button
                onClick={() => setShowAddVolume(true)}
                className="flex items-center gap-1 text-sm text-primary-400 hover:text-primary-300"
              >
                <Plus className="w-4 h-4" />
                Add Volume
              </button>
            </div>
            {app.volumes && app.volumes.length > 0 ? (
              <div className="space-y-2">
                {app.volumes.map((volume) => (
                  <div key={volume.id} className="flex items-center justify-between p-3 bg-slate-700/50 rounded">
                    <div className="flex items-center gap-2 min-w-0">
                      <HardDrive className="w-4 h-4 text-slate-400 flex-shrink-0" />
                      <div className="min-w-0">
                        <div className="text-white text-sm truncate">{volume.container_path}</div>
                        <div className="text-xs text-slate-500 truncate">{volume.host_path}</div>
                      </div>
                    </div>
                    <div className="flex items-center gap-2 flex-shrink-0">
                      {volume.size_mb != null && volume.size_mb > 0 && (
                        <span className="text-slate-400 text-sm">
                          {(volume.size_mb / 1024).toFixed(1)} GB
                        </span>
                      )}
                      <button
                        onClick={() => {
                          if (confirm(`Remove volume "${volume.container_path}"?\nThe data on disk will be preserved.`)) {
                            deleteVolumeMutation.mutate(volume.id)
                          }
                        }}
                        className="p-1 text-slate-400 hover:text-red-400"
                        title="Remove volume"
                      >
                        <Trash2 className="w-4 h-4" />
                      </button>
                    </div>
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
              disabled={app.status === 'deploying' || app.status === 'restarting'}
              className="flex items-center gap-2 px-3 py-1.5 bg-primary-600 hover:bg-primary-700 text-white rounded text-sm disabled:opacity-50"
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
            <div className="overflow-x-auto">
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
                          deployment.status === 'in_progress' || deployment.status === 'queued' ? 'text-amber-400' :
                          'text-slate-400'
                        }`}>
                          <span className={`w-1.5 h-1.5 rounded-full ${
                            deployment.status === 'success' ? 'bg-green-400' :
                            deployment.status === 'failed' ? 'bg-red-400' :
                            deployment.status === 'in_progress' || deployment.status === 'queued' ? 'bg-amber-400 animate-pulse' :
                            'bg-slate-400'
                          }`} />
                          {deployment.status}
                        </span>
                      </td>
                      <td className="px-4 py-4">
                        <span className="text-white font-mono text-sm">
                          {deployment.commit?.slice(0, 7) || '\u2014'}
                        </span>
                      </td>
                      <td className="px-4 py-4 text-slate-300 text-sm">
                        <span className="flex items-center gap-1">
                          <GitBranch className="w-3 h-3" />
                          {deployment.branch || 'main'}
                        </span>
                      </td>
                      <td className="px-4 py-4 text-slate-300 text-sm">
                        {deployment.commit_author || deployment.triggered_by_user || '\u2014'}
                      </td>
                      <td className="px-4 py-4 text-slate-400 text-sm">
                        {deployment.build_duration_seconds ? `${deployment.build_duration_seconds}s` : '\u2014'}
                      </td>
                      <td className="px-4 py-4 text-slate-400 text-sm">
                        {formatRelativeTime(deployment.started_at)}
                      </td>
                      <td className="px-4 py-4 text-right">
                        <div className="flex items-center justify-end gap-1">
                          {deployment.status === 'success' && (
                            <button
                              onClick={() => {
                                if (confirm('Rollback to this deployment? This will create a new deployment using this commit.')) {
                                  rollbackMutation.mutate(deployment.id)
                                }
                              }}
                              className="p-1.5 text-slate-400 hover:text-amber-400 hover:bg-slate-600 rounded transition-colors"
                              title="Rollback to this deployment"
                            >
                              <History className="w-4 h-4" />
                            </button>
                          )}
                          <button
                            onClick={() => {
                              setExpandedDeployment(expandedDeployment === deployment.id ? null : deployment.id)
                              if (!deploymentLogs[deployment.id]) {
                                fetchDeploymentLogs(deployment.id)
                              }
                            }}
                            className="p-1.5 text-slate-400 hover:text-white hover:bg-slate-600 rounded transition-colors"
                            title="View logs"
                          >
                            <FileText className="w-4 h-4" />
                          </button>
                        </div>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
              {expandedDeployment && (
                <div className="border-t border-slate-700 bg-slate-900/50 p-4">
                  <div className="text-xs text-slate-400 mb-2">Build Logs</div>
                  <pre className="text-xs font-mono text-slate-300 whitespace-pre-wrap max-h-64 overflow-auto bg-slate-900 rounded p-3">
                    {deploymentLogs[expandedDeployment] || 'Loading...'}
                  </pre>
                </div>
              )}
            </div>
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
            <h2 className="text-lg font-medium text-white">Live Logs</h2>
            <div className="flex items-center gap-2">
              <label className="flex items-center gap-2 text-xs text-slate-400">
                <input
                  type="checkbox"
                  checked={autoScroll}
                  onChange={(e) => setAutoScroll(e.target.checked)}
                  className="rounded border-slate-600 bg-slate-700"
                />
                Auto-scroll
              </label>
              <button
                onClick={() => setLiveLogs([])}
                className="px-2 py-1 text-xs text-slate-400 hover:text-white bg-slate-700 hover:bg-slate-600 rounded"
              >
                Clear
              </button>
            </div>
          </div>
          <div className="h-[600px] overflow-auto p-4 font-mono text-xs bg-slate-900">
            {liveLogs.length === 0 ? (
              <p className="text-slate-500">Waiting for logs...</p>
            ) : (
              <>
                {liveLogs.map((line, idx) => (
                  <div key={idx} className="text-slate-300 leading-relaxed whitespace-pre-wrap break-all">
                    {line}
                  </div>
                ))}
                <div ref={logsEndRef} />
              </>
            )}
          </div>
        </div>
      )}

      {activeTab === 'environment' && (
        <div className="bg-slate-800 border border-slate-700 rounded-lg overflow-hidden">
          <div className="p-4 border-b border-slate-700 flex items-center justify-between">
            <h2 className="text-lg font-medium text-white">Environment Variables</h2>
            <button
              onClick={() => setShowImportEnv(true)}
              className="flex items-center gap-1 text-sm text-primary-400 hover:text-primary-300"
            >
              <Upload className="w-4 h-4" />
              Import .env
            </button>
          </div>

          {/* Add new variable */}
          <div className="p-4 border-b border-slate-700 bg-slate-700/30">
            <div className="flex items-center gap-4 flex-wrap">
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
                className="flex-1 min-w-[200px] px-3 py-2 bg-slate-700 border border-slate-600 rounded text-sm text-white placeholder-slate-400"
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
            {envVars.length === 0 ? (
              <div className="p-8 text-center text-slate-400">
                No environment variables yet
              </div>
            ) : (
              envVars.map((envVar) => (
                <div key={envVar.key} className="flex items-center justify-between px-4 py-3 hover:bg-slate-700/30">
                  <div className="flex items-center gap-4 flex-1 min-w-0">
                    <span className="text-primary-400 font-mono text-sm w-48 truncate">{envVar.key}</span>
                    <span className="text-slate-300 font-mono text-sm flex-1 truncate">
                      {envVar.secret && !visibleSecrets.has(envVar.key)
                        ? '\u2022\u2022\u2022\u2022\u2022\u2022\u2022\u2022\u2022\u2022\u2022\u2022'
                        : envVar.value}
                    </span>
                  </div>
                  <div className="flex items-center gap-1 flex-shrink-0">
                    {envVar.secret && (
                      <button
                        onClick={() => toggleSecretVisibility(envVar.key)}
                        className="p-1.5 text-slate-400 hover:text-white hover:bg-slate-600 rounded transition-colors"
                        title={visibleSecrets.has(envVar.key) ? 'Hide' : 'Show'}
                      >
                        {visibleSecrets.has(envVar.key) ? <EyeOff className="w-4 h-4" /> : <Eye className="w-4 h-4" />}
                      </button>
                    )}
                    <button
                      onClick={() => copyToClipboard(envVar.value)}
                      className="p-1.5 text-slate-400 hover:text-white hover:bg-slate-600 rounded transition-colors"
                      title="Copy"
                    >
                      <Copy className="w-4 h-4" />
                    </button>
                    <button
                      onClick={() => {
                        if (confirm(`Delete environment variable "${envVar.key}"?`)) {
                          handleDeleteEnvVar(envVar.key)
                        }
                      }}
                      className="p-1.5 text-slate-400 hover:text-red-400 hover:bg-slate-600 rounded transition-colors"
                      title="Delete"
                    >
                      <Trash2 className="w-4 h-4" />
                    </button>
                  </div>
                </div>
              ))
            )}
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
                <dd className="text-white">{app.source?.repo_url || '\u2014'}</dd>
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
            <div className="flex items-center justify-between flex-wrap gap-4">
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

      {/* Add Domain Modal */}
      {showAddDomain && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50">
          <div className="bg-slate-800 border border-slate-700 rounded-lg p-6 max-w-md w-full mx-4">
            <div className="flex items-center justify-between mb-4">
              <h3 className="text-lg font-medium text-white">Add Domain</h3>
              <button onClick={() => setShowAddDomain(false)} className="text-slate-400 hover:text-white">
                <X className="w-5 h-5" />
              </button>
            </div>
            <div className="space-y-4">
              <div>
                <label className="block text-sm font-medium text-slate-300 mb-1">Domain</label>
                <input
                  type="text"
                  value={newDomain}
                  onChange={(e) => setNewDomain(e.target.value)}
                  placeholder="my-app.example.com"
                  className="w-full px-3 py-2 bg-slate-700 border border-slate-600 rounded text-white placeholder-slate-400"
                />
              </div>
              <label className="flex items-center gap-2 text-sm text-slate-300">
                <input
                  type="checkbox"
                  checked={forceHttps}
                  onChange={(e) => setForceHttps(e.target.checked)}
                  className="rounded border-slate-600 bg-slate-700"
                />
                Force HTTPS (redirect HTTP to HTTPS)
              </label>
              <div className="flex justify-end gap-2 pt-2">
                <button
                  onClick={() => setShowAddDomain(false)}
                  className="px-4 py-2 bg-slate-700 hover:bg-slate-600 text-white rounded text-sm"
                >
                  Cancel
                </button>
                <button
                  onClick={handleAddDomain}
                  disabled={!newDomain.trim() || addDomainMutation.isPending}
                  className="px-4 py-2 bg-primary-600 hover:bg-primary-700 text-white rounded text-sm disabled:opacity-50"
                >
                  {addDomainMutation.isPending ? 'Adding...' : 'Add Domain'}
                </button>
              </div>
            </div>
          </div>
        </div>
      )}

      {/* Add Volume Modal */}
      {showAddVolume && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50">
          <div className="bg-slate-800 border border-slate-700 rounded-lg p-6 max-w-md w-full mx-4">
            <div className="flex items-center justify-between mb-4">
              <h3 className="text-lg font-medium text-white">Add Volume</h3>
              <button onClick={() => setShowAddVolume(false)} className="text-slate-400 hover:text-white">
                <X className="w-5 h-5" />
              </button>
            </div>
            <div className="space-y-4">
              <div>
                <label className="block text-sm font-medium text-slate-300 mb-1">Container Path</label>
                <input
                  type="text"
                  value={newVolumePath}
                  onChange={(e) => setNewVolumePath(e.target.value)}
                  placeholder="/app/data"
                  className="w-full px-3 py-2 bg-slate-700 border border-slate-600 rounded text-white placeholder-slate-400"
                />
                <p className="text-xs text-slate-500 mt-1">The path inside the container to mount the volume to</p>
              </div>
              <div>
                <label className="block text-sm font-medium text-slate-300 mb-1">Name (optional)</label>
                <input
                  type="text"
                  value={newVolumeName}
                  onChange={(e) => setNewVolumeName(e.target.value)}
                  placeholder="data"
                  className="w-full px-3 py-2 bg-slate-700 border border-slate-600 rounded text-white placeholder-slate-400"
                />
                <p className="text-xs text-slate-500 mt-1">A friendly name. Defaults to the container path basename.</p>
              </div>
              <div className="flex justify-end gap-2 pt-2">
                <button
                  onClick={() => setShowAddVolume(false)}
                  className="px-4 py-2 bg-slate-700 hover:bg-slate-600 text-white rounded text-sm"
                >
                  Cancel
                </button>
                <button
                  onClick={handleAddVolume}
                  disabled={!newVolumePath.trim() || createVolumeMutation.isPending}
                  className="px-4 py-2 bg-primary-600 hover:bg-primary-700 text-white rounded text-sm disabled:opacity-50"
                >
                  {createVolumeMutation.isPending ? 'Adding...' : 'Add Volume'}
                </button>
              </div>
            </div>
          </div>
        </div>
      )}

      {/* Connect Service Modal */}
      {showConnectService && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50">
          <div className="bg-slate-800 border border-slate-700 rounded-lg p-6 max-w-md w-full mx-4">
            <div className="flex items-center justify-between mb-4">
              <h3 className="text-lg font-medium text-white">Connect Service</h3>
              <button onClick={() => setShowConnectService(false)} className="text-slate-400 hover:text-white">
                <X className="w-5 h-5" />
              </button>
            </div>
            {availableServices.length === 0 ? (
              <div className="text-center py-6">
                <Database className="w-12 h-12 text-slate-600 mx-auto mb-3" />
                <p className="text-slate-400 mb-2">No available services to connect</p>
                <Link href="/services" className="text-sm text-primary-400 hover:text-primary-300">
                  Create a service \u2192
                </Link>
              </div>
            ) : (
              <div className="space-y-2 max-h-96 overflow-y-auto">
                {availableServices.map((service) => (
                  <button
                    key={service.id}
                    onClick={() => connectServiceMutation.mutate(service.id)}
                    disabled={connectServiceMutation.isPending}
                    className="w-full flex items-center justify-between p-3 bg-slate-700/50 hover:bg-slate-700 rounded text-left transition-colors disabled:opacity-50"
                  >
                    <div className="flex items-center gap-3">
                      <Database className="w-5 h-5 text-slate-400" />
                      <div>
                        <div className="text-white font-medium">{service.name}</div>
                        <div className="text-xs text-slate-400 capitalize">{service.type} \u2022 {service.status}</div>
                      </div>
                    </div>
                    <span className="text-xs text-primary-400">Connect</span>
                  </button>
                ))}
              </div>
            )}
          </div>
        </div>
      )}

      {/* Import .env Modal */}
      {showImportEnv && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50">
          <div className="bg-slate-800 border border-slate-700 rounded-lg p-6 max-w-2xl w-full mx-4 max-h-[90vh] overflow-y-auto">
            <div className="flex items-center justify-between mb-4">
              <h3 className="text-lg font-medium text-white">Import .env File</h3>
              <button onClick={() => setShowImportEnv(false)} className="text-slate-400 hover:text-white">
                <X className="w-5 h-5" />
              </button>
            </div>
            <div className="space-y-4">
              <div>
                <label className="block text-sm font-medium text-slate-300 mb-1">Upload file</label>
                <input
                  type="file"
                  accept=".env,.env.local,.env.production,text/plain"
                  onChange={handleFileUpload}
                  className="w-full text-sm text-slate-400 file:mr-3 file:py-2 file:px-3 file:rounded file:border-0 file:text-sm file:bg-slate-700 file:text-white hover:file:bg-slate-600"
                />
              </div>
              <div>
                <label className="block text-sm font-medium text-slate-300 mb-1">Or paste content</label>
                <textarea
                  value={importEnvContent}
                  onChange={(e) => setImportEnvContent(e.target.value)}
                  rows={12}
                  placeholder={`KEY1=value1\nKEY2=value2\n# Comments are ignored`}
                  className="w-full px-3 py-2 bg-slate-700 border border-slate-600 rounded text-white font-mono text-sm placeholder-slate-400"
                />
                <p className="text-xs text-slate-500 mt-1">
                  Variables matching secret patterns (KEY, SECRET, PASSWORD, etc.) will be automatically encrypted.
                </p>
              </div>
              <div className="flex justify-end gap-2 pt-2">
                <button
                  onClick={() => {
                    setShowImportEnv(false)
                    setImportEnvContent('')
                  }}
                  className="px-4 py-2 bg-slate-700 hover:bg-slate-600 text-white rounded text-sm"
                >
                  Cancel
                </button>
                <button
                  onClick={handleImportEnv}
                  disabled={!importEnvContent.trim() || importEnvMutation.isPending}
                  className="px-4 py-2 bg-primary-600 hover:bg-primary-700 text-white rounded text-sm disabled:opacity-50"
                >
                  {importEnvMutation.isPending ? 'Importing...' : 'Import'}
                </button>
              </div>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}
