'use client'

import { useEffect, useRef, useCallback, useState } from 'react'
import { Terminal as XTerminal } from 'xterm'
import { FitAddon } from '@xterm/addon-fit'

interface TerminalProps {
  target: string
  onConnect?: () => void
  onDisconnect?: () => void
}

export function Terminal({ target, onConnect, onDisconnect }: TerminalProps) {
  const terminalRef = useRef<HTMLDivElement>(null)
  const xtermRef = useRef<XTerminal | null>(null)
  const fitAddonRef = useRef<FitAddon | null>(null)
  const wsRef = useRef<WebSocket | null>(null)
  const [isConnected, setIsConnected] = useState(false)
  const reconnectTimeoutRef = useRef<NodeJS.Timeout | null>(null)

  const connect = useCallback(() => {
    if (!terminalRef.current) return

    // Clean up existing connection
    if (wsRef.current) {
      wsRef.current.close()
    }

    // Create terminal if not exists
    if (!xtermRef.current) {
      const term = new XTerminal({
        cursorBlink: true,
        fontSize: 14,
        fontFamily: '"JetBrains Mono", "Fira Code", "Consolas", monospace',
        theme: {
          background: '#0f172a',
          foreground: '#f8fafc',
          cursor: '#f8fafc',
          cursorAccent: '#0f172a',
          selectionBackground: '#334155',
          black: '#1e293b',
          red: '#ef4444',
          green: '#22c55e',
          yellow: '#eab308',
          blue: '#3b82f6',
          magenta: '#a855f7',
          cyan: '#06b6d4',
          white: '#f8fafc',
          brightBlack: '#64748b',
          brightRed: '#f87171',
          brightGreen: '#4ade80',
          brightYellow: '#facc15',
          brightBlue: '#60a5fa',
          brightMagenta: '#c084fc',
          brightCyan: '#22d3ee',
          brightWhite: '#ffffff',
        },
        cols: 80,
        rows: 24,
        scrollback: 10000,
      })

      const fitAddon = new FitAddon()
      term.loadAddon(fitAddon)

      term.open(terminalRef.current)
      fitAddon.fit()

      xtermRef.current = term
      fitAddonRef.current = fitAddon

      // Handle resize
      const resizeObserver = new ResizeObserver(() => {
        fitAddon.fit()
        if (wsRef.current?.readyState === WebSocket.OPEN) {
          wsRef.current.send(JSON.stringify({
            type: 'resize',
            cols: term.cols,
            rows: term.rows,
          }))
        }
      })
      resizeObserver.observe(terminalRef.current)

      // Handle terminal input
      term.onData((data) => {
        if (wsRef.current?.readyState === WebSocket.OPEN) {
          wsRef.current.send(JSON.stringify({
            type: 'input',
            data,
          }))
        }
      })
    }

    // Connect to ttyd WebSocket
    let wsUrl = process.env.NEXT_PUBLIC_TERMINAL_WS_URL
    if (!wsUrl && typeof window !== 'undefined') {
      const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
      wsUrl = `${protocol}//${window.location.host}/terminalws`
    }
    const ws = new WebSocket(`${wsUrl}?target=${encodeURIComponent(target)}`)

    ws.onopen = () => {
      setIsConnected(true)
      onConnect?.()

      // Send initial resize
      if (xtermRef.current) {
        ws.send(JSON.stringify({
          type: 'resize',
          cols: xtermRef.current.cols,
          rows: xtermRef.current.rows,
        }))
      }
    }

    ws.onclose = () => {
      setIsConnected(false)
      onDisconnect?.()

      // Auto-reconnect after 3 seconds
      reconnectTimeoutRef.current = setTimeout(() => {
        connect()
      }, 3000)
    }

    ws.onerror = (error) => {
      console.error('Terminal WebSocket error:', error)
    }

    ws.onmessage = (event) => {
      try {
        const message = JSON.parse(event.data)
        if (message.type === 'output' && xtermRef.current) {
          xtermRef.current.write(message.data)
        }
      } catch {
        // Treat as raw output if not JSON
        if (xtermRef.current) {
          xtermRef.current.write(event.data)
        }
      }
    }

    wsRef.current = ws
  }, [target, onConnect, onDisconnect])

  useEffect(() => {
    connect()

    return () => {
      if (reconnectTimeoutRef.current) {
        clearTimeout(reconnectTimeoutRef.current)
      }
      if (wsRef.current) {
        wsRef.current.close()
      }
      if (xtermRef.current) {
        xtermRef.current.dispose()
        xtermRef.current = null
      }
    }
  }, [connect])

  // Keyboard shortcuts
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if (!xtermRef.current) return

      // Ctrl+C - Copy selection or send SIGINT
      if (e.ctrlKey && e.key === 'c') {
        const selection = xtermRef.current.getSelection()
        if (selection) {
          navigator.clipboard.writeText(selection)
          xtermRef.current.clearSelection()
        } else {
          // Send Ctrl+C to terminal
          if (wsRef.current?.readyState === WebSocket.OPEN) {
            wsRef.current.send(JSON.stringify({ type: 'input', data: '\x03' }))
          }
        }
        e.preventDefault()
      }

      // Ctrl+V - Paste
      if (e.ctrlKey && e.key === 'v') {
        navigator.clipboard.readText().then((text) => {
          if (wsRef.current?.readyState === WebSocket.OPEN) {
            wsRef.current.send(JSON.stringify({ type: 'input', data: text }))
          }
        })
        e.preventDefault()
      }

      // Ctrl+Shift+C - Copy terminal text
      if (e.ctrlKey && e.shiftKey && e.key === 'C') {
        const selection = xtermRef.current.getSelection()
        if (selection) {
          navigator.clipboard.writeText(selection)
        }
        e.preventDefault()
      }

      // Ctrl+Shift+V - Paste into terminal
      if (e.ctrlKey && e.shiftKey && e.key === 'V') {
        navigator.clipboard.readText().then((text) => {
          if (wsRef.current?.readyState === WebSocket.OPEN) {
            wsRef.current.send(JSON.stringify({ type: 'input', data: text }))
          }
        })
        e.preventDefault()
      }
    }

    window.addEventListener('keydown', handleKeyDown)
    return () => window.removeEventListener('keydown', handleKeyDown)
  }, [])

  return (
    <div className="relative h-full w-full bg-slate-900 rounded-lg overflow-hidden">
      {/* Connection status indicator */}
      <div className="absolute top-2 right-2 z-10 flex items-center gap-2">
        <div className={`w-2 h-2 rounded-full ${isConnected ? 'bg-green-500' : 'bg-red-500'}`} />
        <span className="text-xs text-slate-400">
          {isConnected ? 'Connected' : 'Reconnecting...'}
        </span>
      </div>

      {/* Terminal container */}
      <div
        ref={terminalRef}
        className="h-full w-full p-2"
        style={{ minHeight: '400px' }}
      />
    </div>
  )
}
