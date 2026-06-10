'use client'

import { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import {
  RefreshCw,
  GitBranch,
  FileText,
  Loader2,
  ChevronDown,
  ChevronUp,
  RotateCcw,
  Play
} from 'lucide-react'
import { api } from '@/lib/api'

interface Deployment {
  id: string
  app_id: string
  status: 'pending' | 'building' | 'deploying' | 'success' | 'failed' | 'cancelled'
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

interface DeploymentListProps {
  appId: string
  onDeploymentSelect?: (deploymentId: string) => void
  onRollback?: (deploymentId: string) => void
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

function formatDuration(seconds?: number): string {
  if (!seconds) return '—'
  if (seconds < 60) return `${seconds}s`
  const mins = Math.floor(seconds / 60)
  const secs = seconds % 60
  return `${mins}m ${secs}s`
}

const statusColors: Record<string, string> = {
  success: 'text-green-400 bg-green-500/10',
  failed: 'text-red-400 bg-red-500/10',
  building: 'text-amber-400 bg-amber-500/10',
  deploying: 'text-amber-400 bg-amber-500/10',
  pending: 'text-slate-400 bg-slate-500/10',
  cancelled: 'text-slate-400 bg-slate-500/10',
}

const statusDots: Record<string, string> = {
  success: 'bg-green-400',
  failed: 'bg-red-400',
  building: 'bg-amber-400 animate-pulse',
  deploying: 'bg-amber-400 animate-pulse',
  pending: 'bg-slate-400',
  cancelled: 'bg-slate-400',
}

export function DeploymentList({ appId, onDeploymentSelect, onRollback }: DeploymentListProps) {
  const [expandedDeployment, setExpandedDeployment] = useState<string | null>(null)
  const [deploymentLogs, setDeploymentLogs] = useState<Record<string, string>>({})

  const { data: deployments, isLoading, refetch } = useQuery<Deployment[]>({
    queryKey: ['app-deployments', appId],
    queryFn: () => api.apps.getDeployments(appId) as unknown as Promise<Deployment[]>,
  })

  const fetchDeploymentLogs = async (deploymentId: string) => {
    if (deploymentLogs[deploymentId]) return
    try {
      const response = await api.apps.getDeploymentLogs(appId, deploymentId)
      const data = response as unknown as { deployment_id: string; lines: Array<{ timestamp: string; level: string; message: string }> }
      const logText = data.lines?.map(line => `[${line.level}] ${line.message}`).join('\n') || 'No logs available'
      setDeploymentLogs(prev => ({ ...prev, [deploymentId]: logText }))
    } catch {
      setDeploymentLogs(prev => ({ ...prev, [deploymentId]: 'Failed to fetch logs' }))
    }
  }

  const handleToggleExpand = (deploymentId: string) => {
    if (expandedDeployment === deploymentId) {
      setExpandedDeployment(null)
    } else {
      setExpandedDeployment(deploymentId)
      fetchDeploymentLogs(deploymentId)
    }
  }

  if (isLoading) {
    return (
      <div className="flex items-center justify-center h-32">
        <Loader2 className="w-6 h-6 text-primary-500 animate-spin" />
      </div>
    )
  }

  if (!deployments || deployments.length === 0) {
    return (
      <div className="text-center py-8 text-slate-400">
        No deployments yet
      </div>
    )
  }

  return (
    <div className="space-y-2">
      {deployments.map((deployment) => (
        <div key={deployment.id} className="bg-slate-800 border border-slate-700 rounded-lg overflow-hidden">
          {/* Row */}
          <div
            className="flex items-center px-4 py-3 hover:bg-slate-700/50 cursor-pointer"
            onClick={() => handleToggleExpand(deployment.id)}
          >
            {/* Status */}
            <div className="w-24 flex items-center gap-2">
              <span className={`w-2 h-2 rounded-full ${statusDots[deployment.status]}`} />
              <span className={`text-xs font-medium ${statusColors[deployment.status].split(' ')[0]}`}>
                {deployment.status}
              </span>
            </div>

            {/* Commit */}
            <div className="w-28 flex items-center gap-1 text-white font-mono text-sm">
              <GitBranch className="w-3 h-3 text-slate-400" />
              {deployment.commit?.slice(0, 7) || '—'}
            </div>

            {/* Branch */}
            <div className="w-24 text-slate-300 text-sm">
              {deployment.branch || 'main'}
            </div>

            {/* Author */}
            <div className="w-24 text-slate-400 text-sm truncate">
              {deployment.commit_author || deployment.triggered_by_user || '—'}
            </div>

            {/* Duration */}
            <div className="w-20 text-slate-400 text-sm">
              {formatDuration(deployment.build_duration_seconds)}
            </div>

            {/* Time */}
            <div className="flex-1 text-slate-400 text-sm">
              {formatRelativeTime(deployment.started_at)}
            </div>

            {/* Actions */}
            <div className="flex items-center gap-1" onClick={(e) => e.stopPropagation()}>
              {deployment.status === 'success' && onRollback && (
                <button
                  onClick={() => onRollback(deployment.id)}
                  className="p-1.5 text-slate-400 hover:text-white hover:bg-slate-600 rounded transition-colors"
                  title="Rollback to this deployment"
                >
                  <RotateCcw className="w-4 h-4" />
                </button>
              )}
              <button className="p-1.5 text-slate-400 hover:text-white hover:bg-slate-600 rounded transition-colors">
                <FileText className="w-4 h-4" />
              </button>
              {expandedDeployment === deployment.id ? (
                <ChevronUp className="w-4 h-4 text-slate-400" />
              ) : (
                <ChevronDown className="w-4 h-4 text-slate-400" />
              )}
            </div>
          </div>

          {/* Expanded Logs */}
          {expandedDeployment === deployment.id && (
            <div className="border-t border-slate-700 bg-slate-900">
              <div className="px-4 py-2 border-b border-slate-700 flex items-center justify-between">
                <span className="text-xs text-slate-400">Build Logs</span>
                <button
                  onClick={() => {
                    const logs = deploymentLogs[deployment.id]
                    if (logs) {
                      navigator.clipboard.writeText(logs)
                    }
                  }}
                  className="text-xs text-primary-400 hover:text-primary-300"
                >
                  Copy logs
                </button>
              </div>
              <div className="p-4 max-h-64 overflow-auto font-mono text-xs text-slate-300 whitespace-pre-wrap">
                {deploymentLogs[deployment.id] || (
                  <span className="flex items-center gap-2 text-slate-500">
                    <Loader2 className="w-3 h-3 animate-spin" />
                    Loading logs...
                  </span>
                )}
              </div>
            </div>
          )}
        </div>
      ))}
    </div>
  )
}

export default DeploymentList
