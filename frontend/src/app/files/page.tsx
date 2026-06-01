'use client'

import { useState, useCallback, useEffect } from 'react'
import { FileManager, FileItem } from '@/components/files/FileManager'
import { Server, Box } from 'lucide-react'

interface FileTarget {
  id: string
  name: string
  type: 'host' | 'app' | 'service'
}

// Mock targets - in production these would come from API
const availableTargets: FileTarget[] = [
  { id: 'host', name: 'Host Server', type: 'host' },
  { id: 'app:api-prod', name: 'api-prod', type: 'app' },
  { id: 'app:web-client', name: 'web-client', type: 'app' },
  { id: 'app:worker', name: 'worker', type: 'app' },
  { id: 'svc:main-pg', name: 'main-pg (PostgreSQL)', type: 'service' },
  { id: 'svc:redis-cache', name: 'redis-cache (Redis)', type: 'service' },
]

// Mock file data - in production this would come from API
const mockFiles: FileItem[] = [
  {
    name: 'src',
    path: '/var/panel/apps/api-prod/src',
    isDirectory: true,
    size: 0,
    modified: '2024-06-01T10:00:00Z',
    permissions: 'drwxr-xr-x',
  },
  {
    name: 'public',
    path: '/var/panel/apps/api-prod/public',
    isDirectory: true,
    size: 0,
    modified: '2024-06-01T10:00:00Z',
    permissions: 'drwxr-xr-x',
  },
  {
    name: 'node_modules',
    path: '/var/panel/apps/api-prod/node_modules',
    isDirectory: true,
    size: 0,
    modified: '2024-06-01T10:00:00Z',
    permissions: 'drwxr-xr-x',
  },
  {
    name: 'package.json',
    path: '/var/panel/apps/api-prod/package.json',
    isDirectory: false,
    size: 2345,
    modified: '2024-06-01T10:00:00Z',
    permissions: '-rw-r--r--',
  },
  {
    name: 'server.js',
    path: '/var/panel/apps/api-prod/server.js',
    isDirectory: false,
    size: 5248,
    modified: '2024-06-01T10:00:00Z',
    permissions: '-rw-r--r--',
  },
  {
    name: '.env',
    path: '/var/panel/apps/api-prod/.env',
    isDirectory: false,
    size: 512,
    modified: '2024-06-01T10:00:00Z',
    permissions: '-rw-------',
  },
  {
    name: 'README.md',
    path: '/var/panel/apps/api-prod/README.md',
    isDirectory: false,
    size: 1234,
    modified: '2024-06-01T10:00:00Z',
    permissions: '-rw-r--r--',
  },
]

export default function FilesPage() {
  const [selectedTarget, setSelectedTarget] = useState('host')
  const [currentPath, setCurrentPath] = useState('/')
  const [files, setFiles] = useState<FileItem[]>([])
  const [isLoading, setIsLoading] = useState(false)

  const target = availableTargets.find(t => t.id === selectedTarget)

  // Simulate loading files based on target and path
  useEffect(() => {
    const loadFiles = async () => {
      setIsLoading(true)
      // Simulate API call
      await new Promise(resolve => setTimeout(resolve, 500))
      
      // In production, this would be an API call based on target and path
      // For now, return mock data with adjusted paths
      const basePath = selectedTarget === 'host' 
        ? '/var/panel' 
        : `/var/panel/apps/${selectedTarget.replace('app:', '')}`
      
      const adjustedFiles = mockFiles.map(f => ({
        ...f,
        path: `${basePath}${f.path.replace('/var/panel/apps/api-prod', '')}`
      }))
      
      setFiles(adjustedFiles)
      setIsLoading(false)
    }

    loadFiles()
  }, [selectedTarget, currentPath])

  const handleNavigate = useCallback((path: string) => {
    setCurrentPath(path)
  }, [])

  const handleUpload = useCallback(async (fileList: FileList, path: string) => {
    // In production, this would upload to the API
    console.log('Uploading files:', fileList, 'to', path)
    await new Promise(resolve => setTimeout(resolve, 1000))
  }, [])

  const handleCreateFolder = useCallback(async (path: string, name: string) => {
    // In production, this would call the API
    console.log('Creating folder:', name, 'at', path)
    await new Promise(resolve => setTimeout(resolve, 500))
  }, [])

  const handleCreateFile = useCallback(async (path: string, name: string) => {
    // In production, this would call the API
    console.log('Creating file:', name, 'at', path)
    await new Promise(resolve => setTimeout(resolve, 500))
  }, [])

  const handleDelete = useCallback(async (path: string) => {
    // In production, this would call the API
    console.log('Deleting:', path)
    await new Promise(resolve => setTimeout(resolve, 500))
  }, [])

  const handleDownload = useCallback((path: string) => {
    // In production, this would trigger a download
    console.log('Downloading:', path)
  }, [])

  const handleSaveFile = useCallback(async (path: string, content: string) => {
    // In production, this would save to the API
    console.log('Saving file:', path, 'with content:', content)
    await new Promise(resolve => setTimeout(resolve, 500))
  }, [])

  const getTargetIcon = (t: FileTarget) => {
    if (t.type === 'host') return <Server className="w-4 h-4" />
    return <Box className="w-4 h-4" />
  }

  return (
    <div className="h-[calc(100vh-4rem)] flex flex-col bg-slate-900">
      {/* Header */}
      <div className="px-4 py-3 bg-slate-800 border-b border-slate-700">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-4">
            <h1 className="text-lg font-semibold text-white">File Manager</h1>
            
            {/* Target Selector */}
            <div className="flex items-center gap-2">
              <label className="text-sm text-slate-400">Target:</label>
              <select
                value={selectedTarget}
                onChange={(e) => setSelectedTarget(e.target.value)}
                className="bg-slate-700 text-white text-sm rounded px-2 py-1 border border-slate-600 focus:outline-none focus:border-primary-500"
              >
                {availableTargets.map(t => (
                  <option key={t.id} value={t.id}>
                    {t.name}
                  </option>
                ))}
              </select>
            </div>
          </div>
          
          <div className="text-sm text-slate-400">
            {target?.type === 'host' ? 'Host Server' : `${target?.type}: ${target?.name}`}
          </div>
        </div>
      </div>

      {/* File Manager Component */}
      <div className="flex-1 overflow-hidden relative">
        <FileManager
          target={selectedTarget}
          currentPath={currentPath}
          files={files}
          isLoading={isLoading}
          onNavigate={handleNavigate}
          onUpload={handleUpload}
          onCreateFolder={handleCreateFolder}
          onCreateFile={handleCreateFile}
          onDelete={handleDelete}
          onDownload={handleDownload}
          onSaveFile={handleSaveFile}
        />
      </div>
    </div>
  )
}
