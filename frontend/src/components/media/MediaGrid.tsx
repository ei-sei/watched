import type { MediaItem } from '@/types/media'
import MediaCard from './MediaCard'

interface Props { items: MediaItem[] }

export default function MediaGrid({ items }: Props) {
  return (
    <div className="grid grid-cols-3 sm:grid-cols-4 md:grid-cols-5 lg:grid-cols-6 xl:grid-cols-7 gap-3">
      {items.map((item) => (
        <MediaCard key={item.id} item={item} />
      ))}
    </div>
  )
}
