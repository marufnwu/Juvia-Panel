import { AppDetailClient } from './AppDetailClient'

export function generateStaticParams() {
  return [{ slug: ['_'] }]
}

export default function AppDetailPage() {
  return <AppDetailClient />
}
