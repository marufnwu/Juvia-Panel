'use client'

import { useState, useEffect, useCallback, useRef } from 'react'
import { useRouter } from 'next/navigation'
import {
  Search,
  Command,
  LayoutDashboard,
  AppWindow,
  Database,
  Server,
  Settings,
  Terminal,
  FolderOpen,
  Plus,
  Play,
  RefreshCw,
  Keyboard
} from 'lucide-react'

interface CommandItem {
  id: string
  name: string
  description?: string
  icon: React.ElementType
  action: () => void
  category: string
}

const commands: CommandItem[] = [
  {
    id: 'go-dashboard',
    name: 'Go to Dashboard',
    description: 'View server overview and metrics',
    icon: LayoutDashboard,
    action: () => {},
    category: 'Navigation'
  },
  {
    id: 'go-apps',
    name: 'Go to Apps',
    description: 'Manage deployed applications',
    icon: AppWindow,
    action: () => {},
    category: 'Navigation'
  },
  {
    id: 'go-services',
    name: 'Go to Services',
    description: 'Manage databases and services',
    icon: Database,
    action: () => {},
    category: 'Navigation'
  },
  {
    id: 'go-server',
    name: 'Go to Server',
    description: 'Server settings and monitoring',
    icon: Server,
    action: () => {},
    category: 'Navigation'
  },
  {
    id: 'go-settings',
    name: 'Go to Settings',
    description: 'Panel configuration',
    icon: Settings,
    action: () => {},
    category: 'Navigation'
  },
  {
    id: 'open-terminal',
    name: 'Open Terminal',
    description: 'Access web terminal',
    icon: Terminal,
    action: () => {},
    category: 'Tools'
  },
  {
    id: 'open-logs',
    name: 'View Logs',
    description: 'View application logs',
    icon: FolderOpen,
    action: () => {},
    category: 'Tools'
  },
  {
    id: 'create-app',
    name: 'Create New App',
    description: 'Deploy a new application',
    icon: Plus,
    action: () => {},
    category: 'Actions'
  },
  {
    id: 'restart-app',
    name: 'Restart App',
    description: 'Restart a running application',
    icon: RefreshCw,
    action: () => {},
    category: 'Actions'
  },
  {
    id: 'deploy-app',
    name: 'Deploy App',
    description: 'Trigger a new deployment',
    icon: Play,
    action: () => {},
    category: 'Actions'
  },
  {
    id: 'show-shortcuts',
    name: 'Keyboard Shortcuts',
    description: 'View all available keyboard shortcuts',
    icon: Keyboard,
    action: () => {},
    category: 'Help'
  }
]

export function CommandPalette() {
  const [isOpen, setIsOpen] = useState(false)
  const [query, setQuery] = useState('')
  const [selectedIndex, setSelectedIndex] = useState(0)
  const inputRef = useRef<HTMLInputElement>(null)
  const router = useRouter()

  const filteredCommands = commands.filter(cmd =>
    cmd.name.toLowerCase().includes(query.toLowerCase()) ||
    cmd.description?.toLowerCase().includes(query.toLowerCase())
  )

  const handleSelect = useCallback((command: CommandItem) => {
    setIsOpen(false)
    setQuery('')
    
    // Map commands to actual routes
    switch (command.id) {
      case 'go-dashboard':
        router.push('/')
        break
      case 'go-apps':
        router.push('/apps')
        break
      case 'go-services':
        router.push('/services')
        break
      case 'go-server':
        router.push('/server')
        break
      case 'go-settings':
        router.push('/settings')
        break
      default:
        break
    }
  }, [router])

  // Keyboard shortcut (Cmd+K / Ctrl+K)
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if ((e.metaKey || e.ctrlKey) && e.key === 'k') {
        e.preventDefault()
        setIsOpen(true)
      }
      if (e.key === 'Escape') {
        setIsOpen(false)
      }
    }

    window.addEventListener('keydown', handleKeyDown)
    return () => window.removeEventListener('keydown', handleKeyDown)
  }, [])

  // Arrow navigation
  useEffect(() => {
    if (!isOpen) return

    const handleArrow = (e: KeyboardEvent) => {
      if (e.key === 'ArrowDown') {
        e.preventDefault()
        setSelectedIndex(i => Math.min(i + 1, filteredCommands.length - 1))
      }
      if (e.key === 'ArrowUp') {
        e.preventDefault()
        setSelectedIndex(i => Math.max(i - 1, 0))
      }
      if (e.key === 'Enter' && filteredCommands[selectedIndex]) {
        handleSelect(filteredCommands[selectedIndex])
      }
    }

    window.addEventListener('keydown', handleArrow)
    return () => window.removeEventListener('keydown', handleArrow)
  }, [isOpen, selectedIndex, filteredCommands, handleSelect])

  // Focus input when opened
  useEffect(() => {
    if (isOpen) {
      inputRef.current?.focus()
      setSelectedIndex(0)
    }
  }, [isOpen])

  if (!isOpen) return null

  return (
    <div className="fixed inset-0 z-50 flex items-start justify-center pt-[20vh]">
      {/* Backdrop */}
      <div
        className="absolute inset-0 bg-black/50 backdrop-blur-sm"
        onClick={() => setIsOpen(false)}
      />

      {/* Command Palette */}
      <div className="relative w-full max-w-xl bg-slate-800 border border-slate-700 rounded-lg shadow-2xl overflow-hidden">
        {/* Search Input */}
        <div className="flex items-center gap-3 px-4 py-3 border-b border-slate-700">
          <Search className="w-5 h-5 text-slate-400" />
          <input
            ref={inputRef}
            type="text"
            placeholder="Search commands..."
            value={query}
            onChange={(e) => {
              setQuery(e.target.value)
              setSelectedIndex(0)
            }}
            className="flex-1 bg-transparent text-white placeholder-slate-400 outline-none text-sm"
          />
          <div className="flex items-center gap-1 text-xs text-slate-500">
            <span className="px-1.5 py-0.5 bg-slate-700 rounded">
              {navigator.platform.includes('Mac') ? '⌘' : 'Ctrl'}
            </span>
            <span>K</span>
          </div>
        </div>

        {/* Results */}
        <div className="max-h-80 overflow-y-auto">
          {filteredCommands.length === 0 ? (
            <div className="px-4 py-8 text-center text-slate-400 text-sm">
              No commands found
            </div>
          ) : (
            <div className="py-2">
              {filteredCommands.map((command, index) => {
                const Icon = command.icon
                return (
                  <button
                    key={command.id}
                    onClick={() => handleSelect(command)}
                    className={`
                      w-full flex items-center gap-3 px-4 py-2 text-left transition-colors
                      ${index === selectedIndex ? 'bg-slate-700' : 'hover:bg-slate-700/50'}
                    `}
                  >
                    <div className="p-1.5 bg-slate-700 rounded">
                      <Icon className="w-4 h-4 text-slate-300" />
                    </div>
                    <div className="flex-1 min-w-0">
                      <p className="text-sm font-medium text-white truncate">
                        {command.name}
                      </p>
                      {command.description && (
                        <p className="text-xs text-slate-400 truncate">
                          {command.description}
                        </p>
                      )}
                    </div>
                    <span className="text-xs text-slate-500">
                      {command.category}
                    </span>
                  </button>
                )
              })}
            </div>
          )}
        </div>

        {/* Footer */}
        <div className="flex items-center justify-between px-4 py-2 border-t border-slate-700 text-xs text-slate-500">
          <div className="flex items-center gap-4">
            <span className="flex items-center gap-1">
              <span className="px-1 py-0.5 bg-slate-700 rounded">↑</span>
              <span className="px-1 py-0.5 bg-slate-700 rounded">↓</span>
              <span>to navigate</span>
            </span>
            <span className="flex items-center gap-1">
              <span className="px-1 py-0.5 bg-slate-700 rounded">↵</span>
              <span>to select</span>
            </span>
          </div>
          <span className="flex items-center gap-1">
            <span className="px-1 py-0.5 bg-slate-700 rounded">esc</span>
            <span>to close</span>
          </span>
        </div>
      </div>
    </div>
  )
}

// Export a hook to trigger the command palette
export function useCommandPalette() {
  const [isOpen, setIsOpen] = useState(false)

  const open = useCallback(() => setIsOpen(true), [])
  const close = useCallback(() => setIsOpen(false), [])

  return { isOpen, open, close }
}