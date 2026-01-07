import client from './client'

export interface InviteCode {
  code: string
  created_at: string
  used_at: string | null
}

export interface AdminUser {
  id: number
  username: string
  display_name: string | null
  is_admin: boolean
  is_premium: boolean
  is_public: boolean
  created_at: string
}

export interface AdminStats {
  total_users: number
  total_items: number
  total_episodes: number
  total_chapters: number
  unused_invites: number
  db_size_bytes: number
  db_size_human: string
  db_healthy: boolean
}

export const adminApi = {
  getStats: () => client.get<AdminStats>('/admin/stats'),
  listUsers: () => client.get<AdminUser[]>('/admin/users'),
  updateFlags: (id: number, flags: { is_admin?: boolean; is_premium?: boolean }) =>
    client.patch<AdminUser>(`/admin/users/${id}/flags`, flags),
  deleteUser: (id: number) => client.delete(`/admin/users/${id}`),
  listInvites: () => client.get<InviteCode[]>('/admin/invites'),
  createInvite: (code: string) => client.post<{ code: string }>('/admin/invites', { code }),
  deleteInvite: (code: string) => client.delete(`/admin/invites/${code}`),
}
