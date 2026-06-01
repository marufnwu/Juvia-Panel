'use client'

import { useState, useCallback, useRef } from 'react'
import {
  Folder,
  File,
  FileText,
  Image,
  Upload,
  FolderPlus,
  FilePlus,
  Download,
  Trash2,
  Lock,
  ChevronRight,
  Home,
  RefreshCw,
  Search,
  X
} from 'lucide-react'
import { CodeEditor } from './CodeEditor'

export interface FileItem {
  name: string
  path: string
  isDirectory: boolean
  size: number
  modified: string
  permissions: string
}

interface FileManagerProps {
  target: string
  currentPath: string
  files: FileItem[]
  isLoading: boolean
  onNavigate: (path: string) => void
  onUpload: (files: FileList, path: string) => Promise<void>
  onCreateFolder: (path: string, name: string) => Promise<void>
  onCreateFile: (path: string, name: string) => Promise<void>
  onDelete: (path: string) => Promise<void>
  onDownload: (path: string) => void
  onSaveFile: (path: string, content: string) => Promise<void>
  onPermissionsChange?: (path: string, permissions: string) => Promise<void>
}

function formatFileSize(bytes: number): string {
  if (bytes === 0) return '-'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + ' ' + sizes[i]
}

function formatDate(dateString: string): string {
  const date = new Date(dateString)
  return date.toLocaleDateString('en-US', {
    month: 'short',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit'
  })
}

function getFileIcon({ name, isDirectory }: { name: string; isDirectory: boolean }) {
  if (isDirectory) return <Folder className="w-5 h-5 text-amber-400" />
  
  const ext = name.split('.').pop()?.toLowerCase()
  
  if (['jpg', 'jpeg', 'png', 'gif', 'svg', 'webp'].includes(ext || '')) {
    return <Image className="w-5 h-5 text-green-400" />
  }
  if (['md', 'txt', 'doc', 'docx', 'pdf'].includes(ext || '')) {
    return <FileText className="w-5 h-5 text-blue-400" />
  }
  if (['js', 'ts', 'jsx', 'tsx', 'py', 'go', 'rs', 'java'].includes(ext || '')) {
    return <FileText className="w-5 h-5 text-primary-400" />
  }
  if (['json', 'yaml', 'yml', 'toml', 'ini', 'conf'].includes(ext || '')) {
    return <FileText className="w-5 h-5 text-cyan-400" />
  }
  
  return <File className="w-5 h-5 text-slate-400" />
}

export function FileManager({
  target,
  currentPath,
  files,
  isLoading,
  onNavigate,
  onUpload,
  onCreateFolder,
  onCreateFile,
  onDelete,
  onDownload,
  onSaveFile,
  onPermissionsChange,
}: FileManagerProps) {
  const [selectedFiles, setSelectedFiles] = useState<Set<string>>(new Set())
  const [searchQuery, setSearchQuery] = useState('')
  const [isUploading, setIsUploading] = useState(false)
  const [uploadProgress, setUploadProgress] = useState(0)
  const [showNewFolderInput, setShowNewFolderInput] = useState(false)
  const [showNewFileInput, setShowNewFileInput] = useState(false)
  const [newItemName, setNewItemName] = useState('')
  const [editingFile, setEditingFile] = useState<{ path: string; content: string } | null>(null)
  const [showPermissionsModal, setShowPermissionsModal] = useState<string | null>(null)
  const [deleteConfirm, setDeleteConfirm] = useState<string | null>(null)
  const fileInputRef = useRef<HTMLInputElement>(null)
  const dropZoneRef = useRef<HTMLDivElement>(null)

  const filteredFiles = files.filter(file =>
    file.name.toLowerCase().includes(searchQuery.toLowerCase())
  )

  const pathSegments = currentPath.split('/').filter(Boolean)

  const handleFileClick = useCallback((file: FileItem) => {
    if (file.isDirectory) {
      onNavigate(file.path)
    } else {
      // Open file for editing
      const content = 'Loading...' // In production, fetch file content
      setEditingFile({ path: file.path, content })
    }
  }, [onNavigate])

  const handleFileSelect = useCallback((fileName: string, event: React.MouseEvent) => {
    event.stopPropagation()
    
    setSelectedFiles(prev => {
      const newSet = new Set(prev)
      if (event.shiftKey && prev.size > 0) {
        // Shift-click for range selection
        const fileNames = filteredFiles.map(f => f.name)
        const lastSelected = Array.from(prev).pop()
        const lastIndex = fileNames.indexOf(lastSelected || '')
        const currentIndex = fileNames.indexOf(fileName)
        const start = Math.min(lastIndex, currentIndex)
        const end = Math.max(lastIndex, currentIndex)
        for (let i = start; i <= end; i++) {
          newSet.add(fileNames[i])
        }
      } else if (event.ctrlKey || event.metaKey) {
        // Ctrl/Cmd-click for toggle
        if (newSet.has(fileName)) {
          newSet.delete(fileName)
        } else {
          newSet.add(fileName)
        }
      } else {
        // Regular click
        newSet.clear()
        newSet.add(fileName)
      }
      return newSet
    })
  }, [filteredFiles])

  const handleUpload = useCallback(async (e: React.ChangeEvent<HTMLInputElement>) => {
    const fileList = e.target.files
    if (!fileList || fileList.length === 0) return

    setIsUploading(true)
    setUploadProgress(0)

    try {
      await onUpload(fileList, currentPath)
      setUploadProgress(100)
    } catch (error) {
      console.error('Upload failed:', error)
    } finally {
      setIsUploading(false)
      if (fileInputRef.current) {
        fileInputRef.current.value = ''
      }
    }
  }, [currentPath, onUpload])

  const handleDrop = useCallback(async (e: React.DragEvent) => {
    e.preventDefault()
    const fileList = e.dataTransfer.files
    if (!fileList || fileList.length === 0) return

    setIsUploading(true)
    setUploadProgress(0)

    try {
      await onUpload(fileList, currentPath)
      setUploadProgress(100)
    } catch (error) {
      console.error('Upload failed:', error)
    } finally {
      setIsUploading(false)
    }
  }, [currentPath, onUpload])

  const handleCreateFolder = useCallback(async () => {
    if (!newItemName.trim()) return
    try {
      await onCreateFolder(currentPath, newItemName.trim())
      setNewItemName('')
      setShowNewFolderInput(false)
    } catch (error) {
      console.error('Failed to create folder:', error)
    }
  }, [currentPath, newItemName, onCreateFolder])

  const handleCreateFile = useCallback(async () => {
    if (!newItemName.trim()) return
    try {
      await onCreateFile(currentPath, newItemName.trim())
      setNewItemName('')
      setShowNewFileInput(false)
    } catch (error) {
      console.error('Failed to create file:', error)
    }
  }, [currentPath, newItemName, onCreateFile])

  const handleDelete = useCallback(async (path: string) => {
    try {
      await onDelete(path)
      setDeleteConfirm(null)
      setSelectedFiles(prev => {
        const newSet = new Set(prev)
        newSet.delete(path)
        return newSet
      })
    } catch (error) {
      console.error('Failed to delete:', error)
    }
  }, [onDelete])

  const handleSaveFile = useCallback(async (content: string) => {
    if (!editingFile) return
    await onSaveFile(editingFile.path, content)
    setEditingFile(null)
  }, [editingFile, onSaveFile])

  return (
    <div className="flex flex-col h-full">
      {/* Toolbar */}
      <div className="flex items-center justify-between px-4 py-3 bg-slate-800 border-b border-slate-700">
        <div className="flex items-center gap-2">
          <input
            ref={fileInputRef}
            type="file"
            multiple
            className="hidden"
            onChange={handleUpload}
          />
          
          <button
            onClick={() => fileInputRef.current?.click()}
            disabled={isUploading}
            className="flex items-center gap-1.5 px-3 py-1.5 text-sm bg-primary-600 hover:bg-primary-700 text-white rounded transition-colors disabled:opacity-50"
          >
            <Upload className="w-4 h-4" />
            {isUploading ? 'Uploading...' : 'Upload'}
          </button>
          
          <button
            onClick={() => {
              setShowNewFolderInput(true)
              setShowNewFileInput(false)
            }}
            className="flex items-center gap-1.5 px-3 py-1.5 text-sm text-slate-300 hover:text-white hover:bg-slate-700 rounded transition-colors"
          >
            <FolderPlus className="w-4 h-4" />
            New Folder
          </button>
          
          <button
            onClick={() => {
              setShowNewFileInput(true)
              setShowNewFolderInput(false)
            }}
            className="flex items-center gap-1.5 px-3 py-1.5 text-sm text-slate-300 hover:text-white hover:bg-slate-700 rounded transition-colors"
          >
            <FilePlus className="w-4 h-4" />
            New File
          </button>
          
          {selectedFiles.size > 0 && (
            <>
              <button
                onClick={() => {
                  if (selectedFiles.size === 1) {
                    const file = files.find(f => f.name === Array.from(selectedFiles)[0])
                    if (file) onDownload(file.path)
                  }
                }}
                className="flex items-center gap-1.5 px-3 py-1.5 text-sm text-slate-300 hover:text-white hover:bg-slate-700 rounded transition-colors"
              >
                <Download className="w-4 h-4" />
                Download
              </button>
              
              <button
                onClick={() => setDeleteConfirm(Array.from(selectedFiles)[0])}
                className="flex items-center gap-1.5 px-3 py-1.5 text-sm text-red-400 hover:text-red-300 hover:bg-slate-700 rounded transition-colors"
              >
                <Trash2 className="w-4 h-4" />
                Delete ({selectedFiles.size})
              </button>
            </>
          )}
        </div>
        
        <div className="flex items-center gap-2">
          <div className="relative">
            <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-slate-400" />
            <input
              type="text"
              placeholder="Search files..."
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              className="pl-9 pr-4 py-1.5 w-64 bg-slate-700 text-white text-sm rounded border border-slate-600 focus:outline-none focus:border-primary-500"
            />
          </div>
        </div>
      </div>

      {/* Upload Progress */}
      {isUploading && (
        <div className="px-4 py-2 bg-slate-800 border-b border-slate-700">
          <div className="flex items-center gap-3">
            <div className="flex-1 h-2 bg-slate-700 rounded-full overflow-hidden">
              <div
                className="h-full bg-primary-500 transition-all duration-300"
                style={{ width: `${uploadProgress}%` }}
              />
            </div>
            <span className="text-sm text-slate-400">{uploadProgress}%</span>
          </div>
        </div>
      )}

      {/* Breadcrumb */}
      <div className="flex items-center gap-1 px-4 py-2 bg-slate-800 border-b border-slate-700 text-sm">
        <button
          onClick={() => onNavigate('/')}
          className="text-slate-400 hover:text-white transition-colors"
        >
          <Home className="w-4 h-4" />
        </button>
        {pathSegments.map((segment, index) => (
          <div key={index} className="flex items-center">
            <ChevronRight className="w-4 h-4 text-slate-600 mx-1" />
            <button
              onClick={() => {
                const newPath = '/' + pathSegments.slice(0, index + 1).join('/')
                onNavigate(newPath)
              }}
              className="text-slate-400 hover:text-white transition-colors capitalize"
            >
              {segment}
            </button>
          </div>
        ))}
        <span className="text-white ml-2">{target}</span>
      </div>

      {/* File List */}
      <div
        ref={dropZoneRef}
        onDragOver={(e) => e.preventDefault()}
        onDrop={handleDrop}
        className="flex-1 overflow-auto"
      >
        {isLoading ? (
          <div className="flex items-center justify-center h-full">
            <RefreshCw className="w-6 h-6 text-slate-400 animate-spin" />
          </div>
        ) : filteredFiles.length === 0 ? (
          <div className="flex flex-col items-center justify-center h-full text-slate-400">
            <Folder className="w-12 h-12 mb-4" />
            <p>No files found</p>
          </div>
        ) : (
          <table className="w-full">
            <thead className="sticky top-0 bg-slate-800">
              <tr className="text-left text-sm text-slate-400 border-b border-slate-700">
                <th className="px-4 py-2 w-8"></th>
                <th className="px-4 py-2">Name</th>
                <th className="px-4 py-2 w-24">Size</th>
                <th className="px-4 py-2 w-40">Modified</th>
                <th className="px-4 py-2 w-28">Permissions</th>
              </tr>
            </thead>
            <tbody>
              {filteredFiles.map((file) => (
                <tr
                  key={file.path}
                  onClick={() => handleFileClick(file)}
                  onClickCapture={(e) => handleFileSelect(file.name, e)}
                  className={`
                    border-b border-slate-700/50 cursor-pointer transition-colors
                    ${selectedFiles.has(file.name) ? 'bg-slate-700' : 'hover:bg-slate-800'}
                  `}
                >
                  <td className="px-4 py-2">
                    <input
                      type="checkbox"
                      checked={selectedFiles.has(file.name)}
                      onChange={() => {}}
                      className="rounded border-slate-600"
                    />
                  </td>
                  <td className="px-4 py-2">
                    <div className="flex items-center gap-3">
                      {getFileIcon(file)}
                      <span className="text-white">{file.name}</span>
                    </div>
                  </td>
                  <td className="px-4 py-2 text-slate-400 text-sm">
                    {file.isDirectory ? '-' : formatFileSize(file.size)}
                  </td>
                  <td className="px-4 py-2 text-slate-400 text-sm">
                    {formatDate(file.modified)}
                  </td>
                  <td className="px-4 py-2 text-slate-400 text-sm font-mono">
                    {file.permissions}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>

      {/* New Folder Input */}
      {showNewFolderInput && (
        <div className="absolute bottom-4 left-4 right-4 bg-slate-800 border border-slate-700 rounded-lg p-4 shadow-xl">
          <div className="flex items-center gap-3">
            <FolderPlus className="w-5 h-5 text-amber-400" />
            <input
              type="text"
              placeholder="Folder name"
              value={newItemName}
              onChange={(e) => setNewItemName(e.target.value)}
              onKeyDown={(e) => {
                if (e.key === 'Enter') handleCreateFolder()
                if (e.key === 'Escape') setShowNewFolderInput(false)
              }}
              className="flex-1 px-3 py-2 bg-slate-700 text-white rounded border border-slate-600 focus:outline-none focus:border-primary-500"
              autoFocus
            />
            <button
              onClick={handleCreateFolder}
              className="px-4 py-2 bg-primary-600 hover:bg-primary-700 text-white rounded transition-colors"
            >
              Create
            </button>
            <button
              onClick={() => setShowNewFolderInput(false)}
              className="p-2 text-slate-400 hover:text-white hover:bg-slate-700 rounded transition-colors"
            >
              <X className="w-5 h-5" />
            </button>
          </div>
        </div>
      )}

      {/* New File Input */}
      {showNewFileInput && (
        <div className="absolute bottom-4 left-4 right-4 bg-slate-800 border border-slate-700 rounded-lg p-4 shadow-xl">
          <div className="flex items-center gap-3">
            <FilePlus className="w-5 h-5 text-blue-400" />
            <input
              type="text"
              placeholder="File name"
              value={newItemName}
              onChange={(e) => setNewItemName(e.target.value)}
              onKeyDown={(e) => {
                if (e.key === 'Enter') handleCreateFile()
                if (e.key === 'Escape') setShowNewFileInput(false)
              }}
              className="flex-1 px-3 py-2 bg-slate-700 text-white rounded border border-slate-600 focus:outline-none focus:border-primary-500"
              autoFocus
            />
            <button
              onClick={handleCreateFile}
              className="px-4 py-2 bg-primary-600 hover:bg-primary-700 text-white rounded transition-colors"
            >
              Create
            </button>
            <button
              onClick={() => setShowNewFileInput(false)}
              className="p-2 text-slate-400 hover:text-white hover:bg-slate-700 rounded transition-colors"
            >
              <X className="w-5 h-5" />
            </button>
          </div>
        </div>
      )}

      {/* Delete Confirmation */}
      {deleteConfirm && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50">
          <div className="bg-slate-800 border border-slate-700 rounded-lg p-6 w-96 shadow-xl">
            <h3 className="text-lg font-medium text-white mb-2">Confirm Delete</h3>
            <p className="text-slate-400 mb-4">
              Are you sure you want to delete "{deleteConfirm}"? This action cannot be undone.
            </p>
            <div className="flex justify-end gap-3">
              <button
                onClick={() => setDeleteConfirm(null)}
                className="px-4 py-2 text-slate-300 hover:text-white hover:bg-slate-700 rounded transition-colors"
              >
                Cancel
              </button>
              <button
                onClick={() => handleDelete(deleteConfirm)}
                className="px-4 py-2 bg-red-600 hover:bg-red-700 text-white rounded transition-colors"
              >
                Delete
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Permissions Modal */}
      {showPermissionsModal && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50">
          <div className="bg-slate-800 border border-slate-700 rounded-lg p-6 w-96 shadow-xl">
            <h3 className="text-lg font-medium text-white mb-4">Change Permissions</h3>
            <p className="text-slate-400 mb-4 text-sm">{showPermissionsModal}</p>
            <div className="flex justify-end gap-3">
              <button
                onClick={() => setShowPermissionsModal(null)}
                className="px-4 py-2 text-slate-300 hover:text-white hover:bg-slate-700 rounded transition-colors"
              >
                Cancel
              </button>
              <button
                onClick={() => setShowPermissionsModal(null)}
                className="px-4 py-2 bg-primary-600 hover:bg-primary-700 text-white rounded transition-colors"
              >
                Save
              </button>
            </div>
          </div>
        </div>
      )}

      {/* File Editor */}
      {editingFile && (
        <CodeEditor
          filePath={editingFile.path}
          initialContent={editingFile.content}
          onSave={handleSaveFile}
          onClose={() => setEditingFile(null)}
        />
      )}
    </div>
  )
}
