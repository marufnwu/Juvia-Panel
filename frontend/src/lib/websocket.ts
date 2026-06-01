// WebSocket client for real-time updates
// Native browser WebSocket API (no Socket.IO per spec)

import type { ServerMetrics, WSDeploymentUpdate, WSNotification } from '@/types'

type MessageHandler = (data: unknown) => void
type ConnectionHandler = () => void
type MetricsHandler = (data: ServerMetrics) => void
type DeploymentHandler = (data: WSDeploymentUpdate) => void
type NotificationHandler = (data: WSNotification) => void

export type WebSocketMessageType = 
  | 'server.metrics'
  | 'app.deploy.started'
  | 'app.deploy.progress'
  | 'app.deploy.success'
  | 'app.deploy.failed'
  | 'app.status_changed'
  | 'app.logs'
  | 'service.metrics'
  | 'notification'

interface WebSocketMessage {
  type: string
  payload: unknown
}

interface Subscription {
  channel: string
  handler: MessageHandler
}

// Event payload types
interface ServerMetricsPayload extends ServerMetrics {
  timestamp: number
}

interface DeploymentPayload extends WSDeploymentUpdate {
  timestamp: number
}

interface AppLogsPayload {
  app_id: string
  logs: string
  stream: 'stdout' | 'stderr'
  timestamp: number
}

interface AppStatusPayload {
  app_id: string
  status: 'running' | 'stopped' | 'deploying' | 'failed'
  timestamp: number
}

function getDefaultWebSocketUrl(): string {
  if (typeof window !== 'undefined') {
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
    return `${protocol}//${window.location.host}/api/v1/stream`
  }
  return process.env.NEXT_PUBLIC_WS_URL || 'ws://127.0.0.1:9090/api/v1/stream'
}

class PanelWebSocket {
  private ws: WebSocket | null = null
  private url: string
  private handlers: Map<string, Set<MessageHandler>> = new Map()
  private onConnectHandlers: Set<ConnectionHandler> = new Set()
  private onDisconnectHandlers: Set<ConnectionHandler> = new Set()
  private reconnectAttempts = 0
  private maxReconnectAttempts = 5
  private reconnectDelay = 1000
  private shouldReconnect = true
  private isConnecting = false
  private subscriptions: Set<string> = new Set()

  constructor(url?: string) {
    this.url = url || getDefaultWebSocketUrl()
  }

  connect(): void {
    if (this.ws?.readyState === WebSocket.OPEN || this.isConnecting) {
      return
    }

    this.isConnecting = true

    try {
      // Browser WebSocket automatically sends httpOnly cookies with same-origin requests
      // No need to pass token in URL
      this.ws = new WebSocket(this.url)

      this.ws.onopen = () => {
        this.isConnecting = false
        this.reconnectAttempts = 0
        this.onConnectHandlers.forEach(handler => handler())
        
        // Resubscribe to channels after reconnect
        this.subscriptions.forEach(channel => {
          this.send('subscribe', { channel })
        })
      }

      this.ws.onclose = () => {
        this.isConnecting = false
        this.onDisconnectHandlers.forEach(handler => handler())

        if (this.shouldReconnect && this.reconnectAttempts < this.maxReconnectAttempts) {
          this.reconnectAttempts++
          const delay = this.reconnectDelay * Math.pow(2, this.reconnectAttempts - 1)
          setTimeout(() => this.connect(), delay)
        }
      }

      this.ws.onerror = (error) => {
        console.error('WebSocket error:', error)
      }

      this.ws.onmessage = (event) => {
        try {
          const message: WebSocketMessage = JSON.parse(event.data)
          
          // Handle the message based on type
          switch (message.type) {
            case 'server.metrics':
              this.handleServerMetrics(message.payload as ServerMetricsPayload)
              break
            case 'app.deploy.started':
            case 'app.deploy.progress':
            case 'app.deploy.success':
            case 'app.deploy.failed':
              this.handleDeploymentUpdate(message.type, message.payload as DeploymentPayload)
              break
            case 'app.logs':
              this.handleAppLogs(message.payload as AppLogsPayload)
              break
            case 'app.status_changed':
              this.handleAppStatusChange(message.payload as AppStatusPayload)
              break
            case 'notification':
              this.handleNotification(message.payload as WSNotification)
              break
            default:
              // Generic handler for other message types
              const handlers = this.handlers.get(message.type)
              if (handlers) {
                handlers.forEach(handler => handler(message.payload))
              }
          }
        } catch (e) {
          console.error('Failed to parse WebSocket message:', e)
        }
      }
    } catch (error) {
      this.isConnecting = false
      console.error('Failed to create WebSocket:', error)
    }
  }

  private handleServerMetrics(payload: ServerMetricsPayload): void {
    // Update all server metrics handlers
    const handlers = this.handlers.get('server.metrics')
    if (handlers) {
      handlers.forEach(handler => handler(payload))
    }
    
    // Also call specific metrics handlers if registered
    const metricsHandlers = this.handlers.get('metrics')
    if (metricsHandlers) {
      metricsHandlers.forEach(handler => handler(payload))
    }
  }

  private handleDeploymentUpdate(type: string, payload: DeploymentPayload): void {
    const handlers = this.handlers.get(type)
    if (handlers) {
      handlers.forEach(handler => handler(payload))
    }
    
    // Also call generic deployment handlers
    const deploymentHandlers = this.handlers.get('deployment')
    if (deploymentHandlers) {
      deploymentHandlers.forEach(handler => handler({ type, ...payload }))
    }
  }

  private handleAppLogs(payload: AppLogsPayload): void {
    const handlers = this.handlers.get('app.logs')
    if (handlers) {
      handlers.forEach(handler => handler(payload))
    }
    
    // Also call app-specific handlers if registered
    const appHandlers = this.handlers.get(`app.${payload.app_id}.logs`)
    if (appHandlers) {
      appHandlers.forEach(handler => handler(payload))
    }
  }

  private handleAppStatusChange(payload: AppStatusPayload): void {
    const handlers = this.handlers.get('app.status_changed')
    if (handlers) {
      handlers.forEach(handler => handler(payload))
    }
    
    // Also call app-specific handlers
    const appHandlers = this.handlers.get(`app.${payload.app_id}.status`)
    if (appHandlers) {
      appHandlers.forEach(handler => handler(payload))
    }
  }

  private handleNotification(payload: WSNotification): void {
    const handlers = this.handlers.get('notification')
    if (handlers) {
      handlers.forEach(handler => handler(payload))
    }
  }

  disconnect(): void {
    this.shouldReconnect = false
    if (this.ws) {
      this.ws.close()
      this.ws = null
    }
  }

  send(type: string, payload?: unknown): void {
    if (this.ws?.readyState === WebSocket.OPEN) {
      this.ws.send(JSON.stringify({ type, payload }))
    }
  }

  // Subscribe to a channel
  subscribe(channel: string): void {
    this.subscriptions.add(channel)
    this.send('subscribe', { channel })
  }

  // Unsubscribe from a channel
  unsubscribe(channel: string): void {
    this.subscriptions.delete(channel)
    this.send('unsubscribe', { channel })
  }

  // Subscribe to server metrics updates
  onServerMetrics(handler: MetricsHandler): () => void {
    return this.on('server.metrics', handler as MessageHandler)
  }

  // Subscribe to deployment updates for a specific app
  onAppDeployment(appId: string, handler: DeploymentHandler): () => void {
    return this.on(`app.deploy.${appId}`, handler as MessageHandler)
  }

  // Subscribe to app logs
  onAppLogs(appId: string, handler: (logs: string, stream: 'stdout' | 'stderr') => void): () => void {
    const wrappedHandler = (payload: unknown) => {
      const data = payload as AppLogsPayload
      handler(data.logs, data.stream)
    }
    return this.on(`app.${appId}.logs`, wrappedHandler)
  }

  // Subscribe to app status changes
  onAppStatus(appId: string, handler: (status: string) => void): () => void {
    const wrappedHandler = (payload: unknown) => {
      const data = payload as AppStatusPayload
      handler(data.status)
    }
    return this.on(`app.${appId}.status`, wrappedHandler)
  }

  // Subscribe to notifications
  onNotification(handler: NotificationHandler): () => void {
    return this.on('notification', handler as MessageHandler)
  }

  on(type: string, handler: MessageHandler): () => void {
    if (!this.handlers.has(type)) {
      this.handlers.set(type, new Set())
    }
    this.handlers.get(type)!.add(handler)

    // Return unsubscribe function
    return () => {
      this.handlers.get(type)?.delete(handler)
    }
  }

  onConnect(handler: ConnectionHandler): () => void {
    this.onConnectHandlers.add(handler)
    return () => {
      this.onConnectHandlers.delete(handler)
    }
  }

  onDisconnect(handler: ConnectionHandler): () => void {
    this.onDisconnectHandlers.add(handler)
    return () => {
      this.onDisconnectHandlers.delete(handler)
    }
  }

  isConnected(): boolean {
    return this.ws?.readyState === WebSocket.OPEN
  }
}

// Singleton instance
let wsInstance: PanelWebSocket | null = null

export function getWebSocket(): PanelWebSocket {
  if (!wsInstance) {
    wsInstance = new PanelWebSocket()
  }
  return wsInstance
}

// React Query integration helpers
export function useServerMetricsSubscription(
  onMetrics: (data: ServerMetrics) => void,
  enabled = true
): void {
  const ws = getWebSocket()

  useEffect(() => {
    if (!enabled) return

    // Connect if not connected
    ws.connect()

    // Subscribe to metrics
    const unsubscribe = ws.onServerMetrics(onMetrics as (data: ServerMetrics) => void)

    return () => {
      unsubscribe()
    }
  }, [enabled, onMetrics, ws])
}

// Hook for deployment notifications
export function useDeploymentNotifications(
  onDeployment: (data: { appId: string; status: string; deploymentId?: string }) => void,
  enabled = true
): void {
  const ws = getWebSocket()
  const { addToast } = useToastStore()

  useEffect(() => {
    if (!enabled) return

    ws.connect()

    const unsubStarted = ws.on('app.deploy.started', (payload) => {
      const data = payload as DeploymentPayload
      addToast({
        type: 'info',
        title: 'Deployment Started',
        message: `Deploying ${data.app_id}...`,
      })
      onDeployment({ appId: data.app_id, status: 'deploying', deploymentId: data.deployment_id })
    })

    const unsubProgress = ws.on('app.deploy.progress', (payload) => {
      const data = payload as DeploymentPayload
      onDeployment({ appId: data.app_id, status: 'deploying', deploymentId: data.deployment_id })
    })

    const unsubSuccess = ws.on('app.deploy.success', (payload) => {
      const data = payload as DeploymentPayload
      addToast({
        type: 'success',
        title: 'Deployment Successful',
        message: `${data.app_id} deployed successfully`,
      })
      onDeployment({ appId: data.app_id, status: 'running', deploymentId: data.deployment_id })
    })

    const unsubFailed = ws.on('app.deploy.failed', (payload) => {
      const data = payload as DeploymentPayload
      addToast({
        type: 'error',
        title: 'Deployment Failed',
        message: `${data.app_id} deployment failed`,
      })
      onDeployment({ appId: data.app_id, status: 'failed', deploymentId: data.deployment_id })
    })

    return () => {
      unsubStarted()
      unsubProgress()
      unsubSuccess()
      unsubFailed()
    }
  }, [enabled, onDeployment, addToast, ws])
}

// Import useToastStore from stores
import { useToastStore } from '@/stores'

// Import useEffect for React hooks
import { useEffect, useState, useCallback } from 'react'

// Hook for WebSocket connection state
export function useWebSocket() {
  const [isConnected, setIsConnected] = useState(false)
  const ws = getWebSocket()

  useEffect(() => {
    // Set initial state
    setIsConnected(ws.isConnected())

    // Subscribe to connect/disconnect events
    const unsubConnect = ws.onConnect(() => {
      setIsConnected(true)
    })

    const unsubDisconnect = ws.onDisconnect(() => {
      setIsConnected(false)
    })

    // Connect if not connected
    ws.connect()

    return () => {
      unsubConnect()
      unsubDisconnect()
    }
  }, [ws])

  const reconnect = useCallback(() => {
    ws.disconnect()
    setTimeout(() => ws.connect(), 500)
  }, [ws])

  return { isConnected, reconnect, ws }
}

export type { WebSocketMessage }
export default PanelWebSocket
