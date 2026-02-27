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
  refreshFromTMDB: (id: number) => client.post<MediaItem>(`/media/${id}/refresh`),
  setSeasonProgress: (id: number, season: number, count: number) =>
    client.put(`/media/${id}/episodes/progress`, { season, count }),
  listEpisodeStamps: (id: number) => client.get<EpisodeStamp[]>(`/media/${id}/episode-stamps`),
  listRewatches: (id: number) => client.get<Rewatch[]>(`/media/${id}/rewatches`),
  createRewatch: (id: number, data: { started_at?: string | null; finished_at?: string | null }) =>
    client.post<Rewatch>(`/media/${id}/rewatches`, data),
  deleteRewatch: (rewatchId: number) => client.delete(`/rewatches/${rewatchId}`),
}

export interface EpisodeStamp {
  season: number
  episode: number
  watched_at: string
}

export interface Rewatch {
  id: number
  user_id: number
  media_id: number
  started_at: string | null
  finished_at: string | null
  created_at: string
}
