import { useLiveQuery } from 'dexie-react-hooks'
import { db } from '@/offline/db'
import type { MediaItem } from '@/types/media'
import type { MediaListParams } from '@/types/api'

export function useLocalMediaList(params: MediaListParams) {
  return useLiveQuery(async () => {
    // Base collection filtered by type index when available
    let items: MediaItem[]
    if (params.media_type) {
      items = await db.mediaItems.where('media_type').equals(params.media_type).toArray()
    } else {
      items = await db.mediaItems.toArray()
    }

    // Status filter
    if (params.status) {
      items = items.filter((i) => i.status === params.status)
    }

    // Text search on title
    if (params.q) {
      const q = params.q.toLowerCase()
      items = items.filter((i) => i.title.toLowerCase().includes(q))
    }

    // Sort
    const sortKey = (params.sort ?? 'created_at') as keyof MediaItem
    const asc = params.order === 'asc'
    items.sort((a, b) => {
      const av = a[sortKey] ?? ''
      const bv = b[sortKey] ?? ''
      if (av === bv) return 0
      const gt = av > bv
      return asc ? (gt ? 1 : -1) : (gt ? -1 : 1)
    })

    // Paginate
    const page = params.page ?? 1
    const perPage = params.per_page ?? 50
    const total = items.length
    const pages = Math.max(1, Math.ceil(total / perPage))
    const sliced = items.slice((page - 1) * perPage, page * perPage)

    return { items: sliced, total, page, per_page: perPage, pages }
  }, [params.media_type, params.status, params.q, params.sort, params.order, params.page, params.per_page])
}

export function useLocalMediaItem(id: number) {
  return useLiveQuery(() => db.mediaItems.get(id), [id])
}
