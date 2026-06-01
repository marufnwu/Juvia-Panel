'use client'

import { useState } from 'react'
import { useRouter } from 'next/navigation'
import Link from 'next/link'
import { useMutation } from '@tanstack/react-query'
import {
  ArrowLeft,
  GitBranch,
  Upload,
  FileCode,
  ChevronRight,
  ChevronLeft,
  Loader2,
  Check,
  X,
  Plus,
  Trash2
} from 'lucide-react'
import { api, ApiError, CreateAppInput } from '@/lib/api'
import { useToastStore } from '@/stores'

type Step = 'source' | 'configure' | 'basic' | 'review'
type SourceType = 'git' | 'upload' | 'docker' | null

interface AppConfig {
  sourceType: SourceType
  gitUrl: string
  branch: string
  buildStrategy: 'auto' | 'nixpacks' | 'dockerfile' | 'static'
  appName: string
  domain: string
  envVars: Array<{ key: string; value: string; secret: boolean }>
}

const initialConfig: AppConfig = {
  sourceType: null,
  gitUrl: '',
  branch: 'main',
  buildStrategy: 'auto',
  appName: '',
  domain: '',
  envVars: [],
}

function StepIndicator({ currentStep, steps }: { currentStep: Step; steps: { key: Step; label: string }[] }) {
  const stepIndex = steps.findIndex(s => s.key === currentStep)
  
  return (
    <div className="flex items-center justify-center mb-8">
      {steps.map((step, index) => (
        <div key={step.key} className="flex items-center">
          <div className="flex flex-col items-center">
            <div className={`w-8 h-8 rounded-full flex items-center justify-center text-sm font-medium ${
              index < stepIndex
                ? 'bg-green-500 text-white'
                : index === stepIndex
                ? 'bg-primary-600 text-white'
                : 'bg-slate-700 text-slate-400'
            }`}>
              {index < stepIndex ? <Check className="w-4 h-4" /> : index + 1}
            </div>
            <span className={`text-xs mt-1 ${
              index <= stepIndex ? 'text-white' : 'text-slate-500'
            }`}>
              {step.label}
            </span>
          </div>
          {index < steps.length - 1 && (
            <div className={`w-16 h-0.5 mx-2 ${
              index < stepIndex ? 'bg-green-500' : 'bg-slate-700'
            }`} />
          )}
        </div>
      ))}
    </div>
  )
}

export default function CreateAppPage() {
  const router = useRouter()
  const { addToast } = useToastStore()
  const [step, setStep] = useState<Step>('source')
  const [config, setConfig] = useState<AppConfig>(initialConfig)
  const [newEnvKey, setNewEnvKey] = useState('')
  const [newEnvValue, setNewEnvValue] = useState('')
  const [newEnvSecret, setNewEnvSecret] = useState(false)

  const steps = [
    { key: 'source' as Step, label: 'Source' },
    { key: 'configure' as Step, label: 'Configure' },
    { key: 'basic' as Step, label: 'Basic' },
    { key: 'review' as Step, label: 'Review' },
  ]

  // Create app mutation
  const createAppMutation = useMutation({
    mutationFn: async (data: CreateAppInput) => {
      return api.apps.create(data)
    },
    onSuccess: (response) => {
      const data = response as unknown as { id: string; name: string; deployment_id?: string }
      addToast({
        type: 'success',
        title: 'App created',
        message: `${data.name} has been created. Deployment started.`
      })
      router.push(`/apps/${data.id}`)
    },
    onError: (error: ApiError) => {
      addToast({
        type: 'error',
        title: 'Failed to create app',
        message: error.message
      })
    },
  })

  const updateConfig = <K extends keyof AppConfig>(key: K, value: AppConfig[K]) => {
    setConfig(prev => ({ ...prev, [key]: value }))
  }

  const handleSourceSelect = (type: SourceType) => {
    updateConfig('sourceType', type)
    setStep('configure')
  }

  const handleAddEnvVar = () => {
    if (!newEnvKey.trim()) return
    updateConfig('envVars', [...config.envVars, {
      key: newEnvKey.trim().toUpperCase(),
      value: newEnvValue,
      secret: newEnvSecret,
    }])
    setNewEnvKey('')
    setNewEnvValue('')
    setNewEnvSecret(false)
  }

  const handleRemoveEnvVar = (key: string) => {
    updateConfig('envVars', config.envVars.filter(v => v.key !== key))
  }

  const handleCreate = () => {
    const data: CreateAppInput = {
      name: config.appName.toLowerCase().replace(/[^a-z0-9-]/g, '-'),
      build_strategy: config.buildStrategy,
      branch: config.branch,
      git_url: config.gitUrl,
      domain: config.domain || undefined,
      env_vars: config.envVars,
    }
    createAppMutation.mutate(data)
  }

  const canProceed = () => {
    switch (step) {
      case 'source':
        return config.sourceType !== null
      case 'configure':
        if (config.sourceType === 'git') {
          return config.gitUrl.trim() !== ''
        }
        return true
      case 'basic':
        return config.appName.trim() !== ''
      case 'review':
        return true
      default:
        return false
    }
  }

  return (
    <div className="p-6 max-w-3xl mx-auto">
      {/* Header */}
      <div className="flex items-center gap-4 mb-8">
        <Link href="/apps" className="p-2 text-slate-400 hover:text-white hover:bg-slate-800 rounded-md transition-colors">
          <ArrowLeft className="w-5 h-5" />
        </Link>
        <div>
          <h1 className="text-2xl font-semibold text-white">Create New App</h1>
          <p className="text-sm text-slate-400 mt-1">
            Deploy your application in a few steps
          </p>
        </div>
      </div>

      {/* Step Indicator */}
      <StepIndicator currentStep={step} steps={steps} />

      {/* Step Content */}
      <div className="bg-slate-800 border border-slate-700 rounded-lg p-6">
        {/* Step 1: Choose Source */}
        {step === 'source' && (
          <div>
            <h2 className="text-lg font-medium text-white mb-4">Choose Source</h2>
            <p className="text-slate-400 mb-6">How do you want to deploy your app?</p>
            
            <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
              <button
                onClick={() => handleSourceSelect('git')}
                className="flex flex-col items-center p-6 bg-slate-700/50 hover:bg-slate-700 border border-slate-600 hover:border-primary-500 rounded-lg transition-colors"
              >
                <GitBranch className="w-10 h-10 text-primary-400 mb-3" />
                <span className="font-medium text-white">Git Repository</span>
                <span className="text-xs text-slate-400 mt-1">GitHub, GitLab, Bitbucket</span>
              </button>

              <button
                onClick={() => handleSourceSelect('upload')}
                className="flex flex-col items-center p-6 bg-slate-700/50 hover:bg-slate-700 border border-slate-600 hover:border-primary-500 rounded-lg transition-colors"
              >
                <Upload className="w-10 h-10 text-primary-400 mb-3" />
                <span className="font-medium text-white">Upload Files</span>
                <span className="text-xs text-slate-400 mt-1">ZIP or tar.gz archive</span>
              </button>

              <button
                onClick={() => handleSourceSelect('docker')}
                className="flex flex-col items-center p-6 bg-slate-700/50 hover:bg-slate-700 border border-slate-600 hover:border-primary-500 rounded-lg transition-colors"
              >
                <FileCode className="w-10 h-10 text-primary-400 mb-3" />
                <span className="font-medium text-white">Docker Compose</span>
                <span className="text-xs text-slate-400 mt-1">Custom Docker configuration</span>
              </button>
            </div>
          </div>
        )}

        {/* Step 2: Configure Source */}
        {step === 'configure' && (
          <div>
            <h2 className="text-lg font-medium text-white mb-4">Configure Source</h2>
            
            {config.sourceType === 'git' && (
              <div className="space-y-4">
                <div>
                  <label className="block text-sm font-medium text-slate-300 mb-1">
                    Git Repository URL
                  </label>
                  <input
                    type="text"
                    value={config.gitUrl}
                    onChange={(e) => updateConfig('gitUrl', e.target.value)}
                    placeholder="https://github.com/username/repository"
                    className="w-full px-3 py-2 bg-slate-700 border border-slate-600 rounded-md text-white placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-primary-500"
                  />
                </div>

                <div>
                  <label className="block text-sm font-medium text-slate-300 mb-1">
                    Branch
                  </label>
                  <div className="flex gap-2">
                    <input
                      type="text"
                      value={config.branch}
                      onChange={(e) => updateConfig('branch', e.target.value)}
                      placeholder="main"
                      className="flex-1 px-3 py-2 bg-slate-700 border border-slate-600 rounded-md text-white placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-primary-500"
                    />
                    <select
                      value={config.branch}
                      onChange={(e) => updateConfig('branch', e.target.value)}
                      className="px-3 py-2 bg-slate-700 border border-slate-600 rounded-md text-white"
                    >
                      <option value="main">main</option>
                      <option value="master">master</option>
                      <option value="develop">develop</option>
                    </select>
                  </div>
                </div>

                <div>
                  <label className="block text-sm font-medium text-slate-300 mb-1">
                    Build Strategy
                  </label>
                  <select
                    value={config.buildStrategy}
                    onChange={(e) => updateConfig('buildStrategy', e.target.value as AppConfig['buildStrategy'])}
                    className="w-full px-3 py-2 bg-slate-700 border border-slate-600 rounded-md text-white"
                  >
                    <option value="auto">Auto-detect</option>
                    <option value="nixpacks">Nixpacks</option>
                    <option value="dockerfile">Dockerfile</option>
                    <option value="static">Static Site</option>
                  </select>
                  <p className="text-xs text-slate-500 mt-1">
                    {config.buildStrategy === 'auto' && 'Automatically detect the best build strategy based on your project files'}
                    {config.buildStrategy === 'nixpacks' && 'Use Nixpacks for Node.js, Python, Go, and more'}
                    {config.buildStrategy === 'dockerfile' && 'Build using a Dockerfile in your repository'}
                    {config.buildStrategy === 'static' && 'Serve as static files without a build step'}
                  </p>
                </div>
              </div>
            )}

            {config.sourceType === 'upload' && (
              <div className="text-center py-8">
                <Upload className="w-12 h-12 text-slate-400 mx-auto mb-4" />
                <p className="text-slate-400 mb-2">Drag and drop files here or click to upload</p>
                <p className="text-xs text-slate-500">Supported: .zip, .tar.gz, .tar</p>
                <button className="mt-4 px-4 py-2 bg-slate-700 hover:bg-slate-600 text-white rounded-md text-sm">
                  Browse Files
                </button>
              </div>
            )}

            {config.sourceType === 'docker' && (
              <div>
                <label className="block text-sm font-medium text-slate-300 mb-1">
                  docker-compose.yml Content
                </label>
                <textarea
                  placeholder="Paste your docker-compose.yml content here..."
                  rows={10}
                  className="w-full px-3 py-2 bg-slate-700 border border-slate-600 rounded-md text-white font-mono text-sm placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-primary-500"
                />
              </div>
            )}
          </div>
        )}

        {/* Step 3: Basic Configuration */}
        {step === 'basic' && (
          <div>
            <h2 className="text-lg font-medium text-white mb-4">Basic Configuration</h2>
            
            <div className="space-y-4">
              <div>
                <label className="block text-sm font-medium text-slate-300 mb-1">
                  App Name
                </label>
                <input
                  type="text"
                  value={config.appName}
                  onChange={(e) => updateConfig('appName', e.target.value.toLowerCase().replace(/[^a-z0-9-]/g, '-'))}
                  placeholder="my-awesome-app"
                  className="w-full px-3 py-2 bg-slate-700 border border-slate-600 rounded-md text-white placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-primary-500"
                />
                <p className="text-xs text-slate-500 mt-1">
                  Lowercase letters, numbers, and hyphens only
                </p>
              </div>

              <div>
                <label className="block text-sm font-medium text-slate-300 mb-1">
                  Domain (optional)
                </label>
                <input
                  type="text"
                  value={config.domain}
                  onChange={(e) => updateConfig('domain', e.target.value)}
                  placeholder="my-app.example.com"
                  className="w-full px-3 py-2 bg-slate-700 border border-slate-600 rounded-md text-white placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-primary-500"
                />
                <p className="text-xs text-slate-500 mt-1">
                  Leave empty to use auto-generated subdomain
                </p>
              </div>

              <div>
                <label className="block text-sm font-medium text-slate-300 mb-2">
                  Environment Variables
                </label>
                
                {/* Existing env vars */}
                <div className="space-y-2 mb-4">
                  {config.envVars.map((envVar) => (
                    <div key={envVar.key} className="flex items-center gap-2 p-2 bg-slate-700/50 rounded">
                      <span className="text-primary-400 font-mono text-sm w-32">{envVar.key}</span>
                      <span className="text-slate-300 font-mono text-sm flex-1 truncate">
                        {envVar.secret ? '••••••••' : envVar.value}
                      </span>
                      <button
                        onClick={() => handleRemoveEnvVar(envVar.key)}
                        className="p-1 text-slate-400 hover:text-red-400"
                      >
                        <Trash2 className="w-4 h-4" />
                      </button>
                    </div>
                  ))}
                </div>

                {/* Add new env var */}
                <div className="flex items-center gap-2">
                  <input
                    type="text"
                    value={newEnvKey}
                    onChange={(e) => setNewEnvKey(e.target.value.toUpperCase())}
                    placeholder="KEY"
                    className="w-32 px-2 py-1.5 bg-slate-700 border border-slate-600 rounded text-sm text-white placeholder-slate-400"
                  />
                  <input
                    type="text"
                    value={newEnvValue}
                    onChange={(e) => setNewEnvValue(e.target.value)}
                    placeholder="VALUE"
                    className="flex-1 px-2 py-1.5 bg-slate-700 border border-slate-600 rounded text-sm text-white placeholder-slate-400"
                  />
                  <label className="flex items-center gap-1 text-xs text-slate-400">
                    <input
                      type="checkbox"
                      checked={newEnvSecret}
                      onChange={(e) => setNewEnvSecret(e.target.checked)}
                      className="rounded"
                    />
                    Secret
                  </label>
                  <button
                    onClick={handleAddEnvVar}
                    disabled={!newEnvKey.trim()}
                    className="p-1.5 text-primary-400 hover:text-primary-300 disabled:opacity-50"
                  >
                    <Plus className="w-4 h-4" />
                  </button>
                </div>
              </div>
            </div>
          </div>
        )}

        {/* Step 4: Review */}
        {step === 'review' && (
          <div>
            <h2 className="text-lg font-medium text-white mb-4">Review Configuration</h2>
            
            <div className="space-y-4">
              <div className="bg-slate-700/50 rounded-lg p-4">
                <dl className="space-y-2">
                  <div className="flex justify-between">
                    <dt className="text-slate-400">Source</dt>
                    <dd className="text-white">
                      {config.sourceType === 'git' && `${config.gitUrl} (${config.branch})`}
                      {config.sourceType === 'upload' && 'Uploaded files'}
                      {config.sourceType === 'docker' && 'Docker Compose'}
                    </dd>
                  </div>
                  <div className="flex justify-between">
                    <dt className="text-slate-400">Build Strategy</dt>
                    <dd className="text-white capitalize">{config.buildStrategy}</dd>
                  </div>
                  <div className="flex justify-between">
                    <dt className="text-slate-400">App Name</dt>
                    <dd className="text-white">{config.appName}</dd>
                  </div>
                  <div className="flex justify-between">
                    <dt className="text-slate-400">Domain</dt>
                    <dd className="text-white">{config.domain || 'Auto-generated subdomain'}</dd>
                  </div>
                  <div className="flex justify-between">
                    <dt className="text-slate-400">Environment Variables</dt>
                    <dd className="text-white">{config.envVars.length} variable{config.envVars.length !== 1 ? 's' : ''}</dd>
                  </div>
                </dl>
              </div>

              <label className="flex items-center gap-2 text-sm text-slate-300">
                <input type="checkbox" defaultChecked className="rounded" />
                Auto-deploy on future pushes to {config.branch || 'main'} branch
              </label>
            </div>
          </div>
        )}

        {/* Navigation */}
        <div className="flex justify-between mt-8 pt-6 border-t border-slate-700">
          <button
            onClick={() => {
              const stepIndex = steps.findIndex(s => s.key === step)
              if (stepIndex > 0) {
                setStep(steps[stepIndex - 1].key)
              } else {
                router.push('/apps')
              }
            }}
            className="flex items-center gap-2 px-4 py-2 text-slate-400 hover:text-white transition-colors"
          >
            <ChevronLeft className="w-4 h-4" />
            {steps.findIndex(s => s.key === step) === 0 ? 'Cancel' : 'Back'}
          </button>

          {step !== 'review' ? (
            <button
              onClick={() => {
                const stepIndex = steps.findIndex(s => s.key === step)
                setStep(steps[stepIndex + 1].key)
              }}
              disabled={!canProceed()}
              className="flex items-center gap-2 px-4 py-2 bg-primary-600 hover:bg-primary-700 text-white rounded-md font-medium transition-colors disabled:opacity-50"
            >
              Continue
              <ChevronRight className="w-4 h-4" />
            </button>
          ) : (
            <button
              onClick={handleCreate}
              disabled={createAppMutation.isPending || !canProceed()}
              className="flex items-center gap-2 px-4 py-2 bg-primary-600 hover:bg-primary-700 text-white rounded-md font-medium transition-colors disabled:opacity-50"
            >
              {createAppMutation.isPending ? (
                <>
                  <Loader2 className="w-4 h-4 animate-spin" />
                  Creating...
                </>
              ) : (
                <>
                  Deploy App
                  <ChevronRight className="w-4 h-4" />
                </>
              )}
            </button>
          )}
        </div>
      </div>
    </div>
  )
}
