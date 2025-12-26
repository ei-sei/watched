import { useQuery } from '@tanstack/react-query'
import { episodesApi } from '@/api/episodes'
import type { MediaItem } from '@/types/media'

interface Season {
  season_number: number
  episode_count: number
}

export default function TVSeasonProgress({ item }: { item: MediaItem }) {
  const seasons = (item.metadata?.seasons as Season[] | undefined) ?? []

  const { data: episodes = [] } = useQuery({
    queryKey: ['episodes', item.id],
    queryFn: () => episodesApi.list(item.id).then((r) => r.data),
  })

  const watchedBySeason = episodes.reduce<Record<number, number>>((acc, ep) => {
    acc[ep.season_number] = (acc[ep.season_number] ?? 0) + 1
    return acc
  }, {})

  const totalWatched = episodes.length
  const totalEps = item.total_progress ?? seasons.reduce((a, s) => a + s.episode_count, 0)

  if (seasons.length === 0) {
    const pct = totalEps > 0 ? Math.min(100, Math.round((totalWatched / totalEps) * 100)) : null
    return (
      <div className="space-y-1">
        <div className="flex justify-between text-xs text-zinc-500 mb-1">
          <span>{totalWatched} / {totalEps || '?'} episodes watched</span>
          {pct !== null && <span>{pct}%</span>}
        </div>
        <div className="h-1.5 bg-white/[0.06] rounded-full overflow-hidden">
          <div
            className="h-full bg-indigo-500 rounded-full transition-all"
            style={{ width: pct !== null ? `${pct}%` : '0%' }}
          />
        </div>
      </div>
    )
  }

  const overallPct = totalEps > 0 ? Math.min(100, Math.round((totalWatched / totalEps) * 100)) : 0

  return (
    <div className="space-y-2.5">
      <div>
        <div className="flex justify-between text-xs text-zinc-400 mb-1">
          <span>Overall</span>
          <span>{totalWatched} / {totalEps} episodes</span>
        </div>
        <div className="h-1.5 bg-white/[0.06] rounded-full overflow-hidden">
          <div
            className="h-full bg-indigo-500 rounded-full transition-all"
            style={{ width: `${overallPct}%` }}
          />
        </div>
      </div>

      {seasons.map((s) => {
        const watched = watchedBySeason[s.season_number] ?? 0
        const pct = s.episode_count > 0 ? Math.min(100, Math.round((watched / s.episode_count) * 100)) : 0
        const done = watched >= s.episode_count
        return (
          <div key={s.season_number}>
            <div className="flex justify-between text-xs text-zinc-500 mb-0.5">
              <span>Season {s.season_number}</span>
              <span>{watched} / {s.episode_count}</span>
            </div>
            <div className="h-1 bg-white/[0.04] rounded-full overflow-hidden">
              <div
                className={`h-full rounded-full transition-all ${done ? 'bg-green-500' : 'bg-indigo-400'}`}
                style={{ width: `${pct}%` }}
              />
            </div>
          </div>
        )
      })}
    </div>
  )
}
