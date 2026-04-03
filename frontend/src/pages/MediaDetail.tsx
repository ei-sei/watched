import { useParams, useNavigate } from 'react-router-dom'
import { useMediaItem, useUpdateMedia, useDeleteMedia } from '@/hooks/useMedia'
import EpisodeTracker from '@/components/tracking/EpisodeTracker'
import ChapterTracker from '@/components/tracking/ChapterTracker'
import StatusBadge from '@/components/ui/StatusBadge'
import RatingDisplay from '@/components/ui/RatingDisplay'
import LoadingSpinner from '@/components/ui/LoadingSpinner'
import { formatDate } from '@/utils/formatters'
import type { MediaStatus } from '@/types/media'

const STATUSES: MediaStatus[] = ['want_to', 'in_progress', 'completed', 'dropped', 'on_hold']

export default function MediaDetail() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const { data: item, isLoading } = useMediaItem(Number(id))
  const update = useUpdateMedia()
  const remove = useDeleteMedia()

  if (isLoading) return <LoadingSpinner />
  if (!item) return <div className="text-slate-400">Not found</div>

  const handleDelete = async () => {
    if (!confirm('Remove from library?')) return
    await remove.mutateAsync(item.id)
    navigate(-1)
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

      {item.media_type === 'tv_show' && <EpisodeTracker mediaId={item.id} />}
      {item.media_type === 'book' && <ChapterTracker mediaId={item.id} />}
    </div>
  )
}
