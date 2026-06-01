'use client'

import { useState, useCallback, useEffect } from 'react'
import Link from 'next/link'
import { Plus, CheckCircle, XCircle, Clock, MoreVertical, Play, Pause, Trash2 } from 'lucide-react'
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
  }
  next_run?: string
  notify_on_failure: boolean
  log_retention: boolean
  created_at: string
}

// Mock data
const mockCronJobs: CronJob[] = [
  {
    id: 'cron_001',
    name: 'db-cleanup',
    target: 'host',
    target_type: 'host',
    schedule: '0 2 * * *',
    schedule_description: 'Daily at 2:00 AM',
    command: 'npm run cleanup',
    status: 'active',
    last_run: { status: 'success', time: '2024-06-01T02:00:00Z', duration: 45 },
    next_run: '2024-06-02T02:00:00Z',
    notify_on_failure: true,
    log_retention: true,
    created_at: '2024-05-01T00:00:00Z',
  },
  {
    id: 'cron_002',
    name: 'sitemap-gen',
    target: 'app:api-prod',
    target_type: 'app',
    schedule: '0 */6 * * *',
    schedule_description: 'Every 6 hours',
    command: 'php artisan sitemap:generate',
    status: 'active',
    last_run: { status: 'success', time: '2024-06-01T12:00:00Z', duration: 12 },
    next_run: '2024-06-01T18:00:00Z',
    notify_on_failure: true,
    log_retention: true,
    created_at: '2024-05-15T00:00:00Z',
  },
  {
    id: 'cron_003',
    name: 'report-gen',
    target: 'app:api-prod',
    target_type: 'app',
    schedule: '0 0 * * 0',
    schedule_description: 'Weekly on Sunday',
    command: 'python report.py',
    status: 'failed',
    last_run: { status: 'failed', time: '2024-05-26T00:00:00Z' },
    next_run: '2024-06-02T00:00:00Z',
    notify_on_failure: true,
    log_retention: true,
    created_at: '2024-05-01T00:00:00Z',
  },
  {
    id: 'cron_004',
    name: 'health-check',
    target: 'host',
    target_type: 'host',
    schedule: '*/5 * * * *',
    schedule_description: 'Every 5 minutes',
    command: 'curl -f https://api.example.com/health',
    status: 'paused',
    last_run: { status: 'success', time: '2024-05-30T10:00:00Z', duration: 3 },
    notify_on_failure: false,
    log_retention: false,
    created_at: '2024-05-01T00:00:00Z',
  },
]

function formatDate(dateString: string): string {
  const date = new Date(dateString)
  const now = new Date()
  const diffMs = now.getTime() - date.getTime()
  const diffHours = Math.floor(diffMs / (1000 * 60 * 60))
  const diffDays = Math.floor(diffHours / 24)

  if (diffHours < 1) return 'Just now'
  if (diffHours < 24) return `${diffHours}h ago`
  if (diffDays < 7) return `${diffDays}d ago`
  
  return date.toLocaleDateString('en-US', {
    month: 'short',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit'
  })
}

function formatNextRun(dateString: string): string {
  const date = new Date(dateString)
  const now = new Date()
  const diffMs = date.getTime() - now.getTime()
  const diffHours = Math.floor(diffMs / (1000 * 60 * 60))
  const diffDays = Math.floor(diffHours / 24)

  if (diffHours < 1) return 'In less than an hour'
  if (diffHours < 24) return `In ${diffHours} hours`
  if (diffDays === 1) return 'Tomorrow'
  return `In ${diffDays} days`
}

export default function CronJobsPage() {
  const [cronJobs, setCronJobs] = useState<CronJob[]>([])
  const [isLoading, setIsLoading] = useState(true)
  const [showModal, setShowModal] = useState(false)
  const [editingJob, setEditingJob] = useState<CronJobData | undefined>()
  const [activeMenu, setActiveMenu] = useState<string | null>(null)

  useEffect(() => {
    const loadCronJobs = async () => {
      setIsLoading(true)
      await new Promise(resolve => setTimeout(resolve, 500))
      setCronJobs(mockCronJobs)
      setIsLoading(false)
    }
    loadCronJobs()
  }, [])

  const handleSave = useCallback(async (data: CronJobData) => {
    console.log('Saving cron job:', data)
    // In production, this would call the API
    await new Promise(resolve => setTimeout(resolve, 500))
  }, [])

  const handleToggleStatus = useCallback((jobId: string) => {
    setCronJobs(prev => prev.map(job => {
      if (job.id === jobId) {
        return {
          ...job,
          status: job.status === 'active' ? 'paused' : 'active'
        }
      }
      return job
    }))
    setActiveMenu(null)
  }, [])

  const handleDelete = useCallback((jobId: string) => {
    setCronJobs(prev => prev.filter(job => job.id !== jobId))
    setActiveMenu(null)
  }, [])

  const handleRunNow = useCallback((jobId: string) => {
    console.log('Running cron job:', jobId)
    setActiveMenu(null)
  }, [])

  const getStatusIcon = (status: CronJob['status']) => {
    switch (status) {
      case 'active':
        return <CheckCircle className="w-4 h-4 text-green-500" />
      case 'paused':
        return <Pause className="w-4 h-4 text-amber-500" />
      case 'failed':
        return <XCircle className="w-4 h-4 text-red-500" />
    }
  }

  return (
    <div className="min-h-screen bg-slate-900">
      {/* Header */}
      <div className="px-6 py-4 bg-slate-800 border-b border-slate-700">
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-xl font-semibold text-white">Cron Jobs</h1>
            <p className="text-sm text-slate-400 mt-1">
              Schedule and manage automated tasks
            </p>
          </div>
          
          <button
            onClick={() => {
              setEditingJob(undefined)
              setShowModal(true)
            }}
            className="flex items-center gap-2 px-4 py-2 bg-primary-600 hover:bg-primary-700 text-white rounded-lg transition-colors"
          >
            <Plus className="w-4 h-4" />
            New Cron Job
          </button>
        </div>
      </div>

      {/* Stats */}
      <div className="px-6 py-4 border-b border-slate-700">
        <div className="grid grid-cols-4 gap-4">
          <div className="p-4 bg-slate-800 rounded-lg border border-slate-700">
            <div className="text-2xl font-semibold text-white">{cronJobs.length}</div>
            <div className="text-sm text-slate-400">Total Jobs</div>
          </div>
          <div className="p-4 bg-slate-800 rounded-lg border border-slate-700">
            <div className="text-2xl font-semibold text-green-500">
              {cronJobs.filter(j => j.status === 'active').length}
            </div>
            <div className="text-sm text-slate-400">Active</div>
          </div>
          <div className="p-4 bg-slate-800 rounded-lg border border-slate-700">
            <div className="text-2xl font-semibold text-amber-500">
              {cronJobs.filter(j => j.status === 'paused').length}
            </div>
            <div className="text-sm text-slate-400">Paused</div>
          </div>
          <div className="p-4 bg-slate-800 rounded-lg border border-slate-700">
            <div className="text-2xl font-semibold text-red-500">
              {cronJobs.filter(j => j.status === 'failed').length}
            </div>
            <div className="text-sm text-slate-400">Failed</div>
          </div>
        </div>
      </div>

      {/* Cron Jobs List */}
      <div className="p-6">
        {isLoading ? (
          <div className="flex items-center justify-center py-12">
            <div className="w-6 h-6 border-2 border-primary-500 border-t-transparent rounded-full animate-spin" />
          </div>
        ) : cronJobs.length === 0 ? (
          <div className="flex flex-col items-center justify-center py-12 text-slate-400">
            <Clock className="w-12 h-12 mb-4" />
            <p className="text-lg mb-2">No cron jobs yet</p>
            <p className="text-sm mb-4">Create your first scheduled task</p>
            <button
              onClick={() => setShowModal(true)}
              className="flex items-center gap-2 px-4 py-2 bg-primary-600 hover:bg-primary-700 text-white rounded-lg transition-colors"
            >
              <Plus className="w-4 h-4" />
              Create Cron Job
            </button>
          </div>
        ) : (
          <div className="space-y-3">
            {cronJobs.map(job => (
              <div
                key={job.id}
                className="bg-slate-800 rounded-lg border border-slate-700 hover:border-slate-600 transition-colors"
              >
                <div className="flex items-center justify-between p-4">
                  <div className="flex items-center gap-4">
                    {getStatusIcon(job.status)}
                    
                    <div>
                      <div className="flex items-center gap-2">
                        <Link
                          href={`/cron/${job.id}`}
                          className="font-medium text-white hover:text-primary-400 transition-colors"
                        >
                          {job.name}
                        </Link>
                        <span className="px-2 py-0.5 text-xs bg-slate-700 text-slate-300 rounded">
                          {job.target_type}
                        </span>
                      </div>
                      <div className="flex items-center gap-3 mt-1 text-sm text-slate-400">
                        <span className="font-mono text-xs">{job.schedule}</span>
                        <span>•</span>
                        <span>{job.schedule_description}</span>
                      </div>
                    </div>
                  </div>

                  <div className="flex items-center gap-6">
                    <div className="text-right">
                      <p className="text-sm text-slate-400">Last Run</p>
                      {job.last_run ? (
                        <div className="flex items-center gap-2">
                          {job.last_run.status === 'success' ? (
                            <CheckCircle className="w-3 h-3 text-green-500" />
                          ) : (
                            <XCircle className="w-3 h-3 text-red-500" />
                          )}
                          <span className="text-sm text-white">
                            {formatDate(job.last_run.time)}
                            {job.last_run.duration && ` (${job.last_run.duration}s)`}
                          </span>
                        </div>
                      ) : (
                        <span className="text-sm text-slate-500">Never</span>
                      )}
                    </div>

                    <div className="text-right">
                      <p className="text-sm text-slate-400">Next Run</p>
                      {job.next_run ? (
                        <span className="text-sm text-white">
                          {formatNextRun(job.next_run)}
                        </span>
                      ) : (
                        <span className="text-sm text-slate-500">Paused</span>
                      )}
                    </div>

                    <div className="relative">
                      <button
                        onClick={() => setActiveMenu(activeMenu === job.id ? null : job.id)}
                        className="p-2 text-slate-400 hover:text-white hover:bg-slate-700 rounded transition-colors"
                      >
                        <MoreVertical className="w-4 h-4" />
                      </button>

                      {activeMenu === job.id && (
                        <div className="absolute right-0 mt-1 w-48 bg-slate-700 border border-slate-600 rounded-lg shadow-xl py-1 z-10">
                          <button
                            onClick={() => handleRunNow(job.id)}
                            className="w-full flex items-center gap-2 px-4 py-2 text-sm text-slate-300 hover:text-white hover:bg-slate-600 transition-colors"
                          >
                            <Play className="w-4 h-4" />
                            Run Now
                          </button>
                          <button
                            onClick={() => {
                              setEditingJob({
                                name: job.name,
                                target: job.target,
                                target_type: job.target_type,
                                schedule: {
                                  minute: job.schedule.split(' ')[0],
                                  hour: job.schedule.split(' ')[1],
                                  day: job.schedule.split(' ')[2],
                                  month: job.schedule.split(' ')[3],
                                  weekday: job.schedule.split(' ')[4],
                                },
                                command: job.command,
                                notify_on_failure: job.notify_on_failure,
                                log_retention: job.log_retention,
                              })
                              setShowModal(true)
                              setActiveMenu(null)
                            }}
                            className="w-full flex items-center gap-2 px-4 py-2 text-sm text-slate-300 hover:text-white hover:bg-slate-600 transition-colors"
                          >
                            Edit
                          </button>
                          <button
                            onClick={() => handleToggleStatus(job.id)}
                            className="w-full flex items-center gap-2 px-4 py-2 text-sm text-slate-300 hover:text-white hover:bg-slate-600 transition-colors"
                          >
                            {job.status === 'active' ? (
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
                            onClick={() => handleDelete(job.id)}
                            className="w-full flex items-center gap-2 px-4 py-2 text-sm text-red-400 hover:text-red-300 hover:bg-red-500/10 transition-colors"
                          >
                            <Trash2 className="w-4 h-4" />
                            Delete
                          </button>
                        </div>
                      )}
                    </div>
                  </div>
                </div>
              </div>
            ))}
          </div>
        )}
      </div>

      {/* Cron Job Modal */}
      <CronJobModal
        isOpen={showModal}
        onClose={() => {
          setShowModal(false)
          setEditingJob(undefined)
        }}
        onSave={handleSave}
        initialData={editingJob}
      />
    </div>
  )
}
