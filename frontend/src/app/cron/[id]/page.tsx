'use client'

import { useState, useCallback, useEffect } from 'react'
import { useParams, useRouter } from 'next/navigation'

export const dynamic = 'force-dynamic'
export const dynamicParams = true

import Link from 'next/link'

import {
  ArrowLeft,
  CheckCircle,
  XCircle,
  Clock,
  Play,
  Pause,
  Trash2,
  Settings,
  FileText,
  ChevronDown,
  ChevronRight,
  Loader2
} from 'lucide-react'
import { CronJobModal, CronJobData } from '@/components/cron/CronJobModal'

interface CronJob {
  id: string
  name: string
  target: string
  target_type: 'host' | 'app' | 'service'
  schedule: string
  schedule_description: string
  command: string
  status: 'active' | 'paused' | 'failed'
  last_run?: {
    status: 'success' | 'failed'
    time: string
    duration?: number
    output?: string
  }
  next_run?: string
  notify_on_failure: boolean
  log_retention: boolean
  created_at: string
}

interface ExecutionLog {
  id: string
  cron_job_id: string
  status: 'success' | 'failed' | 'running'
  started_at: string
  finished_at?: string
  duration?: number
  output?: string
  error?: string
}

// Mock data
const mockCronJob: CronJob = {
  id: 'cron_001',
  name: 'db-cleanup',
  target: 'host',
  target_type: 'host',
  schedule: '0 2 * * *',
  schedule_description: 'Daily at 2:00 AM',
  command: 'npm run cleanup && npm run backup',
  status: 'active',
  last_run: { status: 'success', time: '2024-06-01T02:00:00Z', duration: 45 },
  next_run: '2024-06-02T02:00:00Z',
  notify_on_failure: true,
  log_retention: true,
  created_at: '2024-05-01T00:00:00Z',
}

const mockExecutionLogs: ExecutionLog[] = [
  {
    id: 'exec_001',
    cron_job_id: 'cron_001',
    status: 'success',
    started_at: '2024-06-01T02:00:00Z',
    finished_at: '2024-06-01T02:00:45Z',
    duration: 45,
    output: 'Starting cleanup...\nCleaning expired sessions...\nRemoved 156 sessions\nCleaning temporary files...\nRemoved 23 files\nCleanup complete!\nStarting backup...\nBackup created: backup-2024-06-01.sql.gz\nBackup complete!',
  },
  {
    id: 'exec_002',
    cron_job_id: 'cron_001',
    status: 'success',
    started_at: '2024-05-31T02:00:00Z',
    finished_at: '2024-05-31T02:00:42Z',
    duration: 42,
    output: 'Starting cleanup...\nCleaning expired sessions...\nRemoved 89 sessions\nCleaning temporary files...\nRemoved 15 files\nCleanup complete!\nStarting backup...\nBackup created: backup-2024-05-31.sql.gz\nBackup complete!',
  },
  {
    id: 'exec_003',
    cron_job_id: 'cron_001',
    status: 'failed',
    started_at: '2024-05-30T02:00:00Z',
    finished_at: '2024-05-30T02:00:30Z',
    duration: 30,
    output: 'Starting cleanup...',
    error: 'Error: ENOSPC - No space left on device\nCleanup failed!',
  },
  {
    id: 'exec_004',
    cron_job_id: 'cron_001',
    status: 'success',
    started_at: '2024-05-29T02:00:00Z',
    finished_at: '2024-05-29T02:00:38Z',
    duration: 38,
    output: 'Starting cleanup...\nCleaning expired sessions...\nRemoved 201 sessions\nCleaning temporary files...\nRemoved 45 files\nCleanup complete!\nStarting backup...\nBackup created: backup-2024-05-29.sql.gz\nBackup complete!',
  },
]

function formatDate(dateString: string): string {
  const date = new Date(dateString)
  return date.toLocaleDateString('en-US', {
    month: 'short',
    day: 'numeric',
    year: 'numeric',
    hour: '2-digit',
    minute: '2-digit'
  })
}

function formatDuration(seconds: number): string {
  if (seconds < 60) return `${seconds}s`
  const minutes = Math.floor(seconds / 60)
  const remainingSeconds = seconds % 60
  return `${minutes}m ${remainingSeconds}s`
}

export default function CronJobDetailPage() {
  const params = useParams()
  const router = useRouter()
  const cronJobId = params.id as string

  const [cronJob, setCronJob] = useState<CronJob | null>(null)
  const [executionLogs, setExecutionLogs] = useState<ExecutionLog[]>([])
  const [isLoading, setIsLoading] = useState(true)
  const [showEditModal, setShowEditModal] = useState(false)
  const [expandedLog, setExpandedLog] = useState<string | null>(null)
  const [isRunning, setIsRunning] = useState(false)

  useEffect(() => {
    const loadData = async () => {
      setIsLoading(true)
      await new Promise(resolve => setTimeout(resolve, 500))
      setCronJob(mockCronJob)
      setExecutionLogs(mockExecutionLogs)
      setIsLoading(false)
    }
    loadData()
  }, [cronJobId])

  const handleRunNow = useCallback(async () => {
    setIsRunning(true)
    await new Promise(resolve => setTimeout(resolve, 2000))
    setIsRunning(false)
  }, [])

  const handleToggleStatus = useCallback(() => {
    if (!cronJob) return
    setCronJob({
      ...cronJob,
      status: cronJob.status === 'active' ? 'paused' : 'active'
    })
  }, [cronJob])

  const handleSave = useCallback(async (data: CronJobData) => {
    console.log('Saving cron job:', data)
    await new Promise(resolve => setTimeout(resolve, 500))
  }, [])

  if (isLoading) {
    return (
      <div className="min-h-screen bg-slate-900 flex items-center justify-center">
        <Loader2 className="w-8 h-8 text-slate-400 animate-spin" />
      </div>
    )
  }

  if (!cronJob) {
    return (
      <div className="min-h-screen bg-slate-900 flex flex-col items-center justify-center">
        <p className="text-white text-lg mb-4">Cron job not found</p>
        <Link href="/cron" className="text-primary-400 hover:text-primary-300">
          Back to Cron Jobs
        </Link>
      </div>
    )
  }

  return (
    <div className="min-h-screen bg-slate-900">
      {/* Header */}
      <div className="px-6 py-4 bg-slate-800 border-b border-slate-700">
        <div className="flex items-center gap-4 mb-4">
          <Link
            href="/cron"
            className="text-slate-400 hover:text-white transition-colors"
          >
            <ArrowLeft className="w-5 h-5" />
          </Link>
          <div className="flex-1">
            <div className="flex items-center gap-3">
              <h1 className="text-xl font-semibold text-white">{cronJob.name}</h1>
              <span className={`px-2 py-0.5 text-xs rounded ${
                cronJob.status === 'active' ? 'bg-green-500/20 text-green-400' :
                cronJob.status === 'paused' ? 'bg-amber-500/20 text-amber-400' :
                'bg-red-500/20 text-red-400'
              }`}>
                {cronJob.status}
              </span>
            </div>
            <p className="text-sm text-slate-400 mt-1">
              {cronJob.schedule_description} ({cronJob.schedule})
            </p>
          </div>
          
          <div className="flex items-center gap-3">
            <button
              onClick={handleRunNow}
              disabled={isRunning}
              className="flex items-center gap-2 px-4 py-2 text-sm bg-primary-600 hover:bg-primary-700 text-white rounded-lg transition-colors disabled:opacity-50"
            >
              {isRunning ? (
                <Loader2 className="w-4 h-4 animate-spin" />
              ) : (
                <Play className="w-4 h-4" />
              )}
              Run Now
            </button>
            
            <button
              onClick={handleToggleStatus}
              className={`flex items-center gap-2 px-4 py-2 text-sm rounded-lg transition-colors ${
                cronJob.status === 'active'
                  ? 'bg-amber-500/20 text-amber-400 hover:bg-amber-500/30'
                  : 'bg-green-500/20 text-green-400 hover:bg-green-500/30'
              }`}
            >
              {cronJob.status === 'active' ? (
                <>
                  <Pause className="w-4 h-4" />
                  Pause
                </>
              ) : (
                <>
                  <Play className="w-4 h-4" />
                  Resume
                </>
              )}
            </button>
            
            <button
              onClick={() => setShowEditModal(true)}
              className="flex items-center gap-2 px-4 py-2 text-sm text-slate-300 hover:text-white hover:bg-slate-700 rounded-lg transition-colors"
            >
              <Settings className="w-4 h-4" />
              Settings
            </button>
          </div>
        </div>
      </div>

      {/* Content */}
      <div className="p-6">
        <div className="grid grid-cols-3 gap-6">
          {/* Main Content */}
          <div className="col-span-2 space-y-6">
            {/* Execution History */}
            <div className="bg-slate-800 rounded-lg border border-slate-700">
              <div className="px-6 py-4 border-b border-slate-700">
                <h2 className="font-medium text-white">Execution History</h2>
              </div>
              
              <div className="divide-y divide-slate-700/50">
                {executionLogs.map(log => (
                  <div key={log.id} className="px-6 py-4">
                    <div
                      className="flex items-center justify-between cursor-pointer"
                      onClick={() => setExpandedLog(expandedLog === log.id ? null : log.id)}
                    >
                      <div className="flex items-center gap-3">
                        {log.status === 'success' && (
                          <CheckCircle className="w-4 h-4 text-green-500" />
                        )}
                        {log.status === 'failed' && (
                          <XCircle className="w-4 h-4 text-red-500" />
                        )}
                        {log.status === 'running' && (
                          <Loader2 className="w-4 h-4 text-blue-500 animate-spin" />
                        )}
                        <div>
                          <p className="text-white">{formatDate(log.started_at)}</p>
                          {log.duration && (
                            <p className="text-xs text-slate-400">
                              Duration: {formatDuration(log.duration)}
                            </p>
                          )}
                        </div>
                      </div>
                      
                      <div className="flex items-center gap-2">
                        {log.error && (
                          <span className="px-2 py-0.5 text-xs bg-red-500/20 text-red-400 rounded">
                            {log.error.slice(0, 50)}...
                          </span>
                        )}
                        {expandedLog === log.id ? (
                          <ChevronDown className="w-4 h-4 text-slate-400" />
                        ) : (
                          <ChevronRight className="w-4 h-4 text-slate-400" />
                        )}
                      </div>
                    </div>
                    
                    {expandedLog === log.id && (
                      <div className="mt-4 p-4 bg-slate-900 rounded border border-slate-700">
                        <div className="flex items-center justify-between mb-2">
                          <span className="text-xs text-slate-500">Output</span>
                          <button className="text-xs text-primary-400 hover:text-primary-300">
                            Copy
                          </button>
                        </div>
                        <pre className="text-sm text-slate-300 font-mono whitespace-pre-wrap overflow-x-auto">
                          {log.output || log.error || 'No output'}
                        </pre>
                      </div>
                    )}
                  </div>
                ))}
              </div>
            </div>
          </div>

          {/* Sidebar */}
          <div className="space-y-6">
            {/* Details */}
            <div className="bg-slate-800 rounded-lg border border-slate-700 p-6">
              <h3 className="font-medium text-white mb-4">Details</h3>
              
              <dl className="space-y-3 text-sm">
                <div className="flex justify-between">
                  <dt className="text-slate-400">Target</dt>
                  <dd className="text-white">{cronJob.target} ({cronJob.target_type})</dd>
                </div>
                <div className="flex justify-between">
                  <dt className="text-slate-400">Schedule</dt>
                  <dd className="text-white font-mono text-xs">{cronJob.schedule}</dd>
                </div>
                <div className="flex justify-between">
                  <dt className="text-slate-400">Next Run</dt>
                  <dd className="text-white">
                    {cronJob.next_run ? formatDate(cronJob.next_run) : 'Paused'}
                  </dd>
                </div>
                <div className="flex justify-between">
                  <dt className="text-slate-400">Notify on Failure</dt>
                  <dd className="text-white">{cronJob.notify_on_failure ? 'Yes' : 'No'}</dd>
                </div>
                <div className="flex justify-between">
                  <dt className="text-slate-400">Log Retention</dt>
                  <dd className="text-white">{cronJob.log_retention ? '10 runs' : 'Disabled'}</dd>
                </div>
                <div className="flex justify-between">
                  <dt className="text-slate-400">Created</dt>
                  <dd className="text-white">{formatDate(cronJob.created_at)}</dd>
                </div>
              </dl>
            </div>

            {/* Command */}
            <div className="bg-slate-800 rounded-lg border border-slate-700 p-6">
              <h3 className="font-medium text-white mb-4">Command</h3>
              <div className="p-3 bg-slate-900 rounded border border-slate-700">
                <code className="text-sm text-slate-300 font-mono break-all">
                  {cronJob.command}
                </code>
              </div>
            </div>
          </div>
        </div>
      </div>

      {/* Edit Modal */}
      {showEditModal && (
        <CronJobModal
          isOpen={showEditModal}
          onClose={() => setShowEditModal(false)}
          onSave={handleSave}
          initialData={{
            name: cronJob.name,
            target: cronJob.target,
            target_type: cronJob.target_type,
            schedule: {
              minute: cronJob.schedule.split(' ')[0],
              hour: cronJob.schedule.split(' ')[1],
              day: cronJob.schedule.split(' ')[2],
              month: cronJob.schedule.split(' ')[3],
              weekday: cronJob.schedule.split(' ')[4],
            },
            command: cronJob.command,
            notify_on_failure: cronJob.notify_on_failure,
            log_retention: cronJob.log_retention,
          }}
        />
      )}
    </div>
  )
}
