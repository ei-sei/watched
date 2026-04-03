import { useState, useRef, useEffect, useMemo } from 'react'
import { useNavigate } from 'react-router-dom'
import { useLiveQuery } from 'dexie-react-hooks'
import { useSearch } from '@/hooks/useSearch'
import { useCreateMedia } from '@/hooks/useMedia'
import { useToast } from '@/components/ui/Toast'
import LoadingSpinner from '@/components/ui/LoadingSpinner'
import { Clock, X, Film, Tv, BookOpen } from 'lucide-react'
import { db } from '@/offline/db'
import type { MediaType, MediaItem, SearchResult } from '@/types/media'

const GROUP_ORDER: SearchResult['media_type'][] = ['film', 'tv_show', 'anime', 'book']
const GROUP_LABELS: Record<string, string> = {
  film:    'Movies',
  tv_show: 'TV Shows',
  anime:   'Anime',
  book:    'Books',
}
const TYPE_LABELS: Record<string, string> = {
  film:    'Movie',
  tv_show: 'TV Series',
  book:    'Book',
  anime:   'Anime',
}
const TYPE_PREFIX: Record<string, string> = {
  film: 'films', tv_show: 'tv', book: 'books', anime: 'anime',
}
const FALLBACK_ICONS: Record<string, React.ReactNode> = {
  film:    <Film size={18} className="text-zinc-700" />,
  tv_show: <Tv size={18} className="text-zinc-700" />,
  anime:   <Tv size={18} className="text-zinc-700" />,
  book:    <BookOpen size={18} className="text-zinc-700" />,
}

function friendlyMeta(result: SearchResult): string {
  const parts: string[] = []
  parts.push(TYPE_LABELS[result.media_type] ?? result.media_type)
  if (result.year) parts.push(String(result.year))
  if (result.media_type === 'book') {
    const authors = result.extra?.authors
    if (Array.isArray(authors) && authors.length > 0) {
      parts.push(authors.slice(0, 2).join(', '))
    }
    const publisher = result.extra?.publisher
    if (typeof publisher === 'string' && publisher) parts.push(publisher)
    const pages = result.extra?.page_count
    if (typeof pages === 'number' && pages > 0) parts.push(`${pages} pages`)
  }
  if (result.media_type === 'anime' || result.media_type === 'tv_show') {
    const episodes = result.extra?.episodes
    if (typeof episodes === 'number' && episodes > 0) {
      parts.push(`${episodes} eps`)
    }
  }
  return parts.join(' · ')
}

const RECENTS_KEY = 'search_recents'
const MAX_RECENTS = 8

function getRecents(): string[] {
  try { return JSON.parse(localStorage.getItem(RECENTS_KEY) ?? '[]') } catch { return [] }
}
function saveRecent(q: string) {
  const trimmed = q.trim()
  if (!trimmed) return
  const prev = getRecents().filter((r) => r !== trimmed)
  localStorage.setItem(RECENTS_KEY, JSON.stringify([trimmed, ...prev].slice(0, MAX_RECENTS)))
}
function removeRecent(q: string) {
  localStorage.setItem(RECENTS_KEY, JSON.stringify(getRecents().filter((r) => r !== q)))
}

const TABS = [
  { value: 'multi',   label: 'All' },
  { value: 'film',    label: 'Movies' },
  { value: 'tv_show', label: 'TV' },
  { value: 'book',    label: 'Books' },
  { value: 'anime',   label: 'Anime' },
] as const

type Tab = typeof TABS[number]['value']

function ResultCard({
  result,
  resultIndex,
  focusedIndex,
  addingId,
  existing,
  onAdd,
  onNavigate,
  onSelect,
}: {
  result: SearchResult
  resultIndex: number
  focusedIndex: number | null
  addingId: string | null
  existing: MediaItem | undefined
  onAdd: (r: SearchResult) => void
  onNavigate: (item: MediaItem) => void
  onSelect: (r: SearchResult) => void
}) {
  const isFocused = focusedIndex === resultIndex
  const isAdding = addingId === result.external_id
  const divRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    if (isFocused) divRef.current?.scrollIntoView({ block: 'nearest', behavior: 'smooth' })
  }, [isFocused])

  return (
    <div
      ref={divRef}
      className={`flex gap-3 rounded-lg p-3 items-start ring-1 transition-colors ${
        isFocused
          ? 'bg-[#1f1f2e] ring-indigo-500/50'
          : 'bg-[#1a1a1a] ring-white/[0.06]'
      }`}
    >
      <button onClick={() => onSelect(result)} className="flex-shrink-0">
        {result.poster_url ? (
          <img src={result.poster_url} alt="" className="w-12 h-16 object-cover rounded" />
        ) : (
          <div className="w-12 h-16 bg-[#222] rounded flex items-center justify-center">
            {FALLBACK_ICONS[result.media_type]}
          </div>
        )}
      </button>
      <button className="flex-1 min-w-0 text-left" onClick={() => onSelect(result)}>
        <p className="font-medium text-zinc-100 text-sm">{result.title}</p>
        <p className="text-xs text-zinc-500 mt-0.5">{friendlyMeta(result)}</p>
        {result.description && (
          <p className="text-xs text-zinc-600 mt-1.5 line-clamp-2 leading-relaxed">{result.description}</p>
        )}
      </button>
      {existing ? (
        <button
          onClick={() => onNavigate(existing)}
          className="flex-shrink-0 bg-indigo-500/10 hover:bg-indigo-500/20 text-indigo-400 text-xs px-3 py-1.5 rounded-md transition-colors whitespace-nowrap"
        >
          In Library →
        </button>
      ) : (
        <button
          onClick={() => onAdd(result)}
          disabled={isAdding}
          className="flex-shrink-0 bg-white/[0.08] hover:bg-white/[0.12] disabled:opacity-50 text-zinc-200 text-xs px-3 py-1.5 rounded-md transition-colors"
        >
          {isAdding ? '…' : 'Add'}
        </button>
      )}
    </div>
  )
}

function SearchDetailSheet({ result, existing, onClose, onAdd, onNavigate, addingId }: {
  result: SearchResult
  existing: MediaItem | undefined
  onClose: () => void
  onAdd: (r: SearchResult) => void
  onNavigate: (item: MediaItem) => void
  addingId: string | null
}) {
  const isAdding = addingId === result.external_id
  return (
    <div className="fixed inset-0 bg-black/60 flex items-end sm:items-center justify-center z-50 px-0 sm:px-4" onClick={onClose}>
      <div
        className="bg-[#1a1a1a] rounded-t-2xl sm:rounded-xl w-full sm:max-w-sm ring-1 ring-white/[0.08] overflow-hidden flex flex-col min-h-[42vh] sm:min-h-0"
        onClick={e => e.stopPropagation()}
      >
        <div className="flex items-start gap-3 p-4 border-b border-white/[0.06] flex-shrink-0">
          {result.poster_url ? (
            <img src={result.poster_url} alt={result.title} className="w-14 h-[84px] rounded object-cover flex-shrink-0" />
          ) : (
            <div className="w-14 h-[84px] rounded bg-white/[0.04] flex-shrink-0 flex items-center justify-center">
              {FALLBACK_ICONS[result.media_type]}
            </div>
          )}
          <div className="flex-1 min-w-0 pt-0.5">
            <p className="text-white font-semibold text-sm leading-snug">{result.title}</p>
            <p className="text-xs text-zinc-500 mt-0.5">{friendlyMeta(result)}</p>
          </div>
          <button onClick={onClose} className="text-zinc-600 hover:text-zinc-400 transition-colors flex-shrink-0 mt-0.5">
            <X size={18} />
          </button>
        </div>

        <div className="p-4 flex flex-col flex-1 gap-4">
          {result.description && (
            <p className="text-xs text-zinc-400 leading-relaxed flex-1 overflow-y-auto">{result.description}</p>
          )}
          <div className="flex-shrink-0 pb-2">
            {existing ? (
              <button
                onClick={() => { onClose(); onNavigate(existing) }}
                className="w-full py-2.5 bg-white/[0.06] hover:bg-white/[0.10] text-zinc-300 text-sm font-medium rounded-lg transition-colors"
              >
                Already in your library →
              </button>
            ) : (
              <button
                onClick={() => onAdd(result)}
                disabled={isAdding}
                className="w-full py-2.5 bg-indigo-600 hover:bg-indigo-700 disabled:opacity-50 text-white text-sm font-medium rounded-lg transition-colors"
              >
                {isAdding ? 'Adding…' : 'Add to library'}
              </button>
            )}
          </div>
        </div>
      </div>
    </div>
  )
}

export default function Search() {
  const [query, setQueryState] = useState<string>(() => sessionStorage.getItem('search_q') ?? '')
  const [tab, setTabState] = useState<Tab>(() => (sessionStorage.getItem('search_tab') ?? 'multi') as Tab)

  const setQuery = (q: string) => { setQueryState(q); sessionStorage.setItem('search_q', q) }
  const setTab = (t: Tab) => { setTabState(t); sessionStorage.setItem('search_tab', t) }

  const { data, suggestion, isFetching } = useSearch(query, tab)
  const create = useCreateMedia()
  const { show } = useToast()
  const navigate = useNavigate()
  const [focused, setFocused] = useState(false)
  const [recents, setRecents] = useState<string[]>(getRecents)
  const [focusedIndex, setFocusedIndex] = useState<number | null>(null)
  const [addingId, setAddingId] = useState<string | null>(null)
  const [selected, setSelected] = useState<SearchResult | null>(null)
  const inputRef = useRef<HTMLInputElement>(null)
  const dropdownRef = useRef<HTMLDivElement>(null)

  const showRecents = focused && query.length === 0 && recents.length > 0

  // Library lookup — build a Map of external_id → MediaItem for O(1) checks
  const libraryMap = useLiveQuery(
    () => db.mediaItems
      .filter((m) => m.external_id !== null && m.external_id !== undefined)
      .toArray()
      .then((items) => new Map(items.filter((m) => m.external_id).map((m) => [m.external_id!, m]))),
    []
  ) ?? new Map<string, MediaItem>()

  // Flat ordered result list matching visual render order (for keyboard nav)
  const flatResults = useMemo(() => {
    if (!data) return []
    if (tab !== 'multi') return data
    return GROUP_ORDER.flatMap((type) => data.filter((r) => r.media_type === type))
  }, [data, tab])

  // Reset keyboard focus when results change
  useEffect(() => { setFocusedIndex(null) }, [flatResults])

  useEffect(() => {
    const handler = (e: MouseEvent) => {
      if (
        inputRef.current && !inputRef.current.contains(e.target as Node) &&
        dropdownRef.current && !dropdownRef.current.contains(e.target as Node)
      ) {
        setFocused(false)
      }
    }
    document.addEventListener('mousedown', handler)
    return () => document.removeEventListener('mousedown', handler)
  }, [])

  const applyRecent = (q: string) => {
    setQuery(q)
    setFocused(false)
    inputRef.current?.blur()
  }

  const deleteRecent = (q: string, e: React.MouseEvent) => {
    e.stopPropagation()
    removeRecent(q)
    setRecents(getRecents())
  }

  const handleAdd = async (result: SearchResult) => {
    saveRecent(query.trim())
    setAddingId(result.external_id)
    try {
      const totalProgress =
        (typeof result.extra?.episodes === 'number' && result.extra.episodes > 0 ? result.extra.episodes : undefined) ??
        (typeof result.extra?.page_count === 'number' && result.extra.page_count > 0 ? result.extra.page_count : undefined)
      await create.mutateAsync({
        media_type: result.media_type as MediaType,
        external_id: result.external_id,
        title: result.title,
        year: result.year ?? undefined,
        poster_url: result.poster_url ?? undefined,
        metadata: result.extra,
        status: 'want_to',
        total_progress: totalProgress,
      })
      show(`"${result.title}" added to library`, 'success')
      setSelected(null)
    } catch (err: unknown) {
      const status = (err as { response?: { status?: number } })?.response?.status
      if (status === 409) {
        show('Already in library', 'error')
      } else if (status === 422) {
        show('Invalid data', 'error')
      } else {
        show('Server error — is the API running?', 'error')
      }
    } finally {
      setAddingId(null)
    }
  }

  const handleNavigate = (item: MediaItem) => {
    navigate(`/${TYPE_PREFIX[item.media_type] ?? 'films'}/${item.id}`)
  }

  const handleKeyDown = (e: React.KeyboardEvent<HTMLInputElement>) => {
    if (e.key === 'ArrowDown' && flatResults.length > 0) {
      e.preventDefault()
      setFocusedIndex((prev) => (prev === null ? 0 : Math.min(prev + 1, flatResults.length - 1)))
      return
    }
    if (e.key === 'ArrowUp' && flatResults.length > 0) {
      e.preventDefault()
      setFocusedIndex((prev) => (prev === null || prev === 0 ? null : prev - 1))
      return
    }
    if (e.key === 'Escape') {
      setFocusedIndex(null)
      setFocused(false)
      inputRef.current?.blur()
      return
    }
    if (e.key === 'Enter') {
      if (focusedIndex !== null && flatResults[focusedIndex]) {
        const result = flatResults[focusedIndex]
        const existing = libraryMap.get(result.external_id)
        if (existing) handleNavigate(existing)
        else handleAdd(result)
        return
      }
      if (query.trim()) {
        saveRecent(query.trim())
        setRecents(getRecents())
        setFocused(false)
      }
    }
  }

  const renderResults = () => {
    if (!data || data.length === 0) {
      if (query.length >= 2 && !isFetching) {
        return <p className="text-zinc-600 text-sm text-center py-12">No results for "{query}"</p>
      }
      return null
    }

    let idx = 0
    const card = (result: SearchResult) => {
      const i = idx++
      return (
        <ResultCard
          key={result.external_id}
          result={result}
          resultIndex={i}
          focusedIndex={focusedIndex}
          addingId={addingId}
          existing={libraryMap.get(result.external_id)}
          onAdd={handleAdd}
          onNavigate={handleNavigate}
          onSelect={setSelected}
        />
      )
    }

    if (tab !== 'multi') {
      return <div className="space-y-2">{data.map(card)}</div>
    }

    const grouped = GROUP_ORDER.reduce<Record<string, SearchResult[]>>((acc, type) => {
      const items = data.filter((r) => r.media_type === type)
      if (items.length > 0) acc[type] = items
      return acc
    }, {})

    return (
      <>
        {Object.entries(grouped).map(([type, items]) => (
          <div key={type} className="space-y-2">
            <p className="text-xs font-medium text-zinc-500 uppercase tracking-wider">{GROUP_LABELS[type]}</p>
            {items.map(card)}
          </div>
        ))}
      </>
    )
  }

  return (
    <div className="space-y-6 max-w-2xl">
      <h1 className="text-2xl font-semibold text-white tracking-tight">Search</h1>

      <div className="relative">
        <input
          ref={inputRef}
          value={query}
          onChange={(e) => { setQuery(e.target.value); setFocusedIndex(null) }}
          onFocus={() => setFocused(true)}
          onKeyDown={handleKeyDown}
          placeholder="Search movies, TV shows, books, anime…"
          className="w-full bg-[#1a1a1a] text-zinc-200 rounded-lg px-4 py-3 border border-white/[0.08] focus:outline-none focus:border-white/20 text-sm placeholder:text-zinc-600 pr-9"
        />
        {query.length > 0 && (
          <button
            onMouseDown={(e) => e.preventDefault()} // keep input focused
            onClick={() => { setQuery(''); setFocusedIndex(null); inputRef.current?.focus() }}
            className="absolute right-3 top-1/2 -translate-y-1/2 text-zinc-600 hover:text-zinc-400 transition-colors"
          >
            <X size={14} />
          </button>
        )}
        {showRecents && (
          <div ref={dropdownRef} className="absolute top-full left-0 right-0 mt-1 bg-[#1a1a1a] border border-white/[0.08] rounded-lg overflow-hidden z-20 shadow-lg">
            <p className="text-[10px] text-zinc-600 uppercase tracking-wider px-3 pt-2 pb-1">Recent</p>
            {recents.map((r) => (
              <button
                key={r}
                onClick={() => applyRecent(r)}
                className="w-full flex items-center gap-2.5 px-3 py-2 text-sm text-zinc-300 hover:bg-white/[0.04] transition-colors text-left"
              >
                <Clock size={13} className="text-zinc-600 flex-shrink-0" />
                <span className="flex-1 truncate">{r}</span>
                <span
                  onClick={(e) => deleteRecent(r, e)}
                  className="text-zinc-700 hover:text-zinc-400 transition-colors p-0.5"
                >
                  <X size={12} />
                </span>
              </button>
            ))}
          </div>
        )}
      </div>

      <div className="flex gap-1.5">
        {TABS.map(({ value, label }) => (
          <button
            key={value}
            onClick={() => setTab(value)}
            className={`px-3 py-1 rounded-md text-xs font-medium transition-colors ${
              tab === value
                ? 'bg-white/10 text-white'
                : 'text-zinc-500 hover:bg-white/5 hover:text-zinc-300'
            }`}
          >
            {label}
          </button>
        ))}
      </div>

      {isFetching && <LoadingSpinner />}

      {!isFetching && suggestion && (
        <p className="text-sm text-zinc-500">
          Did you mean{' '}
          <button
            onClick={() => { setQuery(suggestion); setFocusedIndex(null) }}
            className="text-indigo-400 hover:text-indigo-300 underline underline-offset-2 transition-colors"
          >
            {suggestion}
          </button>
          ?
        </p>
      )}

      {!isFetching && (
        <div className="space-y-6">
          {renderResults()}
        </div>
      )}

      {selected && (
        <SearchDetailSheet
          result={selected}
          existing={libraryMap.get(selected.external_id)}
          onClose={() => setSelected(null)}
          onAdd={handleAdd}
          onNavigate={handleNavigate}
          addingId={addingId}
        />
      )}
    </div>
  )
}
