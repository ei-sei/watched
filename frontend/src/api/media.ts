import client from './client'
import type { MediaItem, PaginatedResponse } from '@/types/media'
import type { MediaListParams } from '@/types/api'

export const mediaApi = {
  list: (params: MediaListParams) => client.get<PaginatedResponse<MediaItem>>('/media', { params }),
  get: (id: number) => client.get<MediaItem>(`/media/${id}`),
  create: (data: Partial<MediaItem> & { media_type: string; title: string }) =>
    client.post<MediaItem>('/media', data),
  update: (id: number, data: Partial<MediaItem>) => client.patch<MediaItem>(`/media/${id}`, data),
  delete: (id: number) => client.delete(`/media/${id}`),
}
