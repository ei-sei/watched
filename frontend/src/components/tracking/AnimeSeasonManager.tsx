import { useState } from 'react'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { Plus, Trash2, X } from 'lucide-react'
import { mediaApi } from '@/api/media'
import { useToast } from '@/components/ui/Toast'
import type { MediaItem } from '@/types/media'

interface JikanResult {
  mal_id: number
  title: string
  year: number | null
  episodes: number | null
  images: { jpg: { large_image_url: string | null } }
}

interface Props { item: MediaItem }

export default function AnimeSeasonManager({ item }: Props) {
  const qc = useQueryClient()
  const { show } = useToast()
  const [open, setOpen] = useState(false)
  const [query, setQuery] = useState('')
  const [results, setResults] = useState<JikanResult[]>([])
  const [searching, setSearching] = useState(false)

  const openAndSearch = async () => {
    setOpen(true)
    setQuery(item.title)
    setSearching(true)
    try {
      const res = await fetch(
        `https://api.jikan.moe/v4/anime?q=${encodeURIComponent(item.title)}&limit=10&sfw=false`
      )
      const json = await res.json()
      setResults(json.data ?? [])
    } catch {
      show('Search failed', 'error')
    } finally {
      setSearching(false)
    }
  }

  const seasons = (item.metadata?.seasons as { season_number: number; episode_count: number; mal_id?: number }[] | undefined) ?? []

  const addSeason = useMutation({
    mutationFn: (malId: number) => mediaApi.addSeason(item.id, malId),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['media', item.id] })
      setOpen(false)
      setQuery('')
      setResults([])
      show('Season added', 'success')
    },
    onError: () => show('Could not fetch season data from MAL', 'error'),
  })

  const removeSeason = useMutation({
    mutationFn: (seasonNum: number) => mediaApi.removeSeason(item.id, seasonNum),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['media', item.id] }),
    onError: () => show('Failed to remove season', 'error'),
  })

  const search = async () => {
    if (!query.trim()) return
    setSearching(true)
    try {
      const res = await fetch(
        `https://api.jikan.moe/v4/anime?q=${encodeURIComponent(query)}&limit=10&sfw=false`
      )
      const json = await res.json()
      setResults(json.data ?? [])
    } catch {
      show('Search failed', 'error')
    } finally {
      setSearching(false)
    }
  }

  return (
    <div className="space-y-2">
      <div className="flex items-center justify-between">
        <h3 className="text-sm font-semibold text-zinc-300 uppercase tracking-wider">Seasons</h3>
        <button
          onClick={() => open ? setOpen(false) : openAndSearch()}
          className="flex items-center gap-1 text-xs text-zinc-500 hover:text-zinc-300 transition-colors"
        >
          <Plus size={12} /> Add season
        </button>
      </div>

      {seasons.length > 0 && (
        <div className="space-y-1">
          {seasons.map((s) => (
            <div key={s.season_number} className="flex items-center justify-between text-sm">
              <span className="text-zinc-400">Season {s.season_number}</span>
              <div className="flex items-center gap-3">
                <span className="text-zinc-600 tabular-nums">{s.episode_count} eps</span>
                {seasons.length > 1 && (
                  <button
                    onClick={() => removeSeason.mutate(s.season_number)}
                    disabled={removeSeason.isPending}
                    className="text-zinc-700 hover:text-red-400 transition-colors disabled:opacity-30"
                  >
                    <Trash2 size={12} />
                  </button>
                )}
              </div>
            </div>
          ))}
        </div>
      )}

      {open && (
        <div className="border border-white/[0.08] rounded-lg p-3 space-y-3 bg-[#111]">
          <div className="flex gap-2">
            <input
              type="text"
              value={query}
              onChange={(e) => setQuery(e.target.value)}
              onKeyDown={(e) => e.key === 'Enter' && search()}
              placeholder="Search MAL for season…"
              className="flex-1 bg-[#1a1a1a] text-sm text-white placeholder-zinc-600 rounded px-3 py-1.5 border border-white/[0.06] focus:outline-none focus:border-indigo-500"
            />
            <button
              onClick={search}
              disabled={searching}
              className="px-3 py-1.5 bg-indigo-600 hover:bg-indigo-700 text-white text-sm rounded disabled:opacity-50"
            >
              {searching ? '…' : 'Search'}
            </button>
            <button onClick={() => { setOpen(false); setQuery(''); setResults([]) }} className="text-zinc-600 hover:text-zinc-300">
              <X size={16} />
            </button>
          </div>

          {results.length > 0 && (
            <div className="space-y-1 max-h-64 overflow-y-auto">
              {results.map((r) => (
                <div key={r.mal_id} className="flex items-center gap-3 p-2 rounded hover:bg-white/[0.04] group">
                  {r.images.jpg.large_image_url && (
                    <img src={r.images.jpg.large_image_url} className="w-8 h-10 object-cover rounded flex-shrink-0" />
                  )}
                  <div className="flex-1 min-w-0">
                    <p className="text-sm text-zinc-200 truncate">{r.title}</p>
                    <p className="text-xs text-zinc-600">{r.year ?? '—'} · {r.episodes ?? '?'} eps</p>
                  </div>
                  <button
                    onClick={() => addSeason.mutate(r.mal_id)}
                    disabled={addSeason.isPending}
                    className="text-xs text-indigo-400 hover:text-indigo-300 disabled:opacity-50 flex-shrink-0"
                  >
                    Add as Season {seasons.length + 1}
                  </button>
                </div>
              ))}
            </div>
          )}
        </div>
      )}
    </div>
  )
}
