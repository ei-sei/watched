import { useState } from 'react'
import { useAuth } from '@/hooks/useAuth'
import { useTrending, type TrendingCategory, type TrendingItem, type TrendingSection } from '@/hooks/useTrending'
import LoadingSpinner from '@/components/ui/LoadingSpinner'
import { Lock, TrendingUp } from 'lucide-react'

const TABS: { key: TrendingCategory; label: string }[] = [
  { key: 'anime',  label: 'Anime'    },
  { key: 'movies', label: 'Movies'   },
  { key: 'tv',     label: 'TV Shows' },
]

export default function Trending() {
  const { user } = useAuth()
  const [tab, setTab] = useState<TrendingCategory>('anime')

  if (!user?.is_premium && !user?.is_admin) {
    return (
      <div className="max-w-2xl">
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
    <div className="space-y-5 max-w-2xl">
      <h1 className="text-2xl font-semibold text-white tracking-tight">Trending</h1>

      {/* Tab bar */}
      <div className="flex gap-1 bg-white/[0.04] rounded-lg p-1">
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

      <TrendingContent category={tab} />
    </div>
  )
}

function TrendingContent({ category }: { category: TrendingCategory }) {
  const { data, isLoading, isError } = useTrending(category)

  if (isLoading) return <LoadingSpinner />
  if (isError) return (
    <p className="text-sm text-zinc-600 text-center py-8">Failed to load trending data</p>
  )
  if (!data?.length) return null

  return (
    <div className="space-y-7">
      {data.map(section => (
        <TrendingRow key={section.label} section={section} />
      ))}
    </div>
  )
}

function TrendingRow({ section }: { section: TrendingSection }) {
  return (
    <div className="space-y-3">
      <p className="text-xs text-zinc-500 font-medium uppercase tracking-wider">{section.label}</p>
      <div className="flex gap-3 overflow-x-auto pb-1 [&::-webkit-scrollbar]:hidden [-ms-overflow-style:none] [scrollbar-width:none]">
        {section.items.map(item => (
          <PosterCard key={item.id} item={item} />
        ))}
      </div>
    </div>
  )
}

function PosterCard({ item }: { item: TrendingItem }) {
  return (
    <div className="flex-none w-[88px]">
      <div className="relative w-full aspect-[2/3] rounded-lg overflow-hidden bg-white/[0.04] ring-1 ring-white/[0.06]">
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
    </div>
  )
}
