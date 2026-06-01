'use client'

import { useState, useCallback } from 'react'
import {
  Save,
  TestTube,
  CheckCircle,
  XCircle,
  Loader2,
  HardDrive,
  Link,
  Cloud
} from 'lucide-react'

interface S3Config {
  endpoint: string
  bucket: string
  region: string
  access_key: string
  secret_key: string
  prefix: string
}

export default function BackupSettingsPage() {
  const [s3Config, setS3Config] = useState<S3Config>({
    endpoint: '',
    bucket: '',
    region: 'us-east-1',
    access_key: '',
    secret_key: '',
    prefix: 'backups',
  })
  const [isTesting, setIsTesting] = useState(false)
  const [isSaving, setIsSaving] = useState(false)
  const [testResult, setTestResult] = useState<'success' | 'error' | null>(null)
  const [testMessage, setTestMessage] = useState('')

  const [defaultSchedule, setDefaultSchedule] = useState('daily')
  const [defaultRetention, setDefaultRetention] = useState('7')
  const [enableEncryption, setEnableEncryption] = useState(true)
  const [compressionLevel, setCompressionLevel] = useState('6')

  const handleTestConnection = useCallback(async () => {
    setIsTesting(true)
    setTestResult(null)
    setTestMessage('')
    
    try {
      // In production, this would call the API to test S3 connection
      await new Promise(resolve => setTimeout(resolve, 2000))
      
      // Simulate success/failure
      if (s3Config.bucket && s3Config.access_key) {
        setTestResult('success')
        setTestMessage('Successfully connected to S3 bucket')
      } else {
        setTestResult('error')
        setTestMessage('Failed to connect: Invalid credentials or bucket not found')
      }
    } catch (error) {
      setTestResult('error')
      setTestMessage('Connection test failed: Network error')
    } finally {
      setIsTesting(false)
    }
  }, [s3Config])

  const handleSave = useCallback(async () => {
    setIsSaving(true)
    
    try {
      // In production, this would save to the API
      await new Promise(resolve => setTimeout(resolve, 1000))
      setTestResult('success')
      setTestMessage('Settings saved successfully')
    } catch (error) {
      setTestResult('error')
      setTestMessage('Failed to save settings')
    } finally {
      setIsSaving(false)
    }
  }, [])

  return (
    <div className="min-h-screen bg-slate-900">
      {/* Header */}
      <div className="px-6 py-4 bg-slate-800 border-b border-slate-700">
        <h1 className="text-xl font-semibold text-white">Backup Settings</h1>
        <p className="text-sm text-slate-400 mt-1">
          Configure backup destinations and default settings
        </p>
      </div>

      <div className="p-6 max-w-4xl space-y-8">
        {/* S3 Configuration */}
        <section className="bg-slate-800 rounded-lg border border-slate-700 overflow-hidden">
          <div className="px-6 py-4 border-b border-slate-700 flex items-center gap-3">
            <div className="p-2 bg-orange-500/20 rounded-lg">
              <Cloud className="w-5 h-5 text-orange-500" />
            </div>
            <div>
              <h2 className="font-medium text-white">S3-Compatible Storage</h2>
              <p className="text-sm text-slate-400">Configure S3 destination for backup storage</p>
            </div>
          </div>
          
          <div className="p-6 space-y-4">
            <div className="grid grid-cols-2 gap-4">
              <div>
                <label className="block text-sm font-medium text-slate-300 mb-1">
                  Endpoint
                </label>
                <input
                  type="text"
                  value={s3Config.endpoint}
                  onChange={(e) => setS3Config(prev => ({ ...prev, endpoint: e.target.value }))}
                  placeholder="https://s3.amazonaws.com"
                  className="w-full px-3 py-2 bg-slate-700 text-white rounded border border-slate-600 focus:outline-none focus:border-primary-500"
                />
                <p className="mt-1 text-xs text-slate-500">
                  Leave empty for AWS S3, or enter custom endpoint for MinIO, Backblaze, etc.
                </p>
              </div>
              
              <div>
                <label className="block text-sm font-medium text-slate-300 mb-1">
                  Bucket Name
                </label>
                <input
                  type="text"
                  value={s3Config.bucket}
                  onChange={(e) => setS3Config(prev => ({ ...prev, bucket: e.target.value }))}
                  placeholder="my-backup-bucket"
                  className="w-full px-3 py-2 bg-slate-700 text-white rounded border border-slate-600 focus:outline-none focus:border-primary-500"
                />
              </div>
              
              <div>
                <label className="block text-sm font-medium text-slate-300 mb-1">
                  Region
                </label>
                <select
                  value={s3Config.region}
                  onChange={(e) => setS3Config(prev => ({ ...prev, region: e.target.value }))}
                  className="w-full px-3 py-2 bg-slate-700 text-white rounded border border-slate-600 focus:outline-none focus:border-primary-500"
                >
                  <option value="us-east-1">US East (N. Virginia)</option>
                  <option value="us-west-2">US West (Oregon)</option>
                  <option value="eu-west-1">EU (Ireland)</option>
                  <option value="eu-central-1">EU (Frankfurt)</option>
                  <option value="ap-northeast-1">Asia Pacific (Tokyo)</option>
                  <option value="ap-southeast-1">Asia Pacific (Singapore)</option>
                </select>
              </div>
              
              <div>
                <label className="block text-sm font-medium text-slate-300 mb-1">
                  Path Prefix
                </label>
                <input
                  type="text"
                  value={s3Config.prefix}
                  onChange={(e) => setS3Config(prev => ({ ...prev, prefix: e.target.value }))}
                  placeholder="backups"
                  className="w-full px-3 py-2 bg-slate-700 text-white rounded border border-slate-600 focus:outline-none focus:border-primary-500"
                />
                <p className="mt-1 text-xs text-slate-500">
                  Prefix for all backup files in the bucket
                </p>
              </div>
              
              <div>
                <label className="block text-sm font-medium text-slate-300 mb-1">
                  Access Key ID
                </label>
                <input
                  type="text"
                  value={s3Config.access_key}
                  onChange={(e) => setS3Config(prev => ({ ...prev, access_key: e.target.value }))}
                  placeholder="AKIAIOSFODNN7EXAMPLE"
                  className="w-full px-3 py-2 bg-slate-700 text-white rounded border border-slate-600 focus:outline-none focus:border-primary-500"
                />
              </div>
              
              <div>
                <label className="block text-sm font-medium text-slate-300 mb-1">
                  Secret Access Key
                </label>
                <input
                  type="password"
                  value={s3Config.secret_key}
                  onChange={(e) => setS3Config(prev => ({ ...prev, secret_key: e.target.value }))}
                  placeholder="••••••••••••••••"
                  className="w-full px-3 py-2 bg-slate-700 text-white rounded border border-slate-600 focus:outline-none focus:border-primary-500"
                />
              </div>
            </div>
            
            <div className="flex items-center gap-4 pt-4">
              <button
                onClick={handleTestConnection}
                disabled={isTesting || !s3Config.bucket || !s3Config.access_key}
                className="flex items-center gap-2 px-4 py-2 text-sm bg-slate-700 hover:bg-slate-600 text-white rounded border border-slate-600 transition-colors disabled:opacity-50"
              >
                {isTesting ? (
                  <Loader2 className="w-4 h-4 animate-spin" />
                ) : (
                  <TestTube className="w-4 h-4" />
                )}
                Test Connection
              </button>
              
              {testResult && (
                <div className={`flex items-center gap-2 ${testResult === 'success' ? 'text-green-500' : 'text-red-500'}`}>
                  {testResult === 'success' ? (
                    <CheckCircle className="w-4 h-4" />
                  ) : (
                    <XCircle className="w-4 h-4" />
                  )}
                  <span className="text-sm">{testMessage}</span>
                </div>
              )}
            </div>
          </div>
        </section>

        {/* Default Settings */}
        <section className="bg-slate-800 rounded-lg border border-slate-700 overflow-hidden">
          <div className="px-6 py-4 border-b border-slate-700 flex items-center gap-3">
            <div className="p-2 bg-blue-500/20 rounded-lg">
              <HardDrive className="w-5 h-5 text-blue-500" />
            </div>
            <div>
              <h2 className="font-medium text-white">Default Backup Settings</h2>
              <p className="text-sm text-slate-400">Configure default behavior for new backups</p>
            </div>
          </div>
          
          <div className="p-6 space-y-4">
            <div className="grid grid-cols-2 gap-4">
              <div>
                <label className="block text-sm font-medium text-slate-300 mb-1">
                  Default Schedule
                </label>
                <select
                  value={defaultSchedule}
                  onChange={(e) => setDefaultSchedule(e.target.value)}
                  className="w-full px-3 py-2 bg-slate-700 text-white rounded border border-slate-600 focus:outline-none focus:border-primary-500"
                >
                  <option value="hourly">Every hour</option>
                  <option value="daily">Daily at 2:00 AM</option>
                  <option value="weekly">Weekly (Sunday midnight)</option>
                  <option value="monthly">Monthly (1st of month)</option>
                </select>
              </div>
              
              <div>
                <label className="block text-sm font-medium text-slate-300 mb-1">
                  Default Retention
                </label>
                <select
                  value={defaultRetention}
                  onChange={(e) => setDefaultRetention(e.target.value)}
                  className="w-full px-3 py-2 bg-slate-700 text-white rounded border border-slate-600 focus:outline-none focus:border-primary-500"
                >
                  <option value="1">Keep for 1 day</option>
                  <option value="7">Keep for 7 days</option>
                  <option value="14">Keep for 14 days</option>
                  <option value="30">Keep for 30 days</option>
                  <option value="90">Keep for 90 days</option>
                </select>
              </div>
              
              <div>
                <label className="block text-sm font-medium text-slate-300 mb-1">
                  Compression Level
                </label>
                <select
                  value={compressionLevel}
                  onChange={(e) => setCompressionLevel(e.target.value)}
                  className="w-full px-3 py-2 bg-slate-700 text-white rounded border border-slate-600 focus:outline-none focus:border-primary-500"
                >
                  <option value="1">Fastest (Low compression)</option>
                  <option value="6">Balanced</option>
                  <option value="9">Best compression (Slow)</option>
                </select>
              </div>
              
              <div>
                <label className="block text-sm font-medium text-slate-300 mb-1">
                  Default Destination
                </label>
                <select
                  className="w-full px-3 py-2 bg-slate-700 text-white rounded border border-slate-600 focus:outline-none focus:border-primary-500"
                >
                  <option value="local">Local only</option>
                  <option value="s3">S3 (if configured)</option>
                  <option value="both">Local + S3</option>
                </select>
              </div>
            </div>
            
            <div className="pt-4">
              <label className="flex items-center gap-3 cursor-pointer">
                <input
                  type="checkbox"
                  checked={enableEncryption}
                  onChange={(e) => setEnableEncryption(e.target.checked)}
                  className="w-4 h-4 rounded border-slate-600 bg-slate-700 text-primary-600 focus:ring-primary-500"
                />
                <div>
                  <span className="text-sm text-white">Enable encryption</span>
                  <p className="text-xs text-slate-400">Encrypt backups before uploading to S3</p>
                </div>
              </label>
            </div>
          </div>
        </section>

        {/* Save Button */}
        <div className="flex justify-end">
          <button
            onClick={handleSave}
            disabled={isSaving}
            className="flex items-center gap-2 px-6 py-2 bg-primary-600 hover:bg-primary-700 text-white rounded-lg transition-colors disabled:opacity-50"
          >
            {isSaving ? (
              <Loader2 className="w-4 h-4 animate-spin" />
            ) : (
              <Save className="w-4 h-4" />
            )}
            Save Settings
          </button>
        </div>
      </div>
    </div>
  )
}
