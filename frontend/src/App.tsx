import { useState, useEffect, type ReactNode } from 'react'
import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'

import { AuthContext, type AuthContextValue } from '@/hooks/useAuth'
import { ToastProvider } from '@/components/ui/Toast'
import { authApi } from '@/api/auth'
import { storage } from '@/platform/storage'
import { registerSyncOnReconnect } from '@/offline/sync'

import Layout from '@/components/layout/Layout'
import Login from '@/pages/Login'
import Register from '@/pages/Register'
import Dashboard from '@/pages/Dashboard'
import MediaLibrary from '@/pages/MediaLibrary'
import MediaDetail from '@/pages/MediaDetail'
import Search from '@/pages/Search'
import Stats from '@/pages/Stats'
import Settings from '@/pages/Settings'
import SharedList from '@/pages/SharedList'

import type { User } from '@/types/auth'

const qc = new QueryClient({
  defaultOptions: { queries: { staleTime: 1000 * 60 * 5, retry: 1 } },
})

function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<User | null>(null)
  const [isLoading, setIsLoading] = useState(true)

  useEffect(() => {
    registerSyncOnReconnect()
    authApi.me()
      .then((r) => setUser(r.data))
      .catch(() => setUser(null))
      .finally(() => setIsLoading(false))
  }, [])

  const login = async (username: string, password: string) => {
    const { data } = await authApi.login({ username, password })
    await storage.set('access_token', data.access_token)
    const { data: me } = await authApi.me()
    setUser(me)
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
  const token = localStorage.getItem('access_token')
  if (!token) return <Navigate to="/login" replace />
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
              </Route>
            </Routes>
          </BrowserRouter>
        </ToastProvider>
      </AuthProvider>
    </QueryClientProvider>
  )
}
