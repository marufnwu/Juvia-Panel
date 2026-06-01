'use client'

import { useState, useEffect, useCallback } from 'react'
import Editor from '@monaco-editor/react'
import { Save, X, AlertCircle } from 'lucide-react'

interface CodeEditorProps {
  filePath: string
  initialContent: string
  language?: string
  onSave: (content: string) => Promise<void>
  onClose: () => void
}

const languageMap: Record<string, string> = {
  js: 'javascript',
  jsx: 'javascript',
  ts: 'typescript',
  tsx: 'typescript',
  py: 'python',
  rb: 'ruby',
  go: 'go',
  rs: 'rust',
  java: 'java',
  c: 'c',
  cpp: 'cpp',
  cs: 'csharp',
  php: 'php',
  html: 'html',
  css: 'css',
  scss: 'scss',
  json: 'json',
  yaml: 'yaml',
  yml: 'yaml',
  md: 'markdown',
  sh: 'shell',
  bash: 'shell',
  zsh: 'shell',
  sql: 'sql',
  dockerfile: 'dockerfile',
  env: 'shell',
  xml: 'xml',
  toml: 'ini',
  ini: 'ini',
  conf: 'ini',
  cfg: 'ini',
}

function detectLanguage(filePath: string): string {
  const ext = filePath.split('.').pop()?.toLowerCase() || ''
  
  // Special filenames
  const filename = filePath.split('/').pop()?.toLowerCase() || ''
  if (filename === 'dockerfile') return 'dockerfile'
  if (filename === 'makefile') return 'makefile'
  if (filename === '.gitignore' || filename === '.dockerignore') return 'ini'
  if (filename === '.env') return 'shell'
  
  return languageMap[ext] || 'plaintext'
}

export function CodeEditor({
  filePath,
  initialContent,
  language,
  onSave,
  onClose
}: CodeEditorProps) {
  const [content, setContent] = useState(initialContent)
  const [originalContent, setOriginalContent] = useState(initialContent)
  const [isSaving, setIsSaving] = useState(false)
  const [hasChanges, setHasChanges] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const detectedLanguage = language || detectLanguage(filePath)
  const fileName = filePath.split('/').pop() || filePath

  useEffect(() => {
    setContent(initialContent)
    setOriginalContent(initialContent)
    setHasChanges(false)
  }, [initialContent, filePath])

  const handleEditorChange = useCallback((value: string | undefined) => {
    const newContent = value || ''
    setContent(newContent)
    setHasChanges(newContent !== originalContent)
    setError(null)
  }, [originalContent])

  const handleSave = useCallback(async () => {
    if (!hasChanges) return
    
    setIsSaving(true)
    setError(null)
    
    try {
      await onSave(content)
      setOriginalContent(content)
      setHasChanges(false)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to save file')
    } finally {
      setIsSaving(false)
    }
  }, [content, hasChanges, onSave])

  const handleDiscard = useCallback(() => {
    if (hasChanges) {
      if (confirm('You have unsaved changes. Are you sure you want to discard them?')) {
        setContent(originalContent)
        setHasChanges(false)
        onClose()
      }
    } else {
      onClose()
    }
  }, [hasChanges, originalContent, onClose])

  // Keyboard shortcut for save
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if ((e.ctrlKey || e.metaKey) && e.key === 's') {
        e.preventDefault()
        handleSave()
      }
      if (e.key === 'Escape') {
        handleDiscard()
      }
    }
    
    window.addEventListener('keydown', handleKeyDown)
    return () => window.removeEventListener('keydown', handleKeyDown)
  }, [handleSave, handleDiscard])

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50">
      <div className="w-[90vw] h-[90vh] bg-slate-900 rounded-lg shadow-2xl flex flex-col overflow-hidden">
        {/* Header */}
        <div className="flex items-center justify-between px-4 py-3 bg-slate-800 border-b border-slate-700">
          <div className="flex items-center gap-3">
            <span className="text-white font-medium">{fileName}</span>
            {hasChanges && (
              <span className="px-2 py-0.5 text-xs bg-amber-500/20 text-amber-400 rounded">
                Modified
              </span>
            )}
            <span className="text-sm text-slate-400">
              {detectedLanguage}
            </span>
          </div>
          
          <div className="flex items-center gap-2">
            {error && (
              <div className="flex items-center gap-2 text-red-400 text-sm mr-4">
                <AlertCircle className="w-4 h-4" />
                <span>{error}</span>
              </div>
            )}
            
            <button
              onClick={handleDiscard}
              className="flex items-center gap-1.5 px-3 py-1.5 text-sm text-slate-300 hover:text-white hover:bg-slate-700 rounded transition-colors"
            >
              <X className="w-4 h-4" />
              Discard
            </button>
            
            <button
              onClick={handleSave}
              disabled={!hasChanges || isSaving}
              className={`
                flex items-center gap-1.5 px-3 py-1.5 text-sm rounded transition-colors
                ${hasChanges && !isSaving
                  ? 'bg-primary-600 text-white hover:bg-primary-700'
                  : 'bg-slate-700 text-slate-500 cursor-not-allowed'
                }
              `}
            >
              <Save className="w-4 h-4" />
              {isSaving ? 'Saving...' : 'Save'}
            </button>
          </div>
        </div>

        {/* Editor */}
        <div className="flex-1 overflow-hidden">
          <Editor
            height="100%"
            language={detectedLanguage}
            value={content}
            onChange={handleEditorChange}
            theme="vs-dark"
            options={{
              minimap: { enabled: true },
              fontSize: 14,
              fontFamily: '"JetBrains Mono", "Fira Code", "Consolas", monospace',
              lineNumbers: 'on',
              renderWhitespace: 'selection',
              tabSize: 2,
              insertSpaces: true,
              wordWrap: 'on',
              automaticLayout: true,
              scrollBeyondLastLine: false,
              padding: { top: 16 },
            }}
          />
        </div>

        {/* Footer */}
        <div className="px-4 py-2 bg-slate-800 border-t border-slate-700 text-xs text-slate-500 flex justify-between">
          <span>Path: {filePath}</span>
          <span>Ctrl+S to save • Esc to close</span>
        </div>
      </div>
    </div>
  )
}
