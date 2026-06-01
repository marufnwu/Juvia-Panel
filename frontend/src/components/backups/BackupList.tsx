'use client'

import { useState, useCallback } from 'react'
import {
  CheckCircle,
  XCircle,
  Clock,
  Download,
  RotateCcw,
  Trash2,
  Loader2,
  AlertTriangle
} from 'lucide-react'

export interface Backup {
  id: string
  name: string
  type: 'app' | 'service'
  target_id: string
  target_name: string
  status: 'completed' | 'failed' | 'in_progress'
  size?: number
  location?: string
  backup_type: 'manual' | 'scheduled'
  created_at: string
}

interface BackupListProps {
  backups: Backup[]
  isLoading: boolean
  onRestore: (backupId: string) => void
  onDelete: (backupId: string) => void
  onDownload?: (backupId: string) => void
}

function formatFileSize(bytes?: number): string {
  if (!bytes) return '-'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + ' ' + sizes[i]
}

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
    year: date.getFullYear() !== now.getFullYear() ? 'numeric' : undefined
  })
}

function getStatusIcon(status: Backup['status']) {
  switch (status) {
    case 'completed':
      return <CheckCircle className="w-4 h-4 text-green-500" />
    case 'failed':
      return <XCircle className="w-4 h-4 text-red-500" />
    case 'in_progress':
      return <Clock className="w-4 h-4 text-amber-500 animate-pulse" />
  }
}

export function BackupList({
  backups,
  isLoading,
  onRestore,
  onDelete,
  onDownload
}: BackupListProps) {
  const [restoreConfirm, setRestoreConfirm] = useState<string | null>(null)
  const [deleteConfirm, setDeleteConfirm] = useState<string | null>(null)

  const handleRestore = useCallback((backupId: string) => {
    setRestoreConfirm(backupId)
  }, [])

  const confirmRestore = useCallback(() => {
    if (restoreConfirm) {
      onRestore(restoreConfirm)
      setRestoreConfirm(null)
    }
  }, [restoreConfirm, onRestore])

  const handleDelete = useCallback((backupId: string) => {
    setDeleteConfirm(backupId)
  }, [])

  const confirmDelete = useCallback(() => {
    if (deleteConfirm) {
      onDelete(deleteConfirm)
      setDeleteConfirm(null)
    }
  }, [deleteConfirm, onDelete])

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-12">
        <Loader2 className="w-6 h-6 text-slate-400 animate-spin" />
      </div>
    )
  }

  if (backups.length === 0) {
    return (
      <div className="flex flex-col items-center justify-center py-12 text-slate-400">
        <Download className="w-12 h-12 mb-4" />
        <p className="text-lg mb-2">No backups yet</p>
        <p className="text-sm">Backups will appear here once created</p>
      </div>
    )
  }

  return (
    <div className="space-y-3">
      {backups.map((backup) => (
        <div
          key={backup.id}
          className="flex items-center justify-between p-4 bg-slate-800 rounded-lg border border-slate-700 hover:border-slate-600 transition-colors"
        >
          <div className="flex items-center gap-4">
            {getStatusIcon(backup.status)}
            
            <div>
              <div className="flex items-center gap-2">
                <span className="font-medium text-white">{backup.name}</span>
                <span className={`px-2 py-0.5 text-xs rounded ${
                  backup.backup_type === 'manual'
                    ? 'bg-blue-500/20 text-blue-400'
                    : 'bg-purple-500/20 text-purple-400'
                }`}>
                  {backup.backup_type}
                </span>
                <span className="px-2 py-0.5 text-xs bg-slate-700 text-slate-300 rounded">
                  {backup.type === 'app' ? 'App' : 'Service'}
                </span>
              </div>
              <div className="flex items-center gap-3 mt-1 text-sm text-slate-400">
                <span>{backup.target_name}</span>
                <span>•</span>
                <span>{formatDate(backup.created_at)}</span>
                {backup.size && (
                  <>
                    <span>•</span>
                    <span>{formatFileSize(backup.size)}</span>
                  </>
                )}
                {backup.location && (
                  <>
                    <span>•</span>
                    <span>{backup.location}</span>
                  </>
                )}
              </div>
            </div>
          </div>

          <div className="flex items-center gap-2">
            {backup.status === 'completed' && (
              <>
                <button
                  onClick={() => handleRestore(backup.id)}
                  className="flex items-center gap-1.5 px-3 py-1.5 text-sm text-slate-300 hover:text-white hover:bg-slate-700 rounded transition-colors"
                  title="Restore backup"
                >
                  <RotateCcw className="w-4 h-4" />
                  Restore
                </button>
                
                {onDownload && (
                  <button
                    onClick={() => onDownload(backup.id)}
                    className="flex items-center gap-1.5 px-3 py-1.5 text-sm text-slate-300 hover:text-white hover:bg-slate-700 rounded transition-colors"
                    title="Download backup"
                  >
                    <Download className="w-4 h-4" />
                  </button>
                )}
              </>
            )}
            
            <button
              onClick={() => handleDelete(backup.id)}
              className="flex items-center gap-1.5 px-3 py-1.5 text-sm text-red-400 hover:text-red-300 hover:bg-slate-700 rounded transition-colors"
              title="Delete backup"
            >
              <Trash2 className="w-4 h-4" />
            </button>
          </div>
        </div>
      ))}

      {/* Restore Confirmation Modal */}
      {restoreConfirm && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50">
          <div className="bg-slate-800 border border-slate-700 rounded-lg p-6 w-[450px] shadow-xl">
            <div className="flex items-center gap-3 mb-4">
              <div className="p-2 bg-amber-500/20 rounded-lg">
                <AlertTriangle className="w-6 h-6 text-amber-500" />
              </div>
              <div>
                <h3 className="text-lg font-medium text-white">Restore Backup</h3>
                <p className="text-sm text-slate-400">This action cannot be undone</p>
              </div>
            </div>
            
            <p className="text-slate-300 mb-4">
              This will overwrite the current data with the backup. A new backup of the 
              current state will be created automatically before restoring.
            </p>
            
            <div className="flex justify-end gap-3">
              <button
                onClick={() => setRestoreConfirm(null)}
                className="px-4 py-2 text-slate-300 hover:text-white hover:bg-slate-700 rounded transition-colors"
              >
                Cancel
              </button>
              <button
                onClick={confirmRestore}
                className="px-4 py-2 bg-amber-600 hover:bg-amber-700 text-white rounded transition-colors"
              >
                Restore Backup
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Delete Confirmation Modal */}
      {deleteConfirm && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50">
          <div className="bg-slate-800 border border-slate-700 rounded-lg p-6 w-[450px] shadow-xl">
            <div className="flex items-center gap-3 mb-4">
              <div className="p-2 bg-red-500/20 rounded-lg">
                <Trash2 className="w-6 h-6 text-red-500" />
              </div>
              <div>
                <h3 className="text-lg font-medium text-white">Delete Backup</h3>
                <p className="text-sm text-slate-400">This action cannot be undone</p>
              </div>
            </div>
            
            <p className="text-slate-300 mb-4">
              Are you sure you want to delete this backup? This backup will be 
              permanently removed and cannot be recovered.
            </p>
            
            <div className="flex justify-end gap-3">
              <button
                onClick={() => setDeleteConfirm(null)}
                className="px-4 py-2 text-slate-300 hover:text-white hover:bg-slate-700 rounded transition-colors"
              >
                Cancel
              </button>
              <button
                onClick={confirmDelete}
                className="px-4 py-2 bg-red-600 hover:bg-red-700 text-white rounded transition-colors"
              >
                Delete Backup
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}
