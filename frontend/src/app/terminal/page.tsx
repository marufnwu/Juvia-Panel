'use client'

import { useState, useCallback, useEffect } from 'react'
import dynamic from 'next/dynamic'
import {
  Plus,
  Copy,
  Clipboard,
  Maximize2,
  Minimize2,
  Trash2,
  Server,
  Box
} from 'lucide-react'

// Dynamic import to avoid SSR issues with xterm.js
const Terminal = dynamic(
  () => import('@/components/terminal/Terminal').then(mod => mod.Terminal),
  { 
    ssr: false,
    loading: () => (
      <div className="h-full w-full bg-slate-900 rounded-lg flex items-center justify-center">
        <div className="text-slate-400">Loading terminal...</div>
      </div>
    )
  }
)

interface TerminalTab {
  id: string
  title: string
  target: string
}

interface TerminalTarget {
  id: string
  name: string
  type: 'host' | 'app' | 'service'
}

// Mock targets - in production these would come from API
const availableTargets: TerminalTarget[] = [
  { id: 'host', name: 'Host Server', type: 'host' },
  { id: 'app:api-prod', name: 'api-prod', type: 'app' },
  { id: 'app:web-client', name: 'web-client', type: 'app' },
  { id: 'app:worker', name: 'worker', type: 'app' },
  { id: 'svc:main-pg', name: 'main-pg (PostgreSQL)', type: 'service' },
  { id: 'svc:redis-cache', name: 'redis-cache (Redis)', type: 'service' },
]

export default function TerminalPage() {
  const [tabs, setTabs] = useState<TerminalTab[]>([
    { id: '1', title: 'Host Server', target: 'host' }
  ])
  const [activeTabId, setActiveTabId] = useState('1')
  const [selectedTarget, setSelectedTarget] = useState('host')
  const [isFullscreen, setIsFullscreen] = useState(false)
  const [mounted, setMounted] = useState(false)

  useEffect(() => {
    setMounted(true)
  }, [])

  const activeTab = tabs.find(t => t.id === activeTabId)

  const handleTargetChange = useCallback((targetId: string) => {
    setSelectedTarget(targetId)
    const target = availableTargets.find(t => t.id === targetId)
    if (target && activeTab) {
      setTabs(prev => prev.map(tab =>
        tab.id === activeTabId
          ? { ...tab, title: target.name, target: targetId }
          : tab
      ))
    }
  }, [activeTab, activeTabId])

  const addNewTab = useCallback(() => {
    const newId = Date.now().toString()
    setTabs(prev => [...prev, {
      id: newId,
      title: 'New Terminal',
      target: 'host'
    }])
    setActiveTabId(newId)
    setSelectedTarget('host')
  }, [])

  const closeTab = useCallback((tabId: string) => {
    if (tabs.length === 1) return // Don't close last tab
    
    const tabIndex = tabs.findIndex(t => t.id === tabId)
    const newTabs = tabs.filter(t => t.id !== tabId)
    setTabs(newTabs)
    
    if (activeTabId === tabId) {
      // Switch to adjacent tab
      const newActiveIndex = Math.min(tabIndex, newTabs.length - 1)
      setActiveTabId(newTabs[newActiveIndex].id)
      setSelectedTarget(newTabs[newActiveIndex].target)
    }
  }, [tabs, activeTabId])

  const copySelection = useCallback(() => {
    // Copy is handled by Ctrl+C in the terminal
  }, [])

  const pasteClipboard = useCallback(async () => {
    // Paste is handled by Ctrl+V in the terminal
  }, [])

  const toggleFullscreen = useCallback(() => {
    setIsFullscreen(prev => !prev)
  }, [])

  const getTargetIcon = (target: TerminalTarget) => {
    if (target.type === 'host') return <Server className="w-4 h-4" />
    if (target.type === 'app') return <Box className="w-4 h-4" />
    return <Box className="w-4 h-4" />
  }

  if (!mounted) {
    return (
      <div className="flex flex-col h-screen bg-slate-900">
        <div className="flex items-center justify-center h-full">
          <div className="text-slate-400">Loading terminal...</div>
        </div>
      </div>
    )
  }

  return (
    <div className={`flex flex-col h-screen bg-slate-900 ${isFullscreen ? 'fixed inset-0 z-50' : ''}`}>
      {/* Header */}
      <div className="flex items-center justify-between px-4 py-2 bg-slate-800 border-b border-slate-700">
        <div className="flex items-center gap-4">
          {/* Target Selector */}
          <div className="flex items-center gap-2">
            <label className="text-sm text-slate-400">Target:</label>
            <select
              value={selectedTarget}
              onChange={(e) => handleTargetChange(e.target.value)}
              className="bg-slate-700 text-white text-sm rounded px-2 py-1 border border-slate-600 focus:outline-none focus:border-primary-500"
            >
              {availableTargets.map(target => (
                <option key={target.id} value={target.id}>
                  {target.name}
                </option>
              ))}
            </select>
          </div>
        </div>

        {/* Tab Actions */}
        <div className="flex items-center gap-2">
          <button
            onClick={addNewTab}
            className="p-1.5 text-slate-400 hover:text-white hover:bg-slate-700 rounded transition-colors"
            title="New Tab (Ctrl+T)"
          >
            <Plus className="w-4 h-4" />
          </button>
          <button
            onClick={copySelection}
            className="p-1.5 text-slate-400 hover:text-white hover:bg-slate-700 rounded transition-colors"
            title="Copy (Ctrl+Shift+C)"
          >
            <Copy className="w-4 h-4" />
          </button>
          <button
            onClick={pasteClipboard}
            className="p-1.5 text-slate-400 hover:text-white hover:bg-slate-700 rounded transition-colors"
            title="Paste (Ctrl+Shift+V)"
          >
            <Clipboard className="w-4 h-4" />
          </button>
          <button
            onClick={toggleFullscreen}
            className="p-1.5 text-slate-400 hover:text-white hover:bg-slate-700 rounded transition-colors"
            title="Fullscreen (F11)"
          >
            {isFullscreen ? (
              <Minimize2 className="w-4 h-4" />
            ) : (
              <Maximize2 className="w-4 h-4" />
            )}
          </button>
        </div>
      </div>

      {/* Tabs */}
      <div className="flex items-center px-2 bg-slate-800 border-b border-slate-700 overflow-x-auto">
        {tabs.map(tab => (
          <div
            key={tab.id}
            className={`
              flex items-center gap-2 px-3 py-2 text-sm cursor-pointer border-b-2 transition-colors
              ${tab.id === activeTabId
                ? 'text-white border-primary-500'
                : 'text-slate-400 border-transparent hover:text-white hover:bg-slate-700'
              }
            `}
            onClick={() => {
              setActiveTabId(tab.id)
              setSelectedTarget(tab.target)
            }}
          >
            {getTargetIcon(availableTargets.find(t => t.id === tab.target) || availableTargets[0])}
            <span>{tab.title}</span>
            {tabs.length > 1 && (
              <button
                onClick={(e) => {
                  e.stopPropagation()
                  closeTab(tab.id)
                }}
                className="ml-1 p-0.5 hover:bg-slate-600 rounded"
              >
                <Trash2 className="w-3 h-3" />
              </button>
            )}
          </div>
        ))}
      </div>

      {/* Terminal Content */}
      <div className="flex-1 overflow-hidden">
        {activeTab && (
          <Terminal
            key={activeTab.id}
            target={activeTab.target}
          />
        )}
      </div>

      {/* Keyboard Shortcuts Help */}
      <div className="px-4 py-2 bg-slate-800 border-t border-slate-700 text-xs text-slate-500">
        <span className="mr-4">Ctrl+C: Copy/SIGINT</span>
        <span className="mr-4">Ctrl+V: Paste</span>
        <span className="mr-4">Ctrl+Shift+C: Copy terminal</span>
        <span className="mr-4">Ctrl+Shift+V: Paste terminal</span>
        <span className="mr-4">F11: Fullscreen</span>
        <span>Ctrl+T: New Tab</span>
      </div>
    </div>
  )
}
