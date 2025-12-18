import client from './client'
import type { ChapterLog } from '@/types/media'

export const chaptersApi = {
  list: (mediaId: number) => client.get<ChapterLog[]>(`/media/${mediaId}/chapters`),
  log: (mediaId: number, data: Partial<ChapterLog> & { chapter_number: number }) =>
    client.post<ChapterLog>(`/media/${mediaId}/chapters`, data),
  batch: (mediaId: number, data: { chapters: Partial<ChapterLog>[] }) =>
    client.post(`/media/${mediaId}/chapters/batch`, data),
  update: (mediaId: number, chId: number, data: Partial<ChapterLog>) =>
    client.patch<ChapterLog>(`/media/${mediaId}/chapters/${chId}`, data),
  delete: (mediaId: number, chId: number) => client.delete(`/media/${mediaId}/chapters/${chId}`),
}
