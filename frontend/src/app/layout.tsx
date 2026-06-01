import type { Metadata } from 'next'
import './globals.css'
import { Providers } from './providers'
import { Navbar } from '@/components/layout/Navbar'
import { Sidebar } from '@/components/layout/Sidebar'

export const metadata: Metadata = {
  title: 'Juvia Panel',
  description: 'Single-server self-hosted PaaS panel',
}

export default function RootLayout({
  children,
}: {
  children: React.ReactNode
}) {
  return (
    <html lang="en" className="dark">
      <body className="min-h-screen bg-slate-900 text-slate-50">
        <Providers>
          <Navbar />
          <div className="pt-16">
            <Sidebar />
            <main className="lg:pl-64">
              {children}
            </main>
          </div>
        </Providers>
      </body>
    </html>
  )
}