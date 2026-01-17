import { useQuery } from '@tanstack/react-query'
import api from '@/api/client'

export type TrendingItem = {
  id: number
  title: string
  poster: string
  score: number
  year: number
}

export type TrendingSection = {
  label: string
  items: TrendingItem[]
}

export type TrendingCategory = 'movies' | 'tv' | 'anime'

const STALE_TIME: Record<TrendingCategory, number> = {
  movies: 1000 * 60 * 60 * 24, // 24h — matches server TTL
  tv:     1000 * 60 * 60 * 24,
  anime:  1000 * 60 * 60 * 12, // 12h
}

export function useTrending(category: TrendingCategory) {
  return useQuery({
    queryKey: ['trending', category],
    queryFn: async () => {
      const { data } = await api.get<TrendingSection[]>(`/trending/${category}`)
      return data
    },
    staleTime: STALE_TIME[category],
  })
}
