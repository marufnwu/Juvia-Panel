import type { Metadata } from 'next'
import './globals.css'
import { Providers } from './providers'
import { LayoutShell } from '@/components/layout/LayoutShell'

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
          <LayoutShell>
            {children}
          </LayoutShell>
        </Providers>
      </body>
    </html>
  )
}