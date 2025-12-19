import { useState } from 'react'
import { useSearch } from '@/hooks/useSearch'
import { useCreateMedia } from '@/hooks/useMedia'
import { useToast } from '@/components/ui/Toast'
import LoadingSpinner from '@/components/ui/LoadingSpinner'
import type { SearchResult as ApiSearchResult } from '@/api/search'
import type { MediaType } from '@/types/media'

const TABS = [
  { value: 'multi', label: 'All' },
  { value: 'film',  label: 'Movies' },
  { value: 'tv',    label: 'TV' },
  { value: 'book',  label: 'Books' },
  { value: 'anime', label: 'Anime' },
] as const

type Tab = typeof TABS[number]['value']

export default function Search() {
  const [tab, setTab] = useState<Tab>('multi')
  const { query, setQuery, data, isFetching } = useSearch(tab)
  const create = useCreateMedia()
  const { show } = useToast()

  const handleAdd = async (result: ApiSearchResult) => {
    try {
      const totalProgress = typeof result.extra?.episodes === 'number' && result.extra.episodes > 0
        ? result.extra.episodes
        : undefined

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

      <input
        value={query}
        onChange={(e) => setQuery(e.target.value)}
        placeholder="Search movies, TV shows, books, anime…"
        className="w-full bg-[#1a1a1a] text-zinc-200 rounded-lg px-4 py-3 border border-white/[0.08] focus:outline-none focus:border-white/20 text-sm placeholder:text-zinc-600"
      />

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
        <div className="space-y-2">
          {data.map((result) => (
            <div key={result.external_id} className="flex gap-3 bg-[#1a1a1a] rounded-lg p-3 items-start ring-1 ring-white/[0.06]">
              {result.poster_url ? (
                <img src={result.poster_url} alt={result.title} className="w-12 h-16 object-cover rounded flex-shrink-0" />
              ) : (
                <div className="w-12 h-16 bg-[#222] rounded flex-shrink-0" />
              )}
              <div className="flex-1 min-w-0">
                <p className="font-medium text-zinc-100 text-sm">{result.title}</p>
                <p className="text-xs text-zinc-500 mt-0.5">
                  {result.year} · {result.media_type.replace('_', ' ')} · {result.source}
                </p>
                {result.description && (
                  <p className="text-xs text-zinc-600 mt-1.5 line-clamp-2 leading-relaxed">{result.description}</p>
                )}
              </div>
              <button
                onClick={() => handleAdd(result)}
                className="flex-shrink-0 bg-white/8 hover:bg-white/12 text-zinc-200 text-xs px-3 py-1.5 rounded-md transition-colors"
              >
                Add
              </button>
            </div>
          ))}
          {data.length === 0 && query.length >= 2 && (
            <p className="text-zinc-600 text-sm text-center py-12">No results for "{query}"</p>
          )}
        </div>
      )}
    </div>
  )
}
