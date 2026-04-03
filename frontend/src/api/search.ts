import client from './client'

export interface SearchResult {
  source: string
  media_type: string
  external_id: string
  title: string
  year: number | null
  poster_url: string | null
  description: string | null
  extra: Record<string, unknown>
}

export const searchApi = {
  search: (q: string, type?: string) =>
    client.get<SearchResult[]>('/search', { params: { q, ...(type ? { type } : {}) } }),
}
