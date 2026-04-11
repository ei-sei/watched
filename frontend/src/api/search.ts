import client from './client'
import type { SearchResult } from '@/types/media'

export const searchApi = {
  search: (q: string, type?: string) =>
    client.get<SearchResult[]>('/search', { params: { q, ...(type ? { type } : {}) } }),
}
