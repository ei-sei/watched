import { useState, useEffect } from 'react'
import { useMediaList } from '@/hooks/useMedia'
import MediaGrid from '@/components/media/MediaGrid'
import LoadingSpinner from '@/components/ui/LoadingSpinner'
import EmptyState from '@/components/ui/EmptyState'
import type { MediaType, MediaStatus } from '@/types/media'
import { MEDIA_TYPE_LABELS } from '@/utils/constants'

interface Props { type: MediaType }

const STATUSES: { value: MediaStatus | ''; label: string }[] = [
  { value: '', label: 'All' },
  { value: 'want_to', label: 'Want to' },
  { value: 'in_progress', label: 'In Progress' },
  { value: 'completed', label: 'Completed' },
  { value: 'dropped', label: 'Dropped' },
  { value: 'on_hold', label: 'On Hold' },
]

export default function MediaLibrary({ type }: Props) {
  const [status, setStatus] = useState<MediaStatus | ''>('')
  const [sort, setSort] = useState('created_at')
  const [page, setPage] = useState(1)
  const [search, setSearch] = useState('')
  const [debouncedSearch, setDebouncedSearch] = useState('')

  useEffect(() => {
    const t = setTimeout(() => { setDebouncedSearch(search); setPage(1) }, 350)
    return () => clearTimeout(t)
  }, [search])

  const { data, isLoading } = useMediaList({
    media_type: type,
    status: status || undefined,
    sort,
    order: 'desc',
    page,
    per_page: 50,
    q: debouncedSearch || undefined,
  })

  return (
    <div className="space-y-6">
      {/* Page header */}
      <div className="flex items-baseline justify-between">
        <h1 className="text-2xl font-semibold text-white tracking-tight">
          {MEDIA_TYPE_LABELS[type]}
        </h1>
        {data && (
          <span className="text-sm text-zinc-500">{data.total} titles</span>
        )}
      </div>

      {/* Search input */}
      <input
        value={search}
        onChange={(e) => setSearch(e.target.value)}
        placeholder={`Search ${MEDIA_TYPE_LABELS[type].toLowerCase()}…`}
        className="w-full bg-[#1a1a1a] text-zinc-200 rounded-lg px-4 py-2.5 text-sm border border-white/[0.08] focus:outline-none focus:border-white/20 placeholder:text-zinc-600"
      />

      {/* Filters */}
      <div className="flex items-center gap-1.5 flex-wrap">
        {STATUSES.map(({ value, label }) => (
          <button
            key={value}
            onClick={() => { setStatus(value as MediaStatus | ''); setPage(1) }}
            className={`px-3 py-1 rounded-md text-xs font-medium transition-colors ${
              status === value
                ? 'bg-white/10 text-white'
                : 'text-zinc-500 hover:bg-white/5 hover:text-zinc-300'
            }`}
          >
            {label}
          </button>
        ))}
        <select
          value={sort}
          onChange={(e) => setSort(e.target.value)}
          className="ml-auto bg-transparent text-zinc-500 text-xs px-2 py-1 rounded-md hover:bg-white/5 transition-colors outline-none cursor-pointer"
        >
          <option value="created_at">Date added</option>
          <option value="rating">Rating</option>
          <option value="title">Title</option>
          <option value="year">Year</option>
        </select>
      </div>

      {/* Content */}
      {isLoading && <LoadingSpinner />}
      {!isLoading && data?.items.length === 0 && (
        <EmptyState message={search ? `No results for "${search}"` : `No ${MEDIA_TYPE_LABELS[type].toLowerCase()} yet`} />
      )}
      {!isLoading && data && data.items.length > 0 && (
        <>
          <MediaGrid items={data.items} />
          {data.pages > 1 && (
            <div className="flex justify-center gap-1.5 pt-4">
              {Array.from({ length: data.pages }, (_, i) => i + 1).map((p) => (
                <button
                  key={p}
                  onClick={() => setPage(p)}
                  className={`w-8 h-8 rounded-md text-xs font-medium transition-colors ${
                    p === page
                      ? 'bg-white/10 text-white'
                      : 'text-zinc-500 hover:bg-white/5 hover:text-zinc-300'
                  }`}
                >
                  {p}
                </button>
              ))}
            </div>
          )}
        </>
      )}
    </div>
  )
}
