// TypeScript types for Server Panel

// User roles
export type UserRole = 'owner' | 'admin' | 'developer' | 'viewer'

// App statuses
export type AppStatus = 'running' | 'stopped' | 'deploying' | 'failed' | 'restarting'

// Service types
export type ServiceType = 'postgresql' | 'mysql' | 'redis' | 'mongodb' | 'minio' | 'custom'
export type ServiceStatus = 'running' | 'stopped' | 'starting' | 'failed'

// Deployment statuses
export type DeploymentStatus = 'queued' | 'in_progress' | 'success' | 'failed' | 'cancelled'

// Build strategies
export type BuildStrategy = 'auto' | 'nixpacks' | 'dockerfile' | 'static'

// User
export interface User {
  id: string
  email: string
  name: string
  role: UserRole
  created_at: string
  last_active: string
}

// App
export interface App {
  id: string
  name: string
  status: AppStatus
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

// Create app input
export interface CreateAppInput {
  name: string
  source: {
    type: 'git' | 'upload' | 'docker_compose'
    repo_url?: string
    branch?: string
    auto_deploy?: boolean
  }
  build?: {
    strategy?: BuildStrategy
    build_command?: string
    start_command?: string
  }
  domain?: {
    primary?: string
    force_https?: boolean
  }
  environment?: Record<string, string>
}

// Environment variable
export interface EnvVariable {
  key: string
  value: string
  secret: boolean
}

export interface EnvVariableInput {
  key: string
  value: string
  secret?: boolean
}

// Deployment
export interface Deployment {
  id: string
  app_id: string
  status: DeploymentStatus
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

// Service
export interface Service {
  id: string
  name: string
  type: ServiceType
  status: ServiceStatus
  host: string
  port: number
  database?: string
  username?: string
  size?: number
  connected_apps: string[]
  created_at: string
}

// Create service input
export interface CreateServiceInput {
  name: string
  type: ServiceType
  version?: string
}

// Backup
export interface Backup {
  id: string
  service_id: string
  status: 'completed' | 'failed' | 'in_progress'
  size?: number
  location?: string
  type: 'manual' | 'scheduled'
  created_at: string
}

// Server metrics
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

// Process info
export interface ProcessInfo {
  pid: number
  name: string
  cpu_percent: number
  mem_percent: number
  user: string
  time: string
}

// Disk info
export interface DiskInfo {
  mount: string
  used: number
  total: number
}

// Network stats
export interface NetworkStats {
  inbound: number
  outbound: number
}

// Activity event
export interface ActivityEvent {
  id: string
  type: string
  message: string
  target_type?: 'app' | 'service' | 'server'
  target_id?: string
  user?: string
  ip?: string
  created_at: string
}

// API responses
export interface AppsResponse {
  data: App[]
  meta: {
    total: number
    page: number
    per_page: number
    total_pages: number
  }
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

export interface ActivityResponse {
  events: ActivityEvent[]
  total: number
  page: number
  limit: number
}

// WebSocket message types
export interface WSDeploymentUpdate {
  app_id: string
  deployment_id: string
  status: DeploymentStatus
  logs?: string
}

export interface WSMetricsUpdate {
  cpu: { current_percent: number }
  memory: { current_mb: number; total_mb: number; percent: number }
  disk: { percent: number; total_gb: number; used_gb: number }
  network: { inbound_mbps: number; outbound_mbps: number }
}

export interface WSNotification {
  id: string
  type: 'success' | 'warning' | 'error' | 'info'
  title: string
  message: string
  created_at: string
}

// Toast notification
export interface Toast {
  id: string
  type: 'success' | 'warning' | 'error' | 'info' | 'loading' | 'progress'
  title: string
  message?: string
  duration?: number
  action?: {
    label: string
    onClick: () => void
  }
  undoAction?: () => void
  progress?: number
}