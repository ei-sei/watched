import { useState, useRef, useEffect } from 'react'
import { useSearch } from '@/hooks/useSearch'
import { useCreateMedia } from '@/hooks/useMedia'
import { useToast } from '@/components/ui/Toast'
import LoadingSpinner from '@/components/ui/LoadingSpinner'
import { Clock, X } from 'lucide-react'
import type { MediaType, SearchResult } from '@/types/media'

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

function friendlyMeta(result: SearchResult): string {
  const parts: string[] = []
  parts.push(TYPE_LABELS[result.media_type] ?? result.media_type)
  if (result.year) parts.push(String(result.year))
  if (result.media_type === 'book') {
    const authors = result.extra?.authors
    if (Array.isArray(authors) && authors.length > 0) {
      parts.push(authors.slice(0, 2).join(', '))
    }
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

function ResultCard({ result, isBestMatch, onAdd }: { result: SearchResult; isBestMatch: boolean; onAdd: (r: SearchResult) => void }) {
  return (
    <div className={`flex gap-3 rounded-lg p-3 items-start ring-1 ${isBestMatch ? 'bg-[#1f1f2e] ring-indigo-500/30' : 'bg-[#1a1a1a] ring-white/[0.06]'}`}>
      {result.poster_url ? (
        <img src={result.poster_url} alt="" className="w-12 h-16 object-cover rounded flex-shrink-0" />
      ) : (
        <div className="w-12 h-16 bg-[#222] rounded flex-shrink-0" />
      )}
      <div className="flex-1 min-w-0">
        <div className="flex items-center gap-2">
          <p className="font-medium text-zinc-100 text-sm">{result.title}</p>
          {isBestMatch && (
            <span className="text-[10px] font-medium text-indigo-400 bg-indigo-500/10 px-1.5 py-0.5 rounded">Best Match</span>
          )}
        </div>
        <p className="text-xs text-zinc-500 mt-0.5">{friendlyMeta(result)}</p>
        {result.description && (
          <p className="text-xs text-zinc-600 mt-1.5 line-clamp-2 leading-relaxed">{result.description}</p>
        )}
      </div>
      <button
        onClick={() => onAdd(result)}
        className="flex-shrink-0 bg-white/8 hover:bg-white/12 text-zinc-200 text-xs px-3 py-1.5 rounded-md transition-colors"
      >
        Add
      </button>
    </div>
  )
}

function renderResults(data: SearchResult[], tab: Tab, onAdd: (r: SearchResult) => void) {
  // On specific tabs, flat list with best match highlight
  if (tab !== 'multi') {
    return (
      <div className="space-y-2">
        {data.map((result, i) => (
          <ResultCard key={result.external_id} result={result} isBestMatch={i === 0 && data.length > 1} onAdd={onAdd} />
        ))}
      </div>
    )
  }

  // On "All" tab, group by media type
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
          {items.map((result, i) => (
            <ResultCard key={result.external_id} result={result} isBestMatch={i === 0 && items.length > 1} onAdd={onAdd} />
          ))}
        </div>
      ))}
    </>
  )
}

export default function Search() {
  const [tab, setTab] = useState<Tab>('multi')
  const { query, setQuery, data, isFetching } = useSearch(tab)
  const create = useCreateMedia()
  const { show } = useToast()
  const [focused, setFocused] = useState(false)
  const [recents, setRecents] = useState<string[]>(getRecents)
  const inputRef = useRef<HTMLInputElement>(null)
  const dropdownRef = useRef<HTMLDivElement>(null)

  const showRecents = focused && query.length === 0 && recents.length > 0

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
    } catch (err: unknown) {
      const status = (err as { response?: { status?: number } })?.response?.status
      if (status === 409) {
        show('Already in library', 'error')
      } else if (status === 422) {
        show('Invalid data', 'error')
      } else {
        show('Server error — is the API running?', 'error')
      }
    }
  }

  return (
    <div className="space-y-6 max-w-2xl">
      <h1 className="text-2xl font-semibold text-white tracking-tight">Search</h1>

      <div className="relative">
        <input
          ref={inputRef}
          value={query}
          onChange={(e) => setQuery(e.target.value)}
          onFocus={() => setFocused(true)}
          onKeyDown={(e) => {
            if (e.key === 'Enter' && query.trim()) {
              saveRecent(query.trim())
              setRecents(getRecents())
              setFocused(false)
            }
          }}
          placeholder="Search movies, TV shows, books, anime…"
          className="w-full bg-[#1a1a1a] text-zinc-200 rounded-lg px-4 py-3 border border-white/[0.08] focus:outline-none focus:border-white/20 text-sm placeholder:text-zinc-600"
        />
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

      {data && !isFetching && (
        <div className="space-y-6">
          {data.length === 0 && query.length >= 2 && (
            <p className="text-zinc-600 text-sm text-center py-12">No results for "{query}"</p>
          )}
          {renderResults(data, tab, handleAdd)}
        </div>
      )}
    </div>
  )
}
