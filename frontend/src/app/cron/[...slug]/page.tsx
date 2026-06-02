import { CronJobDetailClient } from './CronJobDetailClient'

export function generateStaticParams() {
  return [{ slug: ['_'] }]
}

export default function CronJobDetailPage() {
  return <CronJobDetailClient />
}
