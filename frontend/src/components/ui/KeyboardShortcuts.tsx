'use client'

import { useState, useEffect, useCallback } from 'react'
import { X, Search, Command, Keyboard } from 'lucide-react'
import { useCommandPaletteStore } from '@/stores'

interface ShortcutCategory {
  title: string
  shortcuts: {
    keys: string[]
    description: string
  }[]
}

const shortcutCategories: ShortcutCategory[] = [
  {
    title: 'Global',
    shortcuts: [
      { keys: ['Cmd', 'K'], description: 'Open command palette' },
      { keys: ['Cmd', '/'], description: 'Show keyboard shortcuts' },
      { keys: ['Esc'], description: 'Close modal / drawer / dropdown' },
      { keys: ['Cmd', '1'], description: 'Go to Dashboard' },
      { keys: ['Cmd', '2'], description: 'Go to Apps' },
      { keys: ['Cmd', '3'], description: 'Go to Services' },
      { keys: ['Cmd', '4'], description: 'Go to Server' },
      { keys: ['Cmd', 'T'], description: 'Open Terminal' },
      { keys: ['Cmd', 'N'], description: 'Create new app' },
      { keys: ['Cmd', 'B'], description: 'Toggle sidebar' },
    ],
  },
  {
    title: 'App Detail',
    shortcuts: [
      { keys: ['D'], description: 'Deploy app' },
      { keys: ['R'], description: 'Restart app' },
      { keys: ['L'], description: 'Focus logs tab' },
      { keys: ['E'], description: 'Focus environment tab' },
      { keys: ['S'], description: 'Focus settings tab' },
      { keys: ['Shift', 'D'], description: 'Open deploy modal' },
    ],
  },
  {
    title: 'Terminal',
    shortcuts: [
      { keys: ['Cmd', 'C'], description: 'Copy selection' },
      { keys: ['Cmd', 'V'], description: 'Paste' },
      { keys: ['Cmd', 'Shift', 'C'], description: 'Copy terminal text' },
      { keys: ['Cmd', 'Shift', 'V'], description: 'Paste into terminal' },
      { keys: ['Cmd', 'T'], description: 'New terminal tab' },
      { keys: ['Cmd', 'W'], description: 'Close terminal tab' },
      { keys: ['F11'], description: 'Toggle full screen' },
    ],
  },
]

interface KeyboardShortcutsProps {
  open?: boolean
  onClose?: () => void
}

export function KeyboardShortcuts({ open = true, onClose }: KeyboardShortcutsProps) {
  const [search, setSearch] = useState('')
  const [isOpen, setIsOpen] = useState(open)
  const store = useCommandPaletteStore()

  // Handle keyboard shortcut to open
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      // Cmd/Ctrl + / to open shortcuts modal
      if ((e.metaKey || e.ctrlKey) && e.key === '/') {
        e.preventDefault()
        setIsOpen(true)
      }
      
      // Cmd/Ctrl + K to open command palette
      if ((e.metaKey || e.ctrlKey) && e.key === 'k') {
        e.preventDefault()
        setIsOpen(false)
        store.open()
      }
    }

    window.addEventListener('keydown', handleKeyDown)
    return () => window.removeEventListener('keydown', handleKeyDown)
  }, [store])

  // Filter shortcuts by search
  const filteredCategories = shortcutCategories.map(category => ({
    ...category,
    shortcuts: category.shortcuts.filter(shortcut =>
      shortcut.description.toLowerCase().includes(search.toLowerCase())
    ),
  })).filter(category => category.shortcuts.length > 0)

  const handleClose = () => {
    setIsOpen(false)
    onClose?.()
  }

  if (!isOpen) return null

  return (
    <div className="fixed inset-0 z-50 flex items-start justify-center pt-20">
      {/* Backdrop */}
      <div 
        className="absolute inset-0 bg-black/50" 
        onClick={handleClose}
      />

      {/* Modal */}
      <div className="relative bg-slate-800 border border-slate-700 rounded-xl shadow-2xl w-full max-w-2xl mx-4 max-h-[60vh] overflow-hidden flex flex-col">
        {/* Header */}
        <div className="flex items-center justify-between px-6 py-4 border-b border-slate-700">
          <div className="flex items-center gap-3">
            <Keyboard className="w-5 h-5 text-slate-400" />
            <h2 className="text-lg font-semibold text-white">Keyboard Shortcuts</h2>
          </div>
          <button
            onClick={handleClose}
            className="p-1 text-slate-400 hover:text-white hover:bg-slate-700 rounded transition-colors"
          >
            <X className="w-5 h-5" />
          </button>
        </div>

        {/* Search */}
        <div className="px-6 py-3 border-b border-slate-700">
          <div className="relative">
            <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-slate-400" />
            <input
              type="text"
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              placeholder="Search shortcuts..."
              className="w-full pl-10 pr-4 py-2 bg-slate-900 border border-slate-700 rounded-md text-white placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-primary-500"
              autoFocus
            />
          </div>
        </div>

        {/* Shortcuts List */}
        <div className="flex-1 overflow-y-auto p-6">
          <div className="space-y-6">
            {filteredCategories.map((category) => (
              <div key={category.title}>
                <h3 className="text-sm font-medium text-slate-400 uppercase tracking-wider mb-3">
                  {category.title}
                </h3>
                <div className="space-y-2">
                  {category.shortcuts.map((shortcut, idx) => (
                    <div
                      key={idx}
                      className="flex items-center justify-between py-2 px-3 rounded-lg hover:bg-slate-700/50"
                    >
                      <span className="text-sm text-slate-300">{shortcut.description}</span>
                      <div className="flex items-center gap-1">
                        {shortcut.keys.map((key, keyIdx) => (
                          <span key={keyIdx}>
                            <kbd className="px-2 py-1 bg-slate-700 border border-slate-600 rounded text-xs text-white font-mono">
                              {key}
                            </kbd>
                            {keyIdx < shortcut.keys.length - 1 && (
                              <span className="mx-1 text-slate-500">+</span>
                            )}
                          </span>
                        ))}
                      </div>
                    </div>
                  ))}
                </div>
              </div>
            ))}

            {filteredCategories.length === 0 && (
              <div className="text-center py-8">
                <p className="text-slate-400">No shortcuts found for &quot;{search}&quot;</p>
              </div>
            )}
          </div>
        </div>

        {/* Footer */}
        <div className="px-6 py-3 border-t border-slate-700 bg-slate-800/50">
          <p className="text-xs text-slate-500 text-center">
            Press <kbd className="px-1.5 py-0.5 bg-slate-700 rounded">Esc</kbd> to close
          </p>
        </div>
      </div>
    </div>
  )
}

export default KeyboardShortcuts