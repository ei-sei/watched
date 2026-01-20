import client from '@/api/client'
import { db } from './db'
import { getPending, dequeue, incrementRetry } from './queue'
import type { MediaItem, EpisodeLog, ChapterLog, PaginatedResponse } from '@/types/media'

const MAX_RETRIES = 5
const FLUSH_CONCURRENCY = 3

// ── Queue flush ──────────────────────────────────────────────────────────────

export async function flushQueue(): Promise<void> {
  const pending = await getPending()

  const processOne = async (item: (typeof pending)[number]) => {
    if (item.retries >= MAX_RETRIES) {
      await dequeue(item.id!)
      return
    }
    try {
      await client.request({ method: item.method, url: item.url, data: item.data })
      await dequeue(item.id!)
    } catch (err: any) {
      const status = err?.response?.status
      // Non-retryable client errors — drop immediately rather than burning retries
      if (status && status >= 400 && status < 500 && status !== 429) {
        await dequeue(item.id!)
        return
      }
      await incrementRetry(item.id!)
    }
  }

  // Process in batches of FLUSH_CONCURRENCY instead of one-by-one
  for (let i = 0; i < pending.length; i += FLUSH_CONCURRENCY) {
    await Promise.all(pending.slice(i, i + FLUSH_CONCURRENCY).map(processOne))
  }
}

// ── Stale-aware sync ─────────────────────────────────────────────────────────

let lastSyncAt = 0
const SYNC_COOLDOWN_MS = 30_000 // 30 seconds

/** Runs syncMedia only if more than 30 s have passed since the last sync. */
export function syncMediaIfStale(): void {
  if (Date.now() - lastSyncAt < SYNC_COOLDOWN_MS) return
  syncMedia().catch(console.error)
}

// ── Full media sync ──────────────────────────────────────────────────────────

export async function syncMedia(): Promise<void> {
  window.dispatchEvent(new Event('watched:sync-start'))
  // Flush any queued mutations before pulling — ensures server state reflects
  // local changes before we overwrite IndexedDB with the server response.
  await flushQueue()
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
    lastSyncAt = Date.now()

    // Prune items removed on the server
    const serverIds = new Set(allItems.map((i) => i.id))
    const localIds = await db.mediaItems.toCollection().primaryKeys() as number[]
    const stale = localIds.filter((id) => !serverIds.has(id))
    if (stale.length > 0) await db.mediaItems.bulkDelete(stale)
  } catch {
    // Offline or auth not ready — skip silently
  } finally {
    window.dispatchEvent(new Event('watched:sync-end'))
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

// ── Debounced item re-fetch (for status auto-transitions) ────────────────────
// After logging/deleting an episode or chapter the backend may auto-transition
// the item's status. We re-fetch the media item to pick up any change, but we
// debounce per item so rapid toggles only trigger one request.

const pendingRefetch = new Map<number, number>()

export function scheduleItemRefetch(mediaId: number): void {
  clearTimeout(pendingRefetch.get(mediaId))
  const t = window.setTimeout(async () => {
    pendingRefetch.delete(mediaId)
    try {
      const { data } = await client.get<MediaItem>(`/media/${mediaId}`)
      await db.mediaItems.put(data)
    } catch {
      // offline or auth issue — local state is fine
    }
  }, 400)
  pendingRefetch.set(mediaId, t)
}

// ── Reconnect handler ────────────────────────────────────────────────────────

export function registerSyncOnReconnect(): void {
  window.addEventListener('online', () => {
    flushQueue().catch(console.error)
    syncMedia().catch(console.error)
  })
}
