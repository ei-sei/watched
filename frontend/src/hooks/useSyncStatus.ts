import { useState, useEffect, useRef } from 'react'
import { useLiveQuery } from 'dexie-react-hooks'
import { db } from '@/offline/db'

// 'idle' = synced and nothing to show; only shown briefly after a sync finishes
export type SyncState = 'offline' | 'pending' | 'syncing' | 'synced' | 'idle'

const SYNCED_LINGER_MS = 3000 // how long "Synced" stays visible after a sync

export function useSyncStatus() {
  const [online, setOnline] = useState(navigator.onLine)
  const [syncing, setSyncing] = useState(false)
  const [justSynced, setJustSynced] = useState(false)
  const lingerTimer = useRef<ReturnType<typeof setTimeout> | null>(null)

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

  useEffect(() => {
    const onStart = () => {
      setSyncing(true)
      setJustSynced(false)
      if (lingerTimer.current) clearTimeout(lingerTimer.current)
    }
    const onEnd = () => {
      setSyncing(false)
      setJustSynced(true)
      lingerTimer.current = setTimeout(() => setJustSynced(false), SYNCED_LINGER_MS)
    }
    window.addEventListener('watched:sync-start', onStart)
    window.addEventListener('watched:sync-end',   onEnd)
    return () => {
      window.removeEventListener('watched:sync-start', onStart)
      window.removeEventListener('watched:sync-end',   onEnd)
      if (lingerTimer.current) clearTimeout(lingerTimer.current)
    }
  }, [])

  const state: SyncState = !online
    ? 'offline'
    : syncing
    ? 'syncing'
    : pendingCount > 0
    ? 'pending'
    : justSynced
    ? 'synced'
    : 'idle'

  return { state, pendingCount, online }
}
