import { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { useLocalEpisodes } from '@/hooks/useLocalEpisodes'
import { mediaApi, type EpisodeStamp } from '@/api/media'
import type { MediaItem } from '@/types/media'

interface Season {
  season_number: number
  episode_count: number
}

function fmtDate(iso: string) {
  return new Date(iso).toLocaleDateString(undefined, { day: 'numeric', month: 'short', year: 'numeric' })
}

// Dots overlaid on a progress bar. For < 50 total episodes: one dot per episode.
// For >= 50: single dot at the last watched episode only.
function StampedBar({
  pct,
  stamps,
  totalEps,
  color = 'bg-indigo-500',
  height = 'h-1.5',
}: {
  pct: number
  stamps: EpisodeStamp[]
  totalEps: number
  color?: string
  height?: string
}) {
  const [activeEp, setActiveEp] = useState<number | null>(null)

  const dots: EpisodeStamp[] = totalEps >= 50
    ? stamps.length > 0 ? [stamps[stamps.length - 1]] : []
    : stamps

  return (
    <div className="relative flex items-center" style={{ height: '16px' }}>
      {/* Bar */}
      <div className={`absolute inset-x-0 ${height} bg-white/[0.06] rounded-full overflow-hidden`}>
        <div className={`h-full ${color} rounded-full transition-all`} style={{ width: `${pct}%` }} />
      </div>

      {/* Dots */}
      {totalEps > 0 && dots.map((s) => {
        const left = `${Math.min(99, (s.episode / totalEps) * 100)}%`
        const isActive = activeEp === s.episode
        return (
          <div
            key={s.episode}
            className="absolute -translate-x-1/2 cursor-pointer group"
            style={{ left }}
            onMouseEnter={() => setActiveEp(s.episode)}
            onMouseLeave={() => setActiveEp(null)}
            onClick={() => setActiveEp(isActive ? null : s.episode)}
          >
            <div className={`w-2 h-2 rounded-full border border-black/30 transition-transform ${isActive ? 'scale-125' : ''} bg-white/80`} />
            {isActive && (
              <div className="absolute bottom-full left-1/2 -translate-x-1/2 mb-1.5 whitespace-nowrap bg-zinc-800 text-zinc-100 text-[10px] px-2 py-1 rounded shadow-lg z-10 pointer-events-none">
                Ep {s.episode} · {fmtDate(s.watched_at)}
              </div>
            )}
          </div>
        )
      })}
    </div>
  )
}

export default function TVSeasonProgress({ item }: { item: MediaItem }) {
  const seasons = (item.metadata?.seasons as Season[] | undefined) ?? []
  const episodes = useLocalEpisodes(item.id) ?? []

  const { data: stamps = [] } = useQuery({
    queryKey: ['episode-stamps', item.id],
    queryFn: () => mediaApi.listEpisodeStamps(item.id).then(r => r.data),
    staleTime: 1000 * 60 * 5,
  })

  const watchedBySeason = episodes.reduce<Record<number, number>>((acc, ep) => {
    acc[ep.season_number] = (acc[ep.season_number] ?? 0) + 1
    return acc
  }, {})

  const stampsBySeason = stamps.reduce<Record<number, EpisodeStamp[]>>((acc, s) => {
    ;(acc[s.season] ??= []).push(s)
    return acc
  }, {})

  const totalWatched = episodes.length
  const totalEps = item.total_progress ?? seasons.reduce((a, s) => a + s.episode_count, 0)

  const ongoingSingle = seasons.length === 1 && seasons[0].episode_count === 0
  const unknownLabel = item.media_type === 'anime' ? '∞' : '?'

  if (seasons.length === 0 || ongoingSingle) {
    const pct = totalEps > 0 ? Math.min(100, Math.round((totalWatched / totalEps) * 100)) : 0
    return (
      <div className="space-y-1">
        <div className="flex justify-between text-xs text-zinc-500 mb-1">
          <span>{totalWatched} / {totalEps > 0 ? totalEps : unknownLabel} episodes watched</span>
          {totalEps > 0 && <span>{pct}%</span>}
        </div>
        <StampedBar pct={pct} stamps={stamps} totalEps={totalEps} />
      </div>
    )
  }

  const overallPct = totalEps > 0 ? Math.min(100, Math.round((totalWatched / totalEps) * 100)) : 0

  return (
    <div className="space-y-2.5">
      <div>
        <div className="flex justify-between text-xs text-zinc-400 mb-1">
          <span>Overall</span>
          <span>{totalWatched} / {totalEps > 0 ? totalEps : unknownLabel} episodes</span>
        </div>
        <StampedBar pct={overallPct} stamps={stamps} totalEps={totalEps} />
      </div>

      {item.media_type !== 'anime' && seasons.map((s) => {
        const watched = watchedBySeason[s.season_number] ?? 0
        const pct = s.episode_count > 0 ? Math.min(100, Math.round((watched / s.episode_count) * 100)) : 0
        const done = s.episode_count > 0 && watched >= s.episode_count
        const seasonStamps = stampsBySeason[s.season_number] ?? []
        return (
          <div key={s.season_number}>
            <div className="flex justify-between text-xs text-zinc-500 mb-0.5">
              <span>Season {s.season_number}</span>
              <span>{watched} / {s.episode_count > 0 ? s.episode_count : unknownLabel}</span>
            </div>
            <StampedBar
              pct={pct}
              stamps={seasonStamps}
              totalEps={s.episode_count}
              color={done ? 'bg-green-500' : 'bg-indigo-400'}
              height="h-1"
            />
          </div>
        )
      })}
    </div>
  )
}
