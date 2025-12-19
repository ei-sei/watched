import client from './client'
import type { ChapterLog } from '@/types/media'

export interface ImportResult {
  source: 'openlibrary' | 'manual'
  imported: number
}

export const chaptersApi = {
  list: (mediaId: number) =>
    client.get<ChapterLog[]>(`/media/${mediaId}/chapters`),
  upsert: (mediaId: number, data: {
    chapter_number: number
    chapter_title?: string
    start_page?: number
    end_page?: number
    status?: string
    note?: string
  }) => client.put<ChapterLog>(`/media/${mediaId}/chapters`, data),
  delete: (mediaId: number, chId: number) =>
    client.delete(`/media/${mediaId}/chapters/${chId}`),
  import: (mediaId: number, count?: number) =>
    client.post<ImportResult>(`/media/${mediaId}/chapters/import`, { count: count ?? 0 }),
}
