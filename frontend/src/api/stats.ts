import client from './client'
import type { StatsSummary } from '@/types/media'

export const statsApi = {
  summary: () => client.get<StatsSummary>('/stats/summary'),
}
