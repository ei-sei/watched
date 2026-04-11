import { useState } from 'react'
import { Link } from 'react-router-dom'
import type { MediaItem, MediaStatus } from '@/types/media'
import StatusBadge from '@/components/ui/StatusBadge'
import RatingDisplay from '@/components/ui/RatingDisplay'
import StatusSheet from '@/components/ui/StatusSheet'
import { useUpdateMedia } from '@/hooks/useMedia'

const ROUTE: Record<string, string> = { film: 'films', tv_show: 'tv', book: 'books', anime: 'anime' }

interface Props { item: MediaItem }

export default function MediaCard({ item }: Props) {
  const [sheetOpen, setSheetOpen] = useState(false)
  const update = useUpdateMedia()

  return (
    <>
      <Link to={`/${ROUTE[item.media_type]}/${item.id}`} className="group block">
        <div className="rounded-lg bg-[#1a1a1a] hover:bg-[#202020] transition-colors ring-1 ring-white/[0.06] hover:ring-white/[0.12]">
          <div className="aspect-[2/3] bg-[#222222] relative overflow-hidden rounded-t-lg">
            {item.poster_url ? (
              <img
                src={item.poster_url}
                alt={item.title}
                className="w-full h-full object-cover opacity-90 group-hover:opacity-100 transition-opacity"
                loading="lazy"
              />
            ) : (
              <div className="w-full h-full flex items-center justify-center text-zinc-600 text-xs text-center px-3 leading-relaxed">
                {item.title}
              </div>
            )}
          </div>
          <div className="px-2.5 py-2 space-y-1.5">
            <p className="text-xs font-medium text-zinc-200 truncate leading-snug">{item.title}</p>
            <div className="flex items-end justify-between pt-0.5">
              {/* Mobile: tap badge opens sheet; desktop: badge is decorative */}
              <button
                className="md:hidden"
                onClick={(e) => { e.preventDefault(); setSheetOpen(true) }}
              >
                <StatusBadge status={item.status} mediaType={item.media_type} />
              </button>
              <span className="hidden md:inline-flex">
                <StatusBadge status={item.status} mediaType={item.media_type} />
              </span>
              <span className="flex-shrink-0"><RatingDisplay rating={item.rating} size="sm" /></span>
            </div>
          </div>
        </div>
      </Link>

      <StatusSheet
        open={sheetOpen}
        current={item.status as MediaStatus}
        mediaType={item.media_type}
        onSelect={(s) => {
          const today = new Date().toISOString().slice(0, 10)
          const extra: Partial<MediaItem> = {}
          if (s === 'completed' && !item.completed_at) extra.completed_at = today
          if (s === 'in_progress' && !item.started_at) extra.started_at = today
          update.mutate({ id: item.id, data: { status: s, ...extra } })
        }}
        onClose={() => setSheetOpen(false)}
      />
    </>
  )
}
