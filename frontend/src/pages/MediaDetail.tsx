import { useState } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import { useQueryClient } from '@tanstack/react-query'
import { useMediaItem, useUpdateMedia, useDeleteMedia } from '@/hooks/useMedia'
import EpisodeTracker from '@/components/tracking/EpisodeTracker'
import ChapterTracker from '@/components/tracking/ChapterTracker'
import ProgressTracker from '@/components/tracking/ProgressTracker'
import TVSeasonProgress from '@/components/tracking/TVSeasonProgress'
import StatusBadge from '@/components/ui/StatusBadge'
import RatingDisplay from '@/components/ui/RatingDisplay'
import LoadingSpinner from '@/components/ui/LoadingSpinner'
import { useToast } from '@/components/ui/Toast'
import { formatDate } from '@/utils/formatters'
import { mediaApi } from '@/api/media'
import { RefreshCw } from 'lucide-react'
import type { MediaStatus } from '@/types/media'

const REFRESH_COOLDOWN_MS = 60 * 60 * 1000 // 1 hour

function getLastRefresh(itemId: number): number {
  return parseInt(localStorage.getItem(`tmdb_refresh_${itemId}`) ?? '0', 10)
}
function setLastRefresh(itemId: number) {
  localStorage.setItem(`tmdb_refresh_${itemId}`, Date.now().toString())
}

const STATUSES: MediaStatus[] = ['want_to', 'in_progress', 'completed', 'dropped', 'on_hold']

export default function MediaDetail() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const { data: item, isLoading } = useMediaItem(Number(id))
  const update = useUpdateMedia()
  const remove = useDeleteMedia()
  const qc = useQueryClient()
  const { show } = useToast()
  const [refreshing, setRefreshing] = useState(false)

  if (isLoading) return <LoadingSpinner />
  if (!item) return <div className="text-slate-400">Not found</div>

  const lastRefresh = getLastRefresh(item.id)
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
      setLastRefresh(item.id)
      qc.invalidateQueries({ queryKey: ['media', item.id] })
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
      <div className="flex gap-6">
        {item.poster_url && (
          <img src={item.poster_url} alt={item.title} className="w-32 sm:w-48 rounded-lg object-cover flex-shrink-0" />
        )}
        <div className="space-y-3 flex-1 min-w-0">
          <h1 className="text-2xl font-bold text-white">{item.title}</h1>
          {item.year && <p className="text-slate-400">{item.year}</p>}
          <div className="flex items-center gap-3 flex-wrap">
            <StatusBadge status={item.status} />
            <RatingDisplay rating={item.rating} />
          </div>
          <p className="text-xs text-slate-500">Added {formatDate(item.created_at)}</p>

          <div className="space-y-2 pt-2">
            <div className="flex gap-2 items-center">
              <label className="text-sm text-slate-400 w-16">Status</label>
              <select value={item.status}
                onChange={(e) => update.mutate({ id: item.id, data: { status: e.target.value as MediaStatus } })}
                className="bg-slate-800 text-white rounded px-2 py-1 text-sm">
                {STATUSES.map((s) => <option key={s} value={s}>{s.replace('_', ' ')}</option>)}
              </select>
            </div>
            <div className="flex gap-2 items-center">
              <label className="text-sm text-slate-400 w-16">Rating</label>
              <input type="number" min={1} max={10} step={0.5} value={item.rating ?? ''}
                onChange={(e) => update.mutate({ id: item.id, data: { rating: e.target.value ? +e.target.value : null } })}
                className="w-20 bg-slate-800 text-white rounded px-2 py-1 text-sm" placeholder="1–10" />
            </div>
          </div>

          <div className="pt-2">
            <label className="block text-sm text-slate-400 mb-1">Review</label>
            <textarea defaultValue={item.review_text ?? ''}
              onBlur={(e) => update.mutate({ id: item.id, data: { review_text: e.target.value || null } })}
              rows={4} placeholder="Your thoughts…"
              className="w-full bg-slate-800 text-white rounded px-3 py-2 text-sm resize-none border border-slate-700 focus:outline-none focus:border-indigo-500" />
          </div>

          <button onClick={handleDelete} className="text-sm text-red-400 hover:text-red-300 transition-colors">
            Remove from library
          </button>
        </div>
      </div>

      {(item.media_type === 'book' || item.media_type === 'tv_show' || item.media_type === 'anime') && (
        <div className="space-y-2">
          <h3 className="text-sm font-semibold text-zinc-300 uppercase tracking-wider">Progress</h3>
          {item.media_type === 'tv_show' ? (
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
        </div>
      )}
      {item.media_type === 'tv_show' && <EpisodeTracker mediaId={item.id} />}
      {item.media_type === 'book' && <ChapterTracker mediaId={item.id} />}
    </div>
  )
}
