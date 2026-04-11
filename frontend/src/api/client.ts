import axios from 'axios'
import { storage } from '@/platform/storage'

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
  const { data } = await client.post<{ access_token: string }>('/auth/refresh')
  await storage.set('access_token', data.access_token)
  return data.access_token
}

client.interceptors.request.use(async (config) => {
  let token = await storage.get('access_token')

  // Proactively refresh if the token expires within 5 minutes
  if (token && tokenExpiresAt(token) - Date.now() < 5 * 60 * 1000) {
    if (!isRefreshing) {
      isRefreshing = true
      try {
        token = await doRefresh()
        refreshQueue.forEach((cb) => cb(token!))
        refreshQueue = []
      } catch {
        // Refresh failed — let the request go out with the old token;
        // the 401 interceptor below will handle it
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
    if (error.response?.status === 401 && !original._retry) {
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
