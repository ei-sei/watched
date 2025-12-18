import client from './client'
import type { EpisodeLog } from '@/types/media'

export const episodesApi = {
  list: (mediaId: number) => client.get<EpisodeLog[]>(`/media/${mediaId}/episodes`),
  log: (mediaId: number, data: { season_number: number; episode_number: number; rating?: number; note?: string }) =>
    client.post<EpisodeLog>(`/media/${mediaId}/episodes`, data),
  batch: (mediaId: number, data: { season_number: number; episodes: number[] }) =>
    client.post(`/media/${mediaId}/episodes/batch`, data),
  update: (mediaId: number, epId: number, data: Partial<EpisodeLog>) =>
    client.patch<EpisodeLog>(`/media/${mediaId}/episodes/${epId}`, data),
  delete: (mediaId: number, epId: number) => client.delete(`/media/${mediaId}/episodes/${epId}`),
}
