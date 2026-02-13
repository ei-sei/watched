import { useLiveQuery } from 'dexie-react-hooks'
import { db } from '@/offline/db'
import type { MediaItem } from '@/types/media'
import type { MediaListParams } from '@/types/api'

export function useLocalMediaList(params: MediaListParams) {
  return useLiveQuery(async () => {
    // Use the most selective index available to avoid loading the full table.
    let items: MediaItem[]
    if (params.media_type && params.status) {
      // Compound index — pushes both filters into IndexedDB
      items = await db.mediaItems
        .where('[media_type+status]')
        .equals([params.media_type, params.status])
        .toArray()
    } else if (params.media_type) {
      items = await db.mediaItems.where('media_type').equals(params.media_type).toArray()
    } else if (params.status) {
      items = await db.mediaItems.where('status').equals(params.status).toArray()
    } else {
      items = await db.mediaItems.toArray()
    }

    // Text search on title + romaji alternative title
    if (params.q) {
      const q = params.q.toLowerCase()
      items = items.filter((i) => {
        if (i.title.toLowerCase().includes(q)) return true
        const romaji = i.metadata?.title_romaji
        if (typeof romaji === 'string' && romaji.toLowerCase().includes(q)) return true
        return false
      })
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

    const total = items.length

    // Paginate only when explicitly requested (Dashboard uses per_page: 8)
    if (params.per_page) {
      const page = params.page ?? 1
      const perPage = params.per_page
      const pages = Math.max(1, Math.ceil(total / perPage))
      const sliced = items.slice((page - 1) * perPage, page * perPage)
      return { items: sliced, total, page, per_page: perPage, pages }
    }

    return { items, total, page: 1, per_page: total, pages: 1 }
  }, [params.media_type, params.status, params.q, params.sort, params.order, params.page, params.per_page])
}

export function useLocalMediaItem(id: number) {
  return useLiveQuery(() => db.mediaItems.get(id), [id])
}
