'use client'

import { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import {
  RefreshCw,
  Boxes,
  ExternalLink,
  Copy,
  Check,
  Loader2
} from 'lucide-react'
import { api, ApiError, Template } from '@/lib/api'
import { useToastStore } from '@/stores'

interface TemplateListResponse {
  data: Template[]
}

const categoryColors: Record<string, { bg: string; text: string }> = {
  cms: { bg: 'bg-purple-500/10', text: 'text-purple-400' },
  database: { bg: 'bg-blue-500/10', text: 'text-blue-400' },
  cache: { bg: 'bg-amber-500/10', text: 'text-amber-400' },
  analytics: { bg: 'bg-green-500/10', text: 'text-green-400' },
  framework: { bg: 'bg-red-500/10', text: 'text-red-400' },
}

const categoryLabels: Record<string, string> = {
  cms: 'CMS',
  database: 'Database',
  cache: 'Cache',
  analytics: 'Analytics',
  framework: 'Framework',
}

const runtimeColors: Record<string, string> = {
  php: 'bg-orange-500/20 text-orange-400',
  nodejs: 'bg-green-500/20 text-green-400',
  python: 'bg-blue-500/20 text-blue-400',
  postgres: 'bg-blue-600/20 text-blue-400',
  mysql: 'bg-slate-500/20 text-slate-300',
  redis: 'bg-red-500/20 text-red-400',
  elixir: 'bg-purple-500/20 text-purple-400',
  mongodb: 'bg-green-500/20 text-green-400',
  minio: 'bg-slate-500/20 text-slate-300',
  custom: 'bg-slate-500/20 text-slate-400',
}

function TemplateIcon({ icon }: { icon: string }) {
  return (
    <div className="w-12 h-12 bg-slate-700 rounded-lg flex items-center justify-center text-lg font-bold text-white">
      {icon}
    </div>
  )
}

export default function TemplatesPage() {
  const { addToast } = useToastStore()
  const [copiedId, setCopiedId] = useState<string | null>(null)

  const { data, isLoading, error, refetch } = useQuery({
    queryKey: ['templates'],
    queryFn: async () => {
      const response = await api.templates.list()
      return response as unknown as TemplateListResponse
    },
  })

  const templates = data?.data || []

  const handleCopyUrl = (id: string, url: string) => {
    navigator.clipboard.writeText(url)
    setCopiedId(id)
    addToast({ type: 'success', title: 'URL copied', message: 'Docker Compose URL copied to clipboard.' })
    setTimeout(() => setCopiedId(null), 2000)
  }

  return (
    <div className="p-6">
      <div className="flex items-center justify-between mb-6">
        <div>
          <h1 className="text-2xl font-semibold text-white">Templates</h1>
          <p className="text-sm text-slate-400 mt-1">
            Pre-configured application templates with Docker Compose
          </p>
        </div>
        <button
          onClick={() => refetch()}
          className="p-2 text-slate-400 hover:text-white hover:bg-slate-800 rounded-md transition-colors"
          title="Refresh"
        >
          <RefreshCw className={`w-4 h-4 ${isLoading ? 'animate-spin' : ''}`} />
        </button>
      </div>

      {isLoading ? (
        <div className="flex items-center justify-center h-64">
          <Loader2 className="w-8 h-8 text-primary-500 animate-spin" />
        </div>
      ) : error ? (
        <div className="flex flex-col items-center justify-center h-64 text-slate-400">
          <p className="mb-2">Failed to load templates</p>
          <button
            onClick={() => refetch()}
            className="text-primary-500 hover:text-primary-400 text-sm"
          >
            Try again
          </button>
        </div>
      ) : templates.length === 0 ? (
        <div className="flex flex-col items-center justify-center py-16">
          <div className="w-16 h-16 bg-slate-700 rounded-full flex items-center justify-center mb-4">
            <Boxes className="w-8 h-8 text-slate-400" />
          </div>
          <h3 className="text-lg font-medium text-white mb-2">No templates available</h3>
          <p className="text-sm text-slate-400 text-center max-w-md">
            Templates will appear here once they are configured.
          </p>
        </div>
      ) : (
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
          {templates.map((template) => (
            <div
              key={template.id}
              className="bg-slate-800 border border-slate-700 rounded-lg p-5 hover:border-slate-600 transition-colors"
            >
              <div className="flex items-start gap-4 mb-4">
                <TemplateIcon icon={template.icon} />
                <div className="flex-1 min-w-0">
                  <h3 className="text-lg font-medium text-white mb-1">{template.name}</h3>
                  <p className="text-sm text-slate-400 line-clamp-2">{template.description}</p>
                </div>
              </div>

              <div className="flex flex-wrap items-center gap-2 mb-4">
                {template.category && (
                  <span className={`px-2 py-0.5 rounded text-xs font-medium ${categoryColors[template.category]?.bg || 'bg-slate-600/20'} ${categoryColors[template.category]?.text || 'text-slate-400'}`}>
                    {categoryLabels[template.category] || template.category}
                  </span>
                )}
                {template.runtimes?.map((runtime) => (
                  <span
                    key={runtime}
                    className={`px-2 py-0.5 rounded text-xs font-medium ${runtimeColors[runtime] || 'bg-slate-500/20 text-slate-400'}`}
                  >
                    {runtime}
                  </span>
                ))}
              </div>

              {template.variables && template.variables.length > 0 && (
                <div className="mb-4">
                  <p className="text-xs text-slate-500 mb-2">Variables:</p>
                  <div className="space-y-1">
                    {template.variables.slice(0, 3).map((variable) => (
                      <div key={variable.key} className="flex items-center gap-2 text-xs">
                        <code className="text-primary-400 bg-primary-500/10 px-1.5 py-0.5 rounded">
                          {variable.key}
                        </code>
                        {variable.required && (
                          <span className="text-red-400">*</span>
                        )}
                        <span className="text-slate-500 truncate">{variable.label}</span>
                      </div>
                    ))}
                    {template.variables.length > 3 && (
                      <p className="text-xs text-slate-500">
                        +{template.variables.length - 3} more variables
                      </p>
                    )}
                  </div>
                </div>
              )}

              <div className="flex items-center gap-2 pt-3 border-t border-slate-700">
                <a
                  href={template.docker_compose_url}
                  target="_blank"
                  rel="noopener noreferrer"
                  className="flex items-center gap-1.5 px-3 py-1.5 bg-primary-600 hover:bg-primary-700 text-white rounded text-sm font-medium transition-colors flex-1 justify-center"
                >
                  <ExternalLink className="w-4 h-4" />
                  View Compose
                </a>
                <button
                  onClick={() => handleCopyUrl(template.id, template.docker_compose_url)}
                  className="p-1.5 text-slate-400 hover:text-white hover:bg-slate-700 rounded transition-colors"
                  title="Copy URL"
                >
                  {copiedId === template.id ? (
                    <Check className="w-4 h-4 text-green-400" />
                  ) : (
                    <Copy className="w-4 h-4" />
                  )}
                </button>
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  )
}