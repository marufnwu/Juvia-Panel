'use client'

import { useState, useCallback, useEffect } from 'react'
import {
  Server,
  Cpu,
  HardDrive,
  Network,
  Shield,
  RefreshCw,
  Search,
  Trash2,
  AlertTriangle,
  CheckCircle,
  XCircle,
  Loader2,
  Plus,
  X,
  Zap,
  Clock,
  Activity
} from 'lucide-react'
import {
  PieChart,
  Pie,
  Cell,
  LineChart,
  Line,
  XAxis,
  YAxis,
  Tooltip,
  ResponsiveContainer
} from 'recharts'

type TabType = 'overview' | 'processes' | 'disks' | 'network' | 'updates' | 'firewall'

// Mock data
const mockMetrics = {
  cpu_percent: 34,
  cpu_cores: 4,
  ram_used: 4.8,
  ram_total: 8,
  disk_used: 45,
  disk_total: 100,
  load_avg: [0.45, 0.52, 0.38] as [number, number, number],
  uptime: 1234567,
  network_in: 2.4,
  network_out: 1.2,
}

const mockProcesses = [
  { pid: 1234, name: 'node', cpu_percent: 12, mem_percent: 8, user: 'app_001', time: '2d 04:12' },
  { pid: 5678, name: 'postgres', cpu_percent: 5, mem_percent: 15, user: 'postgres', time: '14d 00:00' },
  { pid: 9012, name: 'nginx', cpu_percent: 2, mem_percent: 1, user: 'www-data', time: '14d 00:00' },
  { pid: 3456, name: 'redis-server', cpu_percent: 1, mem_percent: 3, user: 'redis', time: '14d 00:00' },
  { pid: 7890, name: 'docker', cpu_percent: 1, mem_percent: 2, user: 'root', time: '14d 00:00' },
]

const mockDisks = [
  { mount: '/', used: 45, total: 100 },
  { mount: '/var/panel/apps', used: 20, total: 50 },
  { mount: '/var/panel/services', used: 10, total: 20 },
]

const mockNetwork = {
  bandwidth_in: 1024,
  bandwidth_out: 512,
  connections: [
    { local: '192.168.1.100:8080', remote: '10.0.0.1:443', state: 'ESTABLISHED' },
    { local: '192.168.1.100:8080', remote: '10.0.0.2:443', state: 'ESTABLISHED' },
  ],
  ports: [
    { port: 22, protocol: 'TCP', service: 'SSH', status: 'open' },
    { port: 80, protocol: 'TCP', service: 'HTTP', status: 'open' },
    { port: 443, protocol: 'TCP', service: 'HTTPS', status: 'open' },
    { port: 3000, protocol: 'TCP', service: 'api-prod', status: 'open' },
  ],
}

const mockUpdates = [
  { name: 'openssl', version: '3.0.2-0ubuntu1.12', type: 'security', status: 'pending' },
  { name: 'curl', version: '7.81.0-1ubuntu1.15', type: 'security', status: 'pending' },
  { name: 'git', version: '1:2.34.1-1ubuntu1.11', type: 'standard', status: 'pending' },
]

const mockFirewallRules = [
  { port: 22, protocol: 'TCP', source: 'Anywhere', action: 'ALLOW', app: 'SSH' },
  { port: 80, protocol: 'TCP', source: 'Anywhere', action: 'ALLOW', app: 'HTTP' },
  { port: 443, protocol: 'TCP', source: 'Anywhere', action: 'ALLOW', app: 'HTTPS' },
  { port: 3000, protocol: 'TCP', source: '10.0.0.0/8', action: 'ALLOW', app: 'api-prod' },
]

const mockRecentBlocks = [
  { time: '12:34:56', source: '192.168.1.45', port: 22, protocol: 'TCP', reason: 'Brute force' },
  { time: '11:22:33', source: '10.0.0.99', port: 3389, protocol: 'TCP', reason: 'Port scan' },
]

function formatUptime(seconds: number): string {
  const days = Math.floor(seconds / 86400)
  const hours = Math.floor((seconds % 86400) / 3600)
  const minutes = Math.floor((seconds % 3600) / 60)
  return `${days}d ${hours}h ${minutes}m`
}

function formatBytes(bytes: number): string {
  if (bytes < 1024) return bytes + ' B'
  if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(1) + ' KB'
  if (bytes < 1024 * 1024 * 1024) return (bytes / (1024 * 1024)).toFixed(1) + ' MB'
  return (bytes / (1024 * 1024 * 1024)).toFixed(1) + ' GB'
}

export default function ServerPage() {
  const [activeTab, setActiveTab] = useState<TabType>('overview')
  const [metrics, setMetrics] = useState(mockMetrics)
  const [processes, setProcesses] = useState(mockProcesses)
  const [disks] = useState(mockDisks)
  const [network] = useState(mockNetwork)
  const [updates] = useState(mockUpdates)
  const [firewallRules] = useState(mockFirewallRules)
  const [recentBlocks] = useState(mockRecentBlocks)
  const [isLoading, setIsLoading] = useState(false)
  const [searchQuery, setSearchQuery] = useState('')
  const [killConfirm, setKillConfirm] = useState<number | null>(null)
  const [timeRange, setTimeRange] = useState<'1h' | '6h' | '24h' | '7d'>('1h')
  const [showAddRule, setShowAddRule] = useState(false)

  const tabs = [
    { id: 'overview', label: 'Overview', icon: Activity },
    { id: 'processes', label: 'Processes', icon: Cpu },
    { id: 'disks', label: 'Disks', icon: HardDrive },
    { id: 'network', label: 'Network', icon: Network },
    { id: 'updates', label: 'Updates', icon: Zap },
    { id: 'firewall', label: 'Firewall', icon: Shield },
  ]

  const cpuData = [
    { name: 'Used', value: metrics.cpu_percent },
    { name: 'Free', value: 100 - metrics.cpu_percent },
  ]

  const ramData = [
    { name: 'Used', value: metrics.ram_used },
    { name: 'Free', value: metrics.ram_total - metrics.ram_used },
  ]

  const diskData = [
    { name: 'Used', value: metrics.disk_used },
    { name: 'Free', value: metrics.disk_total - metrics.disk_used },
  ]

  const filteredProcesses = processes.filter(p =>
    p.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
    p.user.toLowerCase().includes(searchQuery.toLowerCase()) ||
    p.pid.toString().includes(searchQuery)
  )

  const handleKillProcess = useCallback((pid: number) => {
    console.log('Killing process:', pid)
    setProcesses(prev => prev.filter(p => p.pid !== pid))
    setKillConfirm(null)
  }, [])

  const handleInstallUpdates = useCallback(async () => {
    setIsLoading(true)
    await new Promise(resolve => setTimeout(resolve, 2000))
    setIsLoading(false)
  }, [])

  return (
    <div className="min-h-screen bg-slate-900">
      {/* Header */}
      <div className="px-6 py-4 bg-slate-800 border-b border-slate-700">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-4">
            <div className="p-2 bg-primary-500/20 rounded-lg">
              <Server className="w-6 h-6 text-primary-500" />
            </div>
            <div>
              <h1 className="text-xl font-semibold text-white">Server</h1>
              <p className="text-sm text-slate-400">
                my-vps-01 • Ubuntu 24.04 LTS • {metrics.cpu_cores} CPU • {metrics.ram_total} GB RAM
              </p>
            </div>
          </div>
          
          <div className="flex items-center gap-4">
            <div className="text-right">
              <p className="text-sm text-slate-400">Uptime</p>
              <p className="text-white font-medium">{formatUptime(metrics.uptime)}</p>
            </div>
            <button
              onClick={() => setIsLoading(true)}
              className="p-2 text-slate-400 hover:text-white hover:bg-slate-700 rounded transition-colors"
            >
              <RefreshCw className={`w-5 h-5 ${isLoading ? 'animate-spin' : ''}`} />
            </button>
          </div>
        </div>
      </div>

      {/* Tabs */}
      <div className="px-6 bg-slate-800 border-b border-slate-700">
        <div className="flex items-center gap-1">
          {tabs.map(tab => {
            const Icon = tab.icon
            return (
              <button
                key={tab.id}
                onClick={() => setActiveTab(tab.id as TabType)}
                className={`
                  flex items-center gap-2 px-4 py-3 text-sm font-medium transition-colors border-b-2
                  ${activeTab === tab.id
                    ? 'text-white border-primary-500'
                    : 'text-slate-400 border-transparent hover:text-white'
                  }
                `}
              >
                <Icon className="w-4 h-4" />
                {tab.label}
              </button>
            )
          })}
        </div>
      </div>

      {/* Tab Content */}
      <div className="p-6">
        {activeTab === 'overview' && (
          <div className="space-y-6">
            {/* Resource Cards */}
            <div className="grid grid-cols-3 gap-6">
              {/* CPU */}
              <div className="bg-slate-800 rounded-lg border border-slate-700 p-6">
                <div className="flex items-center justify-between mb-4">
                  <h3 className="text-sm font-medium text-slate-400">CPU Usage</h3>
                  <div className="flex items-center gap-1">
                    {[1, 6, 24, 168].map(h => (
                      <button
                        key={h}
                        onClick={() => setTimeRange(h === 1 ? '1h' : h === 6 ? '6h' : h === 24 ? '24h' : '7d')}
                        className={`px-2 py-0.5 text-xs rounded ${
                          (h === 1 && timeRange === '1h') ||
                          (h === 6 && timeRange === '6h') ||
                          (h === 24 && timeRange === '24h') ||
                          (h === 168 && timeRange === '7d')
                            ? 'bg-primary-500/20 text-primary-400'
                            : 'text-slate-500 hover:text-white'
                        }`}
                      >
                        {h === 168 ? '7d' : h === 1 ? '1h' : `${h}h`}
                      </button>
                    ))}
                  </div>
                </div>
                <div className="flex items-center gap-6">
                  <ResponsiveContainer width={100} height={100}>
                    <PieChart>
                      <Pie
                        data={cpuData}
                        cx="50%"
                        cy="50%"
                        innerRadius={30}
                        outerRadius={45}
                        dataKey="value"
                      >
                        <Cell fill="#3b82f6" />
                        <Cell fill="#334155" />
                      </Pie>
                    </PieChart>
                  </ResponsiveContainer>
                  <div>
                    <div className="text-3xl font-semibold text-white">{metrics.cpu_percent}%</div>
                    <div className="text-sm text-slate-400">{metrics.cpu_cores} cores</div>
                  </div>
                </div>
              </div>

              {/* RAM */}
              <div className="bg-slate-800 rounded-lg border border-slate-700 p-6">
                <h3 className="text-sm font-medium text-slate-400 mb-4">Memory Usage</h3>
                <div className="flex items-center gap-6">
                  <ResponsiveContainer width={100} height={100}>
                    <PieChart>
                      <Pie
                        data={ramData}
                        cx="50%"
                        cy="50%"
                        innerRadius={30}
                        outerRadius={45}
                        dataKey="value"
                      >
                        <Cell fill="#22c55e" />
                        <Cell fill="#334155" />
                      </Pie>
                    </PieChart>
                  </ResponsiveContainer>
                  <div>
                    <div className="text-3xl font-semibold text-white">{metrics.ram_used.toFixed(1)} GB</div>
                    <div className="text-sm text-slate-400">of {metrics.ram_total} GB</div>
                  </div>
                </div>
              </div>

              {/* Disk */}
              <div className="bg-slate-800 rounded-lg border border-slate-700 p-6">
                <h3 className="text-sm font-medium text-slate-400 mb-4">Disk Usage</h3>
                <div className="flex items-center gap-6">
                  <ResponsiveContainer width={100} height={100}>
                    <PieChart>
                      <Pie
                        data={diskData}
                        cx="50%"
                        cy="50%"
                        innerRadius={30}
                        outerRadius={45}
                        dataKey="value"
                      >
                        <Cell fill="#eab308" />
                        <Cell fill="#334155" />
                      </Pie>
                    </PieChart>
                  </ResponsiveContainer>
                  <div>
                    <div className="text-3xl font-semibold text-white">{metrics.disk_used} GB</div>
                    <div className="text-sm text-slate-400">of {metrics.disk_total} GB</div>
                  </div>
                </div>
              </div>
            </div>

            {/* Load Average */}
            <div className="bg-slate-800 rounded-lg border border-slate-700 p-6">
              <h3 className="text-sm font-medium text-slate-400 mb-4">Load Average</h3>
              <div className="flex items-center gap-8">
                <div>
                  <p className="text-xs text-slate-500 mb-1">1 min</p>
                  <p className="text-xl font-semibold text-white">{metrics.load_avg[0].toFixed(2)}</p>
                </div>
                <div>
                  <p className="text-xs text-slate-500 mb-1">5 min</p>
                  <p className="text-xl font-semibold text-white">{metrics.load_avg[1].toFixed(2)}</p>
                </div>
                <div>
                  <p className="text-xs text-slate-500 mb-1">15 min</p>
                  <p className="text-xl font-semibold text-white">{metrics.load_avg[2].toFixed(2)}</p>
                </div>
              </div>
            </div>

            {/* Top Processes */}
            <div className="bg-slate-800 rounded-lg border border-slate-700 p-6">
              <h3 className="text-sm font-medium text-slate-400 mb-4">Top Processes by CPU</h3>
              <table className="w-full">
                <thead>
                  <tr className="text-left text-xs text-slate-500 border-b border-slate-700">
                    <th className="pb-2">PID</th>
                    <th className="pb-2">Name</th>
                    <th className="pb-2">CPU%</th>
                    <th className="pb-2">RAM%</th>
                    <th className="pb-2">User</th>
                    <th className="pb-2">Time</th>
                  </tr>
                </thead>
                <tbody>
                  {mockProcesses.slice(0, 5).map(proc => (
                    <tr key={proc.pid} className="border-b border-slate-700/50">
                      <td className="py-2 text-white font-mono">{proc.pid}</td>
                      <td className="py-2 text-white">{proc.name}</td>
                      <td className="py-2 text-amber-400">{proc.cpu_percent}%</td>
                      <td className="py-2 text-green-400">{proc.mem_percent}%</td>
                      <td className="py-2 text-slate-400">{proc.user}</td>
                      <td className="py-2 text-slate-400 font-mono">{proc.time}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </div>
        )}

        {activeTab === 'processes' && (
          <div className="space-y-4">
            <div className="flex items-center justify-between">
              <div className="relative">
                <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-slate-400" />
                <input
                  type="text"
                  placeholder="Search processes..."
                  value={searchQuery}
                  onChange={(e) => setSearchQuery(e.target.value)}
                  className="pl-9 pr-4 py-2 w-64 bg-slate-800 text-white rounded border border-slate-700 focus:outline-none focus:border-primary-500"
                />
              </div>
              <label className="flex items-center gap-2 text-sm text-slate-400">
                <input type="checkbox" className="rounded" />
                Auto-refresh
              </label>
            </div>

            <div className="bg-slate-800 rounded-lg border border-slate-700 overflow-hidden">
              <table className="w-full">
                <thead>
                  <tr className="text-left text-xs text-slate-500 border-b border-slate-700 bg-slate-800/50">
                    <th className="px-4 py-3 font-medium">PID</th>
                    <th className="px-4 py-3 font-medium">Name</th>
                    <th className="px-4 py-3 font-medium">CPU%</th>
                    <th className="px-4 py-3 font-medium">MEM%</th>
                    <th className="px-4 py-3 font-medium">User</th>
                    <th className="px-4 py-3 font-medium">Time</th>
                    <th className="px-4 py-3 font-medium"></th>
                  </tr>
                </thead>
                <tbody>
                  {filteredProcesses.map(proc => (
                    <tr key={proc.pid} className="border-b border-slate-700/50 hover:bg-slate-700/30">
                      <td className="px-4 py-3 text-white font-mono">{proc.pid}</td>
                      <td className="px-4 py-3 text-white">{proc.name}</td>
                      <td className="px-4 py-3">
                        <span className={proc.cpu_percent > 50 ? 'text-red-400' : 'text-amber-400'}>
                          {proc.cpu_percent}%
                        </span>
                      </td>
                      <td className="px-4 py-3 text-green-400">{proc.mem_percent}%</td>
                      <td className="px-4 py-3 text-slate-400">{proc.user}</td>
                      <td className="px-4 py-3 text-slate-400 font-mono">{proc.time}</td>
                      <td className="px-4 py-3">
                        <button
                          onClick={() => setKillConfirm(proc.pid)}
                          className="p-1 text-red-400 hover:text-red-300 hover:bg-red-500/10 rounded transition-colors"
                          title="Kill process"
                        >
                          <X className="w-4 h-4" />
                        </button>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>

            {/* Kill Confirmation Modal */}
            {killConfirm && (
              <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50">
                <div className="bg-slate-800 border border-slate-700 rounded-lg p-6 w-[400px] shadow-xl">
                  <div className="flex items-center gap-3 mb-4">
                    <div className="p-2 bg-red-500/20 rounded-lg">
                      <AlertTriangle className="w-6 h-6 text-red-500" />
                    </div>
                    <div>
                      <h3 className="text-lg font-medium text-white">Kill Process</h3>
                      <p className="text-sm text-slate-400">PID: {killConfirm}</p>
                    </div>
                  </div>
                  <p className="text-slate-300 mb-6">
                    Are you sure you want to kill this process? This may cause data loss.
                  </p>
                  <div className="flex justify-end gap-3">
                    <button
                      onClick={() => setKillConfirm(null)}
                      className="px-4 py-2 text-slate-300 hover:text-white hover:bg-slate-700 rounded transition-colors"
                    >
                      Cancel
                    </button>
                    <button
                      onClick={() => handleKillProcess(killConfirm)}
                      className="px-4 py-2 bg-red-600 hover:bg-red-700 text-white rounded transition-colors"
                    >
                      Kill Process
                    </button>
                  </div>
                </div>
              </div>
            )}
          </div>
        )}

        {activeTab === 'disks' && (
          <div className="space-y-6">
            <div className="grid gap-4">
              {disks.map(disk => (
                <div key={disk.mount} className="bg-slate-800 rounded-lg border border-slate-700 p-6">
                  <div className="flex items-center justify-between mb-4">
                    <h3 className="font-medium text-white">{disk.mount}</h3>
                    <span className="text-sm text-slate-400">
                      {disk.used} GB / {disk.total} GB
                    </span>
                  </div>
                  <div className="h-2 bg-slate-700 rounded-full overflow-hidden">
                    <div
                      className={`h-full transition-all ${
                        (disk.used / disk.total) > 0.9 ? 'bg-red-500' :
                        (disk.used / disk.total) > 0.7 ? 'bg-amber-500' : 'bg-green-500'
                      }`}
                      style={{ width: `${(disk.used / disk.total) * 100}%` }}
                    />
                  </div>
                </div>
              ))}
            </div>

            <div className="bg-slate-800 rounded-lg border border-slate-700 p-6">
              <h3 className="font-medium text-white mb-4">Cleanup Suggestions</h3>
              <div className="space-y-3">
                <button className="w-full flex items-center justify-between p-3 bg-slate-700/50 hover:bg-slate-700 rounded transition-colors">
                  <div className="flex items-center gap-3">
                    <Trash2 className="w-4 h-4 text-slate-400" />
                    <span className="text-white">Docker system prune</span>
                  </div>
                  <span className="text-sm text-slate-400">Free ~2.3 GB</span>
                </button>
                <button className="w-full flex items-center justify-between p-3 bg-slate-700/50 hover:bg-slate-700 rounded transition-colors">
                  <div className="flex items-center gap-3">
                    <Trash2 className="w-4 h-4 text-slate-400" />
                    <span className="text-white">Clear old logs</span>
                  </div>
                  <span className="text-sm text-slate-400">Free ~500 MB</span>
                </button>
              </div>
            </div>
          </div>
        )}

        {activeTab === 'network' && (
          <div className="space-y-6">
            <div className="grid grid-cols-2 gap-6">
              <div className="bg-slate-800 rounded-lg border border-slate-700 p-6">
                <h3 className="text-sm font-medium text-slate-400 mb-4">Bandwidth</h3>
                <div className="flex items-center gap-8">
                  <div>
                    <p className="text-xs text-slate-500 mb-1">Inbound</p>
                    <p className="text-2xl font-semibold text-green-400">{formatBytes(network.bandwidth_in)}/s</p>
                  </div>
                  <div>
                    <p className="text-xs text-slate-500 mb-1">Outbound</p>
                    <p className="text-2xl font-semibold text-blue-400">{formatBytes(network.bandwidth_out)}/s</p>
                  </div>
                </div>
              </div>
              <div className="bg-slate-800 rounded-lg border border-slate-700 p-6">
                <h3 className="text-sm font-medium text-slate-400 mb-4">Open Ports</h3>
                <div className="space-y-2">
                  {mockNetwork.ports.slice(0, 4).map(port => (
                    <div key={port.port} className="flex items-center justify-between">
                      <span className="text-white">{port.port}/{port.protocol}</span>
                      <span className="text-sm text-slate-400">{port.service}</span>
                    </div>
                  ))}
                </div>
              </div>
            </div>

            <div className="bg-slate-800 rounded-lg border border-slate-700 p-6">
              <h3 className="text-sm font-medium text-slate-400 mb-4">Active Connections</h3>
              <table className="w-full">
                <thead>
                  <tr className="text-left text-xs text-slate-500 border-b border-slate-700">
                    <th className="pb-2">Local</th>
                    <th className="pb-2">Remote</th>
                    <th className="pb-2">State</th>
                  </tr>
                </thead>
                <tbody>
                  {mockNetwork.connections.map((conn, i) => (
                    <tr key={i} className="border-b border-slate-700/50">
                      <td className="py-2 text-white font-mono text-sm">{conn.local}</td>
                      <td className="py-2 text-white font-mono text-sm">{conn.remote}</td>
                      <td className="py-2">
                        <span className="px-2 py-0.5 text-xs bg-green-500/20 text-green-400 rounded">
                          {conn.state}
                        </span>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </div>
        )}

        {activeTab === 'updates' && (
          <div className="space-y-6">
            <div className="bg-slate-800 rounded-lg border border-slate-700 p-6">
              <div className="flex items-center justify-between mb-6">
                <div>
                  <h3 className="font-medium text-white">Available Updates</h3>
                  <p className="text-sm text-slate-400">{updates.length} updates available</p>
                </div>
                <button
                  onClick={handleInstallUpdates}
                  disabled={isLoading}
                  className="flex items-center gap-2 px-4 py-2 bg-primary-600 hover:bg-primary-700 text-white rounded-lg transition-colors disabled:opacity-50"
                >
                  {isLoading ? (
                    <Loader2 className="w-4 h-4 animate-spin" />
                  ) : (
                    <Zap className="w-4 h-4" />
                  )}
                  Install Security Updates
                </button>
              </div>

              <table className="w-full">
                <thead>
                  <tr className="text-left text-xs text-slate-500 border-b border-slate-700">
                    <th className="pb-2">Package</th>
                    <th className="pb-2">Version</th>
                    <th className="pb-2">Type</th>
                    <th className="pb-2">Status</th>
                  </tr>
                </thead>
                <tbody>
                  {updates.map(update => (
                    <tr key={update.name} className="border-b border-slate-700/50">
                      <td className="py-3 text-white">{update.name}</td>
                      <td className="py-3 text-slate-400 font-mono text-sm">{update.version}</td>
                      <td className="py-3">
                        <span className={`px-2 py-0.5 text-xs rounded ${
                          update.type === 'security'
                            ? 'bg-red-500/20 text-red-400'
                            : 'bg-blue-500/20 text-blue-400'
                        }`}>
                          {update.type}
                        </span>
                      </td>
                      <td className="py-3">
                        <span className="px-2 py-0.5 text-xs bg-amber-500/20 text-amber-400 rounded">
                          {update.status}
                        </span>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>

            <div className="bg-slate-800 rounded-lg border border-slate-700 p-6">
              <div className="flex items-center gap-3 mb-4">
                <CheckCircle className="w-5 h-5 text-green-500" />
                <div>
                  <h3 className="font-medium text-white">Panel is up to date</h3>
                  <p className="text-sm text-slate-400">Version 1.0.0</p>
                </div>
              </div>
            </div>
          </div>
        )}

        {activeTab === 'firewall' && (
          <div className="space-y-6">
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-3">
                <div className="flex items-center gap-2">
                  <CheckCircle className="w-5 h-5 text-green-500" />
                  <span className="text-white font-medium">Firewall Active (UFW)</span>
                </div>
                <button className="px-3 py-1 text-sm text-red-400 hover:text-red-300 hover:bg-red-500/10 rounded transition-colors">
                  Disable Firewall
                </button>
              </div>
              <button
                onClick={() => setShowAddRule(true)}
                className="flex items-center gap-2 px-4 py-2 bg-primary-600 hover:bg-primary-700 text-white rounded-lg transition-colors"
              >
                <Plus className="w-4 h-4" />
                Add Rule
              </button>
            </div>

            <div className="bg-slate-800 rounded-lg border border-slate-700 overflow-hidden">
              <table className="w-full">
                <thead>
                  <tr className="text-left text-xs text-slate-500 border-b border-slate-700 bg-slate-800/50">
                    <th className="px-4 py-3 font-medium">Port</th>
                    <th className="px-4 py-3 font-medium">Protocol</th>
                    <th className="px-4 py-3 font-medium">Source</th>
                    <th className="px-4 py-3 font-medium">Action</th>
                    <th className="px-4 py-3 font-medium">App</th>
                    <th className="px-4 py-3 font-medium"></th>
                  </tr>
                </thead>
                <tbody>
                  {firewallRules.map((rule, i) => (
                    <tr key={i} className="border-b border-slate-700/50 hover:bg-slate-700/30">
                      <td className="px-4 py-3 text-white font-mono">{rule.port}</td>
                      <td className="px-4 py-3 text-slate-400">{rule.protocol}</td>
                      <td className="px-4 py-3 text-slate-400">{rule.source}</td>
                      <td className="px-4 py-3">
                        <span className="px-2 py-0.5 text-xs bg-green-500/20 text-green-400 rounded">
                          {rule.action}
                        </span>
                      </td>
                      <td className="px-4 py-3 text-slate-400">{rule.app}</td>
                      <td className="px-4 py-3">
                        <button className="p-1 text-slate-400 hover:text-red-400 hover:bg-red-500/10 rounded transition-colors">
                          <X className="w-4 h-4" />
                        </button>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>

            <div className="bg-slate-800 rounded-lg border border-slate-700 p-6">
              <h3 className="font-medium text-white mb-4">Recent Blocks (Last 24h)</h3>
              <table className="w-full">
                <thead>
                  <tr className="text-left text-xs text-slate-500 border-b border-slate-700">
                    <th className="pb-2">Time</th>
                    <th className="pb-2">Source IP</th>
                    <th className="pb-2">Port</th>
                    <th className="pb-2">Protocol</th>
                    <th className="pb-2">Reason</th>
                  </tr>
                </thead>
                <tbody>
                  {recentBlocks.map((block, i) => (
                    <tr key={i} className="border-b border-slate-700/50">
                      <td className="py-2 text-white font-mono">{block.time}</td>
                      <td className="py-2 text-red-400 font-mono">{block.source}</td>
                      <td className="py-2 text-white">{block.port}</td>
                      <td className="py-2 text-slate-400">{block.protocol}</td>
                      <td className="py-2 text-amber-400">{block.reason}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>

            {/* Add Rule Modal */}
            {showAddRule && (
              <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50">
                <div className="bg-slate-800 border border-slate-700 rounded-lg w-[450px] shadow-xl">
                  <div className="flex items-center justify-between px-6 py-4 border-b border-slate-700">
                    <h2 className="text-lg font-semibold text-white">Add Firewall Rule</h2>
                    <button
                      onClick={() => setShowAddRule(false)}
                      className="text-slate-400 hover:text-white"
                    >
                      <X className="w-5 h-5" />
                    </button>
                  </div>
                  <div className="p-6 space-y-4">
                    <div className="grid grid-cols-2 gap-4">
                      <div>
                        <label className="block text-sm text-slate-400 mb-1">Port</label>
                        <input
                          type="text"
                          placeholder="3000"
                          className="w-full px-3 py-2 bg-slate-700 text-white rounded border border-slate-600 focus:outline-none focus:border-primary-500"
                        />
                      </div>
                      <div>
                        <label className="block text-sm text-slate-400 mb-1">Protocol</label>
                        <select className="w-full px-3 py-2 bg-slate-700 text-white rounded border border-slate-600 focus:outline-none focus:border-primary-500">
                          <option>TCP</option>
                          <option>UDP</option>
                          <option>TCP/UDP</option>
                        </select>
                      </div>
                    </div>
                    <div>
                      <label className="block text-sm text-slate-400 mb-1">Source</label>
                      <select className="w-full px-3 py-2 bg-slate-700 text-white rounded border border-slate-600 focus:outline-none focus:border-primary-500">
                        <option>Anywhere</option>
                        <option>My IP</option>
                        <option>Custom</option>
                      </select>
                    </div>
                    <div>
                      <label className="block text-sm text-slate-400 mb-1">Action</label>
                      <select className="w-full px-3 py-2 bg-slate-700 text-white rounded border border-slate-600 focus:outline-none focus:border-primary-500">
                        <option>Allow</option>
                        <option>Deny</option>
                      </select>
                    </div>
                    <div>
                      <label className="block text-sm text-slate-400 mb-1">Description (optional)</label>
                      <input
                        type="text"
                        placeholder="API internal access"
                        className="w-full px-3 py-2 bg-slate-700 text-white rounded border border-slate-600 focus:outline-none focus:border-primary-500"
                      />
                    </div>
                  </div>
                  <div className="flex justify-end gap-3 px-6 py-4 border-t border-slate-700">
                    <button
                      onClick={() => setShowAddRule(false)}
                      className="px-4 py-2 text-slate-300 hover:text-white hover:bg-slate-700 rounded transition-colors"
                    >
                      Cancel
                    </button>
                    <button
                      onClick={() => setShowAddRule(false)}
                      className="px-4 py-2 bg-primary-600 hover:bg-primary-700 text-white rounded transition-colors"
                    >
                      Add Rule
                    </button>
                  </div>
                </div>
              </div>
            )}
          </div>
        )}
      </div>
    </div>
  )
}
