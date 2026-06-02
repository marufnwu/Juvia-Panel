'use client'

import { FileManager } from '@/components/files/FileManager'
import { Server, Box } from 'lucide-react'

interface FileTarget {
  id: string
  name: string
  type: 'host' | 'app' | 'service'
}

const availableTargets: FileTarget[] = [
  { id: 'host', name: 'Host Server', type: 'host' },
]

export default function FilesPage() {
  const getTargetIcon = (t: FileTarget) => {
    if (t.type === 'host') return <Server className="w-4 h-4" />
    return <Box className="w-4 h-4" />
  }

  return (
    <div className="h-[calc(100vh-4rem)] flex flex-col bg-slate-900">
      <div className="px-4 py-3 bg-slate-800 border-b border-slate-700">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-4">
            <h1 className="text-lg font-semibold text-white">File Manager</h1>
            <div className="flex items-center gap-2">
              <label className="text-sm text-slate-400">Target:</label>
              <select
                defaultValue="host"
                className="bg-slate-700 text-white text-sm rounded px-2 py-1 border border-slate-600 focus:outline-none focus:border-primary-500"
              >
                {availableTargets.map(t => (
                  <option key={t.id} value={t.id}>{t.name}</option>
                ))}
              </select>
            </div>
          </div>
        </div>
      </div>

      <div className="flex-1 overflow-hidden">
        <div className="h-full flex items-center justify-center bg-slate-900">
          <div className="text-center">
            <FileManager
              target="host"
              currentPath="/"
              files={[]}
              isLoading={false}
              onNavigate={() => {}}
              onUpload={async () => {}}
              onCreateFolder={async () => {}}
              onCreateFile={async () => {}}
              onDelete={async () => {}}
              onDownload={() => {}}
              onSaveFile={async () => {}}
            />
            <p className="text-slate-500 mt-4">Full file management coming soon</p>
          </div>
        </div>
      </div>
    </div>
  )
}