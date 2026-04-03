import client from './client'
import type { SearchResponse } from '@/types/media'

export const searchApi = {
  search: (q: string, type?: string) =>
    client.get<SearchResponse>('/search', { params: { q, ...(type ? { type } : {}) } }),
}
