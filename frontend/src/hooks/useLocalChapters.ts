import { useLiveQuery } from 'dexie-react-hooks'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { db } from '@/offline/db'
import { enqueue } from '@/offline/queue'
import { chaptersApi } from '@/api/chapters'
import type { ChapterLog } from '@/types/media'

export function useLocalChapters(mediaId: number) {
  return useLiveQuery(
    () => db.chapterLogs.where('media_item_id').equals(mediaId).toArray(),
    [mediaId],
    [] as ChapterLog[],
  )
}

type UpsertData = {
  chapter_number: number
  chapter_title?: string
  start_page?: number
  end_page?: number
  status?: string
  note?: string
}

export function useUpsertChapter(mediaId: number) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: async (data: UpsertData) => {
      // Optimistic: update or insert local record
      const existing = await db.chapterLogs
        .where('[media_item_id+chapter_number]')
        .equals([mediaId, data.chapter_number])
        .first()

      const optimistic: ChapterLog = {
        id: existing?.id ?? -(Date.now()),
        media_item_id: mediaId,
        chapter_number: data.chapter_number,
        chapter_title: data.chapter_title ?? existing?.chapter_title ?? null,
        start_page: data.start_page ?? existing?.start_page ?? null,
        end_page: data.end_page ?? existing?.end_page ?? null,
        status: (data.status as ChapterLog['status']) ?? existing?.status ?? 'unread',
        note: data.note ?? existing?.note ?? null,
        started_at: existing?.started_at ?? null,
        completed_at: existing?.completed_at ?? null,
      }
      await db.chapterLogs.put(optimistic)

      try {
        const { data: ch } = await chaptersApi.upsert(mediaId, data)
        if (optimistic.id < 0) await db.chapterLogs.delete(optimistic.id)
        await db.chapterLogs.put(ch)
        return ch
      } catch {
        await enqueue({
          method: 'PUT',
          url: `/media/${mediaId}/chapters`,
          data,
          createdAt: Date.now(),
          retries: 0,
        })
        return optimistic
      }
    },
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['stats'] })
    },
  })
}

export function useDeleteChapter(mediaId: number) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: async (chId: number) => {
      await db.chapterLogs.delete(chId)

      try {
        await chaptersApi.delete(mediaId, chId)
      } catch {
        if (chId > 0) {
          await enqueue({
            method: 'DELETE',
            url: `/media/${mediaId}/chapters/${chId}`,
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
