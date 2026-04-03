import axios from 'axios'
import client from './client'
import type { UserList } from '@/types/media'

const baseURL = import.meta.env.VITE_API_URL || 'http://localhost:8000'

export const listsApi = {
  getPublic: (id: number) =>
    axios.get<UserList>(`${baseURL}/share/lists/${id}`),
  list: () => client.get<UserList[]>('/lists'),
  get: (id: number) => client.get<UserList>(`/lists/${id}`),
  create: (data: { name: string; description?: string; is_public?: boolean }) =>
    client.post<UserList>('/lists', data),
  update: (id: number, data: Partial<UserList>) => client.patch<UserList>(`/lists/${id}`, data),
  delete: (id: number) => client.delete(`/lists/${id}`),
  addItem: (listId: number, mediaItemId: number) =>
    client.post(`/lists/${listId}/items`, null, { params: { media_item_id: mediaItemId } }),
  removeItem: (listId: number, itemId: number) => client.delete(`/lists/${listId}/items/${itemId}`),
  reorder: (listId: number, itemIds: number[]) =>
    client.patch(`/lists/${listId}/items/reorder`, { item_ids: itemIds }),
}
