import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { useLiveQuery } from 'dexie-react-hooks'
import { useAuth } from '@/hooks/useAuth'
import { useTrending, type TrendingCategory, type TrendingItem, type TrendingSection } from '@/hooks/useTrending'
import { useCreateMedia } from '@/hooks/useMedia'
import { useToast } from '@/components/ui/Toast'
import LoadingSpinner from '@/components/ui/LoadingSpinner'
import { db } from '@/offline/db'
import { Lock, TrendingUp, X } from 'lucide-react'

const TABS: { key: TrendingCategory; label: string }[] = [
  { key: 'anime',  label: 'Anime'    },
  { key: 'movies', label: 'Movies'   },
  { key: 'tv',     label: 'TV Shows' },
]

export default function Trending() {
  const { user } = useAuth()
  const [tab, setTab] = useState<TrendingCategory>('anime')
  const [selected, setSelected] = useState<TrendingItem | null>(null)

  if (!user?.is_premium && !user?.is_admin) {
    return (
      <div className="max-w-sm">
        <h1 className="text-2xl font-semibold text-white tracking-tight mb-5">Trending</h1>
        <div className="bg-[#1a1a1a] rounded-xl p-8 ring-1 ring-white/[0.06] flex flex-col items-center gap-4 text-center">
          <div className="w-12 h-12 rounded-full bg-white/[0.06] flex items-center justify-center">
            <Lock size={20} className="text-zinc-500" />
          </div>
          <div className="space-y-1">
            <p className="text-white font-medium">Premium feature</p>
            <p className="text-sm text-zinc-500">Trending is available to premium users.</p>
          </div>
          <div className="flex items-center gap-2 text-zinc-600">
            <TrendingUp size={14} />
            <span className="text-xs">Anime · Movies · TV Shows</span>
          </div>
        </div>
      </div>
    )
  }

  return (
    <div className="space-y-5">
      <h1 className="text-2xl font-semibold text-white tracking-tight">Trending</h1>

      {/* Tab bar */}
      <div className="flex gap-1 bg-white/[0.04] rounded-lg p-1 max-w-sm">
        {TABS.map(({ key, label }) => (
          <button
            key={key}
            onClick={() => setTab(key)}
            className={`flex-1 py-1.5 text-sm rounded-md transition-colors ${
              tab === key
                ? 'bg-white/10 text-white font-medium'
                : 'text-zinc-500 hover:text-zinc-300'
            }`}
          >
            {label}
          </button>
        ))}
      </div>

      <TrendingContent category={tab} onSelect={setSelected} />

      {selected && (
        <AddModal item={selected} onClose={() => setSelected(null)} />
      )}
    </div>
  )
}

function TrendingContent({ category, onSelect }: { category: TrendingCategory; onSelect: (item: TrendingItem) => void }) {
  const { data, isLoading, isError } = useTrending(category)

  if (isLoading) return <LoadingSpinner />
  if (isError) return (
    <p className="text-sm text-zinc-600 text-center py-8">Failed to load trending data</p>
  )
  if (!data?.length) return null

  return (
    <div className="space-y-7">
      {data.map(section => (
        <TrendingRow key={section.label} section={section} onSelect={onSelect} />
      ))}
    </div>
  )
}

function TrendingRow({ section, onSelect }: { section: TrendingSection; onSelect: (item: TrendingItem) => void }) {
  return (
    <div className="space-y-3">
      <p className="text-xs text-zinc-500 font-medium uppercase tracking-wider">{section.label}</p>
      {/* Mobile: horizontal scroll — Desktop: wrapping grid */}
      <div className="flex gap-3 overflow-x-auto pb-1 [&::-webkit-scrollbar]:hidden [-ms-overflow-style:none] [scrollbar-width:none] md:grid md:grid-cols-7 md:overflow-x-visible">
        {section.items.map(item => (
          <PosterCard key={item.id} item={item} onSelect={onSelect} />
        ))}
      </div>
    </div>
  )
}

function PosterCard({ item, onSelect }: { item: TrendingItem; onSelect: (item: TrendingItem) => void }) {
  return (
    <button
      onClick={() => onSelect(item)}
      className="flex-none w-[88px] md:w-full text-left group"
    >
      <div className="relative w-full aspect-[2/3] rounded-lg overflow-hidden bg-white/[0.04] ring-1 ring-white/[0.06] group-hover:ring-indigo-500/40 transition-all">
        {item.poster ? (
          <img
            src={item.poster}
            alt={item.title}
            className="w-full h-full object-cover"
            loading="lazy"
          />
        ) : (
          <div className="w-full h-full flex items-center justify-center p-2">
            <span className="text-zinc-700 text-[10px] text-center leading-tight">{item.title}</span>
          </div>
        )}
        {item.score > 0 && (
          <div className="absolute bottom-1 right-1 bg-black/75 rounded px-1 py-0.5 backdrop-blur-sm">
            <span className="text-[9px] text-amber-400 font-semibold tabular-nums">
              {item.score.toFixed(1)}
            </span>
          </div>
        )}
      </div>
      <p className="text-[10px] text-zinc-400 mt-1.5 line-clamp-2 leading-tight">{item.title}</p>
    </button>
  )
}

function AddModal({ item, onClose }: { item: TrendingItem; onClose: () => void }) {
  const create = useCreateMedia()
  const { show } = useToast()
  const navigate = useNavigate()

  // Check if already in library
  const existing = useLiveQuery<import('@/types/media').MediaItem | undefined>(
    () => item.external_id
      ? db.mediaItems.filter((m) => m.external_id === item.external_id).first()
      : Promise.resolve(undefined),
    [item.external_id],
  )

  const handleAdd = async () => {
    try {
      const created = await create.mutateAsync({
        media_type: item.media_type as 'film' | 'tv_show' | 'anime' | 'book',
        external_id: item.external_id,
        title: item.title,
        year: item.year || undefined,
        poster_url: item.poster || undefined,
        status: 'want_to',
      })
      show(`"${item.title}" added`, 'success')
      onClose()
      navigate(`/media/${created.id}`)
    } catch (err: unknown) {
      const s = (err as { response?: { status?: number } })?.response?.status
      if (s === 409) {
        show('Already in library', 'error')
      } else {
        show('Failed to add', 'error')
      }
    }
  }

  return (
    <div className="fixed inset-0 bg-black/60 flex items-end sm:items-center justify-center z-50 px-0 sm:px-4" onClick={onClose}>
      <div className="bg-[#1a1a1a] rounded-t-2xl sm:rounded-xl w-full sm:max-w-sm ring-1 ring-white/[0.08] overflow-hidden" onClick={e => e.stopPropagation()}>
        {/* Header */}
        <div className="flex items-start gap-3 p-4 border-b border-white/[0.06]">
          {item.poster ? (
            <img src={item.poster} alt={item.title} className="w-12 h-[72px] rounded object-cover flex-shrink-0" />
          ) : (
            <div className="w-12 h-[72px] rounded bg-white/[0.04] flex-shrink-0" />
          )}
          <div className="flex-1 min-w-0 pt-0.5">
            <p className="text-white font-semibold text-sm leading-snug line-clamp-2">{item.title}</p>
            {item.year > 0 && <p className="text-xs text-zinc-500 mt-0.5">{item.year}</p>}
          </div>
          <button onClick={onClose} className="text-zinc-600 hover:text-zinc-400 transition-colors flex-shrink-0 mt-0.5">
            <X size={18} />
          </button>
        </div>

        <div className="p-4 space-y-4">
          {/* Synopsis */}
          {item.synopsis && (
            <p className="text-xs text-zinc-400 leading-relaxed max-h-36 overflow-y-auto">{item.synopsis}</p>
          )}

          {/* Add / already in library */}
          {existing ? (
            <button
              onClick={() => { onClose(); navigate(`/media/${existing.id}`) }}
              className="w-full py-2.5 bg-white/[0.06] hover:bg-white/[0.10] text-zinc-300 text-sm font-medium rounded-lg transition-colors"
            >
              Already in your library →
            </button>
          ) : (
            <button
              onClick={handleAdd}
              disabled={create.isPending}
              className="w-full py-2.5 bg-indigo-600 hover:bg-indigo-700 disabled:opacity-50 text-white text-sm font-medium rounded-lg transition-colors"
            >
              {create.isPending ? 'Adding…' : 'Add to library'}
            </button>
          )}
        </div>
      </div>
    </div>
  )
}
