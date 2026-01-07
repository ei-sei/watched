import { useMutation, useQueryClient } from '@tanstack/react-query'
import { mediaApi } from '@/api/media'
import { db } from '@/offline/db'
import { enqueue } from '@/offline/queue'
import type { MediaItem } from '@/types/media'

// Read hooks are removed — use useLocalMediaList / useLocalMediaItem from useLocalMedia.ts
// TanStack Query is kept only for server-side operations (search, stats, auth).

export function useUpdateMedia() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: async ({ id, data }: { id: number; data: Partial<MediaItem> }) => {
      // 1. Instant local write
      const existing = await db.mediaItems.get(id)
      if (existing) {
        await db.mediaItems.put({ ...existing, ...data, updated_at: new Date().toISOString() })
      }

      // 2. Attempt server call; enqueue on failure
      try {
        const { data: updated } = await mediaApi.update(id, data)
        await db.mediaItems.put(updated)
        return updated
      } catch {
        await enqueue({ method: 'PATCH', url: `/media/${id}`, data, createdAt: Date.now(), retries: 0 })
        return existing ? { ...existing, ...data } : data
      }
    },
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['stats'] })
    },
  })
}

export function useDeleteMedia() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: async (id: number) => {
      // 1. Remove locally first
      await db.mediaItems.delete(id)
      await db.episodeLogs.where('media_item_id').equals(id).delete()
      await db.chapterLogs.where('media_item_id').equals(id).delete()

      // 2. Server delete; enqueue on failure
      try {
        await mediaApi.delete(id)
      } catch {
        await enqueue({ method: 'DELETE', url: `/media/${id}`, createdAt: Date.now(), retries: 0 })
      }
    },
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['stats'] })
    },
  })
}

export function useCreateMedia() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: async (data: Parameters<typeof mediaApi.create>[0]) => {
      // Creates always require network — the server assigns the real ID
      const { data: created } = await mediaApi.create(data)
      await db.mediaItems.put(created)
      return created
    },
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['stats'] })
    },
  })
}
