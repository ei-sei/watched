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
  is_locked: boolean
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
  db_size_limit_bytes: number
  db_healthy: boolean
  mem_total_kb: number
  mem_used_kb: number
  mem_available_kb: number
  disk_total_bytes: number
  disk_used_bytes: number
}

export interface AdminUserLibraryStat {
  total: number
  completed: number
}

export interface AdminUserStats {
  films: AdminUserLibraryStat
  tv_shows: AdminUserLibraryStat
  anime: AdminUserLibraryStat
  books: AdminUserLibraryStat
  episodes_watched: number
  chapters_read: number
}

export interface ServiceStatus {
  name: string
  ok: boolean
  latency_ms: number
  error?: string
}

export interface FeatureFlag {
  key: string
  is_premium: boolean
  updated_at: string
}

export const adminApi = {
  getStats: () => client.get<AdminStats>('/admin/stats'),
  getHealth: () => client.get<ServiceStatus[]>('/admin/health'),
  listUsers: () => client.get<AdminUser[]>('/admin/users'),
  getUserStats: (id: number) => client.get<AdminUserStats>(`/admin/users/${id}/stats`),
  getUserLibrary: (id: number) => client.get<import('@/types/media').MediaItem[]>(`/admin/users/${id}/library`),
  updateFlags: (id: number, flags: { is_admin?: boolean; is_premium?: boolean }) =>
    client.patch<AdminUser>(`/admin/users/${id}/flags`, flags),
  deleteUser: (id: number) => client.delete(`/admin/users/${id}`),
  resetPassword: (id: number, password: string) =>
    client.post(`/admin/users/${id}/reset-password`, { password }),
  unlockUser: (id: number) => client.post(`/admin/users/${id}/unlock`, {}),
  listInvites: () => client.get<InviteCode[]>('/admin/invites'),
  createInvite: (code: string) => client.post<{ code: string }>('/admin/invites', { code }),
  deleteInvite: (code: string) => client.delete(`/admin/invites/${code}`),
  getFlags: () => client.get<FeatureFlag[]>('/admin/flags'),
  setFlag: (key: string, isPremium: boolean) => client.patch(`/admin/flags/${key}`, { is_premium: isPremium }),
}
