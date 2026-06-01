'use client'

import { useState, useEffect, useCallback } from 'react'
import { useQuery } from '@tanstack/react-query'
import Link from 'next/link'
import {
  Cpu,
  MemoryStick,
  HardDrive,
  Activity,
  ArrowUpRight,
  ArrowDownRight,
  Server,
  AppWindow,
  Database,
  Clock,
  GitBranch,
  CheckCircle,
  XCircle,
  RefreshCw,
  Plus,
  Terminal,
  AlertTriangle,
  Loader2,
  AlertCircle
} from 'lucide-react'
import { AreaChart, Area, XAxis, YAxis, Tooltip, ResponsiveContainer } from 'recharts'
import { api, ApiError } from '@/lib/api'
import { useToastStore } from '@/stores'
import { useWebSocket, useServerMetricsSubscription } from '@/lib/websocket'
import type { ServerMetrics, App, Service, ActivityEvent } from '@/types'

interface MetricsPoint {
  timestamp: number
  cpu: number
  ram: number
  disk: number
}

function generateSparklineData(base: number, variance: number, points: number = 20) {
  const data = []
  let value = base
  for (let i = 0; i < points; i++) {
    value = Math.max(0, Math.min(100, value + (Math.random() - 0.5) * variance))
    data.push({ value: Math.round(value * 10) / 10 })
  }
  return data
}

function formatBytes(bytes: number): string {
  if (bytes === 0) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + ' ' + sizes[i]
}

function formatUptime(seconds: number): string {
  const days = Math.floor(seconds / 86400)
  const hours = Math.floor((seconds % 86400) / 3600)
  const minutes = Math.floor((seconds % 3600) / 60)
  if (days > 0) return `${days}d ${hours}h`
  if (hours > 0) return `${hours}h ${minutes}m`
  return `${minutes}m`
}

// Skeleton loader component
function Skeleton({ className }: { className: string }) {
  return (
    <div className={`animate-pulse bg-slate-700 rounded ${className}`} />
  )
}

// Error state component
function ErrorState({ message, onRetry }: { message: string; onRetry?: () => void }) {
  return (
    <div className="flex flex-col items-center justify-center h-32 text-slate-400">
      <AlertCircle className="w-6 h-6 text-danger-400 mb-2" />
      <p className="text-sm">{message}</p>
      {onRetry && (
        <button
          onClick={onRetry}
          className="mt-2 text-xs text-primary-400 hover:text-primary-300"
        >
          Retry
        </button>
      )}
    </div>
  )
}

// Loading skeleton for resource cards
function ResourceCardSkeleton() {
  return (
    <div className="bg-slate-800 rounded-lg p-4 border border-slate-700">
      <div className="flex items-start justify-between mb-3">
        <div className="flex items-center gap-2">
          <Skeleton className="w-8 h-8 rounded-lg" />
          <Skeleton className="w-20 h-4" />
        </div>
      </div>
      <Skeleton className="w-16 h-8 mb-1" />
      <Skeleton className="w-24 h-3" />
      <Skeleton className="h-12 mt-3 rounded" />
    </div>
  )
}

// Loading skeleton for list items
function ListItemSkeleton() {
  return (
    <div className="flex items-center justify-between p-4">
      <div className="flex items-center gap-3">
        <Skeleton className="w-8 h-8 rounded" />
        <div>
          <Skeleton className="w-24 h-4 mb-1" />
          <Skeleton className="w-16 h-3" />
        </div>
      </div>
      <Skeleton className="w-16 h-5 rounded-full" />
    </div>
  )
}

// Resource Card Component
function ResourceCard({
  title,
  value,
  subValue,
  icon: Icon,
  trend,
  sparklineData,
  color = 'primary'
}: {
  title: string
  value: string
  subValue?: string
  icon: React.ElementType
  trend?: 'up' | 'down'
  sparklineData?: { value: number }[]
  color?: 'primary' | 'success' | 'warning' | 'danger'
}) {
  const colors = {
    primary: 'text-primary-400',
    success: 'text-success-400',
    warning: 'text-warning-400',
    danger: 'text-danger-400',
  }
  const bgColors = {
    primary: 'bg-primary-500/10',
    success: 'bg-success-500/10',
    warning: 'bg-warning-500/10',
    danger: 'bg-danger-500/10',
  }

  return (
    <div className="bg-slate-800 rounded-lg p-4 border border-slate-700">
      <div className="flex items-start justify-between mb-3">
        <div className="flex items-center gap-2">
          <div className={`p-2 rounded-lg ${bgColors[color]}`}>
            <Icon className={`w-5 h-5 ${colors[color]}`} />
          </div>
          <span className="text-sm text-slate-400">{title}</span>
        </div>
        {trend && (
          <div className={`flex items-center gap-1 text-xs ${trend === 'up' ? 'text-danger-400' : 'text-success-400'}`}>
            {trend === 'up' ? <ArrowUpRight className="w-3 h-3" /> : <ArrowDownRight className="w-3 h-3" />}
            {trend === 'up' ? 'High' : 'Low'}
          </div>
        )}
      </div>
      <div className="text-2xl font-semibold text-white mb-1">{value}</div>
      {subValue && <div className="text-xs text-slate-500">{subValue}</div>}
      {sparklineData && (
        <div className="h-12 mt-3">
          <ResponsiveContainer width="100%" height="100%">
            <AreaChart data={sparklineData}>
              <defs>
                <linearGradient id={`gradient-${title}`} x1="0" y1="0" x2="0" y2="1">
                  <stop offset="0%" stopColor={color === 'primary' ? '#2563EB' : color === 'success' ? '#16A34A' : color === 'warning' ? '#D97706' : '#DC2626'} stopOpacity={0.3} />
                  <stop offset="100%" stopColor={color === 'primary' ? '#2563EB' : color === 'success' ? '#16A34A' : color === 'warning' ? '#D97706' : '#DC2626'} stopOpacity={0} />
                </linearGradient>
              </defs>
              <Area
                type="monotone"
                dataKey="value"
                stroke={color === 'primary' ? '#2563EB' : color === 'success' ? '#16A34A' : color === 'warning' ? '#D97706' : '#DC2626'}
                fill={`url(#gradient-${title})`}
                strokeWidth={1.5}
              />
            </AreaChart>
          </ResponsiveContainer>
        </div>
      )}
    </div>
  )
}

// App Status Badge
function AppStatusBadge({ status }: { status: App['status'] }) {
  const config = {
    running: { color: 'bg-success-500', text: 'text-success-400', label: 'Running' },
    stopped: { color: 'bg-slate-500', text: 'text-slate-400', label: 'Stopped' },
    deploying: { color: 'bg-warning-500', text: 'text-warning-400', label: 'Deploying' },
    failed: { color: 'bg-danger-500', text: 'text-danger-400', label: 'Failed' },
  }[status] || { color: 'bg-slate-500', text: 'text-slate-400', label: status }

  return (
    <span className={`inline-flex items-center gap-1.5 px-2 py-0.5 rounded text-xs font-medium ${config.text}`}>
      <span className={`w-1.5 h-1.5 rounded-full ${config.color}`} />
      {config.label}
    </span>
  )
}

// Quick Action Button
function QuickAction({
  icon: Icon,
  label,
  href,
}: {
  icon: React.ElementType
  label: string
  href: string
}) {
  return (
    <Link
      href={href}
      className="flex flex-col items-center gap-2 p-4 bg-slate-800 rounded-lg border border-slate-700 hover:bg-slate-700 transition-colors"
    >
      <Icon className="w-6 h-6 text-slate-400" />
      <span className="text-xs text-slate-400">{label}</span>
    </Link>
  )
}

export default function DashboardPage() {
  const { addToast } = useToastStore()
  const [metricsHistory, setMetricsHistory] = useState<MetricsPoint[]>([])
  const [lastUpdated, setLastUpdated] = useState<Date | null>(null)
  const [isConnected, setIsConnected] = useState(false)

  const { ws } = useWebSocket()

  const handleMetricsUpdate = useCallback((data: ServerMetrics) => {
    setMetricsHistory(prev => {
      const newPoint: MetricsPoint = {
        timestamp: Date.now(),
        cpu: data.cpu.current_percent,
        ram: data.memory.percent,
        disk: data.disk.percent,
      }
      const updated = [...prev, newPoint].slice(-20)
      return updated
    })
    setLastUpdated(new Date())
  }, [])

  useServerMetricsSubscription(handleMetricsUpdate, true)

  useEffect(() => {
    if (ws) {
      const unsubConnect = ws.onConnect(() => setIsConnected(true))
      const unsubDisconnect = ws.onDisconnect(() => setIsConnected(false))
      return () => {
        unsubConnect()
        unsubDisconnect()
      }
    }
  }, [ws])

  const { data: metrics, isLoading: metricsLoading, error: metricsError, refetch: refetchMetrics } = useQuery({
    queryKey: ['server-metrics'],
    queryFn: api.server.metrics,
    refetchInterval: 30000,
  })

  const { data: appsData, isLoading: appsLoading, error: appsError } = useQuery({
    queryKey: ['apps'],
    queryFn: () => api.apps.list({ limit: 10 }),
  })

  const { data: servicesData, isLoading: servicesLoading, error: servicesError } = useQuery({
    queryKey: ['services'],
    queryFn: () => api.services.list({ limit: 10 }),
  })

  const { data: activityData, isLoading: activityLoading, error: activityError } = useQuery({
    queryKey: ['activity'],
    queryFn: () => api.activity.list({ limit: 5 }),
  })

  useEffect(() => {
    if (metrics && !isConnected) {
      setMetricsHistory(prev => {
        const newPoint: MetricsPoint = {
          timestamp: Date.now(),
          cpu: metrics.cpu.current_percent,
          ram: metrics.memory.percent,
          disk: metrics.disk.percent,
        }
        const updated = [...prev, newPoint].slice(-20)
        return updated
      })
      setLastUpdated(new Date())
    }
  }, [metrics, isConnected])

  const apps = appsData?.data || []
  const services = servicesData?.data || []
  const activity = activityData?.events || []

  const metricsDisplay = metrics || {
    cpu: { current_percent: 0, per_core: [], history: [] },
    memory: { current_mb: 0, total_mb: 0, percent: 0, history: [] },
    disk: { percent: 0, total_gb: 0, used_gb: 0, io_read_mbps: 0, io_write_mbps: 0 },
    load: { '1min': 0, '5min': 0, '15min': 0 },
    network: { inbound_mbps: 0, outbound_mbps: 0, connections_active: 0 },
  }

  const cpuData = metricsHistory.length > 0
    ? metricsHistory.map(p => ({ value: p.cpu }))
    : generateSparklineData(metricsDisplay.cpu.current_percent, 20)
  const ramData = metricsHistory.length > 0
    ? metricsHistory.map(p => ({ value: p.ram }))
    : generateSparklineData(metricsDisplay.memory.percent, 15)
  const diskData = metricsHistory.length > 0
    ? metricsHistory.map(p => ({ value: p.disk }))
    : generateSparklineData(metricsDisplay.disk.percent, 10)

  const runningApps = apps.filter(a => a.status === 'running').slice(0, 6)
  const activeServices = services.filter(s => s.status === 'running').slice(0, 4)

  function timeAgo(dateString: string): string {
    const date = new Date(dateString)
    const now = new Date()
    const seconds = Math.floor((now.getTime() - date.getTime()) / 1000)

    if (seconds < 60) return 'just now'
    if (seconds < 3600) return `${Math.floor(seconds / 60)}m ago`
    if (seconds < 86400) return `${Math.floor(seconds / 3600)}h ago`
    return `${Math.floor(seconds / 86400)}d ago`
  }

  const formatLastUpdated = () => {
    if (!lastUpdated) return 'never'
    const seconds = Math.floor((Date.now() - lastUpdated.getTime()) / 1000)
    if (seconds < 60) return 'just now'
    return `${Math.floor(seconds / 60)}m ago`
  }

  return (
    <div className="p-6 space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-white">Dashboard</h1>
          <p className="text-sm text-slate-400 mt-1">
            Server overview and recent activity
          </p>
        </div>
        <div className="flex items-center gap-4">
          <div className="flex items-center gap-2 text-sm">
            <span className={`w-2 h-2 rounded-full ${isConnected ? 'bg-success-500' : 'bg-slate-500'}`} />
            <span className="text-slate-400">{isConnected ? 'Live' : 'Polling'}</span>
          </div>
          <div className="flex items-center gap-2 text-sm text-slate-400">
            <Clock className="w-4 h-4" />
            <span>Updated: {formatLastUpdated()}</span>
          </div>
        </div>
      </div>

      {/* Resource Cards */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
        {metricsLoading ? (
          <>
            <ResourceCardSkeleton />
            <ResourceCardSkeleton />
            <ResourceCardSkeleton />
            <ResourceCardSkeleton />
          </>
        ) : metricsError ? (
          <div className="col-span-4">
            <ErrorState
              message="Failed to load server metrics"
              onRetry={() => refetchMetrics()}
            />
          </div>
        ) : (
          <>
            <ResourceCard
              title="CPU Usage"
              value={`${metricsDisplay.cpu.current_percent.toFixed(1)}%`}
              subValue={`${metricsDisplay.cpu.per_core.length} cores`}
              icon={Cpu}
              trend={metricsDisplay.cpu.current_percent > 80 ? 'up' : undefined}
              sparklineData={cpuData}
              color={metricsDisplay.cpu.current_percent > 80 ? 'danger' : 'primary'}
            />
            <ResourceCard
              title="RAM Usage"
              value={formatBytes(metricsDisplay.memory.current_mb * 1024 * 1024)}
              subValue={`of ${formatBytes(metricsDisplay.memory.total_mb * 1024 * 1024)}`}
              icon={MemoryStick}
              sparklineData={ramData}
              color="primary"
            />
            <ResourceCard
              title="Disk Usage"
              value={`${metricsDisplay.disk.used_gb} GB`}
              subValue={`of ${metricsDisplay.disk.total_gb} GB`}
              icon={HardDrive}
              sparklineData={diskData}
              color="warning"
            />
            <ResourceCard
              title="Load Average"
              value={`${metricsDisplay.load['1min'].toFixed(2)}`}
              subValue={`${metricsDisplay.load['5min'].toFixed(2)} / ${metricsDisplay.load['15min'].toFixed(2)}`}
              icon={Activity}
              color={metricsDisplay.load['1min'] > 4 ? 'danger' : 'success'}
            />
          </>
        )}
      </div>

      {/* Quick Actions */}
      <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
        <QuickAction icon={Terminal} label="Terminal" href="/terminal" />
        <QuickAction icon={Plus} label="New App" href="/apps/new" />
        <QuickAction icon={GitBranch} label="Deployments" href="/deployments" />
        <QuickAction icon={AlertTriangle} label="View Logs" href="/logs" />
      </div>

      {/* Main Content Grid */}
      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        {/* Running Apps */}
        <div className="bg-slate-800 rounded-lg border border-slate-700">
          <div className="flex items-center justify-between p-4 border-b border-slate-700">
            <h2 className="text-lg font-semibold text-white flex items-center gap-2">
              <AppWindow className="w-5 h-5 text-slate-400" />
              Running Apps
            </h2>
            <Link href="/apps" className="text-sm text-primary-400 hover:text-primary-300">
              View all
            </Link>
          </div>
          <div className="divide-y divide-slate-700">
            {appsLoading ? (
              <>
                <ListItemSkeleton />
                <ListItemSkeleton />
                <ListItemSkeleton />
              </>
            ) : appsError ? (
              <div className="p-4">
                <ErrorState message="Failed to load apps" />
              </div>
            ) : runningApps.length === 0 ? (
              <div className="p-4 text-center text-slate-400 text-sm">
                No running apps
              </div>
            ) : (
              runningApps.map((app) => (
                <div key={app.id} className="flex items-center justify-between p-4 hover:bg-slate-700/50 transition-colors">
                  <div className="flex items-center gap-3">
                    <div className="w-8 h-8 rounded bg-slate-700 flex items-center justify-center">
                      <Server className="w-4 h-4 text-slate-400" />
                    </div>
                    <div>
                      <p className="text-sm font-medium text-white">{app.name}</p>
                      <p className="text-xs text-slate-400">{app.runtime}</p>
                    </div>
                  </div>
                  <AppStatusBadge status={app.status} />
                </div>
              ))
            )}
          </div>
        </div>

        {/* Active Services */}
        <div className="bg-slate-800 rounded-lg border border-slate-700">
          <div className="flex items-center justify-between p-4 border-b border-slate-700">
            <h2 className="text-lg font-semibold text-white flex items-center gap-2">
              <Database className="w-5 h-5 text-slate-400" />
              Active Services
            </h2>
            <Link href="/services" className="text-sm text-primary-400 hover:text-primary-300">
              View all
            </Link>
          </div>
          <div className="divide-y divide-slate-700">
            {servicesLoading ? (
              <>
                <ListItemSkeleton />
                <ListItemSkeleton />
                <ListItemSkeleton />
              </>
            ) : servicesError ? (
              <div className="p-4">
                <ErrorState message="Failed to load services" />
              </div>
            ) : activeServices.length === 0 ? (
              <div className="p-4 text-center text-slate-400 text-sm">
                No active services
              </div>
            ) : (
              activeServices.map((service) => (
                <div key={service.id} className="flex items-center justify-between p-4 hover:bg-slate-700/50 transition-colors">
                  <div className="flex items-center gap-3">
                    <div className="w-8 h-8 rounded bg-slate-700 flex items-center justify-center">
                      <Database className="w-4 h-4 text-slate-400" />
                    </div>
                    <div>
                      <p className="text-sm font-medium text-white">{service.name}</p>
                      <p className="text-xs text-slate-400">{service.type}:{service.port}</p>
                    </div>
                  </div>
                  <span className="flex items-center gap-1.5 text-xs text-success-400">
                    <span className="w-1.5 h-1.5 rounded-full bg-success-500" />
                    Running
                  </span>
                </div>
              ))
            )}
          </div>
        </div>

        {/* Recent Activity */}
        <div className="bg-slate-800 rounded-lg border border-slate-700">
          <div className="flex items-center justify-between p-4 border-b border-slate-700">
            <h2 className="text-lg font-semibold text-white flex items-center gap-2">
              <Activity className="w-5 h-5 text-slate-400" />
              Recent Activity
            </h2>
            <Link href="/activity" className="text-sm text-primary-400 hover:text-primary-300">
              View all
            </Link>
          </div>
          <div className="divide-y divide-slate-700">
            {activityLoading ? (
              <>
                <div className="p-4 space-y-3">
                  <Skeleton className="h-4 w-full" />
                  <Skeleton className="h-4 w-3/4" />
                  <Skeleton className="h-4 w-1/2" />
                </div>
              </>
            ) : activityError ? (
              <div className="p-4">
                <ErrorState message="Failed to load activity" />
              </div>
            ) : activity.length === 0 ? (
              <div className="p-4 text-center text-slate-400 text-sm">
                No recent activity
              </div>
            ) : (
              activity.slice(0, 5).map((event) => (
                <div key={event.id} className="p-4">
                  <div className="flex items-start gap-3">
                    <div className={`mt-0.5 p-1 rounded ${
                      event.type === 'deployment' && event.message.includes('failed')
                        ? 'bg-danger-500/10 text-danger-400'
                        : event.type === 'deployment'
                        ? 'bg-success-500/10 text-success-400'
                        : 'bg-slate-700 text-slate-400'
                    }`}>
                      {event.type === 'deployment' ? (
                        event.message.includes('failed') ? (
                          <XCircle className="w-4 h-4" />
                        ) : (
                          <CheckCircle className="w-4 h-4" />
                        )
                      ) : (
                        <RefreshCw className="w-4 h-4" />
                      )}
                    </div>
                    <div className="flex-1 min-w-0">
                      <p className="text-sm text-white">{event.message}</p>
                      <div className="flex items-center gap-2 mt-1 text-xs text-slate-400">
                        {event.user && <span>{event.user}</span>}
                        <span>•</span>
                        <span>{timeAgo(event.created_at)}</span>
                      </div>
                    </div>
                  </div>
                </div>
              ))
            )}
          </div>
        </div>
      </div>

      {/* Server Info Footer */}
      <div className="bg-slate-800 rounded-lg border border-slate-700 p-4">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-4">
            <div className="flex items-center gap-2">
              <Server className="w-5 h-5 text-slate-400" />
              <span className="text-sm text-slate-400">Uptime:</span>
              <span className="text-sm text-white font-medium">{formatUptime(metricsDisplay.uptime)}</span>
            </div>
            <div className="h-4 w-px bg-slate-700" />
            <div className="flex items-center gap-2">
              <span className="text-sm text-slate-400">Network:</span>
              <span className="text-sm text-white font-medium">
                ↓ {formatBytes((metricsDisplay.network_in || 0) * 1024)}/s
              </span>
              <span className="text-sm text-white font-medium">
                ↑ {formatBytes((metricsDisplay.network_out || 0) * 1024)}/s
              </span>
            </div>
          </div>
          <button
            className="flex items-center gap-2 px-3 py-1.5 text-sm text-slate-400 hover:text-white hover:bg-slate-700 rounded transition-colors"
            onClick={() => addToast({ type: 'info', title: 'Refreshing metrics...', duration: 2000 })}
          >
            <RefreshCw className="w-4 h-4" />
            Refresh
          </button>
        </div>
      </div>
    </div>
  )
}