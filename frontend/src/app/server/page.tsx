'use client'

import { useState, useCallback } from 'react'
import { useQuery } from '@tanstack/react-query'
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

import { api, ServerMetrics, ProcessesResponse, DisksResponse, NetworkStats } from '@/lib/api'

export default function ServerPage() {
  const [activeTab, setActiveTab] = useState<TabType>('overview')
  const [searchQuery, setSearchQuery] = useState('')
  const [killConfirm, setKillConfirm] = useState<number | null>(null)
  const [timeRange, setTimeRange] = useState<'1h' | '6h' | '24h' | '7d'>('1h')
  const [showAddRule, setShowAddRule] = useState(false)

  const { data: metrics } = useQuery<ServerMetrics>({
    queryKey: ['server-metrics'],
    queryFn: () => api.server.metrics(),
    refetchInterval: 30000,
  })

  const { data: processesData } = useQuery<ProcessesResponse>({
    queryKey: ['server-processes'],
    queryFn: () => api.server.processes(),
    refetchInterval: 10000,
  })

  const { data: disksData } = useQuery<DisksResponse>({
    queryKey: ['server-disks'],
    queryFn: () => api.server.diskUsage(),
  })

  const { data: networkStats } = useQuery<NetworkStats>({
    queryKey: ['server-network'],
    queryFn: () => api.server.networkStats(),
    refetchInterval: 30000,
  })

  const processes = processesData?.processes || []
  const disks = disksData?.disks || []

  const tabs = [
    { id: 'overview', label: 'Overview', icon: Activity },
    { id: 'processes', label: 'Processes', icon: Cpu },
    { id: 'disks', label: 'Disks', icon: HardDrive },
    { id: 'network', label: 'Network', icon: Network },
    { id: 'updates', label: 'Updates', icon: Zap },
    { id: 'firewall', label: 'Firewall', icon: Shield },
  ]

  const cpuPercent = metrics?.cpu?.current_percent ?? 0
  const cpuCores = metrics?.cpu?.per_core?.length ?? 0
  const ramUsed = metrics?.memory?.current_mb ?? 0
  const ramTotal = metrics?.memory?.total_mb ?? 0
  const diskUsed = metrics?.disk?.used_gb ?? 0
  const diskTotal = metrics?.disk?.total_gb ?? 0

  const cpuData = [
    { name: 'Used', value: cpuPercent },
    { name: 'Free', value: 100 - cpuPercent },
  ]

  const ramData = [
    { name: 'Used', value: ramUsed / 1024 },
    { name: 'Free', value: (ramTotal - ramUsed) / 1024 },
  ]

  const diskData = [
    { name: 'Used', value: diskUsed },
    { name: 'Free', value: diskTotal - diskUsed },
  ]

  const filteredProcesses = processes.filter(p =>
    p.command.toLowerCase().includes(searchQuery.toLowerCase()) ||
    p.user.toLowerCase().includes(searchQuery.toLowerCase()) ||
    p.pid.toString().includes(searchQuery)
  )

  const handleKillProcess = useCallback((pid: number) => {
    console.log('Killing process:', pid)
    setKillConfirm(null)
  }, [])

  const handleRefresh = useCallback(() => {
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
                my-vps-01 • Ubuntu 24.04 LTS • {cpuCores} CPU • {(ramTotal / 1024).toFixed(0)} GB RAM
              </p>
            </div>
          </div>

          <div className="flex items-center gap-4">
            <div className="text-right">
              <p className="text-sm text-slate-400">Uptime</p>
              <p className="text-white font-medium">-</p>
            </div>
            <button
              onClick={handleRefresh}
              className="p-2 text-slate-400 hover:text-white hover:bg-slate-700 rounded transition-colors"
            >
              <RefreshCw className="w-5 h-5" />
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
                    <div className="text-3xl font-semibold text-white">{cpuPercent}%</div>
                    <div className="text-sm text-slate-400">{cpuCores} cores</div>
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
                    <div className="text-3xl font-semibold text-white">{(ramUsed / 1024).toFixed(1)} GB</div>
                    <div className="text-sm text-slate-400">of {(ramTotal / 1024).toFixed(0)} GB</div>
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
                    <div className="text-3xl font-semibold text-white">{diskUsed.toFixed(0)} GB</div>
                    <div className="text-sm text-slate-400">of {diskTotal.toFixed(0)} GB</div>
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
                  <p className="text-xl font-semibold text-white">{(metrics?.load?.['1min'] ?? 0).toFixed(2)}</p>
                </div>
                <div>
                  <p className="text-xs text-slate-500 mb-1">5 min</p>
                  <p className="text-xl font-semibold text-white">{(metrics?.load?.['5min'] ?? 0).toFixed(2)}</p>
                </div>
                <div>
                  <p className="text-xs text-slate-500 mb-1">15 min</p>
                  <p className="text-xl font-semibold text-white">{(metrics?.load?.['15min'] ?? 0).toFixed(2)}</p>
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
                  {processes.slice(0, 5).map(proc => (
                    <tr key={proc.pid} className="border-b border-slate-700/50">
                      <td className="py-2 text-white font-mono">{proc.pid}</td>
                      <td className="py-2 text-white">{proc.command}</td>
                      <td className="py-2 text-amber-400">{proc.cpu}%</td>
                      <td className="py-2 text-green-400">{proc.mem}%</td>
                      <td className="py-2 text-slate-400">{proc.user}</td>
                      <td className="py-2 text-slate-400">-</td>
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
                      <td className="px-4 py-3 text-white">{proc.command}</td>
                      <td className="px-4 py-3">
                        <span className={parseFloat(proc.cpu) > 50 ? 'text-red-400' : 'text-amber-400'}>
                          {proc.cpu}%
                        </span>
                      </td>
                      <td className="px-4 py-3 text-green-400">{proc.mem}%</td>
                      <td className="px-4 py-3 text-slate-400">{proc.user}</td>
                      <td className="px-4 py-3 text-slate-400">-</td>
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
              {disks.length === 0 ? (
                <div className="bg-slate-800 rounded-lg border border-slate-700 p-6 text-center text-slate-400">
                  No disk data available
                </div>
              ) : (
                disks.map(disk => (
                  <div key={disk.mount} className="bg-slate-800 rounded-lg border border-slate-700 p-6">
                    <div className="flex items-center justify-between mb-4">
                      <h3 className="font-medium text-white">{disk.mount}</h3>
                      <span className="text-sm text-slate-400">
                        {disk.used_gb.toFixed(1)} GB / {disk.total_gb.toFixed(1)} GB
                      </span>
                    </div>
                    <div className="h-2 bg-slate-700 rounded-full overflow-hidden">
                      <div
                        className={`h-full transition-all ${
                        (disk.used_gb / disk.total_gb) > 0.9 ? 'bg-red-500' :
                        (disk.used_gb / disk.total_gb) > 0.7 ? 'bg-amber-500' : 'bg-green-500'
                      }`}
                      style={{ width: `${(disk.used_gb / disk.total_gb) * 100}%` }}
                      />
                    </div>
                  </div>
                ))
              )}
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
                <h3 className="text-sm font-medium text-slate-400 mb-4">Bandwidth (24h)</h3>
                <div className="flex items-center gap-8">
                  <div>
                    <p className="text-xs text-slate-500 mb-1">Inbound</p>
                    <p className="text-2xl font-semibold text-green-400">{networkStats?.bandwidth_24h?.inbound_gb?.toFixed(2) ?? '0.00'} GB</p>
                  </div>
                  <div>
                    <p className="text-xs text-slate-500 mb-1">Outbound</p>
                    <p className="text-2xl font-semibold text-blue-400">{networkStats?.bandwidth_24h?.outbound_gb?.toFixed(2) ?? '0.00'} GB</p>
                  </div>
                </div>
              </div>
              <div className="bg-slate-800 rounded-lg border border-slate-700 p-6">
                <h3 className="text-sm font-medium text-slate-400 mb-4">Live Network</h3>
                <div className="grid grid-cols-2 gap-4">
                  <div>
                    <p className="text-xs text-slate-500 mb-1">Inbound</p>
                    <p className="text-lg font-semibold text-green-400">{metrics?.network?.inbound_mbps?.toFixed(2) ?? '0.00'} Mbps</p>
                  </div>
                  <div>
                    <p className="text-xs text-slate-500 mb-1">Outbound</p>
                    <p className="text-lg font-semibold text-blue-400">{metrics?.network?.outbound_mbps?.toFixed(2) ?? '0.00'} Mbps</p>
                  </div>
                </div>
                <p className="text-sm text-slate-400 mt-2">{metrics?.network?.connections_active ?? 0} active connections</p>
              </div>
            </div>

            <div className="bg-slate-800 rounded-lg border border-slate-700 p-6">
              <h3 className="text-sm font-medium text-slate-400 mb-4">Open Ports</h3>
              <table className="w-full">
                <thead>
                  <tr className="text-left text-xs text-slate-500 border-b border-slate-700">
                    <th className="pb-2">Port</th>
                    <th className="pb-2">Protocol</th>
                    <th className="pb-2">Service</th>
                  </tr>
                </thead>
                <tbody>
                  {(networkStats?.open_ports ?? []).map((port, i) => (
                    <tr key={i} className="border-b border-slate-700/50">
                      <td className="py-2 text-white font-mono">{port.port}</td>
                      <td className="py-2 text-slate-400">{port.protocol}</td>
                      <td className="py-2 text-slate-400">{port.service}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>

            <div className="bg-slate-800 rounded-lg border border-slate-700 p-6">
              <h3 className="text-sm font-medium text-slate-400 mb-4">Interfaces</h3>
              <div className="space-y-2">
                {(networkStats?.interfaces ?? []).map((iface, i) => (
                  <div key={i} className="flex items-center justify-between">
                    <span className="text-white">{iface.name}</span>
                    <span className="text-sm text-slate-400">{iface.ipv4 || iface.ipv6 || '-'}</span>
                  </div>
                ))}
              </div>
            </div>
          </div>
        )}

        {activeTab === 'updates' && (
          <div className="space-y-6">
            <div className="bg-slate-800 rounded-lg border border-slate-700 p-6">
              <div className="flex items-center justify-between mb-6">
                <div>
                  <h3 className="font-medium text-white">System Updates</h3>
                  <p className="text-sm text-slate-400">Check for available system updates</p>
                </div>
                <button
                  className="flex items-center gap-2 px-4 py-2 bg-primary-600 hover:bg-primary-700 text-white rounded-lg transition-colors"
                >
                  <Zap className="w-4 h-4" />
                  Check for Updates
                </button>
              </div>
            </div>

            <div className="bg-slate-800 rounded-lg border border-slate-700 p-6">
              <div className="flex items-center gap-3 mb-4">
                <CheckCircle className="w-5 h-5 text-green-500" />
                <div>
                  <h3 className="font-medium text-white">System Up to Date</h3>
                  <p className="text-sm text-slate-400">No security updates available</p>
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
                    <th className="px-4 py-3 font-medium"></th>
                  </tr>
                </thead>
                <tbody>
                  <tr>
                    <td className="px-4 py-3 text-slate-400 text-center" colSpan={5}>
                      Firewall rules managed via dedicated API
                    </td>
                  </tr>
                </tbody>
              </table>
            </div>

            <div className="bg-slate-800 rounded-lg border border-slate-700 p-6">
              <h3 className="font-medium text-white mb-4">Recent Blocks (Last 24h)</h3>
              <p className="text-slate-400">No recent blocks</p>
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
