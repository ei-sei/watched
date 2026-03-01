import { useState } from 'react'
import { Navigate } from 'react-router-dom'
import { useQuery } from '@tanstack/react-query'
import { useAuth } from '@/hooks/useAuth'
import { adminApi } from '@/api/admin'
import { formatDate } from '@/utils/formatters'
import { ChevronDown, ChevronUp } from 'lucide-react'
import type { AdminUser } from '@/api/admin'
import type { MediaItem } from '@/types/media'

const TYPE_LABELS: Record<string, string> = {
  film: 'Films', tv_show: 'TV Shows', anime: 'Anime', book: 'Books',
}
const TYPE_ORDER = ['film', 'tv_show', 'anime', 'book']

function LibraryPanel({ items }: { items: MediaItem[] }) {
  const grouped = TYPE_ORDER.reduce<Record<string, MediaItem[]>>((acc, type) => {
    const matching = items.filter(i => i.media_type === type)
    if (matching.length) acc[type] = matching
    return acc
  }, {})

  if (!items.length) return <p className="text-xs text-zinc-600 pt-2">Empty library.</p>

  return (
    <div className="space-y-4 pt-2">
      {Object.entries(grouped).map(([type, list]) => (
        <div key={type}>
          <p className="text-[10px] text-zinc-500 uppercase tracking-wider mb-1.5">
            {TYPE_LABELS[type]} · {list.length}
          </p>
          <div className="space-y-1">
            {list.map(item => (
              <div key={item.id} className="flex items-center justify-between gap-3">
                <p className="text-xs text-zinc-300 truncate">
                  {item.title}{item.year ? ` (${item.year})` : ''}
                </p>
                <span className="text-[10px] text-zinc-600 flex-shrink-0 capitalize">
                  {item.status.replace('_', ' ')}
                </span>
              </div>
            ))}
          </div>
        </div>
      ))}
    </div>
  )
}

function UserCard({ u }: { u: AdminUser }) {
  const [open, setOpen] = useState(false)

  const library = useQuery({
    queryKey: ['admin', 'user-library', u.id],
    queryFn: () => adminApi.getUserLibrary(u.id).then(r => r.data),
    enabled: open,
    staleTime: 1000 * 60 * 5,
  })

  return (
    <div className="bg-[#1a1a1a] rounded-lg ring-1 ring-white/[0.06] overflow-hidden">
      <button
        onClick={() => setOpen(o => !o)}
        className="w-full flex items-center gap-3 px-4 py-3 text-left group"
      >
        <div className="flex-1 min-w-0">
          <p className="text-sm text-zinc-200 font-medium">{u.username}</p>
          <p className="text-[10px] text-zinc-600 mt-0.5">
            Joined {formatDate(u.created_at)}
            {u.is_premium && <span className="ml-2 text-indigo-400">Premium</span>}
            {u.is_admin && <span className="ml-2 text-amber-400/70">Admin</span>}
          </p>
        </div>
        <span className="text-zinc-600 group-hover:text-zinc-400 transition-colors flex-shrink-0">
          {open ? <ChevronUp size={13} /> : <ChevronDown size={13} />}
        </span>
      </button>

      {open && (
        <div className="px-4 pb-4 border-t border-white/[0.04]">
          {library.isLoading && <p className="text-xs text-zinc-600 pt-2">Loading…</p>}
          {library.isError && <p className="text-xs text-red-400 pt-2">Failed to load</p>}
          {library.data && <LibraryPanel items={library.data} />}
        </div>
      )}
    </div>
  )
}

export default function UsersOverview() {
  const { user } = useAuth()

  const users = useQuery({
    queryKey: ['admin', 'users'],
    queryFn: () => adminApi.listUsers().then(r => r.data),
    staleTime: 1000 * 60 * 2,
    enabled: user?.username === 'admin',
  })

  if (user?.username !== 'admin') return <Navigate to="/" replace />

  return (
    <div className="space-y-4 max-w-2xl">
      <h1 className="text-2xl font-semibold text-white tracking-tight">Users</h1>

      {users.isLoading && <p className="text-zinc-600 text-sm">Loading…</p>}
      {users.isError && <p className="text-red-400 text-sm">Failed to load users.</p>}

      {users.data && (
        <div className="space-y-1.5">
          {users.data.map(u => (
            <UserCard key={u.id} u={u} />
          ))}
        </div>
      )}
    </div>
  )
}
