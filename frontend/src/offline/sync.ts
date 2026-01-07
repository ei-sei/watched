import client from '@/api/client'
import { db } from './db'
import { getPending, dequeue, incrementRetry } from './queue'
import type { MediaItem, EpisodeLog, ChapterLog, PaginatedResponse } from '@/types/media'

const MAX_RETRIES = 3

// ── Queue flush ──────────────────────────────────────────────────────────────

export async function flushQueue(): Promise<void> {
  const pending = await getPending()
  for (const item of pending) {
    if (item.retries >= MAX_RETRIES) {
      await dequeue(item.id!)
      continue
    }
    try {
      await client.request({ method: item.method, url: item.url, data: item.data })
      await dequeue(item.id!)
    } catch {
      await incrementRetry(item.id!)
    }
  }
}

// ── Full media sync ──────────────────────────────────────────────────────────

export async function syncMedia(): Promise<void> {
  try {
    const allItems: MediaItem[] = []
    let page = 1
    const perPage = 200

    while (true) {
      const { data } = await client.get<PaginatedResponse<MediaItem>>('/media', {
        params: { per_page: perPage, page },
      })
      allItems.push(...data.items)
      if (page >= data.pages) break
      page++
    }

    await db.mediaItems.bulkPut(allItems)

    // Prune items removed on the server
    const serverIds = new Set(allItems.map((i) => i.id))
    const localIds = await db.mediaItems.toCollection().primaryKeys() as number[]
    const stale = localIds.filter((id) => !serverIds.has(id))
    if (stale.length > 0) await db.mediaItems.bulkDelete(stale)
  } catch {
    // Offline or auth not ready — skip silently
  }
}

// ── Per-item detail sync (episodes / chapters) ───────────────────────────────

export async function syncItemDetail(mediaId: number): Promise<void> {
  const item = await db.mediaItems.get(mediaId)
  if (!item) return

  try {
    if (item.media_type === 'tv_show' || item.media_type === 'anime') {
      const { data: episodes } = await client.get<EpisodeLog[]>(`/media/${mediaId}/episodes`)
      await db.episodeLogs.where('media_item_id').equals(mediaId).delete()
      if (episodes.length > 0) await db.episodeLogs.bulkPut(episodes)
    } else if (item.media_type === 'book') {
      const { data: chapters } = await client.get<ChapterLog[]>(`/media/${mediaId}/chapters`)
      await db.chapterLogs.where('media_item_id').equals(mediaId).delete()
      if (chapters.length > 0) await db.chapterLogs.bulkPut(chapters)
    }
  } catch {
    // Offline — local data is fine
  }
}

// ── Reconnect handler ────────────────────────────────────────────────────────

export function registerSyncOnReconnect(): void {
  window.addEventListener('online', () => {
    flushQueue().catch(console.error)
    syncMedia().catch(console.error)
  })
}
