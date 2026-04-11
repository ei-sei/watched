import { useState, useEffect } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import { useQueryClient } from '@tanstack/react-query'
import { useLocalMediaItem } from '@/hooks/useLocalMedia'
import { db } from '@/offline/db'
import { useUpdateMedia, useDeleteMedia } from '@/hooks/useMedia'
import { syncItemDetail } from '@/offline/sync'
import EpisodeTracker from '@/components/tracking/EpisodeTracker'
import ChapterTracker from '@/components/tracking/ChapterTracker'
import ProgressTracker from '@/components/tracking/ProgressTracker'
import TVSeasonProgress from '@/components/tracking/TVSeasonProgress'
import { useToast } from '@/components/ui/Toast'
import { formatDate } from '@/utils/formatters'
import { mediaApi } from '@/api/media'
import { RefreshCw, Trash2 } from 'lucide-react'
import { STATUS_LABELS, STATUS_LABELS_BOOK, STATUS_COLOURS } from '@/utils/constants'
import type { MediaItem, MediaStatus } from '@/types/media'

const REFRESH_COOLDOWN_MS = 60 * 60 * 1000 // 1 hour

function refreshKey(item: MediaItem) {
  return `${item.media_type}_refresh_${item.id}`
}
function getLastRefresh(item: MediaItem): number {
  return parseInt(localStorage.getItem(refreshKey(item)) ?? '0', 10)
}
function setLastRefresh(item: MediaItem) {
  localStorage.setItem(refreshKey(item), Date.now().toString())
}

const STATUSES: MediaStatus[] = ['want_to', 'in_progress', 'completed', 'dropped', 'on_hold']

export default function MediaDetail() {
  const { id } = useParams<{ id: string }>()
  const mediaId = Number(id)
  const navigate = useNavigate()
  // useLiveQuery resolves synchronously from IndexedDB — no network spinner
  const item = useLocalMediaItem(mediaId)
  const update = useUpdateMedia()
  const remove = useDeleteMedia()
  const qc = useQueryClient()
  const { show } = useToast()
  const [refreshing, setRefreshing] = useState(false)

  // Pull fresh episodes/chapters for this item in the background
  useEffect(() => {
    if (mediaId > 0) syncItemDetail(mediaId).catch(console.error)
  }, [mediaId])

  // item is undefined only for the one Dexie initialisation tick — treat as loading
  if (item === undefined) return null
  if (item === null) return <div className="text-slate-400">Not found</div>

  const lastRefresh = getLastRefresh(item)
  const cooldownRemaining = REFRESH_COOLDOWN_MS - (Date.now() - lastRefresh)
  const onCooldown = cooldownRemaining > 0

  const handleDelete = async () => {
    if (!confirm('Remove from library?')) return
    await remove.mutateAsync(item.id)
    navigate(-1)
  }

  const handleRefresh = async () => {
    if (onCooldown) {
      const mins = Math.ceil(cooldownRemaining / 60000)
      show(`Please wait ${mins} minute${mins !== 1 ? 's' : ''} before refreshing again`, 'error')
      return
    }
    const oldTotal = item.total_progress
    setRefreshing(true)
    try {
      const { data: updated } = await mediaApi.refreshFromTMDB(item.id)
      await db.mediaItems.put(updated)
      setLastRefresh(item)
      qc.invalidateQueries({ queryKey: ['stats'] })
      if (updated.total_progress === oldTotal) {
        show('No new episodes — count is up to date', 'info')
      } else {
        show(`Updated: ${oldTotal ?? '?'} → ${updated.total_progress} episodes`, 'success')
      }
    } catch {
      show('Could not fetch from TMDB', 'error')
    } finally {
      setRefreshing(false)
    }
  }

  return (
    <div className="max-w-3xl mx-auto space-y-8">
      <div className="flex flex-col sm:flex-row gap-5">
        {item.poster_url && (
          <img src={item.poster_url} alt={item.title} className="w-32 sm:w-44 rounded-lg object-cover flex-shrink-0 shadow-lg self-start" />
        )}
        <div className="flex-1 min-w-0 space-y-4">
          <div className="flex items-start justify-between gap-2">
            <div>
              <h1 className="text-2xl font-bold text-white leading-tight">{item.title}</h1>
              <p className="text-sm text-zinc-500 mt-1">
                {item.year && <span>{item.year} · </span>}
                Added {formatDate(item.created_at)}
              </p>
            </div>
            <button
              onClick={handleDelete}
              className="p-1.5 text-zinc-700 hover:text-red-400 transition-colors flex-shrink-0 mt-0.5"
              title="Remove from library"
            >
              <Trash2 size={16} />
            </button>
          </div>

          {/* Status pills */}
          <div className="flex flex-wrap gap-1.5">
            {STATUSES.map((s) => {
              const active = item.status === s
              const colours = STATUS_COLOURS[s] ?? 'bg-zinc-700 text-zinc-300'
              return (
                <button
                  key={s}
                  onClick={() => {
                    const today = new Date().toISOString().slice(0, 10)
                    const extra: Partial<MediaItem> = {}
                    if (s === 'completed' && !item.completed_at) extra.completed_at = today
                    if (s === 'in_progress' && !item.started_at) extra.started_at = today
                    update.mutate({ id: item.id, data: { status: s, ...extra } })
                  }}
                  className={`px-3 py-1 rounded-full text-xs font-medium transition-all ${
                    active
                      ? colours
                      : 'bg-transparent text-zinc-600 hover:text-zinc-400 border border-white/[0.06] hover:border-white/[0.12]'
                  }`}
                >
                  {(item.media_type === 'book' ? STATUS_LABELS_BOOK : STATUS_LABELS)[s]}
                </button>
              )
            })}
          </div>

          {/* Rating row */}
          <div className="space-y-1.5">
            <p className="text-xs text-zinc-600 uppercase tracking-wider">Rating</p>
            <div className="flex flex-wrap gap-1">
              {Array.from({ length: 10 }, (_, i) => i + 1).map((n) => {
                const active = item.rating !== null && item.rating !== undefined && n <= item.rating
                const isSelected = item.rating === n
                return (
                  <button
                    key={n}
                    onClick={() => update.mutate({ id: item.id, data: { rating: isSelected ? null : n } })}
                    className={`w-7 h-7 rounded text-xs font-semibold transition-all ${
                      isSelected
                        ? 'bg-indigo-500 text-white scale-110'
                        : active
                        ? 'bg-indigo-500/30 text-indigo-300'
                        : 'bg-white/[0.04] text-zinc-600 hover:bg-white/[0.08] hover:text-zinc-400'
                    }`}
                  >
                    {n}
                  </button>
                )
              })}
            </div>
          </div>

          {/* Review */}
          <div className="space-y-1.5">
            <p className="text-xs text-zinc-600 uppercase tracking-wider">Review</p>
            <textarea
              defaultValue={item.review_text ?? ''}
              onBlur={(e) => update.mutate({ id: item.id, data: { review_text: e.target.value || null } })}
              rows={3}
              placeholder="Your thoughts…"
              className="w-full bg-white/[0.03] text-zinc-300 placeholder-zinc-700 rounded-lg px-3 py-2 text-sm resize-none focus:outline-none focus:ring-1 focus:ring-indigo-500/50 transition-all"
            />
          </div>
        </div>
      </div>

      {(item.media_type === 'book' || item.media_type === 'tv_show' || item.media_type === 'anime') && (
        <div className="space-y-2">
          <h3 className="text-sm font-semibold text-zinc-300 uppercase tracking-wider">Progress</h3>
          {(item.media_type === 'tv_show' || item.media_type === 'anime') ? (
            <TVSeasonProgress item={item} />
          ) : (
            <ProgressTracker key={`${item.id}-${item.current_progress}-${item.total_progress}`} item={item} />
          )}
          {item.media_type === 'tv_show' && item.external_id?.startsWith('tmdb:') && (
            <button
              onClick={handleRefresh}
              disabled={refreshing || onCooldown}
              className="flex items-center gap-1.5 text-xs text-zinc-500 hover:text-zinc-300 transition-colors disabled:opacity-40"
            >
              <RefreshCw size={12} className={refreshing ? 'animate-spin' : ''} />
              {refreshing ? 'Updating…' : onCooldown ? `Refresh available in ${Math.ceil(cooldownRemaining / 60000)}m` : 'Refresh episode count from TMDB'}
            </button>
          )}
          {item.media_type === 'anime' && item.external_id?.startsWith('mal:') && (() => {
            const s = (item.metadata?.seasons as { episode_count: number }[] | undefined) ?? []
            return s.length === 1 && s[0]?.episode_count === 0
          })() && (
            <button
              onClick={handleRefresh}
              disabled={refreshing || onCooldown}
              className="flex items-center gap-1.5 text-xs text-zinc-500 hover:text-zinc-300 transition-colors disabled:opacity-40"
            >
              <RefreshCw size={12} className={refreshing ? 'animate-spin' : ''} />
              {refreshing ? 'Updating…' : onCooldown ? `Refresh available in ${Math.ceil(cooldownRemaining / 60000)}m` : 'Refresh episode count from MAL'}
            </button>
          )}
        </div>
      )}
      {(item.media_type === 'tv_show' || item.media_type === 'anime') && <EpisodeTracker item={item} />}
      {item.media_type === 'book' && <ChapterTracker mediaId={item.id} />}
    </div>
  )
}
