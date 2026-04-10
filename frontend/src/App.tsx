import { useState, useEffect, type ReactNode } from 'react'
import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'

import { AuthContext, useAuth, type AuthContextValue } from '@/hooks/useAuth'
import { ToastProvider } from '@/components/ui/Toast'
import { authApi } from '@/api/auth'
import { storage } from '@/platform/storage'
import { syncMedia, registerSyncOnReconnect } from '@/offline/sync'

import Layout from '@/components/layout/Layout'
import Login from '@/pages/Login'
import Register from '@/pages/Register'
import Dashboard from '@/pages/Dashboard'
import MediaLibrary from '@/pages/MediaLibrary'
import MediaDetail from '@/pages/MediaDetail'
import Search from '@/pages/Search'
import Stats from '@/pages/Stats'
import Settings from '@/pages/Settings'
import Admin from '@/pages/Admin'
import SharedList from '@/pages/SharedList'
import PublicProfile from '@/pages/PublicProfile'

import type { User } from '@/types/auth'

const qc = new QueryClient({
  defaultOptions: { queries: { staleTime: 1000 * 60 * 5, retry: 1 } },
})

// Register once at module level — idempotent
registerSyncOnReconnect()

function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<User | null>(null)
  const [isLoading, setIsLoading] = useState(true)

  useEffect(() => {
    const timeout = setTimeout(() => setIsLoading(false), 5000)
    authApi.me()
      .then((r) => {
        setUser(r.data)
        // Kick off background sync as soon as we know the user is authenticated
        syncMedia().catch(console.error)
      })
      .catch(() => setUser(null))
      .finally(() => { clearTimeout(timeout); setIsLoading(false) })
  }, [])

  const login = async (username: string, password: string) => {
    const { data } = await authApi.login({ username, password })
    await storage.set('access_token', data.access_token)
    const { data: me } = await authApi.me()
    setUser(me)
    // Sync after login so library is available immediately
    syncMedia().catch(console.error)
  }

  const logout = async () => {
    await authApi.logout().catch(() => {})
    await storage.remove('access_token')
    setUser(null)
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
                <Route path="films" element={<MediaLibrary type="film" />} />
                <Route path="films/:id" element={<MediaDetail />} />
                <Route path="tv" element={<MediaLibrary type="tv_show" />} />
                <Route path="tv/:id" element={<MediaDetail />} />
                <Route path="books" element={<MediaLibrary type="book" />} />
                <Route path="books/:id" element={<MediaDetail />} />
                <Route path="anime" element={<MediaLibrary type="anime" />} />
                <Route path="anime/:id" element={<MediaDetail />} />
                <Route path="search" element={<Search />} />
                <Route path="stats" element={<Stats />} />
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
