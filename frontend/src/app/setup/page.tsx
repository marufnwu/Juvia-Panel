'use client'

import { useState, useEffect } from 'react'
import { useRouter } from 'next/navigation'
import Link from 'next/link'
import { Server, Loader2, Eye, EyeOff, CheckCircle, ArrowRight, ArrowLeft } from 'lucide-react'
import { useAuthStore } from '@/stores'
import { useToastStore } from '@/stores'

const API_BASE = process.env.NEXT_PUBLIC_API_URL || '/api/v1'

type Step = 'account' | 'complete'

export default function SetupPage() {
  const router = useRouter()
  const { register, checkUsersExist, usersExist, isAuthenticated, setAuth } = useAuthStore()
  const { addToast } = useToastStore()
  
  const [currentStep, setCurrentStep] = useState<Step>('account')
  const [isLoading, setIsLoading] = useState(false)
  const [isCheckingUsers, setIsCheckingUsers] = useState(true)
  
  // Form state
  const [email, setEmail] = useState('')
  const [username, setUsername] = useState('')
  const [password, setPassword] = useState('')
  const [confirmPassword, setConfirmPassword] = useState('')
  const [showPassword, setShowPassword] = useState(false)
  const [formError, setFormError] = useState('')

  // Check if users already exist on mount
  useEffect(() => {
    async function checkUsers() {
      const exists = await checkUsersExist()
      if (exists) {
        // Users exist, redirect to login
        router.push('/login')
      } else {
        setIsCheckingUsers(false)
      }
    }
    checkUsers()
  }, [checkUsersExist, router])

  // If already authenticated, redirect to dashboard
  useEffect(() => {
    if (isAuthenticated) {
      router.push('/')
    }
  }, [isAuthenticated, router])

  const validateForm = () => {
    if (!email || !username || !password || !confirmPassword) {
      setFormError('All fields are required')
      return false
    }
    if (!email.includes('@')) {
      setFormError('Please enter a valid email address')
      return false
    }
    if (username.length < 3) {
      setFormError('Username must be at least 3 characters')
      return false
    }
    if (username.length > 32) {
      setFormError('Username must be at most 32 characters')
      return false
    }
    if (!/^[a-zA-Z0-9_]+$/.test(username)) {
      setFormError('Username can only contain letters, numbers, and underscores')
      return false
    }
    if (password.length < 8) {
      setFormError('Password must be at least 8 characters')
      return false
    }
    if (password !== confirmPassword) {
      setFormError('Passwords do not match')
      return false
    }
    setFormError('')
    return true
  }

  const handleCreateAccount = async () => {
    if (!validateForm()) return
    
    setIsLoading(true)
    setFormError('')

    try {
      const response = await fetch(`${API_BASE}/auth/register`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ email, username, password }),
        credentials: 'include',
      })
      
      if (!response.ok) {
        const errorData = await response.json()
        throw new Error(errorData.message || 'Failed to create account')
      }
      
      const data = await response.json()
      setAuth(data.access_token, data.user ? {
        id: String(data.user.id),
        email: data.user.email,
        name: data.user.username,
        role: data.user.role,
      } : null)
      addToast({ type: 'success', title: 'Account created successfully!' })
      setCurrentStep('complete')
    } catch (err) {
      setFormError(err instanceof Error ? err.message : 'Failed to create account')
      addToast({ type: 'error', title: 'Registration failed', message: err instanceof Error ? err.message : 'Please try again' })
    } finally {
      setIsLoading(false)
    }
  }

  const handleGoToDashboard = () => {
    router.push('/')
  }

  // Show loading while checking
  if (isCheckingUsers) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-slate-900 px-4">
        <div className="text-center">
          <Loader2 className="w-8 h-8 animate-spin text-primary-400 mx-auto mb-4" />
          <p className="text-slate-400">Loading...</p>
        </div>
      </div>
    )
  }

  return (
    <div className="min-h-screen flex items-center justify-center bg-slate-900 px-4 py-12">
      <div className="w-full max-w-md">
        {/* Logo and Title */}
        <div className="text-center mb-8">
          <div className="inline-flex items-center justify-center w-16 h-16 rounded-2xl bg-primary-500/10 mb-4">
            <Server className="w-8 h-8 text-primary-400" />
          </div>
          <h1 className="text-2xl font-bold text-white">Setup Juvia Panel</h1>
          <p className="text-slate-400 mt-2">Create your admin account to get started</p>
        </div>

        {/* Progress Steps */}
        <div className="flex items-center justify-center gap-2 mb-8">
          <div className={`flex items-center gap-2 ${currentStep === 'account' ? 'text-primary-400' : 'text-success-400'}`}>
            <div className={`w-8 h-8 rounded-full flex items-center justify-center text-sm font-medium ${
              currentStep === 'account' ? 'bg-primary-500 text-white' : 'bg-success-500 text-white'
            }`}>
              {currentStep === 'complete' ? <CheckCircle className="w-4 h-4" /> : '1'}
            </div>
            <span className="text-sm font-medium">Account</span>
          </div>
          <div className="w-8 h-px bg-slate-700" />
          <div className={`flex items-center gap-2 ${currentStep === 'complete' ? 'text-success-400' : 'text-slate-500'}`}>
            <div className={`w-8 h-8 rounded-full flex items-center justify-center text-sm font-medium ${
              currentStep === 'complete' ? 'bg-success-500 text-white' : 'bg-slate-700 text-slate-400'
            }`}>
              {currentStep === 'complete' ? '2' : '2'}
            </div>
            <span className="text-sm font-medium">Complete</span>
          </div>
        </div>

        {/* Step 1: Create Account */}
        {currentStep === 'account' && (
          <div className="bg-slate-800 rounded-xl border border-slate-700 p-6">
            <div className="space-y-4">
              {formError && (
                <div className="p-3 rounded-lg bg-danger-500/10 border border-danger-500/20 text-danger-400 text-sm">
                  {formError}
                </div>
              )}

              <div>
                <label htmlFor="email" className="block text-sm font-medium text-slate-300 mb-2">
                  Email
                </label>
                <input
                  id="email"
                  type="email"
                  value={email}
                  onChange={(e) => setEmail(e.target.value)}
                  className="w-full px-4 py-2.5 bg-slate-900 border border-slate-700 rounded-lg text-white placeholder-slate-500 focus:outline-none focus:ring-2 focus:ring-primary-500 focus:border-transparent"
                  placeholder="admin@example.com"
                />
              </div>

              <div>
                <label htmlFor="username" className="block text-sm font-medium text-slate-300 mb-2">
                  Username
                </label>
                <input
                  id="username"
                  type="text"
                  value={username}
                  onChange={(e) => setUsername(e.target.value)}
                  className="w-full px-4 py-2.5 bg-slate-900 border border-slate-700 rounded-lg text-white placeholder-slate-500 focus:outline-none focus:ring-2 focus:ring-primary-500 focus:border-transparent"
                  placeholder="admin"
                />
                <p className="text-xs text-slate-500 mt-1">Letters, numbers, and underscores only</p>
              </div>

              <div>
                <label htmlFor="password" className="block text-sm font-medium text-slate-300 mb-2">
                  Password
                </label>
                <div className="relative">
                  <input
                    id="password"
                    type={showPassword ? 'text' : 'password'}
                    value={password}
                    onChange={(e) => setPassword(e.target.value)}
                    className="w-full px-4 py-2.5 bg-slate-900 border border-slate-700 rounded-lg text-white placeholder-slate-500 focus:outline-none focus:ring-2 focus:ring-primary-500 focus:border-transparent pr-10"
                    placeholder="••••••••"
                  />
                  <button
                    type="button"
                    onClick={() => setShowPassword(!showPassword)}
                    className="absolute right-3 top-1/2 -translate-y-1/2 text-slate-400 hover:text-white transition-colors"
                  >
                    {showPassword ? <EyeOff className="w-5 h-5" /> : <Eye className="w-5 h-5" />}
                  </button>
                </div>
                <p className="text-xs text-slate-500 mt-1">At least 8 characters</p>
              </div>

              <div>
                <label htmlFor="confirmPassword" className="block text-sm font-medium text-slate-300 mb-2">
                  Confirm Password
                </label>
                <input
                  id="confirmPassword"
                  type="password"
                  value={confirmPassword}
                  onChange={(e) => setConfirmPassword(e.target.value)}
                  className="w-full px-4 py-2.5 bg-slate-900 border border-slate-700 rounded-lg text-white placeholder-slate-500 focus:outline-none focus:ring-2 focus:ring-primary-500 focus:border-transparent"
                  placeholder="••••••••"
                />
              </div>

              <button
                type="button"
                onClick={handleCreateAccount}
                disabled={isLoading}
                className="w-full flex items-center justify-center gap-2 px-4 py-2.5 bg-primary-600 hover:bg-primary-700 disabled:bg-primary-600/50 text-white font-medium rounded-lg transition-colors mt-4"
              >
                {isLoading && <Loader2 className="w-4 h-4 animate-spin" />}
                Create Account
                <ArrowRight className="w-4 h-4" />
              </button>
            </div>

            <div className="mt-6 text-center">
              <Link href="/login" className="text-sm text-slate-400 hover:text-primary-400 transition-colors flex items-center justify-center gap-1">
                <ArrowLeft className="w-4 h-4" />
                Back to login
              </Link>
            </div>
          </div>
        )}

        {/* Step 2: Complete */}
        {currentStep === 'complete' && (
          <div className="bg-slate-800 rounded-xl border border-slate-700 p-6">
            <div className="text-center py-8">
              <div className="inline-flex items-center justify-center w-16 h-16 rounded-full bg-success-500/10 mb-4">
                <CheckCircle className="w-8 h-8 text-success-400" />
              </div>
              <h2 className="text-xl font-bold text-white mb-2">You're all set!</h2>
              <p className="text-slate-400 mb-8">
                Your admin account has been created. You can now start managing your server.
              </p>
              
              <button
                type="button"
                onClick={handleGoToDashboard}
                className="inline-flex items-center gap-2 px-6 py-2.5 bg-primary-600 hover:bg-primary-700 text-white font-medium rounded-lg transition-colors"
              >
                Go to Dashboard
                <ArrowRight className="w-4 h-4" />
              </button>
            </div>

            <div className="mt-6 text-center">
              <Link href="/login" className="text-sm text-slate-400 hover:text-primary-400 transition-colors flex items-center justify-center gap-1">
                <ArrowLeft className="w-4 h-4" />
                Back to login
              </Link>
            </div>
          </div>
        )}

        {/* Footer */}
        <p className="text-center text-slate-500 text-sm mt-8">
          Juvia Panel v1.0
        </p>
      </div>
    </div>
  )
}