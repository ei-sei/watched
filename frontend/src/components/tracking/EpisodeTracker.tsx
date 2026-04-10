import { useState } from 'react'
import { useLocalEpisodes, useLogEpisode, useDeleteEpisode } from '@/hooks/useLocalEpisodes'
import { formatDate } from '@/utils/formatters'
import type { MediaItem, EpisodeLog } from '@/types/media'

interface Season {
  season_number: number
  episode_count: number
}

interface Props { item: MediaItem }

export default function EpisodeTracker({ item }: Props) {
  const seasons = (item.metadata?.seasons as Season[] | undefined) ?? []
  const episodes = useLocalEpisodes(item.id)
  const log = useLogEpisode(item.id)
  const remove = useDeleteEpisode(item.id)

  if (seasons.length > 0) {
    return (
      <div className="space-y-1">
        <h3 className="text-sm font-semibold text-zinc-300 uppercase tracking-wider mb-3">Episodes</h3>
        {seasons.map((s) => {
          const watched = episodes
            .filter((e) => e.season_number === s.season_number)
            .sort((a, b) => b.episode_number - a.episode_number)
          const watchedCount = watched.length
          const maxEp = watched[0]?.episode_number ?? 0
          const ongoing = s.episode_count === 0
          const ongoingSingle = ongoing && seasons.length === 1
          const canAdd = ongoing || watchedCount < s.episode_count
          const canRemove = watchedCount > 0
          const isComplete = !ongoing && watchedCount === s.episode_count

          // Derive per-season dates from episode logs so each season is independent.
          // For single-season items, fall back to item-level started_at/completed_at.
          const withDates = watched.filter((e) => e.watched_at)
          const earliestEp = withDates.reduce<EpisodeLog | null>(
            (min, e) => (!min || e.watched_at! < min.watched_at!) ? e : min, null
          )
          const latestEp = withDates.reduce<EpisodeLog | null>(
            (max, e) => (!max || e.watched_at! > max.watched_at!) ? e : max, null
          )
          const isSingleSeason = seasons.length === 1
          const startLabel = earliestEp
            ? formatDate(earliestEp.watched_at)
            : (isSingleSeason && item.started_at ? formatDate(item.started_at) : null)
          const endLabel = isComplete
            ? (latestEp
                ? formatDate(latestEp.watched_at)
                : (isSingleSeason && item.completed_at ? formatDate(item.completed_at) : null))
            : null

          return (
            <div key={s.season_number} className="flex items-center gap-2 sm:gap-3 py-1 flex-wrap">
              {!ongoingSingle && (
                <span className="text-sm text-zinc-400 w-16 sm:w-20 flex-shrink-0">S{s.season_number < 10 ? '0' + s.season_number : s.season_number}</span>
              )}
              <div className="flex items-center gap-2 bg-[#1a1a1a] rounded-md border border-white/[0.08] px-1">
                <button
                  onClick={() => watched[0] && remove.mutate(watched[0].id)}
                  disabled={!canRemove || remove.isPending}
                  className="w-7 h-7 flex items-center justify-center text-zinc-500 hover:text-zinc-200 disabled:opacity-30 transition-colors text-base"
                >−</button>
                <span className="text-sm text-zinc-300 w-20 text-center tabular-nums">
                  {watchedCount} / {ongoing ? (item.media_type === 'anime' ? '∞' : '?') : s.episode_count}
                </span>
                <button
                  onClick={() => log.mutate({ season: s.season_number, episode: maxEp + 1 })}
                  disabled={!canAdd || log.isPending}
                  className="w-7 h-7 flex items-center justify-center text-zinc-500 hover:text-zinc-200 disabled:opacity-30 transition-colors text-base"
                >+</button>
              </div>
              {(startLabel || endLabel) && (
                <span className="text-xs text-zinc-600">
                  {startLabel && endLabel
                    ? `${startLabel} → ${endLabel}`
                    : startLabel ?? endLabel}
                </span>
              )}
            </div>
          )
        })}
      </div>
    )
  }

  // Fallback for items without season data
  return <ManualEpisodeTracker episodes={episodes} log={log} remove={remove} />
}

function ManualEpisodeTracker({ episodes, log, remove }: {
  episodes: EpisodeLog[]
  log: ReturnType<typeof useLogEpisode>
  remove: ReturnType<typeof useDeleteEpisode>
}) {
  const [newSeason, setNewSeason] = useState(1)
  const [newEpisode, setNewEpisode] = useState(1)

  const byseason = episodes.reduce<Record<number, EpisodeLog[]>>((acc, ep) => {
    if (!acc[ep.season_number]) acc[ep.season_number] = []
    acc[ep.season_number].push(ep)
    return acc
  }, {})

  return (
    <div className="space-y-4">
      <h3 className="text-sm font-semibold text-zinc-300 uppercase tracking-wider">Episodes</h3>
      <div className="flex gap-2 items-end">
        <div>
          <label className="block text-xs text-slate-400 mb-1">Season</label>
          <input type="number" min={1} value={newSeason} onChange={(e) => setNewSeason(+e.target.value)}
            className="w-20 bg-slate-700 text-white rounded px-2 py-1 text-sm" />
        </div>
        <div>
          <label className="block text-xs text-slate-400 mb-1">Episode</label>
          <input type="number" min={1} value={newEpisode} onChange={(e) => setNewEpisode(+e.target.value)}
            className="w-20 bg-slate-700 text-white rounded px-2 py-1 text-sm" />
        </div>
        <button onClick={() => log.mutate({ season: newSeason, episode: newEpisode })}
          className="bg-indigo-600 hover:bg-indigo-700 text-white px-3 py-1 rounded text-sm">
          Log
        </button>
      </div>
      {Object.entries(byseason).sort(([a], [b]) => +a - +b).map(([season, eps]) => (
        <div key={season}>
          <p className="text-sm font-medium text-slate-300 mb-2">Season {season}</p>
          <div className="flex flex-wrap gap-2">
            {eps.sort((a, b) => a.episode_number - b.episode_number).map((ep) => (
              <button key={ep.id} onClick={() => remove.mutate(ep.id)}
                title="Click to remove"
                className="w-9 h-9 rounded bg-indigo-600 text-white text-xs font-bold hover:bg-red-600 transition-colors">
                {ep.episode_number}
              </button>
            ))}
          </div>
        </div>
      ))}
    </div>
  )
}
