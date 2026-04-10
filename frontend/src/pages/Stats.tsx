import { useStats } from '@/hooks/useStats'
import LoadingSpinner from '@/components/ui/LoadingSpinner'

function formatTime(minutes: number): string {
  if (minutes < 60) return `${minutes}m`
  const h = Math.floor(minutes / 60)
  if (h < 24) return `${h}h`
  const d = Math.floor(h / 24)
  const rem = h % 24
  return rem > 0 ? `${d}d ${rem}h` : `${d}d`
}

export default function Stats() {
  const { data: s, isLoading } = useStats()

  if (isLoading) return <LoadingSpinner />
  if (!s) return null

  const totalItems = s.films.total + s.tv_shows.total + s.books.total + s.anime.total
  const ratingDist = s.rating_distribution ?? []
  const monthlyActivity = s.monthly_activity ?? []
  const maxRatingCount = Math.max(...ratingDist.map(b => b.count), 1)
  const maxMonthCount = Math.max(...monthlyActivity.map(m => m.count), 1)

  const typeCards = [
    { label: 'Movies',   total: s.films.total,    detail: `${s.films.this_month} this month`,          sub: s.films.avg_rating ? `Avg ${s.films.avg_rating.toFixed(1)} / 10` : null, colour: 'text-blue-400' },
    { label: 'TV Shows', total: s.tv_shows.total,  detail: `${s.tv_shows.in_progress} in progress`,    sub: s.tv_shows.episodes_this_month > 0 ? `${s.tv_shows.episodes_this_month} eps this month` : null, colour: 'text-purple-400' },
    { label: 'Books',    total: s.books.total,     detail: `${s.books.in_progress} in progress`,       sub: s.books.chapters_this_month > 0 ? `${s.books.chapters_this_month} ch this month` : null, colour: 'text-emerald-400' },
    { label: 'Anime',   total: s.anime.total,     detail: `${s.anime.in_progress} in progress`,       sub: s.anime.episodes_this_month > 0 ? `${s.anime.episodes_this_month} eps this month` : null, colour: 'text-pink-400' },
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

      {/* Per-type cards */}
      <div className="grid grid-cols-2 gap-3">
        {typeCards.map(({ label, total, detail, sub, colour }) => (
          <div key={label} className="bg-[#1a1a1a] rounded-xl p-4 ring-1 ring-white/[0.06] space-y-2">
            <p className="text-xs text-zinc-500 font-medium uppercase tracking-wider">{label}</p>
            <p className={`text-3xl font-bold tabular-nums ${colour}`}>{total}</p>
            <div className="space-y-0.5">
              <p className="text-xs text-zinc-500">{detail}</p>
              {sub && <p className="text-xs text-zinc-600">{sub}</p>}
            </div>
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

      {/* Monthly activity */}
      <div className="bg-[#1a1a1a] rounded-xl p-4 ring-1 ring-white/[0.06] space-y-3">
        <p className="text-xs text-zinc-500 font-medium uppercase tracking-wider">Monthly activity</p>
        {monthlyActivity.length === 0 ? (
          <p className="text-xs text-zinc-600 py-4 text-center">No activity data yet</p>
        ) : (
          <div className="flex items-end gap-1 h-20">
            {monthlyActivity.map(({ month, count }) => {
              const height = count > 0 ? Math.max((count / maxMonthCount) * 100, 8) : 0
              const [y, m] = month.split('-')
              const label = new Date(Number(y), Number(m) - 1).toLocaleString('default', { month: 'short' })
              return (
                <div key={month} className="flex-1 flex flex-col items-center gap-1">
                  {count > 0 && (
                    <span className="text-[9px] text-zinc-600 tabular-nums">{count}</span>
                  )}
                  <div className="w-full flex items-end" style={{ height: '64px' }}>
                    <div
                      className="w-full rounded-sm bg-indigo-500/40 transition-all"
                      style={{ height: `${height}%` }}
                    />
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
