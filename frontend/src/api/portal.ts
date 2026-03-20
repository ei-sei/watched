import client from './client'

export interface PortalLink {
  id: number
  name: string
  url: string
  category: 'sources' | 'movies_tv' | 'anime'
  is_starred: boolean
  note?: string | null
  created_by: number
  created_at: string
}

export interface PortalStatus {
  id: number
  ok: boolean
  status_code: number
}

export const portalApi = {
  list: () => client.get<PortalLink[]>('/portal'),
  status: () => client.get<PortalStatus[]>('/portal/status'),
  create: (data: { name: string; url: string; category: string; note?: string | null }) =>
    client.post<PortalLink>('/portal', data),
  update: (id: number, data: { name: string; url: string; category: string; note?: string | null }) =>
    client.patch<PortalLink>(`/portal/${id}`, data),
  delete: (id: number) => client.delete(`/portal/${id}`),
  reorder: (ids: number[]) => client.put('/portal/reorder', { ids }),
  star: (id: number) => client.put(`/portal/${id}/star`, {}),
  unstar: (id: number) => client.delete(`/portal/${id}/star`),
}
