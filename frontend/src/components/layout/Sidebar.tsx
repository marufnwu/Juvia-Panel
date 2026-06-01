'use client'

import { useState } from 'react'
import Link from 'next/link'
import { usePathname } from 'next/navigation'
import {
  ChevronLeft,
  ChevronRight,
  ChevronDown,
  LayoutDashboard,
  AppWindow,
  Database,
  Server,
  GitBranch,
  Settings,
  Shield,
  Terminal,
  Folder,
  Clock,
  HardDrive,
  ShieldCheck,
  Package
} from 'lucide-react'

interface NavItem {
  name: string
  href: string
  icon: React.ComponentType<{ className?: string }>
  badge?: number | string
  alert?: boolean
}

interface NavSection {
  title: string
  items: NavItem[]
  collapsible?: boolean
  defaultExpanded?: boolean
}

const sidebarSections: NavSection[] = [
  {
    title: 'Main',
    items: [
      { name: 'Dashboard', href: '/', icon: LayoutDashboard },
      { name: 'Apps', href: '/apps', icon: AppWindow },
      { name: 'Services', href: '/services', icon: Database },
      { name: 'Templates', href: '/templates', icon: Package },
      { name: 'Server', href: '/server', icon: Server, alert: true },
    ]
  },
  {
    title: 'Server',
    collapsible: true,
    defaultExpanded: false,
    items: [
      { name: 'Overview', href: '/server', icon: Server },
      { name: 'Terminal', href: '/terminal', icon: Terminal },
      { name: 'Files', href: '/files', icon: Folder },
      { name: 'Backups', href: '/backups', icon: HardDrive },
      { name: 'Cron Jobs', href: '/cron', icon: Clock },
      { name: 'Firewall', href: '/server?tab=firewall', icon: ShieldCheck },
    ]
  },
  {
    title: 'Deployments',
    items: [
      { name: 'Git Repositories', href: '/repositories', icon: GitBranch },
    ]
  },
  {
    title: 'System',
    items: [
      { name: 'Settings', href: '/settings', icon: Settings },
      { name: 'Security', href: '/security', icon: Shield },
    ]
  }
]

export function Sidebar() {
  const [collapsed, setCollapsed] = useState(false)
  const [expandedSections, setExpandedSections] = useState<Set<string>>(
    new Set(sidebarSections.filter(s => !s.collapsible || s.defaultExpanded).map(s => s.title))
  )
  const pathname = usePathname()

  const toggleSection = (title: string) => {
    setExpandedSections(prev => {
      const newSet = new Set(prev)
      if (newSet.has(title)) {
        newSet.delete(title)
      } else {
        newSet.add(title)
      }
      return newSet
    })
  }

  const isActive = (href: string) => {
    if (href === '/') {
      return pathname === '/'
    }
    if (href.startsWith('/server')) {
      return pathname.startsWith('/server')
    }
    return pathname.startsWith(href)
  }

  return (
    <aside
      className={`
        fixed left-0 top-16 bottom-0 z-40
        bg-slate-900 border-r border-slate-700
        transition-all duration-200 ease-in-out
        hidden lg:block
        ${collapsed ? 'w-16' : 'w-64'}
      `}
    >
      <div className="flex flex-col h-full">
        {/* Toggle Button */}
        <div className="flex justify-end p-2">
          <button
            onClick={() => setCollapsed(!collapsed)}
            className="p-1.5 text-slate-400 hover:text-white hover:bg-slate-800 rounded transition-colors"
          >
            {collapsed ? (
              <ChevronRight className="w-4 h-4" />
            ) : (
              <ChevronLeft className="w-4 h-4" />
            )}
          </button>
        </div>

        {/* Navigation */}
        <nav className="flex-1 overflow-y-auto px-2 py-4">
          {sidebarSections.map((section) => (
            <div key={section.title} className="mb-6">
              {!collapsed && (
                <button
                  onClick={() => section.collapsible && toggleSection(section.title)}
                  className={`flex items-center justify-between w-full px-3 mb-2 text-xs font-semibold uppercase tracking-wider
                    ${section.collapsible ? 'cursor-pointer text-slate-500 hover:text-slate-400' : 'text-slate-500'}
                  `}
                >
                  <span>{section.title}</span>
                  {section.collapsible && (
                    <ChevronDown
                      className={`w-4 h-4 transition-transform ${
                        expandedSections.has(section.title) ? 'rotate-0' : '-rotate-90'
                      }`}
                    />
                  )}
                </button>
              )}
              
              {(!section.collapsible || expandedSections.has(section.title)) && (
                <div className="space-y-1">
                  {section.items.map((item) => {
                    const Icon = item.icon
                    const active = isActive(item.href)
                    return (
                      <Link
                        key={item.name}
                        href={item.href}
                        className={`
                          flex items-center gap-3 px-3 py-2 rounded-md text-sm font-medium transition-colors
                          ${active
                            ? 'bg-slate-800 text-white'
                            : 'text-slate-400 hover:text-white hover:bg-slate-800'
                          }
                          ${collapsed ? 'justify-center' : ''}
                        `}
                        title={collapsed ? item.name : undefined}
                      >
                        <Icon className="w-5 h-5 flex-shrink-0" />
                        {!collapsed && (
                          <>
                            <span className="flex-1">{item.name}</span>
                            {item.alert && (
                              <span className="w-2 h-2 bg-red-500 rounded-full" />
                            )}
                            {item.badge && (
                              <span className="px-1.5 py-0.5 text-xs bg-slate-700 rounded">
                                {item.badge}
                              </span>
                            )}
                          </>
                        )}
                      </Link>
                    )
                  })}
                </div>
              )}
            </div>
          ))}
        </nav>

        {/* Collapse Indicator */}
        {!collapsed && (
          <div className="p-4 border-t border-slate-700">
            <p className="text-xs text-slate-500 text-center">
              Juvia Panel v1.0.0
            </p>
          </div>
        )}
      </div>
    </aside>
  )
}
