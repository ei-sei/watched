import { useState, useEffect } from 'react'
import { useLiveQuery } from 'dexie-react-hooks'
import { db } from '@/offline/db'

export type SyncState = 'offline' | 'pending' | 'syncing' | 'synced'

export function useSyncStatus() {
  const [online, setOnline] = useState(navigator.onLine)
  const [syncing, setSyncing] = useState(false)

  const pendingCount = useLiveQuery(() => db.mutationQueue.count(), [], 0) ?? 0

  useEffect(() => {
    const onOnline  = () => setOnline(true)
    const onOffline = () => setOnline(false)
    window.addEventListener('online',  onOnline)
    window.addEventListener('offline', onOffline)
    return () => {
      window.removeEventListener('online',  onOnline)
      window.removeEventListener('offline', onOffline)
    }
  }, [])

  // Listen for sync events dispatched by the sync engine
  useEffect(() => {
    const onStart = () => setSyncing(true)
    const onEnd   = () => setSyncing(false)
    window.addEventListener('watched:sync-start', onStart)
    window.addEventListener('watched:sync-end',   onEnd)
    return () => {
      window.removeEventListener('watched:sync-start', onStart)
      window.removeEventListener('watched:sync-end',   onEnd)
    }
  }, [])

  const state: SyncState = !online
    ? 'offline'
    : syncing
    ? 'syncing'
    : pendingCount > 0
    ? 'pending'
    : 'synced'

  return { state, pendingCount, online }
}
