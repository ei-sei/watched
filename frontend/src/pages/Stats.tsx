import { useStats } from '@/hooks/useStats'
import LoadingSpinner from '@/components/ui/LoadingSpinner'
import { useAuth } from '@/hooks/useAuth'
import { BarChart2, Lock } from 'lucide-react'

function Sparkline({ data, colour, id }: { data: number[]; colour: string; id: string }) {
  const w = 200
  const h = 48
  const max = Math.max(...data, 1)
  const pts = data.map((v, i) => ({
    x: (i / Math.max(data.length - 1, 1)) * w,
    y: h - 4 - (v / max) * (h - 8),
  }))

  // Smooth cubic bezier path
  const linePath = pts.reduce((path, pt, i) => {
    if (i === 0) return `M ${pt.x.toFixed(1)} ${pt.y.toFixed(1)}`
    const prev = pts[i - 1]
    const cx = ((prev.x + pt.x) / 2).toFixed(1)
    return `${path} C ${cx} ${prev.y.toFixed(1)}, ${cx} ${pt.y.toFixed(1)}, ${pt.x.toFixed(1)} ${pt.y.toFixed(1)}`
  }, '')

  const last = pts[pts.length - 1]
  const areaPath = `${linePath} L ${last.x.toFixed(1)} ${h} L 0 ${h} Z`
  const gradId = `spark-${id}`

  return (
    <svg viewBox={`0 0 ${w} ${h}`} className="w-full" style={{ height: 48 }} preserveAspectRatio="none">
      <defs>
        <linearGradient id={gradId} x1="0" y1="0" x2="0" y2="1">
          <stop offset="0%" stopColor={colour} stopOpacity="0.35" />
          <stop offset="100%" stopColor={colour} stopOpacity="0.02" />
        </linearGradient>
      </defs>
      <path d={areaPath} fill={`url(#${gradId})`} />
      <path d={linePath} fill="none" stroke={colour} strokeWidth="1.5" />
      <circle cx={last.x.toFixed(1)} cy={last.y.toFixed(1)} r="2.5" fill={colour} />
    </svg>
  )
}

function formatTime(minutes: number): string {
  if (minutes < 60) return `${minutes}m`
  const h = Math.floor(minutes / 60)
  if (h < 24) return `${h}h`
  const d = Math.floor(h / 24)
  const rem = h % 24
  return rem > 0 ? `${d}d ${rem}h` : `${d}d`
}

export default function Stats() {
  const { user } = useAuth()
  const { data: s, isLoading } = useStats()

  if (!user?.is_premium && !user?.is_admin) {
    return (
      <div className="max-w-2xl">
        <h1 className="text-2xl font-semibold text-white tracking-tight mb-5">Stats</h1>
        <div className="bg-[#1a1a1a] rounded-xl p-8 ring-1 ring-white/[0.06] flex flex-col items-center gap-4 text-center">
          <div className="w-12 h-12 rounded-full bg-white/[0.06] flex items-center justify-center">
            <Lock size={20} className="text-zinc-500" />
          </div>
          <div className="space-y-1">
            <p className="text-white font-medium">Premium feature</p>
            <p className="text-sm text-zinc-500">Stats are available to premium users.</p>
          </div>
          <div className="flex items-center gap-2 text-zinc-600">
            <BarChart2 size={14} />
            <span className="text-xs">Rating distribution · Monthly activity · Streaks</span>
          </div>
        </div>
      </div>
    )
  }

  if (isLoading) return <LoadingSpinner />
  if (!s) return null

  const totalItems = s.films.total + s.tv_shows.total + s.books.total + s.anime.total
  const ratingDist = s.rating_distribution ?? []
  const monthlyActivity = s.monthly_activity ?? []
  const maxRatingCount = Math.max(...ratingDist.map(b => b.count), 1)
  const maxMonthTotal = Math.max(...monthlyActivity.map(m => m.films + m.tv + m.books + m.anime), 1)

  const CATEGORY_SEGMENTS = [
    { key: 'anime' as const, colour: 'bg-pink-400',    label: 'Anime'    },
    { key: 'books' as const, colour: 'bg-emerald-400', label: 'Books'    },
    { key: 'tv'    as const, colour: 'bg-purple-400',  label: 'TV'       },
    { key: 'films' as const, colour: 'bg-blue-400',    label: 'Movies'   },
  ]

  const parseMonth = (m: string) => {
    const [y, mo] = m.split('-')
    return new Date(Number(y), Number(mo) - 1).toLocaleString('default', { month: 'short' })
  }
  const firstMonth = monthlyActivity.length > 0 ? parseMonth(monthlyActivity[0].month) : ''
  const lastMonth  = monthlyActivity.length > 0 ? parseMonth(monthlyActivity[monthlyActivity.length - 1].month) : ''

  const typeCards = [
    {
      label: 'Movies',
      total: s.films.total,
      sub: s.films.avg_rating ? `Avg ${s.films.avg_rating.toFixed(1)}/10` : 'No ratings yet',
      highlight: s.films.this_month > 0 ? `+${s.films.this_month} this month` : 'no activity',
      colour: 'text-blue-400',
      hex: '#60a5fa',
      sparkData: monthlyActivity.map(m => m.films),
    },
    {
      label: 'TV',
      total: s.tv_shows.total,
      sub: `${s.tv_shows.in_progress} in progress`,
      highlight: s.tv_shows.episodes_this_month > 0 ? `${s.tv_shows.episodes_this_month} eps this month` : 'no activity',
      colour: 'text-pink-400',
      hex: '#f472b6',
      sparkData: monthlyActivity.map(m => m.tv),
    },
    {
      label: 'Books',
      total: s.books.total,
      sub: `${s.books.in_progress} in progress`,
      highlight: s.books.chapters_this_month > 0 ? `${s.books.chapters_this_month} ch this month` : 'no activity',
      colour: 'text-emerald-400',
      hex: '#34d399',
      sparkData: monthlyActivity.map(m => m.books),
    },
    {
      label: 'Anime',
      total: s.anime.total,
      sub: `${s.anime.in_progress} in progress`,
      highlight: s.anime.episodes_this_month > 0 ? `${s.anime.episodes_this_month} eps this month` : 'no activity',
      colour: 'text-purple-400',
      hex: '#c084fc',
      sparkData: monthlyActivity.map(m => m.anime),
    },
  ]

  return (
    <div className="space-y-5 max-w-2xl">
      <h1 className="text-2xl font-semibold text-white tracking-tight">Stats</h1>

      {/* Top summary row */}
      <div className="grid grid-cols-3 gap-3">
        <div className="bg-[#1a1a1a] rounded-xl p-4 ring-1 ring-white/[0.06] col-span-1">
          <p className="text-3xl font-bold text-white tabular-nums">{totalItems}</p>
          <p className="text-xs text-zinc-500 mt-0.5">Total in library</p>
        </div>
        <div className="bg-[#1a1a1a] rounded-xl p-4 ring-1 ring-white/[0.06] col-span-1">
          <p className="text-3xl font-bold text-indigo-400 tabular-nums">{formatTime(s.estimated_minutes)}</p>
          <p className="text-xs text-zinc-500 mt-0.5">Time spent</p>
        </div>
        <div className="bg-[#1a1a1a] rounded-xl p-4 ring-1 ring-white/[0.06] col-span-1">
          <p className="text-3xl font-bold text-amber-400 tabular-nums">
            {s.completion_rate > 0 ? Math.round(s.completion_rate * 100) : 0}%
          </p>
          <p className="text-xs text-zinc-500 mt-0.5">Completion rate</p>
        </div>
      </div>

      {/* Streaks */}
      <div className="grid grid-cols-2 gap-3">
        <div className="bg-[#1a1a1a] rounded-xl p-4 ring-1 ring-white/[0.06]">
          <p className="text-xs text-zinc-500 font-medium uppercase tracking-wider mb-2">Current streak</p>
          <p className="text-3xl font-bold text-orange-400 tabular-nums">{s.current_streak_days}</p>
          <p className="text-xs text-zinc-600 mt-0.5">{s.current_streak_days === 1 ? 'day' : 'days'}</p>
        </div>
        <div className="bg-[#1a1a1a] rounded-xl p-4 ring-1 ring-white/[0.06]">
          <p className="text-xs text-zinc-500 font-medium uppercase tracking-wider mb-2">Longest streak</p>
          <p className="text-3xl font-bold text-orange-300 tabular-nums">{s.longest_streak_days}</p>
          <p className="text-xs text-zinc-600 mt-0.5">{s.longest_streak_days === 1 ? 'day' : 'days'}</p>
        </div>
      </div>

      {/* Per-type cards with sparklines */}
      <div className="grid grid-cols-2 gap-3">
        {typeCards.map(({ label, total, sub, highlight, colour, hex, sparkData }) => (
          <div key={label} className="bg-[#1a1a1a] rounded-xl ring-1 ring-white/[0.06] overflow-hidden">
            <div className="px-4 pt-4 pb-2">
              <div className="flex items-start justify-between gap-2">
                <p className="text-xs text-zinc-500 font-medium uppercase tracking-wider mt-1">{label}</p>
                <p className={`text-3xl font-bold tabular-nums leading-none ${colour}`}>{total}</p>
              </div>
              <div className="flex items-center justify-between mt-1.5 gap-1">
                <p className="text-xs text-zinc-500 truncate">{sub}</p>
                <p className="text-xs text-emerald-400 whitespace-nowrap">{highlight}</p>
              </div>
            </div>
            <Sparkline data={sparkData} colour={hex} id={label} />
            {firstMonth && (
              <div className="flex justify-between px-4 pb-3 -mt-1">
                <span className="text-[10px] text-zinc-600">{firstMonth}</span>
                <span className="text-[10px] text-zinc-600">{lastMonth}</span>
              </div>
            )}
          </div>
        ))}
      </div>

      {/* Rating distribution */}
      {ratingDist.length > 0 && (
        <div className="bg-[#1a1a1a] rounded-xl p-4 ring-1 ring-white/[0.06] space-y-3">
          <p className="text-xs text-zinc-500 font-medium uppercase tracking-wider">Rating distribution</p>
          <div className="flex items-end gap-1.5 h-20">
            {Array.from({ length: 10 }, (_, i) => i + 1).map((r) => {
              const bucket = ratingDist.find(b => b.rating === r)
              const count = bucket?.count ?? 0
              const height = count > 0 ? Math.max((count / maxRatingCount) * 100, 8) : 0
              return (
                <div key={r} className="flex-1 flex flex-col items-center gap-1">
                  {count > 0 && (
                    <span className="text-[9px] text-zinc-600 tabular-nums">{count}</span>
                  )}
                  <div className="w-full flex items-end" style={{ height: '64px' }}>
                    <div
                      className="w-full rounded-sm bg-indigo-500/60 transition-all"
                      style={{ height: `${height}%` }}
                    />
                  </div>
                  <span className="text-[9px] text-zinc-600">{r}</span>
                </div>
              )
            })}
          </div>
        </div>
      )}

      {/* Monthly activity — stacked bar chart */}
      <div className="bg-[#1a1a1a] rounded-xl p-4 ring-1 ring-white/[0.06] space-y-3">
        <div className="flex items-center justify-between">
          <p className="text-xs text-zinc-500 font-medium uppercase tracking-wider">Monthly activity</p>
          <div className="flex items-center gap-3">
            {CATEGORY_SEGMENTS.slice().reverse().map(({ key, colour, label }) => (
              <span key={key} className="flex items-center gap-1">
                <span className={`w-2 h-2 rounded-full ${colour}`} />
                <span className="text-[10px] text-zinc-500">{label}</span>
              </span>
            ))}
          </div>
        </div>
        {monthlyActivity.length === 0 ? (
          <p className="text-xs text-zinc-600 py-4 text-center">No activity data yet</p>
        ) : (
          <div className="flex items-end gap-1 h-20">
            {monthlyActivity.map((m) => {
              const total = m.films + m.tv + m.books + m.anime
              const barPct = total > 0 ? Math.max((total / maxMonthTotal) * 100, 6) : 0
              const [y, mo] = m.month.split('-')
              const label = new Date(Number(y), Number(mo) - 1).toLocaleString('default', { month: 'short' })
              return (
                <div key={m.month} className="flex-1 flex flex-col items-center gap-1">
                  {total > 0 && (
                    <span className="text-[9px] text-zinc-600 tabular-nums">{total}</span>
                  )}
                  <div className="w-full flex flex-col-reverse overflow-hidden rounded-sm" style={{ height: '64px' }}>
                    {CATEGORY_SEGMENTS.map(({ key, colour }) => {
                      const segPct = total > 0 ? (m[key] / total) * barPct : 0
                      return segPct > 0 ? (
                        <div
                          key={key}
                          className={`w-full ${colour} opacity-70 transition-all`}
                          style={{ height: `${segPct}%` }}
                        />
                      ) : null
                    })}
                  </div>
                  <span className="text-[9px] text-zinc-600">{label}</span>
                </div>
              )
            })}
          </div>
        )}
      </div>

      {totalItems === 0 && (
        <p className="text-zinc-600 text-sm text-center py-12">Nothing in your library yet — start adding some!</p>
      )}
    </div>
  )
}
