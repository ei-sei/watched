import { useSyncStatus } from '@/hooks/useSyncStatus'

export default function SyncIndicator() {
  const { state, pendingCount } = useSyncStatus()

  // Nothing to show when idle — all is well and the last sync was a while ago
  if (state === 'idle') return null

  const configs = {
    offline: {
      dot: 'bg-red-500',
      label: 'Offline',
      title: 'No connection — changes will sync when back online',
    },
    pending: {
      dot: 'bg-amber-400',
      label: `${pendingCount} pending`,
      title: `${pendingCount} change${pendingCount !== 1 ? 's' : ''} waiting to sync`,
    },
    syncing: {
      dot: 'bg-blue-400 animate-pulse',
      label: 'Syncing…',
      title: 'Syncing with server',
    },
    synced: {
      dot: 'bg-emerald-500',
      label: 'Synced',
      title: 'All changes saved',
    },
  }

  const { dot, label, title } = configs[state]

  return (
    <span
      title={title}
      className="flex items-center gap-1.5 text-[10px] text-zinc-600 select-none"
    >
      <span className={`w-1.5 h-1.5 rounded-full flex-shrink-0 ${dot}`} />
      {label}
    </span>
  )
}
