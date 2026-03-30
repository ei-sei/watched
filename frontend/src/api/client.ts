import axios from 'axios'
import { storage } from '@/platform/storage'

// In PWA standalone mode, browsers (especially on Android/Samsung) can clear
// HttpOnly cookies when the app is fully closed. We persist the refresh token
// in localStorage as a fallback and send it via header so sessions survive.
export const PWA_REFRESH_KEY = 'pwa_refresh_token'
export function isPWAStandalone(): boolean {
  return (window.navigator as any).standalone === true ||          // iOS Safari
    window.matchMedia('(display-mode: standalone)').matches        // Android / desktop PWA
}

const client = axios.create({
  baseURL: import.meta.env.VITE_API_URL || 'http://localhost:8000',
  withCredentials: true,
  timeout: 15000,
})

// Decode JWT exp claim client-side (no verification needed — server verifies)
function tokenExpiresAt(token: string): number {
  try {
    const payload = JSON.parse(atob(token.split('.')[1]))
    return (payload.exp ?? 0) * 1000
  } catch { return 0 }
}

let isRefreshing = false
let refreshQueue: Array<(token: string) => void> = []

async function doRefresh(): Promise<string> {
  // Always send the refresh token via header as a fallback in case the
  // HttpOnly cookie was cleared by the browser (common on Android when
  // the app is backgrounded for a long time).
  const headers: Record<string, string> = {}
  const storedToken = localStorage.getItem(PWA_REFRESH_KEY)
  if (storedToken) headers['X-Refresh-Token'] = storedToken
  const { data } = await client.post<{ access_token: string }>('/auth/refresh', undefined, { headers })
  await storage.set('access_token', data.access_token)
  return data.access_token
}

client.interceptors.request.use(async (config) => {
  let token = await storage.get('access_token')

  // Proactively refresh if the token expires within 5 minutes.
  // Skip for /auth/refresh itself to avoid a deadlock where doRefresh()
  // re-enters this interceptor and queues behind itself.
  if (token && tokenExpiresAt(token) - Date.now() < 5 * 60 * 1000 && !config.url?.includes('/auth/refresh')) {
    if (!isRefreshing) {
      isRefreshing = true
      const oldToken = token
      try {
        token = await doRefresh()
        refreshQueue.forEach((cb) => cb(token!))
        refreshQueue = []
      } catch {
        // Refresh failed — proceed with the old token for this request and
        // any queued requests. Without resolving the queue, queued requests
        // (including logout) would hang forever.
        refreshQueue.forEach((cb) => cb(oldToken))
        refreshQueue = []
      } finally {
        isRefreshing = false
      }
    } else {
      // Another request is already refreshing — wait for it
      token = await new Promise<string>((resolve) => refreshQueue.push(resolve))
    }
  }

  if (token) config.headers.Authorization = `Bearer ${token}`
  return config
})

client.interceptors.response.use(
  (response) => response,
  async (error) => {
    const original = error.config

    // If the refresh endpoint itself returns 401, the session is gone.
    // Don't attempt another refresh — just clear the token and redirect.
    if (original.url?.includes('/auth/refresh') && error.response?.status === 401) {
      refreshQueue.forEach((cb) => cb(''))
      refreshQueue = []
      isRefreshing = false
      await storage.remove('access_token')
      window.location.replace('/login')
      return Promise.reject(error)
    }

    if (error.response?.status === 401 && !original._retry && original.headers?.Authorization) {
      original._retry = true
      if (isRefreshing) {
        return new Promise((resolve) => {
          refreshQueue.push((token) => {
            original.headers.Authorization = `Bearer ${token}`
            resolve(client(original))
          })
        })
      }
      isRefreshing = true
      try {
        const token = await doRefresh()
        refreshQueue.forEach((cb) => cb(token))
        refreshQueue = []
        original.headers.Authorization = `Bearer ${token}`
        return client(original)
      } catch (refreshErr: any) {
        // Only force logout on definitive auth failures — not network errors
        const status = refreshErr?.response?.status
        if (status === 401 || status === 403) {
          refreshQueue = []
          await storage.remove('access_token')
          window.location.replace('/login')
        } else {
          refreshQueue = []
        }
      } finally {
        isRefreshing = false
      }
    }
    return Promise.reject(error)
  }
)

export default client
