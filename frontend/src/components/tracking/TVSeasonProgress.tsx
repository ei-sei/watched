import { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { useLocalEpisodes } from '@/hooks/useLocalEpisodes'
import { mediaApi, type EpisodeStamp } from '@/api/media'
import { formatDate } from '@/utils/formatters'
import type { MediaItem } from '@/types/media'

interface Season {
  season_number: number
  episode_count: number
}

type DayGroup = {
  date: string
  watched_at: string
  episodes: number[]
}

function groupByDay(stamps: EpisodeStamp[]): DayGroup[] {
  const map = new Map<string, DayGroup>()
  for (const s of stamps) {
    const day = s.watched_at.slice(0, 10)
    if (!map.has(day)) map.set(day, { date: day, watched_at: s.watched_at, episodes: [] })
    map.get(day)!.episodes.push(s.episode)
  }
  return Array.from(map.values())
    .sort((a, b) => a.date.localeCompare(b.date))
    .map(g => ({ ...g, episodes: g.episodes.sort((a, b) => a - b) }))
}

function epLabel(episodes: number[]): string {
  if (episodes.length === 1) return `Ep ${episodes[0]}`
  const isConsecutive = episodes.every((ep, i) => i === 0 || ep === episodes[i - 1] + 1)
  if (isConsecutive) return `Ep ${episodes[0]}–${episodes[episodes.length - 1]}`
  if (episodes.length <= 4) return `Ep ${episodes.join(', ')}`
  return `Ep ${episodes[0]}, ${episodes[1]} +${episodes.length - 2} more`
}

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
  const [activeKey, setActiveKey] = useState<string | null>(null)

  const groups = groupByDay(stamps)

  // Position each dot by the highest episode in that day's group, on the same
  // scale as the bar fill — so the dot always sits within the blue region.
  const getLeftPct = (episodes: number[]) => {
    if (totalEps <= 0) return 0
    const maxEp = Math.max(...episodes)
    return ((maxEp - 0.5) / totalEps) * 100
  }

  return (
    <div className="relative h-4">
      <div className={`absolute inset-x-0 top-1/2 -translate-y-1/2 ${height} bg-white/[0.06] rounded-full overflow-hidden`}>
        <div className={`h-full ${color} rounded-full transition-all`} style={{ width: `${pct}%` }} />
      </div>

      {groups.map((g) => {
        const leftPct = getLeftPct(g.episodes)
        const isActive = activeKey === g.date
        return (
          <div
            key={g.date}
            className="absolute top-1/2 -translate-y-1/2 -translate-x-1/2 cursor-pointer"
            style={{ left: `${leftPct}%` }}
            onMouseEnter={() => setActiveKey(g.date)}
            onMouseLeave={() => setActiveKey(null)}
            onClick={() => setActiveKey(isActive ? null : g.date)}
          >
            <div className={`w-1.5 h-1.5 rounded-full bg-white/90 border border-black/20 transition-transform ${isActive ? 'scale-125' : ''}`} />
            {isActive && (
              <div className="absolute bottom-full left-1/2 -translate-x-1/2 mb-2 whitespace-nowrap bg-zinc-800 text-zinc-100 text-[10px] px-2 py-1 rounded shadow-lg z-10 pointer-events-none">
                {epLabel(g.episodes)} · {formatDate(g.watched_at)}
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
