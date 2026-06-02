// API client for Server Panel
// Uses native fetch (no Axios per spec)

import { useAuthStore } from '@/stores'

const API_BASE = process.env.NEXT_PUBLIC_API_URL || '/api/v1'

interface RequestOptions extends RequestInit {
  params?: Record<string, string | number | undefined>
}

class ApiError extends Error {
  constructor(
    message: string,
    public status: number,
    public code?: string
  ) {
    super(message)
    this.name = 'ApiError'
  }
}

async function request<T>(
  endpoint: string,
  options: RequestOptions = {},
  retryCount = 0
): Promise<T> {
  const { params, ...fetchOptions } = options

  let url = `${API_BASE}${endpoint}`
  if (params) {
    const searchParams = new URLSearchParams()
    Object.entries(params).forEach(([key, value]) => {
      if (value !== undefined) {
        searchParams.append(key, String(value))
      }
    })
    url += `?${searchParams.toString()}`
  }

  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
    ...(fetchOptions.headers as Record<string, string> || {}),
  }

  const token = useAuthStore.getState().accessToken
  if (token) {
    headers['Authorization'] = `Bearer ${token}`
  }

  const response = await fetch(url, {
    ...fetchOptions,
    headers,
    credentials: 'include',
  })

  if (response.status === 401 && retryCount === 0) {
    const refreshed = await refreshToken()
    if (refreshed) {
      return request(endpoint, options, 1)
    }
  }

  if (!response.ok) {
    const error = await response.json().catch(() => ({ message: 'Request failed' }))
    throw new ApiError(error.message || 'Request failed', response.status, error.code)
  }

  const text = await response.text()
  if (!text) return {} as T
  return JSON.parse(text) as T
}

async function refreshToken(): Promise<boolean> {
  try {
    const response = await fetch(`${API_BASE}/auth/refresh`, {
      method: 'POST',
      credentials: 'include',
    })
    if (!response.ok) {
      useAuthStore.getState().clearAuth()
      // Redirect to login if we're in a browser and not already on auth pages
      if (typeof window !== 'undefined') {
        const path = window.location.pathname
        if (path !== '/login' && path !== '/setup') {
          window.location.href = '/login'
        }
      }
      return false
    }
    const data = await response.json()
    useAuthStore.getState().setAuth(data.access_token, useAuthStore.getState().user)
    return true
  } catch {
    useAuthStore.getState().clearAuth()
    if (typeof window !== 'undefined') {
      const path = window.location.pathname
      if (path !== '/login' && path !== '/setup') {
        window.location.href = '/login'
      }
    }
    return false
  }
}

export const api = {
  // Health check
  health: () => request<{ status: string }>('/health'),

  // Auth
  auth: {
    login: (username: string, password: string) =>
      request<{ access_token: string; user?: { id: number; username: string; email: string; role: string } }>('/auth/login', {
        method: 'POST',
        body: JSON.stringify({ username, password }),
      }),

    refresh: () =>
      request<{ access_token: string }>('/auth/refresh', {
        method: 'POST',
      }),

    logout: () =>
      request<void>('/auth/logout', {
        method: 'POST',
      }),

    register: (email: string, username: string, password: string) =>
      request<RegisterResponse>('/auth/register', {
        method: 'POST',
        body: JSON.stringify({ email, username, password }),
      }),

    status: () =>
      request<{ users_exist: boolean; count: number }>('/auth/status'),
  },

  // Users
  users: {
    me: () => request<User>('/users/me'),

    list: (params?: { page?: number; limit?: number }) =>
      request<User[]>('/users', { params }),

    invite: (email: string, role: string) =>
      request<{ id: string; invite_url?: string }>('/users/invite', {
        method: 'POST',
        body: JSON.stringify({ email, role }),
      }),

    delete: (id: string) =>
      request<void>(`/users/${id}`, {
        method: 'DELETE',
      }),

    updateRole: (userId: string, role: string) =>
      request<void>(`/users/${userId}/role`, {
        method: 'PUT',
        body: JSON.stringify({ role }),
      }),

    // API Keys
    listApiKeys: () =>
      request<{ data: ApiKey[] }>('/users/me/api-keys'),

    createApiKey: (data: { name: string; scopes: string; expires_in?: number }) =>
      request<ApiKeyResponse>('/users/me/api-keys', {
        method: 'POST',
        body: JSON.stringify(data),
      }),

    revokeApiKey: (id: string) =>
      request<void>(`/users/me/api-keys/${id}`, {
        method: 'DELETE',
      }),
  },

  // Apps
  apps: {
    list: (params?: { page?: number; per_page?: number; status?: string; search?: string }) =>
      request<AppsResponse>('/apps', { params }),

    get: (id: string) =>
      request<App>(`/apps/${id}`),

    create: (data: CreateAppInput) =>
      request<{ id: string; name: string; status: string; message: string; deployment_id: string }>('/apps', {
        method: 'POST',
        body: JSON.stringify(data),
      }),

    update: (id: string, data: Partial<App>) =>
      request<{ id: string; name: string; message: string; requires_restart: boolean }>(`/apps/${id}`, {
        method: 'PUT',
        body: JSON.stringify(data),
      }),

    delete: (id: string) =>
      request<void>(`/apps/${id}`, {
        method: 'DELETE',
      }),

    deploy: (id: string, branch?: string) =>
      request<{ deployment_id: string; status: string; message: string }>(`/apps/${id}/deploy`, {
        method: 'POST',
        body: JSON.stringify({ branch }),
      }),

    restart: (id: string) =>
      request<void>(`/apps/${id}/restart`, {
        method: 'POST',
      }),

    stop: (id: string) =>
      request<void>(`/apps/${id}/stop`, {
        method: 'POST',
      }),

    start: (id: string) =>
      request<void>(`/apps/${id}/start`, {
        method: 'POST',
      }),

    rollback: (id: string, deploymentId: string) =>
      request<{ message: string; new_deployment_id: string; target_deployment_id: string }>(`/apps/${id}/rollback`, {
        method: 'POST',
        body: JSON.stringify({ deployment_id: deploymentId }),
      }),

    // Environment variables
    getEnv: (id: string) =>
      request<{ app_id: string; variables: EnvVariable[] }>(`/apps/${id}/env`),

    updateEnv: (id: string, variables: EnvVariable[], deleteKeys: string[] = []) =>
      request<{ message: string; updated_count: number; deleted_count: number }>(`/apps/${id}/env`, {
        method: 'PUT',
        body: JSON.stringify({
          variables: variables.map(v => ({ key: v.key, value: v.value, is_secret: v.secret })),
          delete_keys: deleteKeys,
        }),
      }),

    // Deployments
    getDeployments: (id: string, params?: { page?: number; per_page?: number; status?: string }) =>
      request<{ data: Deployment[]; meta: { total: number; page: number; per_page: number; total_pages: number } }>(`/apps/${id}/deployments`, { params }),

    getDeploymentLogs: (appId: string, deploymentId: string) =>
      request<{ logs: string }>(`/apps/${appId}/deployments/${deploymentId}/logs`),

    // Logs
    getLogs: (id: string, params?: { stream?: string; tail?: number }) =>
      request<{ app_id: string; stream: string; lines: LogLine[] }>(`/apps/${id}/logs`, { params }),
  },

  // Services
  services: {
    list: (params?: { page?: number; per_page?: number; type?: string }) =>
      request<ServicesResponse>('/services', { params }),

    get: (id: string) =>
      request<Service>(`/services/${id}`),

    create: (data: CreateServiceInput) =>
      request<Service>('/services', {
        method: 'POST',
        body: JSON.stringify(data),
      }),

    delete: (id: string) =>
      request<void>(`/services/${id}`, {
        method: 'DELETE',
      }),

    restart: (id: string) =>
      request<void>(`/services/${id}/restart`, {
        method: 'POST',
      }),

    backup: (id: string) =>
      request<Backup>(`/services/${id}/backup`, {
        method: 'POST',
      }),

    getBackups: (id: string) =>
      request<Backup[]>(`/services/${id}/backups`),

    restoreBackup: (id: string, backupId: string) =>
      request<void>(`/services/${id}/backups/${backupId}/restore`, {
        method: 'POST',
      }),
  },

  // Server metrics
  server: {
    metrics: () =>
      request<ServerMetrics>('/server/metrics'),

    processes: () =>
      request<ProcessesResponse>('/server/processes'),

    diskUsage: () =>
      request<DisksResponse>('/server/disks'),

    networkStats: () =>
      request<NetworkStats>('/server/network'),
  },

  // Activity
  activity: {
    list: (params?: { page?: number; per_page?: number; type?: string }) =>
      request<ActivityResponse>('/activity', { params }),
  },

  // Notifications
  notifications: {
    list: (params?: { page?: number; per_page?: number; unread?: boolean }) => {
      const queryParams: Record<string, string | number | undefined> = {}
      if (params) {
        if (params.page !== undefined) queryParams.page = params.page
        if (params.per_page !== undefined) queryParams.per_page = params.per_page
        if (params.unread !== undefined) queryParams.unread = params.unread ? 'true' : 'false'
      }
      return request<NotificationResponse>('/notifications', { params: queryParams })
    },

    getUnreadCount: () =>
      request<{ unread_count: number }>('/notifications/unread-count'),

    markAsRead: (id: string) =>
      request<void>(`/notifications/${id}/read`, { method: 'POST' }),

    markAllAsRead: () =>
      request<void>('/notifications/read-all', { method: 'POST' }),

    delete: (id: string) =>
      request<void>(`/notifications/${id}`, { method: 'DELETE' }),
  },

  // Settings
  settings: {
    testEmail: () =>
      request<{ message: string }>('/settings/notifications/test/email', { method: 'POST' }),

    testWebhook: (url: string) =>
      request<{ message: string }>('/settings/notifications/test/webhook', {
        method: 'POST',
        body: JSON.stringify({ url }),
      }),

    exportData: () =>
      request<{ id: string; message: string }>('/settings/export', { method: 'POST' }),

    getExportStatus: (id: string) =>
      request<{ id: string; status: string; download_url?: string }>(`/settings/export/${id}`),

    downloadExport: (id: string) =>
      request<Blob>(`/settings/export/download/${id}`),
  },

  // Templates
  templates: {
    list: () =>
      request<Template[]>('/templates'),

    get: (id: string) =>
      request<Template>(`/templates/${id}`),
  },
}

// Types
export interface User {
  id: string
  email: string
  name: string
  role: 'owner' | 'admin' | 'developer' | 'viewer'
  created_at: string
  last_active: string
}

export interface RegisterResponse {
  access_token: string
  token_type: string
  expires_in: number
  refresh_token?: string
  user?: {
    id: number
    email: string
    username: string
    role: string
  }
}

export interface App {
  id: string
  name: string
  status: 'running' | 'stopped' | 'deploying' | 'failed' | 'restarting'
  health_status?: string
  runtime: string
  runtime_version?: string
  primary_domain?: string
  domains?: string[]
  source?: {
    type: string
    provider?: string
    repo_url?: string
    branch?: string
    auto_deploy?: boolean
    last_commit?: string
    last_commit_message?: string
    last_commit_author?: string
    last_commit_timestamp?: string
  }
  build_strategy?: string
  container_id?: string
  ports?: { internal: number; external?: number }
  env_count?: number
  volume_count?: number
  resource_usage?: {
    cpu_percent: number
    memory_mb: number
    memory_limit_mb: number
  }
  last_deployed_at?: string
  created_at: string
  updated_at: string
}

export interface AppsResponse {
  data: App[]
  meta: {
    total: number
    page: number
    per_page: number
    total_pages: number
  }
}

export interface CreateAppInput {
  name: string
  source: {
    type: 'git' | 'upload' | 'docker_compose'
    repo_url?: string
    branch?: string
    auto_deploy?: boolean
  }
  build?: {
    strategy?: 'auto' | 'nixpacks' | 'dockerfile' | 'static'
    build_command?: string
    start_command?: string
  }
  domain?: {
    primary?: string
    force_https?: boolean
  }
  environment?: Record<string, string>
}

export interface EnvVariable {
  key: string
  value: string
  secret: boolean
}

export interface Deployment {
  id: string
  app_id: string
  status: 'queued' | 'in_progress' | 'success' | 'failed' | 'cancelled'
  commit?: string
  commit_message?: string
  commit_author?: string
  branch?: string
  build_duration_seconds?: number
  deploy_duration_seconds?: number
  started_at?: string
  completed_at?: string
  triggered_by?: string
  triggered_by_user?: string
}

export interface Service {
  id: string
  name: string
  type: 'postgresql' | 'mysql' | 'redis' | 'mongodb' | 'minio' | 'custom'
  status: 'running' | 'stopped' | 'starting' | 'failed'
  host: string
  port: number
  database?: string
  username?: string
  size?: number
  connected_apps: string[]
  created_at: string
}

export interface ServicesResponse {
  data: Service[]
  meta: {
    total: number
    page: number
    per_page: number
    total_pages: number
  }
}

export interface CreateServiceInput {
  name: string
  type: string
  version?: string
}

export interface Backup {
  id: string
  service_id: string
  status: 'completed' | 'failed' | 'in_progress'
  size?: number
  location?: string
  type: 'manual' | 'scheduled'
  created_at: string
}

export interface ServerMetrics {
  cpu: {
    current_percent: number
    per_core: number[]
    history: { timestamp: string; value: number }[]
  }
  memory: {
    current_mb: number
    total_mb: number
    percent: number
    history: { timestamp: string; value: number }[]
  }
  disk: {
    percent: number
    total_gb: number
    used_gb: number
    io_read_mbps: number
    io_write_mbps: number
  }
  load: {
    '1min': number
    '5min': number
    '15min': number
  }
  network: {
    inbound_mbps: number
    outbound_mbps: number
    connections_active: number
  }
}

export interface ProcessInfo {
  pid: number
  command: string
  cpu: string
  mem: string
  user: string
}

export interface ProcessesResponse {
  processes: ProcessInfo[]
  total_count: number
}

export interface DiskInfo {
  filesystem: string
  mount: string
  used_gb: number
  total_gb: number
  free_gb: number
  percent: string
}

export interface DisksResponse {
  disks: DiskInfo[]
  largest_directories: Array<{ path: string; size: string }>
}

export interface NetworkStats {
  bandwidth_24h: {
    inbound_gb: number
    outbound_gb: number
  }
  interfaces: Array<{
    name: string
    ipv4: string
    ipv6: string
    mac: string
    state: string
  }>
  open_ports: Array<{
    port: number
    protocol: string
    service: string
  }>
}

export interface ActivityResponse {
  data: ActivityEvent[]
  meta: {
    total: number
    page: number
    per_page: number
    total_pages: number
  }
}

export interface ActivityEvent {
  id: string
  user_id?: number
  user_username: string
  action: string
  resource_type: string
  resource_id?: string
  details?: string
  ip_address?: string
  user_agent?: string
  created_at: string
}

export interface LogLine {
  timestamp: string
  stream: string
  message: string
}

export interface Notification {
  id: string
  user_id: number
  title: string
  message: string
  severity: 'info' | 'warning' | 'error' | 'success'
  link?: string
  read_at?: string
  created_at: string
}

export interface NotificationResponse {
  data: Notification[]
  meta: {
    total: number
    page: number
    per_page: number
    total_pages: number
  }
}

export interface ApiKey {
  id: string
  name: string
  scopes: string
  last_used_at: string | null
  expires_at: string | null
  created_at: string
  masked_token: string
}

export interface ApiKeyResponse {
  id: string
  name: string
  scopes: string
  token: string
  expires_at: string | null
  created_at: string
}

export interface TemplateVariable {
  key: string
  label: string
  description: string
  default: string
  required: boolean
}

export interface Template {
  id: string
  name: string
  description: string
  icon: string
  category: string
  runtimes: string[]
  docker_compose_url: string
  variables: TemplateVariable[]
}

export { ApiError }
export default api