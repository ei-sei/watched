import { useState, useEffect } from 'react'
import { useQuery } from '@tanstack/react-query'
import { searchApi } from '@/api/search'

type Tab = 'multi' | 'film' | 'tv_show' | 'book' | 'anime'

export function useSearch(query: string, type: Tab = 'multi') {
  const [debouncedQuery, setDebouncedQuery] = useState(query)

  useEffect(() => {
    const timer = setTimeout(() => setDebouncedQuery(query), 300)
    return () => clearTimeout(timer)
  }, [query])

  const result = useQuery({
    queryKey: ['search', type, debouncedQuery],
    queryFn: () =>
      searchApi.search(debouncedQuery, type === 'multi' ? undefined : type).then((r) => r.data),
    enabled: debouncedQuery.length >= 2,
  })

  return {
    data: result.data?.items,
    suggestion: result.data?.suggestion ?? '',
    isFetching: result.isFetching,
  }
}
