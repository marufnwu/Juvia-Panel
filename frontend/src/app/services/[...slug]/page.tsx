import { ServiceDetailClient } from './ServiceDetailClient'

export function generateStaticParams() {
  return [{ slug: ['_'] }]
}

export default function ServiceDetailPage() {
  return <ServiceDetailClient />
}
