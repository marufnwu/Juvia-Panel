'use client'

import { usePathname } from 'next/navigation'
import { Navbar } from '@/components/layout/Navbar'
import { Sidebar } from '@/components/layout/Sidebar'

const publicRoutes = ['/login', '/setup']

export function LayoutShell({ children }: { children: React.ReactNode }) {
  const pathname = usePathname()
  const isPublicRoute = publicRoutes.includes(pathname)

  if (isPublicRoute) {
    return <>{children}</>
  }

  return (
    <>
      <Navbar />
      <div className="pt-16">
        <Sidebar />
        <main className="lg:pl-64">
          {children}
        </main>
      </div>
    </>
  )
}
