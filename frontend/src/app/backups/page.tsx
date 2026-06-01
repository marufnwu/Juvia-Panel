'use client'

import { useState, useCallback, useEffect } from 'react'
import { BackupList, Backup } from '@/components/backups/BackupList'
import {
  Settings,
  Download,
  Plus,
  Filter,
  RefreshCw,
  Database,
  Box
} from 'lucide-react'

// Mock data - in production these would come from API
const mockBackups: Backup[] = [
  {
    id: 'bkp_001',
    name: 'api-prod-2024-06-01-020000',
    type: 'app',
    target_id: 'app_api-prod',
    target_name: 'api-prod',
    status: 'completed',
    size: 45 * 1024 * 1024,
    location: 'S3',
    backup_type: 'scheduled',
    created_at: '2024-06-01T02:00:00Z',
  },
  {
    id: 'bkp_002',
    name: 'main-pg-2024-06-01-020000',
    type: 'service',
    target_id: 'svc_main-pg',
    target_name: 'main-pg (PostgreSQL)',
    status: 'completed',
    size: 234 * 1024 * 1024,
    location: 'S3',
    backup_type: 'scheduled',
    created_at: '2024-06-01T02:00:00Z',
  },
  {
    id: 'bkp_003',
    name: 'web-client-2024-05-31-140000',
    type: 'app',
    target_id: 'app_web-client',
    target_name: 'web-client',
    status: 'completed',
    size: 12 * 1024 * 1024,
    location: 'Local',
    backup_type: 'manual',
    created_at: '2024-05-31T14:00:00Z',
  },
  {
    id: 'bkp_004',
    name: 'redis-cache-2024-05-31-020000',
    type: 'service',
    target_id: 'svc_redis-cache',
    target_name: 'redis-cache (Redis)',
    status: 'failed',
    size: undefined,
    location: 'S3',
    backup_type: 'scheduled',
    created_at: '2024-05-31T02:00:00Z',
  },
  {
    id: 'bkp_005',
    name: 'api-prod-manual-2024-05-30',
    type: 'app',
    target_id: 'app_api-prod',
    target_name: 'api-prod',
    status: 'completed',
    size: 44 * 1024 * 1024,
    location: 'Local',
    backup_type: 'manual',
    created_at: '2024-05-30T10:30:00Z',
  },
]

type FilterType = 'all' | 'app' | 'service'
type FilterStatus = 'all' | 'completed' | 'failed' | 'in_progress'
type FilterLocation = 'all' | 'local' | 's3'

export default function BackupsPage() {
  const [backups, setBackups] = useState<Backup[]>([])
  const [isLoading, setIsLoading] = useState(true)
  const [typeFilter, setTypeFilter] = useState<FilterType>('all')
  const [statusFilter, setStatusFilter] = useState<FilterStatus>('all')
  const [locationFilter, setLocationFilter] = useState<FilterLocation>('all')
  const [showSettings, setShowSettings] = useState(false)

  // Default backup settings
  const [defaultSchedule, setDefaultSchedule] = useState('daily')
  const [defaultRetention, setDefaultRetention] = useState('7')
  const [defaultDestination, setDefaultDestination] = useState('s3')

  useEffect(() => {
    // Simulate loading backups
    const loadBackups = async () => {
      setIsLoading(true)
      await new Promise(resolve => setTimeout(resolve, 500))
      setBackups(mockBackups)
      setIsLoading(false)
    }
    loadBackups()
  }, [])

  const filteredBackups = backups.filter(backup => {
    if (typeFilter !== 'all' && backup.type !== typeFilter) return false
    if (statusFilter !== 'all' && backup.status !== statusFilter) return false
    if (locationFilter !== 'all' && 
        ((locationFilter === 'local' && backup.location !== 'Local') ||
         (locationFilter === 's3' && backup.location !== 'S3'))) return false
    return true
  })

  const handleRestore = useCallback((backupId: string) => {
    console.log('Restoring backup:', backupId)
    // In production, this would call the API
  }, [])

  const handleDelete = useCallback((backupId: string) => {
    console.log('Deleting backup:', backupId)
    setBackups(prev => prev.filter(b => b.id !== backupId))
    // In production, this would call the API
  }, [])

  const handleDownload = useCallback((backupId: string) => {
    console.log('Downloading backup:', backupId)
    // In production, this would trigger a download
  }, [])

  const totalBackups = backups.length
  const completedBackups = backups.filter(b => b.status === 'completed').length
  const failedBackups = backups.filter(b => b.status === 'failed').length

  return (
    <div className="min-h-screen bg-slate-900">
      {/* Header */}
      <div className="px-6 py-4 bg-slate-800 border-b border-slate-700">
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-xl font-semibold text-white">Backups</h1>
            <p className="text-sm text-slate-400 mt-1">
              Manage backups for all apps and services
            </p>
          </div>
          
          <div className="flex items-center gap-3">
            <button
              onClick={() => setShowSettings(true)}
              className="flex items-center gap-2 px-4 py-2 text-sm text-slate-300 hover:text-white hover:bg-slate-700 rounded-lg transition-colors"
            >
              <Settings className="w-4 h-4" />
              Backup Settings
            </button>
            
            <button className="flex items-center gap-2 px-4 py-2 text-sm bg-primary-600 hover:bg-primary-700 text-white rounded-lg transition-colors">
              <Plus className="w-4 h-4" />
              Create Backup
            </button>
          </div>
        </div>
      </div>

      {/* Stats */}
      <div className="px-6 py-4 border-b border-slate-700">
        <div className="grid grid-cols-3 gap-4">
          <div className="p-4 bg-slate-800 rounded-lg border border-slate-700">
            <div className="text-2xl font-semibold text-white">{totalBackups}</div>
            <div className="text-sm text-slate-400">Total Backups</div>
          </div>
          <div className="p-4 bg-slate-800 rounded-lg border border-slate-700">
            <div className="text-2xl font-semibold text-green-500">{completedBackups}</div>
            <div className="text-sm text-slate-400">Completed</div>
          </div>
          <div className="p-4 bg-slate-800 rounded-lg border border-slate-700">
            <div className="text-2xl font-semibold text-red-500">{failedBackups}</div>
            <div className="text-sm text-slate-400">Failed</div>
          </div>
        </div>
      </div>

      {/* Filters */}
      <div className="px-6 py-4 border-b border-slate-700">
        <div className="flex items-center gap-4">
          <div className="flex items-center gap-2">
            <Filter className="w-4 h-4 text-slate-400" />
            <span className="text-sm text-slate-400">Filters:</span>
          </div>
          
          <select
            value={typeFilter}
            onChange={(e) => setTypeFilter(e.target.value as FilterType)}
            className="bg-slate-700 text-white text-sm rounded px-3 py-1.5 border border-slate-600 focus:outline-none focus:border-primary-500"
          >
            <option value="all">All Types</option>
            <option value="app">Apps</option>
            <option value="service">Services</option>
          </select>
          
          <select
            value={statusFilter}
            onChange={(e) => setStatusFilter(e.target.value as FilterStatus)}
            className="bg-slate-700 text-white text-sm rounded px-3 py-1.5 border border-slate-600 focus:outline-none focus:border-primary-500"
          >
            <option value="all">All Status</option>
            <option value="completed">Completed</option>
            <option value="failed">Failed</option>
            <option value="in_progress">In Progress</option>
          </select>
          
          <select
            value={locationFilter}
            onChange={(e) => setLocationFilter(e.target.value as FilterLocation)}
            className="bg-slate-700 text-white text-sm rounded px-3 py-1.5 border border-slate-600 focus:outline-none focus:border-primary-500"
          >
            <option value="all">All Locations</option>
            <option value="local">Local</option>
            <option value="s3">S3</option>
          </select>
          
          <button
            onClick={() => {
              setTypeFilter('all')
              setStatusFilter('all')
              setLocationFilter('all')
            }}
            className="text-sm text-slate-400 hover:text-white transition-colors"
          >
            Clear filters
          </button>
        </div>
      </div>

      {/* Backup List */}
      <div className="p-6">
        <BackupList
          backups={filteredBackups}
          isLoading={isLoading}
          onRestore={handleRestore}
          onDelete={handleDelete}
          onDownload={handleDownload}
        />
      </div>

      {/* Backup Settings Modal */}
      {showSettings && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50">
          <div className="bg-slate-800 border border-slate-700 rounded-lg w-[600px] shadow-xl">
            <div className="flex items-center justify-between px-6 py-4 border-b border-slate-700">
              <h2 className="text-lg font-semibold text-white">Backup Settings</h2>
              <button
                onClick={() => setShowSettings(false)}
                className="text-slate-400 hover:text-white transition-colors"
              >
                ×
              </button>
            </div>
            
            <div className="p-6 space-y-6">
              <div>
                <h3 className="text-sm font-medium text-white mb-3">Default Schedule</h3>
                <select
                  value={defaultSchedule}
                  onChange={(e) => setDefaultSchedule(e.target.value)}
                  className="w-full bg-slate-700 text-white rounded px-3 py-2 border border-slate-600 focus:outline-none focus:border-primary-500"
                >
                  <option value="hourly">Every hour</option>
                  <option value="daily">Daily at 2:00 AM</option>
                  <option value="weekly">Weekly (Sunday midnight)</option>
                  <option value="monthly">Monthly (1st of month)</option>
                </select>
              </div>
              
              <div>
                <h3 className="text-sm font-medium text-white mb-3">Default Retention</h3>
                <select
                  value={defaultRetention}
                  onChange={(e) => setDefaultRetention(e.target.value)}
                  className="w-full bg-slate-700 text-white rounded px-3 py-2 border border-slate-600 focus:outline-none focus:border-primary-500"
                >
                  <option value="1">Keep for 1 day</option>
                  <option value="7">Keep for 7 days</option>
                  <option value="14">Keep for 14 days</option>
                  <option value="30">Keep for 30 days</option>
                  <option value="90">Keep for 90 days</option>
                </select>
              </div>
              
              <div>
                <h3 className="text-sm font-medium text-white mb-3">Default Destination</h3>
                <select
                  value={defaultDestination}
                  onChange={(e) => setDefaultDestination(e.target.value)}
                  className="w-full bg-slate-700 text-white rounded px-3 py-2 border border-slate-600 focus:outline-none focus:border-primary-500"
                >
                  <option value="local">Local only</option>
                  <option value="s3">S3 (my-backup-bucket)</option>
                  <option value="both">Local + S3</option>
                </select>
              </div>
              
              <div className="pt-4 border-t border-slate-700">
                <button
                  onClick={() => setShowSettings(false)}
                  className="w-full px-4 py-2 bg-primary-600 hover:bg-primary-700 text-white rounded-lg transition-colors"
                >
                  Save Settings
                </button>
              </div>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}
