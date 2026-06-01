'use client'

import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import {
  UserPlus,
  MoreHorizontal,
  Mail,
  Shield,
  Trash2,
  CheckCircle,
  Clock,
  XCircle,
  Loader2
} from 'lucide-react'
import { api, ApiError } from '@/lib/api'
import { useToastStore } from '@/stores'
import { InviteModal } from '@/components/team/InviteModal'
import { RoleSelector } from '@/components/team/RoleSelector'

type UserRole = 'owner' | 'admin' | 'developer' | 'viewer'
type UserStatus = 'active' | 'pending' | 'inactive'

interface TeamMember {
  id: string
  name: string
  email: string
  role: UserRole
  status: UserStatus
  last_active: string | null
  created_at: string
}

interface TeamResponse {
  data: TeamMember[]
  meta: {
    total: number
    page: number
    per_page: number
    total_pages: number
  }
}

const roleColors: Record<UserRole, { bg: string; text: string }> = {
  owner: { bg: 'bg-purple-500/10', text: 'text-purple-400' },
  admin: { bg: 'bg-blue-500/10', text: 'text-blue-400' },
  developer: { bg: 'bg-green-500/10', text: 'text-green-400' },
  viewer: { bg: 'bg-slate-500/10', text: 'text-slate-400' },
}

const statusIcons: Record<UserStatus, React.ComponentType<{ className?: string }>> = {
  active: CheckCircle,
  pending: Clock,
  inactive: XCircle,
}

const statusColors: Record<UserStatus, string> = {
  active: 'text-green-400',
  pending: 'text-amber-400',
  inactive: 'text-slate-400',
}

function formatRelativeTime(dateString: string | null): string {
  if (!dateString) return '—'
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
  return date.toLocaleDateString()
}

export default function TeamPage() {
  const queryClient = useQueryClient()
  const { addToast } = useToastStore()
  const [inviteModalOpen, setInviteModalOpen] = useState(false)
  const [roleSelectorOpen, setRoleSelectorOpen] = useState(false)
  const [selectedMember, setSelectedMember] = useState<TeamMember | null>(null)
  const [menuOpen, setMenuOpen] = useState<string | null>(null)

  // Fetch team members
  const { data, isLoading, error, refetch } = useQuery({
    queryKey: ['team'],
    queryFn: async () => {
      const response = await api.users.list()
      return response as unknown as TeamResponse
    },
  })

  // Remove member mutation
  const removeMutation = useMutation({
    mutationFn: (userId: string) => api.users.delete(userId),
    onSuccess: () => {
      addToast({ type: 'success', title: 'Member removed', message: 'Team member has been removed.' })
      queryClient.invalidateQueries({ queryKey: ['team'] })
      setMenuOpen(null)
    },
    onError: (error: ApiError) => {
      addToast({ type: 'error', title: 'Failed to remove member', message: error.message })
    },
  })

  const members = data?.data || []

  const handleRoleChange = (member: TeamMember) => {
    setSelectedMember(member)
    setRoleSelectorOpen(true)
    setMenuOpen(null)
  }

  const handleRemoveMember = (member: TeamMember) => {
    if (confirm(`Are you sure you want to remove ${member.name} from the team?`)) {
      removeMutation.mutate(member.id)
    }
    setMenuOpen(null)
  }

  return (
    <div className="p-6">
      {/* Header */}
      <div className="flex items-center justify-between mb-6">
        <div>
          <h1 className="text-2xl font-semibold text-white">Team</h1>
          <p className="text-sm text-slate-400 mt-1">
            Manage your team members and their permissions
          </p>
        </div>
        <button
          onClick={() => setInviteModalOpen(true)}
          className="flex items-center gap-2 px-4 py-2 bg-primary-600 hover:bg-primary-700 text-white rounded-md text-sm font-medium transition-colors"
        >
          <UserPlus className="w-4 h-4" />
          Invite Member
        </button>
      </div>

      {/* Role Info */}
      <div className="mb-6 p-4 bg-slate-800 border border-slate-700 rounded-lg">
        <h3 className="text-sm font-medium text-white mb-3">Role Permissions</h3>
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
          <div className="flex items-start gap-3">
            <div className={`p-2 rounded-md ${roleColors.owner.bg}`}>
              <Shield className={`w-4 h-4 ${roleColors.owner.text}`} />
            </div>
            <div>
              <p className={`text-sm font-medium ${roleColors.owner.text}`}>Owner</p>
              <p className="text-xs text-slate-400">Full control, billing, delete server</p>
            </div>
          </div>
          <div className="flex items-start gap-3">
            <div className={`p-2 rounded-md ${roleColors.admin.bg}`}>
              <Shield className={`w-4 h-4 ${roleColors.admin.text}`} />
            </div>
            <div>
              <p className={`text-sm font-medium ${roleColors.admin.text}`}>Admin</p>
              <p className="text-xs text-slate-400">Manage apps, services, users</p>
            </div>
          </div>
          <div className="flex items-start gap-3">
            <div className={`p-2 rounded-md ${roleColors.developer.bg}`}>
              <Shield className={`w-4 h-4 ${roleColors.developer.text}`} />
            </div>
            <div>
              <p className={`text-sm font-medium ${roleColors.developer.text}`}>Developer</p>
              <p className="text-xs text-slate-400">Deploy apps, manage env vars, view logs</p>
            </div>
          </div>
          <div className="flex items-start gap-3">
            <div className={`p-2 rounded-md ${roleColors.viewer.bg}`}>
              <Shield className={`w-4 h-4 ${roleColors.viewer.text}`} />
            </div>
            <div>
              <p className={`text-sm font-medium ${roleColors.viewer.text}`}>Viewer</p>
              <p className="text-xs text-slate-400">Read-only access to all resources</p>
            </div>
          </div>
        </div>
      </div>

      {/* Members Table */}
      <div className="bg-slate-800 border border-slate-700 rounded-lg overflow-hidden">
        {isLoading ? (
          <div className="flex items-center justify-center h-64">
            <Loader2 className="w-8 h-8 text-primary-500 animate-spin" />
          </div>
        ) : error ? (
          <div className="flex flex-col items-center justify-center h-64 text-slate-400">
            <p className="mb-2">Failed to load team members</p>
            <button
              onClick={() => refetch()}
              className="text-primary-500 hover:text-primary-400 text-sm"
            >
              Try again
            </button>
          </div>
        ) : members.length === 0 ? (
          <div className="flex flex-col items-center justify-center py-16">
            <div className="w-16 h-16 bg-slate-700 rounded-full flex items-center justify-center mb-4">
              <Mail className="w-8 h-8 text-slate-400" />
            </div>
            <h3 className="text-lg font-medium text-white mb-2">No team members yet</h3>
            <p className="text-sm text-slate-400 mb-6 text-center max-w-md">
              Invite your team members to collaborate on managing your server.
            </p>
            <button
              onClick={() => setInviteModalOpen(true)}
              className="flex items-center gap-2 px-4 py-2 bg-primary-600 hover:bg-primary-700 text-white rounded-md text-sm font-medium transition-colors"
            >
              <UserPlus className="w-4 h-4" />
              Invite Member
            </button>
          </div>
        ) : (
          <table className="w-full">
            <thead>
              <tr className="border-b border-slate-700">
                <th className="px-4 py-3 text-left text-xs font-medium text-slate-400 uppercase tracking-wider">
                  Member
                </th>
                <th className="px-4 py-3 text-left text-xs font-medium text-slate-400 uppercase tracking-wider">
                  Role
                </th>
                <th className="px-4 py-3 text-left text-xs font-medium text-slate-400 uppercase tracking-wider">
                  Status
                </th>
                <th className="px-4 py-3 text-left text-xs font-medium text-slate-400 uppercase tracking-wider">
                  Last Active
                </th>
                <th className="px-4 py-3 text-right text-xs font-medium text-slate-400 uppercase tracking-wider w-24">
                  Actions
                </th>
              </tr>
            </thead>
            <tbody className="divide-y divide-slate-700">
              {members.map((member) => {
                const StatusIcon = statusIcons[member.status]
                return (
                  <tr key={member.id} className="hover:bg-slate-700/50 transition-colors">
                    <td className="px-4 py-4">
                      <div className="flex items-center gap-3">
                        <div className="w-10 h-10 rounded-full bg-primary-600 flex items-center justify-center">
                          <span className="text-sm font-medium text-white">
                            {member.name.charAt(0).toUpperCase()}
                          </span>
                        </div>
                        <div>
                          <p className="text-sm font-medium text-white">{member.name}</p>
                          <p className="text-xs text-slate-400">{member.email}</p>
                        </div>
                      </div>
                    </td>
                    <td className="px-4 py-4">
                      <span className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium ${roleColors[member.role].bg} ${roleColors[member.role].text}`}>
                        {member.role.charAt(0).toUpperCase() + member.role.slice(1)}
                      </span>
                    </td>
                    <td className="px-4 py-4">
                      <div className="flex items-center gap-2">
                        <StatusIcon className={`w-4 h-4 ${statusColors[member.status]}`} />
                        <span className="text-sm text-slate-300 capitalize">{member.status}</span>
                      </div>
                    </td>
                    <td className="px-4 py-4">
                      <span className="text-sm text-slate-400">
                        {formatRelativeTime(member.last_active)}
                      </span>
                    </td>
                    <td className="px-4 py-4">
                      <div className="flex items-center justify-end gap-1">
                        <button
                          onClick={() => handleRoleChange(member)}
                          className="p-1.5 text-slate-400 hover:text-white hover:bg-slate-600 rounded transition-colors"
                          title="Change role"
                        >
                          <Shield className="w-4 h-4" />
                        </button>
                        <div className="relative">
                          <button
                            onClick={() => setMenuOpen(menuOpen === member.id ? null : member.id)}
                            className="p-1.5 text-slate-400 hover:text-white hover:bg-slate-600 rounded transition-colors"
                            title="More actions"
                          >
                            <MoreHorizontal className="w-4 h-4" />
                          </button>
                          {menuOpen === member.id && (
                            <div className="absolute right-0 mt-1 w-48 bg-slate-700 border border-slate-600 rounded-md shadow-lg py-1 z-10">
                              <button
                                onClick={() => handleRemoveMember(member)}
                                className="flex items-center gap-2 w-full px-4 py-2 text-sm text-danger-500 hover:bg-slate-600"
                              >
                                <Trash2 className="w-4 h-4" />
                                Remove member
                              </button>
                            </div>
                          )}
                        </div>
                      </div>
                    </td>
                  </tr>
                )
              })}
            </tbody>
          </table>
        )}
      </div>

      {/* Invite Modal */}
      <InviteModal
        open={inviteModalOpen}
        onClose={() => setInviteModalOpen(false)}
      />

      {/* Role Selector Modal */}
      {selectedMember && (
        <RoleSelector
          open={roleSelectorOpen}
          onClose={() => {
            setRoleSelectorOpen(false)
            setSelectedMember(null)
          }}
          member={selectedMember}
        />
      )}
    </div>
  )
}