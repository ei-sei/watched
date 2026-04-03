import client from '@/api/client'
import { getPending, dequeue, incrementRetry } from './queue'

const MAX_RETRIES = 3

export async function flushQueue(): Promise<void> {
  const pending = await getPending()
  for (const item of pending) {
    if (item.retries >= MAX_RETRIES) {
      await dequeue(item.id!)
      continue
    }
    try {
      await client.request({ method: item.method, url: item.url, data: item.data })
      await dequeue(item.id!)
    } catch {
      await incrementRetry(item.id!)
    }
  }
}

export function registerSyncOnReconnect(): void {
  window.addEventListener('online', () => {
    flushQueue().catch(console.error)
  })
}
