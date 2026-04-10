import { useLiveQuery } from 'dexie-react-hooks'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { db } from '@/offline/db'
import { enqueue } from '@/offline/queue'
import { episodesApi } from '@/api/episodes'
import type { EpisodeLog } from '@/types/media'

export function useLocalEpisodes(mediaId: number) {
  return useLiveQuery(
    () => db.episodeLogs.where('media_item_id').equals(mediaId).toArray(),
    [mediaId],
    [] as EpisodeLog[],
  )
}

export function useLogEpisode(mediaId: number) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: async ({ season, episode }: { season: number; episode: number }) => {
      // Optimistic: add a temp record so the UI updates instantly
      const tempId = -(Date.now())
      const tempEp: EpisodeLog = {
        id: tempId,
        media_item_id: mediaId,
        season_number: season,
        episode_number: episode,
        watched_at: new Date().toISOString(),
        rating: null,
        note: null,
      }
      await db.episodeLogs.put(tempEp)

      try {
        const { data: ep } = await episodesApi.log(mediaId, { season, episode })
        // Replace temp with real server record
        await db.episodeLogs.delete(tempId)
        await db.episodeLogs.put(ep)
        // Keep media item progress in sync
        const item = await db.mediaItems.get(mediaId)
        if (item) {
          const logged = await db.episodeLogs.where('media_item_id').equals(mediaId).count()
          await db.mediaItems.update(mediaId, { current_progress: logged })
        }
        return ep
      } catch {
        await enqueue({
          method: 'PUT',
          url: `/media/${mediaId}/episodes`,
          data: { season, episode },
          createdAt: Date.now(),
          retries: 0,
        })
        return tempEp
      }
    },
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['stats'] })
    },
  })
}

export function useDeleteEpisode(mediaId: number) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: async (epId: number) => {
      await db.episodeLogs.delete(epId)

      try {
        await episodesApi.delete(mediaId, epId)
        const item = await db.mediaItems.get(mediaId)
        if (item) {
          const logged = await db.episodeLogs.where('media_item_id').equals(mediaId).count()
          await db.mediaItems.update(mediaId, { current_progress: logged })
        }
      } catch {
        // If this was a temp record (negative id) nothing to DELETE on server
        if (epId > 0) {
          await enqueue({
            method: 'DELETE',
            url: `/media/${mediaId}/episodes/${epId}`,
            createdAt: Date.now(),
            retries: 0,
          })
        }
      }
    },
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['stats'] })
    },
  })
}
