import { useQuery } from '@tanstack/react-query'
import { statsApi } from '@/api/stats'

export function useStats() {
  return useQuery({
    queryKey: ['stats', 'summary'],
    queryFn: () => statsApi.summary().then((r) => r.data),
  })
}
