import { useStats } from '@/hooks/useStats'
import LoadingSpinner from '@/components/ui/LoadingSpinner'

export default function Stats() {
  const { data: s, isLoading } = useStats()

  if (isLoading) return <LoadingSpinner />
  if (!s) return null

  const cards = [
    {
      label: 'Movies',
      total: s.films.total,
      detail: s.films.this_month > 0 ? `${s.films.this_month} this month` : 'None this month',
      sub: s.films.avg_rating ? `Avg rating ${s.films.avg_rating.toFixed(1)}` : null,
      colour: 'text-blue-400',
    },
    {
      label: 'TV Shows',
      total: s.tv_shows.total,
      detail: `${s.tv_shows.in_progress} in progress`,
      sub: s.tv_shows.episodes_this_month > 0 ? `${s.tv_shows.episodes_this_month} eps this month` : null,
      colour: 'text-purple-400',
    },
    {
      label: 'Books',
      total: s.books.total,
      detail: `${s.books.in_progress} in progress`,
      sub: s.books.chapters_this_month > 0 ? `${s.books.chapters_this_month} chapters this month` : null,
      colour: 'text-emerald-400',
    },
    {
      label: 'Anime',
      total: s.anime.total,
      detail: `${s.anime.in_progress} in progress`,
      sub: s.anime.episodes_this_month > 0 ? `${s.anime.episodes_this_month} eps this month` : null,
      colour: 'text-pink-400',
    },
  ]

  const totalItems = s.films.total + s.tv_shows.total + s.books.total + s.anime.total

  return (
    <div className="space-y-6 max-w-2xl">
      <h1 className="text-2xl font-semibold text-white tracking-tight">Stats</h1>

      {/* Summary row */}
      <div className="bg-[#1a1a1a] rounded-xl p-4 ring-1 ring-white/[0.06] flex items-center justify-between">
        <div>
          <p className="text-3xl font-bold text-white tabular-nums">{totalItems}</p>
          <p className="text-xs text-zinc-500 mt-0.5">Total in library</p>
        </div>
        {s.films.avg_rating && (
          <div className="text-right">
            <p className="text-3xl font-bold text-yellow-400 tabular-nums">{s.films.avg_rating.toFixed(1)}</p>
            <p className="text-xs text-zinc-500 mt-0.5">Avg movie rating</p>
          </div>
        )}
      </div>

      {/* Per-type cards */}
      <div className="grid grid-cols-2 gap-3">
        {cards.map(({ label, total, detail, sub, colour }) => (
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

      {totalItems === 0 && (
        <p className="text-zinc-600 text-sm text-center py-12">
          Nothing in your library yet — start adding some!
        </p>
      )}
    </div>
  )
}
