import { useStats } from '@/hooks/useStats'
import LoadingSpinner from '@/components/ui/LoadingSpinner'
import { pluralise } from '@/utils/formatters'

export default function Stats() {
  const { data: stats, isLoading } = useStats()

  if (isLoading) return <LoadingSpinner />
  if (!stats) return null

  return (
    <div className="space-y-8 max-w-2xl mx-auto">
      <h1 className="text-2xl font-bold text-white">Stats</h1>

      <div className="grid grid-cols-3 gap-4">
        {[
          { label: 'Films', value: stats.films.total, sub: `${stats.films.this_month} this month` },
          { label: 'TV Shows', value: stats.tv_shows.total, sub: `${stats.tv_shows.in_progress} in progress` },
          { label: 'Books', value: stats.books.total, sub: `${stats.books.in_progress} in progress` },
        ].map(({ label, value, sub }) => (
          <div key={label} className="bg-slate-800 rounded-lg p-4 text-center">
            <p className="text-3xl font-bold text-white">{value}</p>
            <p className="text-sm text-slate-400 mt-1">{label}</p>
            <p className="text-xs text-slate-500 mt-0.5">{sub}</p>
          </div>
        ))}
      </div>

      <div className="bg-slate-800 rounded-lg p-4">
        <p className="text-slate-400 text-sm mb-1">Current streak</p>
        <p className="text-3xl font-bold text-indigo-400">{pluralise(stats.current_streak_days, 'day')}</p>
      </div>

      <div className="grid grid-cols-2 gap-4">
        <div className="bg-slate-800 rounded-lg p-4">
          <p className="text-slate-400 text-sm mb-2">This month — Films</p>
          <p className="text-2xl font-bold text-white">{stats.films.this_month}</p>
          {stats.films.avg_rating && (
            <p className="text-sm text-yellow-400 mt-1">avg ★ {stats.films.avg_rating}</p>
          )}
        </div>
        <div className="bg-slate-800 rounded-lg p-4">
          <p className="text-slate-400 text-sm mb-2">This month — Reading</p>
          <p className="text-2xl font-bold text-white">{stats.books.chapters_this_month} chapters</p>
          <p className="text-sm text-slate-400 mt-1">{stats.books.pages_this_month} pages</p>
        </div>
      </div>
    </div>
  )
}
