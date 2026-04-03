import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { mediaApi } from '@/api/media'
import type { MediaListParams } from '@/types/api'
import type { MediaItem } from '@/types/media'

export function useMediaList(params: MediaListParams) {
  return useQuery({
    queryKey: ['media', params],
    queryFn: () => mediaApi.list(params).then((r) => r.data),
  })
}

export function useMediaItem(id: number) {
  return useQuery({
    queryKey: ['media', id],
    queryFn: () => mediaApi.get(id).then((r) => r.data),
    enabled: id > 0,
  })
}

export function useUpdateMedia() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, data }: { id: number; data: Partial<MediaItem> }) => mediaApi.update(id, data),
    onSuccess: (_, { id }) => {
      qc.invalidateQueries({ queryKey: ['media', id] })
      qc.invalidateQueries({ queryKey: ['media'] })
    },
  })
}

export function useDeleteMedia() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: number) => mediaApi.delete(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['media'] }),
  })
}

export function useCreateMedia() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (data: Parameters<typeof mediaApi.create>[0]) => mediaApi.create(data),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['media'] }),
  })
}
