'use client'

import Link from 'next/link'
import { ChevronRight, Home } from 'lucide-react'
import { usePathname } from 'next/navigation'

interface BreadcrumbItem {
  label: string
  href?: string
}

export function Breadcrumb() {
  const pathname = usePathname()

  // Generate breadcrumb items from pathname
  const items: BreadcrumbItem[] = [
    { label: 'Home', href: '/' }
  ]

  // Parse pathname to build breadcrumbs
  const pathSegments = pathname.split('/').filter(Boolean)
  let currentPath = ''

  pathSegments.forEach((segment, index) => {
    currentPath += `/${segment}`
    
    // Format the label (capitalize, replace hyphens)
    const label = segment
      .replace(/-/g, ' ')
      .replace(/\b\w/g, c => c.toUpperCase())
    
    items.push({
      label,
      href: index < pathSegments.length - 1 ? currentPath : undefined
    })
  })

  // Don't show breadcrumb on home page
  if (items.length <= 1) {
    return null
  }

  return (
    <nav className="flex items-center gap-1 text-sm text-slate-400">
      <Link
        href="/"
        className="p-1 hover:text-white transition-colors"
      >
        <Home className="w-4 h-4" />
      </Link>
      
      {items.slice(1).map((item, index) => (
        <div key={index} className="flex items-center gap-1">
          <ChevronRight className="w-4 h-4 text-slate-600" />
          {item.href ? (
            <Link
              href={item.href}
              className="hover:text-white transition-colors"
            >
              {item.label}
            </Link>
          ) : (
            <span className="text-white">{item.label}</span>
          )}
        </div>
      ))}
    </nav>
  )
}