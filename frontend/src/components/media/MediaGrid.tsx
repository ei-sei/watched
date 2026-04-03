import type { MediaItem } from '@/types/media'
import MediaCard from './MediaCard'

interface Props { items: MediaItem[] }

export default function MediaGrid({ items }: Props) {
  return (
    <div className="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-5 xl:grid-cols-6 gap-4">
      {items.map((item) => (
        <MediaCard key={item.id} item={item} />
      ))}
    </div>
  )
}
