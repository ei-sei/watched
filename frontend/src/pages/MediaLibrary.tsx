import { useState, useEffect, useRef } from 'react'
import { useSearchParams } from 'react-router-dom'
import { ChevronDown } from 'lucide-react'
import { useLocalMediaList } from '@/hooks/useLocalMedia'
import MediaGrid from '@/components/media/MediaGrid'
import EmptyState from '@/components/ui/EmptyState'
import { syncMediaIfStale } from '@/offline/sync'
import type { MediaType, MediaStatus } from '@/types/media'
import { MEDIA_TYPE_LABELS, STATUS_LABELS, STATUS_LABELS_BOOK } from '@/utils/constants'

interface Props { type: MediaType }

const STATUS_OPTIONS: (MediaStatus | '')[] = ['', 'in_progress', 'want_to', 'completed', 'on_hold', 'dropped']

export default function MediaLibrary({ type }: Props) {
  const [searchParams, setSearchParams] = useSearchParams()
  const validStatuses: (MediaStatus | '')[] = ['', 'in_progress', 'want_to', 'completed', 'on_hold', 'dropped']
  const paramStatus = searchParams.get('status') ?? 'in_progress'
  const status: MediaStatus | '' = validStatuses.includes(paramStatus as MediaStatus | '') ? paramStatus as MediaStatus | '' : 'in_progress'

  const setStatus = (value: MediaStatus | '') => {
    setSearchParams(value ? { status: value } : {}, { replace: true })
  }

  const [statusOpen, setStatusOpen] = useState(false)
  const [sort, setSort] = useState('created_at')
  const statusRef = useRef<HTMLDivElement>(null)
  const [page, setPage] = useState(1)
  const [search, setSearch] = useState('')
  const [debouncedSearch, setDebouncedSearch] = useState('')

  // Sync from server on each category visit (no-op if synced recently)
  useEffect(() => { syncMediaIfStale() }, [type])

  // Close status popover on outside click
  useEffect(() => {
    const handler = (e: MouseEvent) => {
      if (statusRef.current && !statusRef.current.contains(e.target as Node)) {
        setStatusOpen(false)
      }
    }
    document.addEventListener('mousedown', handler)
    return () => document.removeEventListener('mousedown', handler)
  }, [])

  useEffect(() => {
    const t = setTimeout(() => { setDebouncedSearch(search); setPage(1) }, 350)
    return () => clearTimeout(t)
  }, [search])

  const data = useLocalMediaList({
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
      <div className="flex items-center gap-2">
        {/* Mobile: compact dropdown toggle */}
        <div ref={statusRef} className="relative md:hidden">
          <button
            onClick={() => setStatusOpen((v) => !v)}
            className="flex items-center gap-1.5 bg-[#1a1a1a] border border-white/[0.08] rounded-lg px-3 py-1.5 text-xs font-medium text-zinc-200 transition-colors hover:border-white/20"
          >
            {status === '' ? 'All' : (type === 'book' ? STATUS_LABELS_BOOK : STATUS_LABELS)[status]}
            <ChevronDown size={12} className={`text-zinc-500 transition-transform ${statusOpen ? 'rotate-180' : ''}`} />
          </button>
          {statusOpen && (
            <div className="absolute top-full left-0 mt-1 bg-[#1a1a1a] border border-white/[0.08] rounded-lg overflow-hidden z-10 min-w-[140px] shadow-lg">
              {STATUS_OPTIONS.map((value) => {
                const labels = type === 'book' ? STATUS_LABELS_BOOK : STATUS_LABELS
                const label = value === '' ? 'All' : labels[value]
                return (
                  <button
                    key={value}
                    onClick={() => { setStatus(value); setPage(1); setStatusOpen(false) }}
                    className={`w-full text-left px-3 py-2 text-xs transition-colors ${
                      status === value
                        ? 'text-white bg-white/10'
                        : 'text-zinc-400 hover:bg-white/[0.04] hover:text-zinc-200'
                    }`}
                  >
                    {label}
                  </button>
                )
              })}
            </div>
          )}
        </div>

        {/* Desktop: pill row (option A) */}
        <div className="hidden md:flex items-center gap-1.5 flex-wrap">
          {STATUS_OPTIONS.map((value) => {
            const labels = type === 'book' ? STATUS_LABELS_BOOK : STATUS_LABELS
            const label = value === '' ? 'All' : labels[value]
            return (
              <button
                key={value}
                onClick={() => { setStatus(value); setPage(1) }}
                className={`px-3 py-1 rounded-md text-xs font-medium transition-colors ${
                  status === value
                    ? 'bg-white/10 text-white'
                    : 'text-zinc-500 hover:bg-white/5 hover:text-zinc-300'
                }`}
              >
                {label}
              </button>
            )
          })}
        </div>

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

      {/* Content — data is undefined only on the very first Dexie tick, then resolves instantly */}
      {data && data.items.length === 0 && (
        <EmptyState message={search ? `No results for "${search}"` : `No ${MEDIA_TYPE_LABELS[type].toLowerCase()} yet`} />
      )}
      {data && data.items.length > 0 && (
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
