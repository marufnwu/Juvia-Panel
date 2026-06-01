'use client'

import { useState } from 'react'
import { useMutation } from '@tanstack/react-query'
import {
  Settings as SettingsIcon,
  Server,
  User,
  Key,
  Bell,
  Shield,
  Loader2,
  Check,
  Copy,
  Eye,
  EyeOff,
  Smartphone,
  Monitor,
  Sun,
  Moon
} from 'lucide-react'
import { api, ApiError } from '@/lib/api'
import { useToastStore, useAuthStore } from '@/stores'
import { TwoFactorSetup } from '@/components/settings/TwoFactorSetup'
import { TwoFactorDisable } from '@/components/settings/TwoFactorDisable'
import { APIKeyList } from '@/components/settings/APIKeyList'

type Tab = 'panel' | 'server' | 'user' | 'api' | 'notifications' | 'security'

const tabs: { id: Tab; label: string; icon: React.ComponentType<{ className?: string }> }[] = [
  { id: 'panel', label: 'Panel', icon: SettingsIcon },
  { id: 'server', label: 'Server', icon: Server },
  { id: 'user', label: 'User', icon: User },
  { id: 'api', label: 'API Keys', icon: Key },
  { id: 'notifications', label: 'Notifications', icon: Bell },
  { id: 'security', label: 'Security', icon: Shield },
]

// Panel Settings Tab
function PanelSettings() {
  const { addToast } = useToastStore()
  const [panelName, setPanelName] = useState('Juvia Panel')
  const [domain, setDomain] = useState('panel.example.com')
  const [subdomainPattern, setSubdomainPattern] = useState('{app}.panel.example.com')
  const [autoUpdate, setAutoUpdate] = useState(true)

  const mutation = useMutation({
    mutationFn: (data: { name: string; domain: string; subdomain_pattern: string; auto_update: boolean }) =>
      api.apps.create(data as any),
    onSuccess: () => {
      addToast({ type: 'success', title: 'Settings saved', message: 'Panel settings have been updated.' })
    },
    onError: (error: ApiError) => {
      addToast({ type: 'error', title: 'Failed to save', message: error.message })
    },
  })

  const handleSave = () => {
    mutation.mutate({ name: panelName, domain, subdomain_pattern: subdomainPattern, auto_update: autoUpdate })
  }

  return (
    <div className="space-y-6">
      <div>
        <h3 className="text-lg font-medium text-white mb-4">Panel Settings</h3>
      </div>

      {/* Panel Name */}
      <div>
        <label className="block text-sm font-medium text-slate-300 mb-2">Panel Name</label>
        <input
          type="text"
          value={panelName}
          onChange={(e) => setPanelName(e.target.value)}
          className="w-full max-w-md px-3 py-2 bg-slate-900 border border-slate-700 rounded-md text-white focus:outline-none focus:ring-2 focus:ring-primary-500"
        />
      </div>

      {/* Domain */}
      <div>
        <label className="block text-sm font-medium text-slate-300 mb-2">Panel Domain</label>
        <input
          type="text"
          value={domain}
          onChange={(e) => setDomain(e.target.value)}
          className="w-full max-w-md px-3 py-2 bg-slate-900 border border-slate-700 rounded-md text-white focus:outline-none focus:ring-2 focus:ring-primary-500"
        />
        <p className="mt-1 text-xs text-slate-500">The domain where the panel is accessible</p>
      </div>

      {/* Subdomain Pattern */}
      <div>
        <label className="block text-sm font-medium text-slate-300 mb-2">App Subdomain Pattern</label>
        <input
          type="text"
          value={subdomainPattern}
          onChange={(e) => setSubdomainPattern(e.target.value)}
          className="w-full max-w-md px-3 py-2 bg-slate-900 border border-slate-700 rounded-md text-white focus:outline-none focus:ring-2 focus:ring-primary-500"
        />
        <p className="mt-1 text-xs text-slate-500">Use {"{app}"} as placeholder for app name</p>
      </div>

      {/* Auto Update */}
      <div className="flex items-center gap-3">
        <label className="relative inline-flex items-center cursor-pointer">
          <input
            type="checkbox"
            checked={autoUpdate}
            onChange={(e) => setAutoUpdate(e.target.checked)}
            className="sr-only peer"
          />
          <div className="w-11 h-6 bg-slate-700 peer-focus:ring-2 peer-focus:ring-primary-500 rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-primary-600"></div>
        </label>
        <span className="text-sm text-slate-300">Automatically check for updates</span>
      </div>

      {/* Save Button */}
      <div className="pt-4 border-t border-slate-700">
        <button
          onClick={handleSave}
          disabled={mutation.isPending}
          className="flex items-center gap-2 px-4 py-2 bg-primary-600 hover:bg-primary-700 disabled:bg-primary-500/50 text-white rounded-md text-sm font-medium transition-colors"
        >
          {mutation.isPending ? <Loader2 className="w-4 h-4 animate-spin" /> : <Check className="w-4 h-4" />}
          Save Changes
        </button>
      </div>
    </div>
  )
}

// Server Settings Tab
function ServerSettings() {
  const { addToast } = useToastStore()
  const [serverName, setServerName] = useState('my-vps-01')
  const [timezone, setTimezone] = useState('UTC')
  const [rebooting, setRebooting] = useState(false)

  const handleReboot = async () => {
    if (!confirm('Are you sure you want to reboot the server? This will disconnect all active sessions.')) {
      return
    }
    setRebooting(true)
    // Simulate reboot
    setTimeout(() => {
      addToast({ type: 'info', title: 'Reboot initiated', message: 'Server is rebooting...' })
      setRebooting(false)
    }, 2000)
  }

  return (
    <div className="space-y-6">
      <div>
        <h3 className="text-lg font-medium text-white mb-4">Server Settings</h3>
      </div>

      {/* Server Name */}
      <div>
        <label className="block text-sm font-medium text-slate-300 mb-2">Server Name</label>
        <input
          type="text"
          value={serverName}
          onChange={(e) => setServerName(e.target.value)}
          className="w-full max-w-md px-3 py-2 bg-slate-900 border border-slate-700 rounded-md text-white focus:outline-none focus:ring-2 focus:ring-primary-500"
        />
      </div>

      {/* Timezone */}
      <div>
        <label className="block text-sm font-medium text-slate-300 mb-2">Timezone</label>
        <select
          value={timezone}
          onChange={(e) => setTimezone(e.target.value)}
          className="w-full max-w-md px-3 py-2 bg-slate-900 border border-slate-700 rounded-md text-white focus:outline-none focus:ring-2 focus:ring-primary-500"
        >
          <option value="UTC">UTC</option>
          <option value="America/New_York">America/New_York</option>
          <option value="America/Los_Angeles">America/Los_Angeles</option>
          <option value="Europe/London">Europe/London</option>
          <option value="Asia/Dhaka">Asia/Dhaka</option>
          <option value="Asia/Tokyo">Asia/Tokyo</option>
        </select>
      </div>

      {/* Reboot */}
      <div className="pt-4 border-t border-slate-700">
        <button
          onClick={handleReboot}
          disabled={rebooting}
          className="flex items-center gap-2 px-4 py-2 bg-danger-600 hover:bg-danger-700 disabled:bg-danger-500/50 text-white rounded-md text-sm font-medium transition-colors"
        >
          {rebooting ? <Loader2 className="w-4 h-4 animate-spin" /> : <Server className="w-4 h-4" />}
          Reboot Server
        </button>
        <p className="mt-2 text-xs text-slate-500">This will restart the server and disconnect all active sessions.</p>
      </div>
    </div>
  )
}

// User Settings Tab
function UserSettings() {
  const { user } = useAuthStore()
  const { addToast } = useToastStore()
  const [name, setName] = useState(user?.name || '')
  const [email, setEmail] = useState(user?.email || '')
  const [currentPassword, setCurrentPassword] = useState('')
  const [newPassword, setNewPassword] = useState('')
  const [confirmPassword, setConfirmPassword] = useState('')
  const [showPassword, setShowPassword] = useState(false)
  const [twoFactorEnabled, setTwoFactorEnabled] = useState(false)
  const [twoFactorSetupOpen, setTwoFactorSetupOpen] = useState(false)
  const [twoFactorDisableOpen, setTwoFactorDisableOpen] = useState(false)
  const [theme, setTheme] = useState<'dark' | 'light' | 'system'>('dark')
  const [timeFormat, setTimeFormat] = useState<'12h' | '24h'>('24h')

  const handlePasswordChange = () => {
    if (newPassword !== confirmPassword) {
      addToast({ type: 'error', title: 'Password mismatch', message: 'New passwords do not match.' })
      return
    }
    if (newPassword.length < 8) {
      addToast({ type: 'error', title: 'Password too short', message: 'Password must be at least 8 characters.' })
      return
    }
    addToast({ type: 'success', title: 'Password updated', message: 'Your password has been changed.' })
    setCurrentPassword('')
    setNewPassword('')
    setConfirmPassword('')
  }

  return (
    <div className="space-y-6">
      <div>
        <h3 className="text-lg font-medium text-white mb-4">User Settings</h3>
      </div>

      {/* Profile */}
      <div className="space-y-4">
        <h4 className="text-sm font-medium text-slate-400 uppercase tracking-wider">Profile</h4>
        
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          <div>
            <label className="block text-sm font-medium text-slate-300 mb-2">Name</label>
            <input
              type="text"
              value={name}
              onChange={(e) => setName(e.target.value)}
              className="w-full px-3 py-2 bg-slate-900 border border-slate-700 rounded-md text-white focus:outline-none focus:ring-2 focus:ring-primary-500"
            />
          </div>
          <div>
            <label className="block text-sm font-medium text-slate-300 mb-2">Email</label>
            <input
              type="email"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              className="w-full px-3 py-2 bg-slate-900 border border-slate-700 rounded-md text-white focus:outline-none focus:ring-2 focus:ring-primary-500"
            />
          </div>
        </div>
      </div>

      {/* Password Change */}
      <div className="space-y-4 pt-4 border-t border-slate-700">
        <h4 className="text-sm font-medium text-slate-400 uppercase tracking-wider">Change Password</h4>
        
        <div>
          <label className="block text-sm font-medium text-slate-300 mb-2">Current Password</label>
          <div className="relative">
            <input
              type={showPassword ? 'text' : 'password'}
              value={currentPassword}
              onChange={(e) => setCurrentPassword(e.target.value)}
              className="w-full max-w-md px-3 py-2 bg-slate-900 border border-slate-700 rounded-md text-white focus:outline-none focus:ring-2 focus:ring-primary-500"
            />
          </div>
        </div>

        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          <div>
            <label className="block text-sm font-medium text-slate-300 mb-2">New Password</label>
            <div className="relative">
              <input
                type={showPassword ? 'text' : 'password'}
                value={newPassword}
                onChange={(e) => setNewPassword(e.target.value)}
                className="w-full px-3 py-2 bg-slate-900 border border-slate-700 rounded-md text-white focus:outline-none focus:ring-2 focus:ring-primary-500"
              />
            </div>
          </div>
          <div>
            <label className="block text-sm font-medium text-slate-300 mb-2">Confirm Password</label>
            <input
              type={showPassword ? 'text' : 'password'}
              value={confirmPassword}
              onChange={(e) => setConfirmPassword(e.target.value)}
              className="w-full px-3 py-2 bg-slate-900 border border-slate-700 rounded-md text-white focus:outline-none focus:ring-2 focus:ring-primary-500"
            />
          </div>
        </div>

        <div className="flex items-center gap-3">
          <label className="relative inline-flex items-center cursor-pointer">
            <input
              type="checkbox"
              checked={showPassword}
              onChange={(e) => setShowPassword(e.target.checked)}
              className="sr-only peer"
            />
            <div className="w-11 h-6 bg-slate-700 peer-focus:ring-2 peer-focus:ring-primary-500 rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-primary-600"></div>
          </label>
          <span className="text-sm text-slate-300">Show passwords</span>
        </div>

        <button
          onClick={handlePasswordChange}
          className="flex items-center gap-2 px-4 py-2 bg-primary-600 hover:bg-primary-700 text-white rounded-md text-sm font-medium transition-colors"
        >
          Update Password
        </button>
      </div>

      {/* 2FA */}
      <div className="space-y-4 pt-4 border-t border-slate-700">
        <h4 className="text-sm font-medium text-slate-400 uppercase tracking-wider">Two-Factor Authentication</h4>
        
        <div className="flex items-center justify-between p-4 bg-slate-800 border border-slate-700 rounded-lg">
          <div className="flex items-center gap-3">
            <Smartphone className="w-5 h-5 text-slate-400" />
            <div>
              <p className="text-sm font-medium text-white">Authenticator App</p>
              <p className="text-xs text-slate-400">
                {twoFactorEnabled ? 'Enabled - Your account is more secure' : 'Not enabled - Add an extra layer of security'}
              </p>
            </div>
          </div>
          {twoFactorEnabled ? (
            <button
              onClick={() => setTwoFactorDisableOpen(true)}
              className="px-4 py-2 text-sm font-medium text-danger-500 hover:text-danger-400 hover:bg-slate-700 rounded-md transition-colors"
            >
              Disable
            </button>
          ) : (
            <button
              onClick={() => setTwoFactorSetupOpen(true)}
              className="flex items-center gap-2 px-4 py-2 bg-primary-600 hover:bg-primary-700 text-white rounded-md text-sm font-medium transition-colors"
            >
              <Smartphone className="w-4 h-4" />
              Enable 2FA
            </button>
          )}
        </div>
      </div>

      {/* Preferences */}
      <div className="space-y-4 pt-4 border-t border-slate-700">
        <h4 className="text-sm font-medium text-slate-400 uppercase tracking-wider">Preferences</h4>
        
        {/* Theme */}
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-3">
            {theme === 'dark' ? <Moon className="w-5 h-5 text-slate-400" /> : theme === 'light' ? <Sun className="w-5 h-5 text-slate-400" /> : <Monitor className="w-5 h-5 text-slate-400" />}
            <div>
              <p className="text-sm font-medium text-white">Theme</p>
              <p className="text-xs text-slate-400">Choose your preferred color scheme</p>
            </div>
          </div>
          <select
            value={theme}
            onChange={(e) => setTheme(e.target.value as 'dark' | 'light' | 'system')}
            className="px-3 py-1.5 bg-slate-800 border border-slate-700 rounded-md text-sm text-white focus:outline-none focus:ring-2 focus:ring-primary-500"
          >
            <option value="dark">Dark</option>
            <option value="light">Light</option>
            <option value="system">System</option>
          </select>
        </div>

        {/* Time Format */}
        <div className="flex items-center justify-between">
          <div>
            <p className="text-sm font-medium text-white">Time Format</p>
            <p className="text-xs text-slate-400">Choose 12-hour or 24-hour format</p>
          </div>
          <div className="flex items-center gap-2">
            <button
              onClick={() => setTimeFormat('12h')}
              className={`px-3 py-1.5 rounded text-sm font-medium transition-colors ${
                timeFormat === '12h' ? 'bg-primary-600 text-white' : 'bg-slate-800 text-slate-400 hover:text-white'
              }`}
            >
              12h
            </button>
            <button
              onClick={() => setTimeFormat('24h')}
              className={`px-3 py-1.5 rounded text-sm font-medium transition-colors ${
                timeFormat === '24h' ? 'bg-primary-600 text-white' : 'bg-slate-800 text-slate-400 hover:text-white'
              }`}
            >
              24h
            </button>
          </div>
        </div>
      </div>

      {/* 2FA Modals */}
      <TwoFactorSetup open={twoFactorSetupOpen} onClose={() => setTwoFactorSetupOpen(false)} />
      <TwoFactorDisable open={twoFactorDisableOpen} onClose={() => setTwoFactorDisableOpen(false)} onDisable={() => setTwoFactorEnabled(false)} />
    </div>
  )
}

// Notifications Tab
function NotificationSettings() {
  const { addToast } = useToastStore()
  const [emailEnabled, setEmailEnabled] = useState(true)
  const [slackWebhook, setSlackWebhook] = useState('')
  const [deployments, setDeployments] = useState(true)
  const [sslExpiring, setSslExpiring] = useState(true)
  const [backupFailed, setBackupFailed] = useState(true)
  const [serverAlert, setServerAlert] = useState(true)

  return (
    <div className="space-y-6">
      <div>
        <h3 className="text-lg font-medium text-white mb-4">Notification Settings</h3>
        <p className="text-sm text-slate-400">Configure how you receive notifications about your server and apps.</p>
      </div>

      {/* Email */}
      <div className="space-y-4">
        <h4 className="text-sm font-medium text-slate-400 uppercase tracking-wider">Email</h4>
        
        <div className="flex items-center gap-3">
          <label className="relative inline-flex items-center cursor-pointer">
            <input
              type="checkbox"
              checked={emailEnabled}
              onChange={(e) => setEmailEnabled(e.target.checked)}
              className="sr-only peer"
            />
            <div className="w-11 h-6 bg-slate-700 peer-focus:ring-2 peer-focus:ring-primary-500 rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-primary-600"></div>
          </label>
          <span className="text-sm text-slate-300">Enable email notifications</span>
        </div>
      </div>

      {/* Slack */}
      <div className="space-y-4 pt-4 border-t border-slate-700">
        <h4 className="text-sm font-medium text-slate-400 uppercase tracking-wider">Slack Integration</h4>
        
        <div>
          <label className="block text-sm font-medium text-slate-300 mb-2">Slack Webhook URL</label>
          <input
            type="text"
            value={slackWebhook}
            onChange={(e) => setSlackWebhook(e.target.value)}
            placeholder="https://hooks.slack.com/services/..."
            className="w-full max-w-md px-3 py-2 bg-slate-900 border border-slate-700 rounded-md text-white placeholder-slate-500 focus:outline-none focus:ring-2 focus:ring-primary-500"
          />
        </div>
      </div>

      {/* Event Types */}
      <div className="space-y-4 pt-4 border-t border-slate-700">
        <h4 className="text-sm font-medium text-slate-400 uppercase tracking-wider">Notification Events</h4>
        
        <div className="space-y-3">
          <div className="flex items-center justify-between p-3 bg-slate-800 rounded-lg">
            <div>
              <p className="text-sm font-medium text-white">Deployment updates</p>
              <p className="text-xs text-slate-400">Get notified when apps are deployed or fail</p>
            </div>
            <label className="relative inline-flex items-center cursor-pointer">
              <input
                type="checkbox"
                checked={deployments}
                onChange={(e) => setDeployments(e.target.checked)}
                className="sr-only peer"
              />
              <div className="w-11 h-6 bg-slate-700 peer-focus:ring-2 peer-focus:ring-primary-500 rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-primary-600"></div>
            </label>
          </div>

          <div className="flex items-center justify-between p-3 bg-slate-800 rounded-lg">
            <div>
              <p className="text-sm font-medium text-white">SSL expiring</p>
              <p className="text-xs text-slate-400">Get warned before SSL certificates expire</p>
            </div>
            <label className="relative inline-flex items-center cursor-pointer">
              <input
                type="checkbox"
                checked={sslExpiring}
                onChange={(e) => setSslExpiring(e.target.checked)}
                className="sr-only peer"
              />
              <div className="w-11 h-6 bg-slate-700 peer-focus:ring-2 peer-focus:ring-primary-500 rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-primary-600"></div>
            </label>
          </div>

          <div className="flex items-center justify-between p-3 bg-slate-800 rounded-lg">
            <div>
              <p className="text-sm font-medium text-white">Backup failed</p>
              <p className="text-xs text-slate-400">Get alerted when a backup fails</p>
            </div>
            <label className="relative inline-flex items-center cursor-pointer">
              <input
                type="checkbox"
                checked={backupFailed}
                onChange={(e) => setBackupFailed(e.target.checked)}
                className="sr-only peer"
              />
              <div className="w-11 h-6 bg-slate-700 peer-focus:ring-2 peer-focus:ring-primary-500 rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-primary-600"></div>
            </label>
          </div>

          <div className="flex items-center justify-between p-3 bg-slate-800 rounded-lg">
            <div>
              <p className="text-sm font-medium text-white">Server alerts</p>
              <p className="text-xs text-slate-400">CPU, RAM, or disk usage warnings</p>
            </div>
            <label className="relative inline-flex items-center cursor-pointer">
              <input
                type="checkbox"
                checked={serverAlert}
                onChange={(e) => setServerAlert(e.target.checked)}
                className="sr-only peer"
              />
              <div className="w-11 h-6 bg-slate-700 peer-focus:ring-2 peer-focus:ring-primary-500 rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-primary-600"></div>
            </label>
          </div>
        </div>
      </div>

      <div className="pt-4 border-t border-slate-700">
        <button
          onClick={() => addToast({ type: 'success', title: 'Settings saved', message: 'Notification preferences updated.' })}
          className="flex items-center gap-2 px-4 py-2 bg-primary-600 hover:bg-primary-700 text-white rounded-md text-sm font-medium transition-colors"
        >
          <Check className="w-4 h-4" />
          Save Preferences
        </button>
      </div>
    </div>
  )
}

// Security Tab
function SecuritySettings() {
  const { addToast } = useToastStore()
  const [sessionTimeout, setSessionTimeout] = useState('24h')
  const [minPasswordLength, setMinPasswordLength] = useState(8)
  const [require2FA, setRequire2FA] = useState(false)
  const [ipAllowlist, setIpAllowlist] = useState<string[]>([])

  return (
    <div className="space-y-6">
      <div>
        <h3 className="text-lg font-medium text-white mb-4">Security Settings</h3>
        <p className="text-sm text-slate-400">Configure security policies for your panel.</p>
      </div>

      {/* Session Timeout */}
      <div className="space-y-4">
        <h4 className="text-sm font-medium text-slate-400 uppercase tracking-wider">Session</h4>
        
        <div>
          <label className="block text-sm font-medium text-slate-300 mb-2">Session Timeout</label>
          <select
            value={sessionTimeout}
            onChange={(e) => setSessionTimeout(e.target.value)}
            className="w-full max-w-xs px-3 py-2 bg-slate-900 border border-slate-700 rounded-md text-white focus:outline-none focus:ring-2 focus:ring-primary-500"
          >
            <option value="1h">1 hour</option>
            <option value="8h">8 hours</option>
            <option value="24h">24 hours</option>
            <option value="7d">7 days</option>
            <option value="30d">30 days</option>
          </select>
        </div>

        <div className="flex items-center gap-3">
          <label className="relative inline-flex items-center cursor-pointer">
            <input
              type="checkbox"
              checked={require2FA}
              onChange={(e) => setRequire2FA(e.target.checked)}
              className="sr-only peer"
            />
            <div className="w-11 h-6 bg-slate-700 peer-focus:ring-2 peer-focus:ring-primary-500 rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-primary-600"></div>
          </label>
          <div>
            <p className="text-sm font-medium text-white">Require 2FA for all users</p>
            <p className="text-xs text-slate-400">All team members must enable two-factor authentication</p>
          </div>
        </div>
      </div>

      {/* Password Policy */}
      <div className="space-y-4 pt-4 border-t border-slate-700">
        <h4 className="text-sm font-medium text-slate-400 uppercase tracking-wider">Password Policy</h4>
        
        <div>
          <label className="block text-sm font-medium text-slate-300 mb-2">Minimum Password Length</label>
          <input
            type="number"
            min={8}
            max={128}
            value={minPasswordLength}
            onChange={(e) => setMinPasswordLength(parseInt(e.target.value))}
            className="w-24 px-3 py-2 bg-slate-900 border border-slate-700 rounded-md text-white focus:outline-none focus:ring-2 focus:ring-primary-500"
          />
        </div>
      </div>

      {/* IP Allowlist */}
      <div className="space-y-4 pt-4 border-t border-slate-700">
        <h4 className="text-sm font-medium text-slate-400 uppercase tracking-wider">IP Allowlist</h4>
        <p className="text-xs text-slate-500">Restrict panel access to specific IP addresses. Leave empty to allow all IPs.</p>
        
        <div className="space-y-2">
          {ipAllowlist.map((ip, idx) => (
            <div key={idx} className="flex items-center gap-2">
              <input
                type="text"
                value={ip}
                onChange={(e) => {
                  const newList = [...ipAllowlist]
                  newList[idx] = e.target.value
                  setIpAllowlist(newList)
                }}
                placeholder="192.168.1.0/24"
                className="flex-1 max-w-xs px-3 py-2 bg-slate-900 border border-slate-700 rounded-md text-white placeholder-slate-500 focus:outline-none focus:ring-2 focus:ring-primary-500"
              />
              <button
                onClick={() => setIpAllowlist(ipAllowlist.filter((_, i) => i !== idx))}
                className="p-2 text-slate-400 hover:text-danger-500 transition-colors"
              >
                Remove
              </button>
            </div>
          ))}
          <button
            onClick={() => setIpAllowlist([...ipAllowlist, ''])}
            className="text-sm text-primary-400 hover:text-primary-300"
          >
            + Add IP range
          </button>
        </div>
      </div>

      {/* Login Attempts Log */}
      <div className="space-y-4 pt-4 border-t border-slate-700">
        <h4 className="text-sm font-medium text-slate-400 uppercase tracking-wider">Recent Login Attempts</h4>
        
        <div className="bg-slate-800 border border-slate-700 rounded-lg overflow-hidden">
          <table className="w-full">
            <thead>
              <tr className="border-b border-slate-700">
                <th className="px-4 py-2 text-left text-xs font-medium text-slate-400">Time</th>
                <th className="px-4 py-2 text-left text-xs font-medium text-slate-400">IP</th>
                <th className="px-4 py-2 text-left text-xs font-medium text-slate-400">Result</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-slate-700">
              <tr>
                <td className="px-4 py-2 text-sm text-slate-400">2024-06-01 12:34:56</td>
                <td className="px-4 py-2 text-sm text-slate-300">192.168.1.100</td>
                <td className="px-4 py-2 text-sm text-green-400">Success</td>
              </tr>
              <tr>
                <td className="px-4 py-2 text-sm text-slate-400">2024-06-01 10:22:11</td>
                <td className="px-4 py-2 text-sm text-slate-300">10.0.0.55</td>
                <td className="px-4 py-2 text-sm text-danger-400">Failed (wrong password)</td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>

      <div className="pt-4 border-t border-slate-700">
        <button
          onClick={() => addToast({ type: 'success', title: 'Settings saved', message: 'Security settings updated.' })}
          className="flex items-center gap-2 px-4 py-2 bg-primary-600 hover:bg-primary-700 text-white rounded-md text-sm font-medium transition-colors"
        >
          <Check className="w-4 h-4" />
          Save Security Settings
        </button>
      </div>
    </div>
  )
}

export default function SettingsPage() {
  const [activeTab, setActiveTab] = useState<Tab>('user')

  return (
    <div className="p-6">
      {/* Header */}
      <div className="mb-6">
        <h1 className="text-2xl font-semibold text-white">Settings</h1>
        <p className="text-sm text-slate-400 mt-1">
          Manage your panel settings and preferences
        </p>
      </div>

      {/* Tabs */}
      <div className="flex items-center gap-1 mb-6 overflow-x-auto pb-2">
        {tabs.map((tab) => {
          const Icon = tab.icon
          return (
            <button
              key={tab.id}
              onClick={() => setActiveTab(tab.id)}
              className={`
                flex items-center gap-2 px-4 py-2 rounded-md text-sm font-medium whitespace-nowrap transition-colors
                ${activeTab === tab.id
                  ? 'bg-primary-600 text-white'
                  : 'text-slate-400 hover:text-white hover:bg-slate-800'
                }
              `}
            >
              <Icon className="w-4 h-4" />
              {tab.label}
            </button>
          )
        })}
      </div>

      {/* Tab Content */}
      <div className="bg-slate-800 border border-slate-700 rounded-lg p-6">
        {activeTab === 'panel' && <PanelSettings />}
        {activeTab === 'server' && <ServerSettings />}
        {activeTab === 'user' && <UserSettings />}
        {activeTab === 'api' && <APIKeyList />}
        {activeTab === 'notifications' && <NotificationSettings />}
        {activeTab === 'security' && <SecuritySettings />}
      </div>
    </div>
  )
}