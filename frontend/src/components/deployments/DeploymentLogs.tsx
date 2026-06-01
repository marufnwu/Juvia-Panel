'use client'

import { useState, useEffect, useRef } from 'react'
import { useQuery } from '@tanstack/react-query'
import {
  Search,
  Download,
  RefreshCw,
  Pause,
  Play,
  Loader2,
  X
} from 'lucide-react'
import { api } from '@/lib/api'

interface LogLine {
  timestamp: string
  stream: 'stdout' | 'stderr'
  message: string
}

interface DeploymentLogsProps {
  appId: string
  deploymentId: string
  isStreaming?: boolean
}

function formatTimestamp(timestamp: string): string {
  const date = new Date(timestamp)
  return date.toLocaleTimeString('en-US', { hour12: false })
}

function highlightErrors(logs: LogLine[]): LogLine[] {
  return logs.map(line => {
    if (line.stream === 'stderr' || line.message.toLowerCase().includes('error')) {
      return { ...line, message: `[ERROR] ${line.message}` }
    }
    return line
  })
}

export function DeploymentLogs({ appId, deploymentId, isStreaming = false }: DeploymentLogsProps) {
  const [logs, setLogs] = useState<LogLine[]>([])
  const [searchQuery, setSearchQuery] = useState('')
  const [autoScroll, setAutoScroll] = useState(true)
  const [streamType, setStreamType] = useState<'both' | 'stdout' | 'stderr'>('both')
  const logsEndRef = useRef<HTMLDivElement>(null)
  const logsContainerRef = useRef<HTMLDivElement>(null)

  // Fetch initial logs
  const { data: initialLogs, isLoading } = useQuery({
    queryKey: ['deployment-logs', deploymentId],
    queryFn: async () => {
      const response = await api.apps.getDeploymentLogs(appId, deploymentId)
      const data = response as unknown as { logs: string }
      // Parse logs - assuming newline-delimited format
      const lines = (data.logs || '').split('\n').filter(Boolean).map((line, i) => {
        try {
          const parsed = JSON.parse(line)
          return {
            timestamp: parsed.timestamp || new Date().toISOString(),
            stream: parsed.stream || 'stdout',
            message: parsed.message || line,
          }
        } catch {
          return {
            timestamp: new Date().toISOString(),
            stream: 'stdout' as const,
            message: line,
          }
        }
      })
      return lines
    },
    enabled: !!deploymentId,
  })

  // Set initial logs
  useEffect(() => {
    if (initialLogs) {
      setLogs(initialLogs)
    }
  }, [initialLogs])

  // Auto-scroll to bottom
  useEffect(() => {
    if (autoScroll && logsEndRef.current) {
      logsEndRef.current.scrollIntoView({ behavior: 'smooth' })
    }
  }, [logs, autoScroll])

  // Handle scroll to detect if user scrolled up
  const handleScroll = () => {
    if (logsContainerRef.current) {
      const { scrollTop, scrollHeight, clientHeight } = logsContainerRef.current
      const isAtBottom = scrollHeight - scrollTop - clientHeight < 50
      setAutoScroll(isAtBottom)
    }
  }

  // Filter logs based on search and stream type
  const filteredLogs = logs.filter(line => {
    // Filter by stream type
    if (streamType === 'stdout' && line.stream === 'stderr') return false
    if (streamType === 'stderr' && line.stream === 'stdout') return false
    
    // Filter by search query
    if (searchQuery && !line.message.toLowerCase().includes(searchQuery.toLowerCase())) {
      return false
    }
    
    return true
  })

  const highlightedLogs = highlightErrors(filteredLogs)

  const handleDownload = () => {
    const logText = highlightedLogs
      .map(line => `[${line.timestamp}] ${line.stream.toUpperCase()}: ${line.message}`)
      .join('\n')
    
    const blob = new Blob([logText], { type: 'text/plain' })
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = `deployment-${deploymentId}.log`
    document.body.appendChild(a)
    a.click()
    document.body.removeChild(a)
    URL.revokeObjectURL(url)
  }

  if (isLoading) {
    return (
      <div className="flex items-center justify-center h-64">
        <Loader2 className="w-6 h-6 text-primary-500 animate-spin" />
      </div>
    )
  }

  return (
    <div className="flex flex-col h-full bg-slate-900 rounded-lg border border-slate-700">
      {/* Header */}
      <div className="flex items-center justify-between px-4 py-2 border-b border-slate-700">
        <div className="flex items-center gap-4">
          {/* Stream filter */}
          <div className="flex items-center gap-1 bg-slate-800 rounded p-0.5">
            {(['both', 'stdout', 'stderr'] as const).map((type) => (
              <button
                key={type}
                onClick={() => setStreamType(type)}
                className={`px-2 py-1 text-xs rounded transition-colors ${
                  streamType === type
                    ? 'bg-slate-700 text-white'
                    : 'text-slate-400 hover:text-white'
                }`}
              >
                {type === 'both' ? 'All' : type.toUpperCase()}
              </button>
            ))}
          </div>

          {/* Search */}
          <div className="relative">
            <Search className="absolute left-2 top-1/2 -translate-y-1/2 w-3 h-3 text-slate-400" />
            <input
              type="text"
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              placeholder="Filter logs..."
              className="pl-7 pr-3 py-1 bg-slate-800 border border-slate-700 rounded text-xs text-white placeholder-slate-400 focus:outline-none focus:ring-1 focus:ring-primary-500 w-48"
            />
            {searchQuery && (
              <button
                onClick={() => setSearchQuery('')}
                className="absolute right-2 top-1/2 -translate-y-1/2"
              >
                <X className="w-3 h-3 text-slate-400 hover:text-white" />
              </button>
            )}
          </div>
        </div>

        <div className="flex items-center gap-2">
          {/* Auto-scroll toggle */}
          <button
            onClick={() => setAutoScroll(!autoScroll)}
            className={`p-1.5 rounded transition-colors ${
              autoScroll ? 'text-primary-400 bg-primary-500/20' : 'text-slate-400 hover:text-white'
            }`}
            title={autoScroll ? 'Auto-scroll enabled' : 'Auto-scroll paused'}
          >
            {autoScroll ? <Play className="w-4 h-4" /> : <Pause className="w-4 h-4" />}
          </button>

          {/* Refresh */}
          <button
            className="p-1.5 text-slate-400 hover:text-white rounded transition-colors"
            title="Refresh logs"
          >
            <RefreshCw className="w-4 h-4" />
          </button>

          {/* Download */}
          <button
            onClick={handleDownload}
            className="p-1.5 text-slate-400 hover:text-white rounded transition-colors"
            title="Download logs"
          >
            <Download className="w-4 h-4" />
          </button>

          {/* Streaming indicator */}
          {isStreaming && (
            <div className="flex items-center gap-1.5 px-2 py-1 bg-green-500/20 text-green-400 rounded text-xs">
              <span className="w-2 h-2 rounded-full bg-green-400 animate-pulse" />
              Streaming
            </div>
          )}
        </div>
      </div>

      {/* Logs content */}
      <div
        ref={logsContainerRef}
        onScroll={handleScroll}
        className="flex-1 overflow-auto p-4 font-mono text-xs"
      >
        {highlightedLogs.length === 0 ? (
          <div className="flex items-center justify-center h-full text-slate-500">
            {searchQuery ? 'No matching logs found' : 'No logs available'}
          </div>
        ) : (
          <div className="space-y-0.5">
            {highlightedLogs.map((line, index) => (
              <div
                key={index}
                className={`flex gap-2 ${
                  line.stream === 'stderr' ? 'text-red-400' : 'text-slate-300'
                }`}
              >
                <span className="text-slate-500 shrink-0">
                  [{formatTimestamp(line.timestamp)}]
                </span>
                <span className={`shrink-0 w-12 text-right ${
                  line.stream === 'stderr' ? 'text-red-500' : 'text-slate-500'
                }`}>
                  {line.stream}
                </span>
                <span className="whitespace-pre-wrap break-all flex-1">
                  {line.message}
                </span>
              </div>
            ))}
            <div ref={logsEndRef} />
          </div>
        )}
      </div>

      {/* Footer */}
      <div className="px-4 py-2 border-t border-slate-700 flex items-center justify-between text-xs text-slate-500">
        <span>
          {filteredLogs.length} lines
          {searchQuery && ` (filtered from ${logs.length})`}
        </span>
        {autoScroll && <span className="text-primary-400">Auto-scrolling</span>}
      </div>
    </div>
  )
}

export default DeploymentLogs
