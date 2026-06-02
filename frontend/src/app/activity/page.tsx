'use client'

import { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import {
  Download,
  Search,
  Filter,
  Calendar,
  ChevronLeft,
  ChevronRight,
  Activity,
  User,
  Box,
  Database,
  Server,
  Loader2
} from 'lucide-react'
import { api, ActivityEvent } from '@/lib/api'

type FilterType = 'all' | 'app' | 'service' | 'user'

interface ActivityResponse {
  events: ActivityEvent[]
  total: number
  page: number
  limit: number
}

function formatDate(dateString: string): string {
  const date = new Date(dateString)
  return date.toLocaleDateString('en-US', {
    year: 'numeric',
    month: 'short',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  })
}

function formatRelativeTime(dateString: string): string {
  const date = new Date(dateString)
  const now = new Date()
  const diffMs = now.getTime() - date.getTime()
  const diffMins = Math.floor(diffMs / 60000)
  const diffHours = Math.floor(diffMins / 60)
  const diffDays = Math.floor(diffHours / 24)

  if (diffMins < 1) return 'Just now'
  if (diffMins < 60) return `${diffMins}m ago`
  if (diffHours < 24) return `${diffHours}h ago`
  if (diffDays < 30) return `${diffDays}d ago`
  return formatDate(dateString)
}

function getEventIcon(type: string) {
  if (type.includes('app') || type.includes('deploy')) return Box
  if (type.includes('service') || type.includes('database')) return Database
  if (type.includes('server') || type.includes('system')) return Server
  return User
}

function getEventColor(type: string): string {
  if (type.includes('failed') || type.includes('error')) return 'text-danger-500'
  if (type.includes('created') || type.includes('deployed')) return 'text-success-500'
  if (type.includes('updated') || type.includes('modified')) return 'text-primary-500'
  return 'text-slate-400'
}

export default function ActivityPage() {
  const [filter, setFilter] = useState<FilterType>('all')
  const [search, setSearch] = useState('')
  const [page, setPage] = useState(1)
  const [dateRange, setDateRange] = useState<{ from?: string; to?: string }>({})

  const { data, isLoading, error } = useQuery({
    queryKey: ['activity', { page, filter, search }],
    queryFn: async () => {
      const params: { page: number; limit: number; action?: string; user_id?: number } = {
        page,
        limit: 50,
      }
      if (filter !== 'all') {
        const prefix = filter === 'app' ? 'app.' : filter === 'service' ? 'service.' : 'user.'
        params.action = prefix
      }
      return api.activity.list(params)
    },
  })

  const events = data?.data || []
  const total = data?.meta?.total || 0
  const totalPages = Math.ceil(total / 50)

  const filteredEvents = events.filter(event => {
    if (filter !== 'all') {
      const filterPrefix = filter === 'app' ? 'app' : filter === 'service' ? 'service' : 'user'
      if (!event.action.startsWith(filterPrefix)) return false
    }
    if (search) {
      const searchLower = search.toLowerCase()
      return event.action.toLowerCase().includes(searchLower) ||
             event.user_username.toLowerCase().includes(searchLower)
    }
    return true
  })

  const handleExportCSV = () => {
    const headers = ['Time', 'User', 'Action', 'Target', 'IP']
    const rows = filteredEvents.map(event => [
      event.created_at,
      event.user_username || 'system',
      event.action,
      event.resource_id || '-',
      event.ip_address || '-',
    ])
    const csv = [headers.join(','), ...rows.map(row => row.join(','))].join('\n')
    const blob = new Blob([csv], { type: 'text/csv' })
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = `activity-log-${new Date().toISOString().split('T')[0]}.csv`
    a.click()
    URL.revokeObjectURL(url)
  }

  return (
    <div className="p-6">
      {/* Header */}
      <div className="flex items-center justify-between mb-6">
        <div>
          <h1 className="text-2xl font-semibold text-white">Activity Log</h1>
          <p className="text-sm text-slate-400 mt-1">
            Track all actions and changes on your server
          </p>
        </div>
        <button
          onClick={handleExportCSV}
          className="flex items-center gap-2 px-4 py-2 bg-slate-700 hover:bg-slate-600 text-white rounded-md text-sm font-medium transition-colors"
        >
          <Download className="w-4 h-4" />
          Export CSV
        </button>
      </div>

      {/* Filters */}
      <div className="flex flex-wrap items-center gap-4 mb-6">
        {/* Search */}
        <div className="relative flex-1 min-w-[240px] max-w-md">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-slate-400" />
          <input
            type="text"
            placeholder="Search activities..."
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            className="w-full pl-10 pr-4 py-2 bg-slate-800 border border-slate-700 rounded-md text-sm text-white placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-primary-500"
          />
        </div>

        {/* Type Filter */}
        <select
          value={filter}
          onChange={(e) => setFilter(e.target.value as FilterType)}
          className="px-3 py-2 bg-slate-800 border border-slate-700 rounded-md text-sm text-white focus:outline-none focus:ring-2 focus:ring-primary-500"
        >
          <option value="all">All Activities</option>
          <option value="app">App Actions</option>
          <option value="service">Service Actions</option>
          <option value="user">User Actions</option>
        </select>

        {/* Date Range */}
        <div className="flex items-center gap-2">
          <input
            type="date"
            value={dateRange.from || ''}
            onChange={(e) => setDateRange({ ...dateRange, from: e.target.value })}
            className="px-3 py-2 bg-slate-800 border border-slate-700 rounded-md text-sm text-white focus:outline-none focus:ring-2 focus:ring-primary-500"
          />
          <span className="text-slate-500">to</span>
          <input
            type="date"
            value={dateRange.to || ''}
            onChange={(e) => setDateRange({ ...dateRange, to: e.target.value })}
            className="px-3 py-2 bg-slate-800 border border-slate-700 rounded-md text-sm text-white focus:outline-none focus:ring-2 focus:ring-primary-500"
          />
        </div>
      </div>

      {/* Activity Table */}
      <div className="bg-slate-800 border border-slate-700 rounded-lg overflow-hidden">
        {isLoading ? (
          <div className="flex items-center justify-center h-64">
            <Loader2 className="w-8 h-8 text-primary-500 animate-spin" />
          </div>
        ) : error ? (
          <div className="flex flex-col items-center justify-center h-64 text-slate-400">
            <p className="mb-2">Failed to load activity log</p>
          </div>
        ) : filteredEvents.length === 0 ? (
          <div className="flex flex-col items-center justify-center py-16">
            <Activity className="w-12 h-12 text-slate-500 mb-4" />
            <h3 className="text-lg font-medium text-white mb-2">No activities found</h3>
            <p className="text-sm text-slate-400">
              {search ? 'Try adjusting your search criteria' : 'Activity will appear here as you use the panel'}
            </p>
          </div>
        ) : (
          <>
            <table className="w-full">
              <thead>
                <tr className="border-b border-slate-700">
                  <th className="px-4 py-3 text-left text-xs font-medium text-slate-400 uppercase tracking-wider">
                    Time
                  </th>
                  <th className="px-4 py-3 text-left text-xs font-medium text-slate-400 uppercase tracking-wider">
                    User
                  </th>
                  <th className="px-4 py-3 text-left text-xs font-medium text-slate-400 uppercase tracking-wider">
                    Action
                  </th>
                  <th className="px-4 py-3 text-left text-xs font-medium text-slate-400 uppercase tracking-wider">
                    Target
                  </th>
                  <th className="px-4 py-3 text-left text-xs font-medium text-slate-400 uppercase tracking-wider">
                    IP
                  </th>
                </tr>
              </thead>
              <tbody className="divide-y divide-slate-700">
                {filteredEvents.map((event) => {
                  const Icon = getEventIcon(event.action)
                  const iconColor = getEventColor(event.action)
                  return (
                    <tr key={event.id} className="hover:bg-slate-700/50 transition-colors">
                      <td className="px-4 py-4">
                        <div className="flex items-center gap-2">
                          <Icon className={`w-4 h-4 ${iconColor}`} />
                          <span className="text-sm text-slate-400" title={formatDate(event.created_at)}>
                            {formatRelativeTime(event.created_at)}
                          </span>
                        </div>
                      </td>
                      <td className="px-4 py-4">
                        <span className="text-sm text-white">{event.user_username || 'system'}</span>
                      </td>
                      <td className="px-4 py-4">
                        <span className="text-sm text-slate-300">{event.action}</span>
                      </td>
                      <td className="px-4 py-4">
                        <span className="text-sm text-slate-400">
                          {event.resource_type ? `${event.resource_type}/${event.resource_id || ''}` : '—'}
                        </span>
                      </td>
                      <td className="px-4 py-4">
                        <span className="text-sm text-slate-500">{event.ip_address || '—'}</span>
                      </td>
                    </tr>
                  )
                })}
              </tbody>
            </table>

            {/* Pagination */}
            {totalPages > 1 && (
              <div className="flex items-center justify-between px-4 py-3 border-t border-slate-700">
                <p className="text-sm text-slate-400">
                  Showing {filteredEvents.length} of {total} activities
                </p>
                <div className="flex items-center gap-2">
                  <button
                    onClick={() => setPage(page - 1)}
                    disabled={page === 1}
                    className="p-1.5 text-slate-400 hover:text-white hover:bg-slate-700 rounded disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
                  >
                    <ChevronLeft className="w-4 h-4" />
                  </button>
                  <span className="text-sm text-slate-300 px-2">
                    Page {page} of {totalPages}
                  </span>
                  <button
                    onClick={() => setPage(page + 1)}
                    disabled={page >= totalPages}
                    className="p-1.5 text-slate-400 hover:text-white hover:bg-slate-700 rounded disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
                  >
                    <ChevronRight className="w-4 h-4" />
                  </button>
                </div>
              </div>
            )}
          </>
        )}
      </div>
    </div>
  )
}