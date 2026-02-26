import { useState, useEffect, useRef, type ReactNode } from 'react'
import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'

import { AuthContext, useAuth, type AuthContextValue } from '@/hooks/useAuth'
import { ToastProvider } from '@/components/ui/Toast'
import { authApi } from '@/api/auth'
import { storage } from '@/platform/storage'
import { syncMedia, registerSyncOnReconnect } from '@/offline/sync'
import { isIOSStandalone, PWA_REFRESH_KEY } from '@/api/client'

import Layout from '@/components/layout/Layout'
import Login from '@/pages/Login'
import Register from '@/pages/Register'
import Dashboard from '@/pages/Dashboard'
import MediaLibrary from '@/pages/MediaLibrary'
import MediaDetail from '@/pages/MediaDetail'
import Search from '@/pages/Search'
import Stats from '@/pages/Stats'
import Trending from '@/pages/Trending'
import Settings from '@/pages/Settings'
import Admin from '@/pages/Admin'
import SharedList from '@/pages/SharedList'
import PublicProfile from '@/pages/PublicProfile'

import type { User } from '@/types/auth'

// ── PWA auto-update ───────────────────────────────────────────────────────────
// When a new SW takes over, reload — but only when the tab is hidden so we
// never interrupt active use. If the tab is currently visible, defer until
// the user next switches away.
if ('serviceWorker' in navigator) {
  navigator.serviceWorker.addEventListener('controllerchange', () => {
    const reload = () => { if (document.visibilityState === 'hidden') window.location.reload() }
    if (document.visibilityState === 'hidden') {
      window.location.reload()
    } else {
      document.addEventListener('visibilitychange', reload, { once: true })
    }
  })
}

// ── User cache (survives network blips on app load) ───────────────────────────
const USER_CACHE_KEY = 'watched_user_cache'
function getCachedUser(): User | null {
  try { return JSON.parse(localStorage.getItem(USER_CACHE_KEY) ?? 'null') } catch { return null }
}
function setCachedUser(u: User | null) {
  if (u) localStorage.setItem(USER_CACHE_KEY, JSON.stringify(u))
  else localStorage.removeItem(USER_CACHE_KEY)
}

const qc = new QueryClient({
  defaultOptions: { queries: { staleTime: 1000 * 60 * 5, retry: 1 } },
})

// Register once at module level — idempotent
registerSyncOnReconnect()

function AuthProvider({ children }: { children: ReactNode }) {
  // Seed from cache so ProtectedRoute never flashes /login on a valid session
  const [user, setUser] = useState<User | null>(getCachedUser)
  const [isLoading, setIsLoading] = useState(true)
  // Track whether we've already resolved the initial me() call
  const initialised = useRef(false)

  const hydrateUser = async () => {
    try {
      const { data: me } = await authApi.me()
      setUser(me)
      setCachedUser(me)
      if (!initialised.current) syncMedia().catch(console.error)
    } catch (err: any) {
      const status = err?.response?.status
      // Only clear the session on definitive auth failures (401/403 on /auth/refresh,
      // which the client interceptor already handles). A plain network error or
      // server blip keeps the cached user intact.
      if (status === 401 || status === 403) {
        setUser(null)
        setCachedUser(null)
      }
    } finally {
      initialised.current = true
      setIsLoading(false)
    }
  }

  useEffect(() => {
    // Safety: if the server is unreachable, don't spin forever
    const timeout = setTimeout(() => setIsLoading(false), 5000)
    hydrateUser().finally(() => clearTimeout(timeout))

    // Re-validate whenever the tab becomes visible again (covers long idle periods)
    const onVisible = () => {
      if (initialised.current && document.visibilityState === 'visible') hydrateUser()
    }
    document.addEventListener('visibilitychange', onVisible)
    return () => document.removeEventListener('visibilitychange', onVisible)
  }, [])

  const login = async (username: string, password: string) => {
    const { data } = await authApi.login({ username, password })
    await storage.set('access_token', data.access_token)
    // iOS PWA: persist refresh token in localStorage so it survives app restarts
    // (iOS clears HttpOnly cookies when the PWA is fully closed)
    if (isIOSStandalone() && data.refresh_token) {
      localStorage.setItem(PWA_REFRESH_KEY, data.refresh_token)
    }
    const { data: me } = await authApi.me()
    setUser(me)
    setCachedUser(me)
    syncMedia().catch(console.error)
  }

  const logout = async () => {
    await authApi.logout().catch(() => {})
    await storage.remove('access_token')
    localStorage.removeItem(PWA_REFRESH_KEY)
    setUser(null)
    setCachedUser(null)
    qc.clear()
  }

  const value: AuthContextValue = { user, isLoading, login, logout }
  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>
}

function ProtectedRoute({ children }: { children: ReactNode }) {
  const { user, isLoading } = useAuth()
  if (isLoading) return (
    <div className="min-h-screen bg-[#0d0d0d] flex items-center justify-center">
      <div className="w-6 h-6 border-2 border-indigo-500 border-t-transparent rounded-full animate-spin" />
    </div>
  )
  if (!user) return <Navigate to="/login" replace />
  return <>{children}</>
}

export default function App() {
  return (
    <QueryClientProvider client={qc}>
      <AuthProvider>
        <ToastProvider>
          <BrowserRouter>
            <Routes>
              <Route path="/login" element={<Login />} />
              <Route path="/register" element={<Register />} />
              <Route path="/share/:id" element={<SharedList />} />
              <Route path="/u/:username" element={<PublicProfile />} />
              <Route path="/" element={<ProtectedRoute><Layout /></ProtectedRoute>}>
                <Route index element={<Dashboard />} />
                <Route path="films" element={<MediaLibrary key="film" type="film" />} />
                <Route path="films/:id" element={<MediaDetail />} />
                <Route path="tv" element={<MediaLibrary key="tv_show" type="tv_show" />} />
                <Route path="tv/:id" element={<MediaDetail />} />
                <Route path="books" element={<MediaLibrary key="book" type="book" />} />
                <Route path="books/:id" element={<MediaDetail />} />
                <Route path="anime" element={<MediaLibrary key="anime" type="anime" />} />
                <Route path="anime/:id" element={<MediaDetail />} />
                <Route path="search" element={<Search />} />
                <Route path="stats" element={<Stats />} />
                <Route path="trending" element={<Trending />} />
                <Route path="settings" element={<Settings />} />
                <Route path="admin" element={<Admin />} />
              </Route>
            </Routes>
          </BrowserRouter>
        </ToastProvider>
      </AuthProvider>
    </QueryClientProvider>
  )
}
