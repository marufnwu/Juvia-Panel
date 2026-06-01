'use client'

import { useState, useCallback } from 'react'
import { X, Clock, AlertCircle } from 'lucide-react'

interface CronJobModalProps {
  isOpen: boolean
  onClose: () => void
  onSave: (data: CronJobData) => Promise<void>
  initialData?: CronJobData
}

export interface CronJobData {
  name: string
  target: string
  target_type: 'host' | 'app' | 'service'
  schedule: {
    minute: string
    hour: string
    day: string
    month: string
    weekday: string
  }
  command: string
  notify_on_failure: boolean
  log_retention: boolean
}

const schedulePresets = [
  { label: 'Every minute', value: '* * * * *' },
  { label: 'Every 5 minutes', value: '*/5 * * * *' },
  { label: 'Every hour', value: '0 * * * *' },
  { label: 'Every day at midnight', value: '0 0 * * *' },
  { label: 'Every day at 2 AM', value: '0 2 * * *' },
  { label: 'Every Sunday at midnight', value: '0 0 * * 0' },
  { label: 'First of month', value: '0 0 1 * *' },
]

const targetOptions = [
  { id: 'host', name: 'Host Server', type: 'host' as const },
  { id: 'app:api-prod', name: 'api-prod', type: 'app' as const },
  { id: 'app:web-client', name: 'web-client', type: 'app' as const },
  { id: 'svc:main-pg', name: 'main-pg (PostgreSQL)', type: 'service' as const },
]

export function CronJobModal({ isOpen, onClose, onSave, initialData }: CronJobModalProps) {
  const [name, setName] = useState(initialData?.name || '')
  const [target, setTarget] = useState(initialData?.target || 'host')
  const [command, setCommand] = useState(initialData?.command || '')
  const [notifyOnFailure, setNotifyOnFailure] = useState(initialData?.notify_on_failure ?? true)
  const [logRetention, setLogRetention] = useState(initialData?.log_retention ?? true)
  const [scheduleMode, setScheduleMode] = useState<'preset' | 'custom'>('preset')
  const [selectedPreset, setSelectedPreset] = useState(schedulePresets[3].value)
  const [customMinute, setCustomMinute] = useState('0')
  const [customHour, setCustomHour] = useState('2')
  const [customDay, setCustomDay] = useState('*')
  const [customMonth, setCustomMonth] = useState('*')
  const [customWeekday, setCustomWeekday] = useState('*')
  const [isSaving, setIsSaving] = useState(false)
  const [errors, setErrors] = useState<Record<string, string>>({})

  const getSchedule = useCallback(() => {
    if (scheduleMode === 'preset') {
      return selectedPreset
    }
    return `${customMinute} ${customHour} ${customDay} ${customMonth} ${customWeekday}`
  }, [scheduleMode, selectedPreset, customMinute, customHour, customDay, customMonth, customWeekday])

  const getScheduleDescription = useCallback((cron: string) => {
    const preset = schedulePresets.find(p => p.value === cron)
    if (preset) return preset.label

    // Simple description generation
    const parts = cron.split(' ')
    if (parts.length !== 5) return cron

    const [min, hour, day, month, wday] = parts
    let desc = []

    if (min === '*' && hour === '*') desc.push('Every minute')
    else if (min.startsWith('*/')) desc.push(`Every ${min.slice(2)} minutes`)
    else if (hour === '*') desc.push(`At minute ${min}`)
    else desc.push(`At ${hour}:${min.padStart(2, '0')}`)

    if (day !== '*') desc.push(`on day ${day}`)
    if (month !== '*') desc.push(`in month ${month}`)
    if (wday !== '*') {
      const days = ['Sunday', 'Monday', 'Tuesday', 'Wednesday', 'Thursday', 'Friday', 'Saturday']
      desc.push(`on ${days[parseInt(wday)] || wday}`)
    }

    return desc.join(' ') || cron
  }, [])

  const validate = useCallback(() => {
    const newErrors: Record<string, string> = {}

    if (!name.trim()) {
      newErrors.name = 'Name is required'
    }

    if (!command.trim()) {
      newErrors.command = 'Command is required'
    }

    setErrors(newErrors)
    return Object.keys(newErrors).length === 0
  }, [name, command])

  const handleSave = useCallback(async () => {
    if (!validate()) return

    setIsSaving(true)

    const targetInfo = targetOptions.find(t => t.id === target)
    const schedule = getSchedule()

    try {
      await onSave({
        name: name.trim(),
        target,
        target_type: targetInfo?.type || 'host',
        schedule: {
          minute: scheduleMode === 'preset' ? selectedPreset.split(' ')[0] : customMinute,
          hour: scheduleMode === 'preset' ? selectedPreset.split(' ')[1] : customHour,
          day: scheduleMode === 'preset' ? selectedPreset.split(' ')[2] : customDay,
          month: scheduleMode === 'preset' ? selectedPreset.split(' ')[3] : customMonth,
          weekday: scheduleMode === 'preset' ? selectedPreset.split(' ')[4] : customWeekday,
        },
        command: command.trim(),
        notify_on_failure: notifyOnFailure,
        log_retention: logRetention,
      })
      onClose()
    } catch (error) {
      console.error('Failed to save cron job:', error)
    } finally {
      setIsSaving(false)
    }
  }, [validate, onSave, onClose, name, target, getSchedule, scheduleMode, selectedPreset, customMinute, customHour, customDay, customMonth, customWeekday, command, notifyOnFailure, logRetention])

  if (!isOpen) return null

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50">
      <div className="bg-slate-800 border border-slate-700 rounded-lg w-[550px] max-h-[90vh] overflow-y-auto shadow-xl">
        {/* Header */}
        <div className="flex items-center justify-between px-6 py-4 border-b border-slate-700">
          <div className="flex items-center gap-3">
            <div className="p-2 bg-primary-500/20 rounded-lg">
              <Clock className="w-5 h-5 text-primary-500" />
            </div>
            <div>
              <h2 className="text-lg font-semibold text-white">
                {initialData ? 'Edit Cron Job' : 'New Cron Job'}
              </h2>
            </div>
          </div>
          <button
            onClick={onClose}
            className="text-slate-400 hover:text-white transition-colors"
          >
            <X className="w-5 h-5" />
          </button>
        </div>

        {/* Content */}
        <div className="p-6 space-y-6">
          {/* Name */}
          <div>
            <label className="block text-sm font-medium text-slate-300 mb-1">
              Name
            </label>
            <input
              type="text"
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder="db-cleanup"
              className={`w-full px-3 py-2 bg-slate-700 text-white rounded border ${
                errors.name ? 'border-red-500' : 'border-slate-600'
              } focus:outline-none focus:border-primary-500`}
            />
            {errors.name && (
              <p className="mt-1 text-xs text-red-400 flex items-center gap-1">
                <AlertCircle className="w-3 h-3" />
                {errors.name}
              </p>
            )}
          </div>

          {/* Target */}
          <div>
            <label className="block text-sm font-medium text-slate-300 mb-1">
              Target
            </label>
            <select
              value={target}
              onChange={(e) => setTarget(e.target.value)}
              className="w-full px-3 py-2 bg-slate-700 text-white rounded border border-slate-600 focus:outline-none focus:border-primary-500"
            >
              {targetOptions.map(opt => (
                <option key={opt.id} value={opt.id}>
                  {opt.name} ({opt.type})
                </option>
              ))}
            </select>
          </div>

          {/* Schedule */}
          <div>
            <label className="block text-sm font-medium text-slate-300 mb-2">
              Schedule
            </label>
            
            <div className="flex items-center gap-4 mb-3">
              <label className="flex items-center gap-2 cursor-pointer">
                <input
                  type="radio"
                  checked={scheduleMode === 'preset'}
                  onChange={() => setScheduleMode('preset')}
                  className="text-primary-600"
                />
                <span className="text-sm text-white">Preset</span>
              </label>
              <label className="flex items-center gap-2 cursor-pointer">
                <input
                  type="radio"
                  checked={scheduleMode === 'custom'}
                  onChange={() => setScheduleMode('custom')}
                  className="text-primary-600"
                />
                <span className="text-sm text-white">Custom (cron expression)</span>
              </label>
            </div>

            {scheduleMode === 'preset' ? (
              <select
                value={selectedPreset}
                onChange={(e) => setSelectedPreset(e.target.value)}
                className="w-full px-3 py-2 bg-slate-700 text-white rounded border border-slate-600 focus:outline-none focus:border-primary-500"
              >
                {schedulePresets.map(preset => (
                  <option key={preset.value} value={preset.value}>
                    {preset.label} ({preset.value})
                  </option>
                ))}
              </select>
            ) : (
              <div className="space-y-3">
                <div className="grid grid-cols-5 gap-2">
                  <div>
                    <label className="block text-xs text-slate-500 mb-1">Minute</label>
                    <input
                      type="text"
                      value={customMinute}
                      onChange={(e) => setCustomMinute(e.target.value)}
                      className="w-full px-2 py-1 bg-slate-700 text-white text-sm rounded border border-slate-600 focus:outline-none focus:border-primary-500"
                    />
                  </div>
                  <div>
                    <label className="block text-xs text-slate-500 mb-1">Hour</label>
                    <input
                      type="text"
                      value={customHour}
                      onChange={(e) => setCustomHour(e.target.value)}
                      className="w-full px-2 py-1 bg-slate-700 text-white text-sm rounded border border-slate-600 focus:outline-none focus:border-primary-500"
                    />
                  </div>
                  <div>
                    <label className="block text-xs text-slate-500 mb-1">Day</label>
                    <input
                      type="text"
                      value={customDay}
                      onChange={(e) => setCustomDay(e.target.value)}
                      className="w-full px-2 py-1 bg-slate-700 text-white text-sm rounded border border-slate-600 focus:outline-none focus:border-primary-500"
                    />
                  </div>
                  <div>
                    <label className="block text-xs text-slate-500 mb-1">Month</label>
                    <input
                      type="text"
                      value={customMonth}
                      onChange={(e) => setCustomMonth(e.target.value)}
                      className="w-full px-2 py-1 bg-slate-700 text-white text-sm rounded border border-slate-600 focus:outline-none focus:border-primary-500"
                    />
                  </div>
                  <div>
                    <label className="block text-xs text-slate-500 mb-1">Weekday</label>
                    <input
                      type="text"
                      value={customWeekday}
                      onChange={(e) => setCustomWeekday(e.target.value)}
                      className="w-full px-2 py-1 bg-slate-700 text-white text-sm rounded border border-slate-600 focus:outline-none focus:border-primary-500"
                    />
                  </div>
                </div>
                <p className="text-xs text-slate-400">
                  Schedule: <span className="text-primary-400 font-mono">{getSchedule()}</span>
                </p>
              </div>
            )}

            <p className="mt-2 text-sm text-slate-400">
              {getScheduleDescription(getSchedule())}
            </p>
          </div>

          {/* Command */}
          <div>
            <label className="block text-sm font-medium text-slate-300 mb-1">
              Command
            </label>
            <textarea
              value={command}
              onChange={(e) => setCommand(e.target.value)}
              placeholder="npm run cleanup"
              rows={3}
              className={`w-full px-3 py-2 bg-slate-700 text-white rounded border font-mono text-sm ${
                errors.command ? 'border-red-500' : 'border-slate-600'
              } focus:outline-none focus:border-primary-500 resize-none`}
            />
            {errors.command && (
              <p className="mt-1 text-xs text-red-400 flex items-center gap-1">
                <AlertCircle className="w-3 h-3" />
                {errors.command}
              </p>
            )}
          </div>

          {/* Options */}
          <div className="space-y-3">
            <label className="flex items-center gap-3 cursor-pointer">
              <input
                type="checkbox"
                checked={notifyOnFailure}
                onChange={(e) => setNotifyOnFailure(e.target.checked)}
                className="w-4 h-4 rounded border-slate-600 bg-slate-700 text-primary-600"
              />
              <div>
                <span className="text-sm text-white">Notify on failure</span>
                <p className="text-xs text-slate-400">Send notification when job fails</p>
              </div>
            </label>
            
            <label className="flex items-center gap-3 cursor-pointer">
              <input
                type="checkbox"
                checked={logRetention}
                onChange={(e) => setLogRetention(e.target.checked)}
                className="w-4 h-4 rounded border-slate-600 bg-slate-700 text-primary-600"
              />
              <div>
                <span className="text-sm text-white">Log output</span>
                <p className="text-xs text-slate-400">Retain last 10 run logs</p>
              </div>
            </label>
          </div>
        </div>

        {/* Footer */}
        <div className="flex justify-end gap-3 px-6 py-4 border-t border-slate-700">
          <button
            onClick={onClose}
            className="px-4 py-2 text-slate-300 hover:text-white hover:bg-slate-700 rounded transition-colors"
          >
            Cancel
          </button>
          <button
            onClick={handleSave}
            disabled={isSaving}
            className="px-4 py-2 bg-primary-600 hover:bg-primary-700 text-white rounded transition-colors disabled:opacity-50"
          >
            {isSaving ? 'Saving...' : initialData ? 'Save Changes' : 'Create Cron Job'}
          </button>
        </div>
      </div>
    </div>
  )
}
