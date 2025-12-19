import { useEffect, useState } from 'react'
import { useParams, Link } from 'react-router-dom'
import client from '@/api/client'
import type { MediaItem, MediaType } from '@/types/media'
import LoadingSpinner from '@/components/ui/LoadingSpinner'

interface ProfileData {
  username: string
  display_name: string | null
  avatar_url: string | null
  media: MediaItem[]
}

const TABS: { type: MediaType; label: string }[] = [
  { type: 'film',    label: 'Films' },
  { type: 'tv_show', label: 'TV Shows' },
  { type: 'book',    label: 'Books' },
  { type: 'anime',   label: 'Anime' },
]

const STATUS_LABEL: Record<string, string> = {
  want_to: 'Want to',
  in_progress: 'In Progress',
  completed: 'Completed',
  dropped: 'Dropped',
  on_hold: 'On Hold',
}

export default function PublicProfile() {
  const { username } = useParams<{ username: string }>()
  const [profile, setProfile] = useState<ProfileData | null>(null)
  const [loading, setLoading] = useState(true)
  const [notFound, setNotFound] = useState(false)
  const [tab, setTab] = useState<MediaType>('film')

  useEffect(() => {
    if (!username) return
    client.get<ProfileData>(`/u/${username}`)
      .then((r) => setProfile(r.data))
      .catch(() => setNotFound(true))
      .finally(() => setLoading(false))
  }, [username])

  if (loading) return (
    <div className="min-h-screen bg-[#0d0d0d] flex items-center justify-center">
      <LoadingSpinner />
    </div>
  )

  if (notFound || !profile) return (
    <div className="min-h-screen bg-[#0d0d0d] flex items-center justify-center">
      <div className="text-center space-y-3">
        <p className="text-zinc-400 text-sm">This profile doesn't exist or isn't public.</p>
        <Link to="/" className="text-indigo-400 text-sm hover:underline">Go to Watched</Link>
      </div>
    </div>
  )

  const filtered = profile.media.filter((m) => m.media_type === tab)

  return (
    <div className="min-h-screen bg-[#0d0d0d] text-zinc-200">
      <div className="max-w-2xl mx-auto px-4 py-12 space-y-8">
        {/* Header */}
        <div className="space-y-1">
          <h1 className="text-2xl font-semibold text-white tracking-tight">
            {profile.display_name ?? profile.username}
          </h1>
          <p className="text-sm text-zinc-600">@{profile.username}</p>
        </div>

        {/* Tabs */}
        <div className="flex items-center gap-1">
          {TABS.map(({ type, label }) => {
            const count = profile.media.filter((m) => m.media_type === type).length
            return (
              <button
                key={type}
                onClick={() => setTab(type)}
                className={`px-3 py-1.5 rounded-md text-xs font-medium transition-colors ${
                  tab === type
                    ? 'bg-white/10 text-white'
                    : 'text-zinc-500 hover:bg-white/5 hover:text-zinc-300'
                }`}
              >
                {label}
                {count > 0 && (
                  <span className="ml-1.5 text-zinc-600">{count}</span>
                )}
              </button>
            )
          })}
        </div>

        {/* Media list */}
        {filtered.length === 0 ? (
          <p className="text-zinc-600 text-sm">Nothing here yet.</p>
        ) : (
          <div className="space-y-1.5">
            {filtered.map((item) => (
              <div key={item.id} className="flex items-center gap-3 bg-[#1a1a1a] rounded-lg p-3 ring-1 ring-white/[0.06]">
                {item.poster_url ? (
                  <img src={item.poster_url} alt={item.title} className="w-8 h-11 object-cover rounded flex-shrink-0" />
                ) : (
                  <div className="w-8 h-11 bg-[#222] rounded flex-shrink-0" />
                )}
                <div className="flex-1 min-w-0">
                  <p className="text-sm font-medium text-zinc-100 truncate">{item.title}</p>
                  <p className="text-xs text-zinc-600 mt-0.5">
                    {item.year && `${item.year} · `}{STATUS_LABEL[item.status] ?? item.status}
                  </p>
                </div>
                {item.rating && (
                  <span className="text-xs text-zinc-500 flex-shrink-0">{item.rating}/10</span>
                )}
              </div>
            ))}
          </div>
        )}

        {/* Footer */}
        <div className="pt-4 border-t border-white/[0.06]">
          <Link to="/" className="text-xs text-zinc-600 hover:text-zinc-400 transition-colors">
            Made with Watched
          </Link>
        </div>
      </div>
    </div>
  )
}
