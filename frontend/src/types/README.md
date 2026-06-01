# Types

TypeScript type definitions for Server Panel.

## Overview

- `index.ts` - Main export, contains all shared types

## Type Categories

### User & Auth
- `User` - User entity with role
- `UserRole` - 'owner' | 'admin' | 'developer' | 'viewer'

### Apps
- `App` - Application entity
- `AppStatus` - 'running' | 'stopped' | 'deploying' | 'failed'
- `CreateAppInput` - Input for creating an app
- `Deployment` - Deployment record
- `DeploymentStatus` - 'pending' | 'building' | 'deploying' | 'success' | 'failed'

### Services
- `Service` - Database/service entity
- `ServiceType` - 'postgresql' | 'mysql' | 'redis' | 'mongodb' | 'minio' | 'custom'
- `ServiceStatus` - 'running' | 'stopped' | 'starting' | 'failed'
- `CreateServiceInput` - Input for creating a service
- `Backup` - Backup record

### Server
- `ServerMetrics` - CPU, RAM, disk metrics
- `ProcessInfo` - Process list item
- `DiskInfo` - Disk usage item
- `NetworkStats` - Network I/O stats

### Activity
- `ActivityEvent` - Activity log entry
- `ActivityResponse` - Paginated activity list

### WebSocket
- `WSDeploymentUpdate` - Real-time deployment update
- `WSMetricsUpdate` - Real-time metrics update
- `WSNotification` - Real-time notification

### UI
- `Toast` - Toast notification for UI