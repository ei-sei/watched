import { Link } from 'react-router-dom'
import type { MediaItem } from '@/types/media'
import StatusBadge from '@/components/ui/StatusBadge'
import RatingDisplay from '@/components/ui/RatingDisplay'

const ROUTE: Record<string, string> = { film: 'films', tv_show: 'tv', book: 'books', anime: 'anime' }

interface Props { item: MediaItem }

export default function MediaCard({ item }: Props) {
  return (
    <Link to={`/${ROUTE[item.media_type]}/${item.id}`} className="group block">
      <div className="rounded-lg overflow-hidden bg-[#1a1a1a] hover:bg-[#202020] transition-colors ring-1 ring-white/[0.06] hover:ring-white/[0.12]">
        <div className="aspect-[2/3] bg-[#222222] relative overflow-hidden">
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
          {item.year && <p className="text-xs text-zinc-600">{item.year}</p>}
          <div className="flex items-center justify-between pt-0.5">
            <StatusBadge status={item.status} />
            <RatingDisplay rating={item.rating} size="sm" />
          </div>
        </div>
      </div>
    </Link>
  )
}
