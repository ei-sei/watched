import client from './client'
import type { User, LoginRequest, RegisterRequest, TokenResponse, ProfileSettings } from '@/types/auth'

export const authApi = {
  login: (data: LoginRequest) => client.post<TokenResponse>('/auth/login', data),
  register: (data: RegisterRequest) => client.post<User>('/auth/register', data),
  refresh: () => client.post<TokenResponse>('/auth/refresh'),
  logout: () => client.post('/auth/logout'),
  me: () => client.get<User>('/users/me'),
  updateMe: (data: { display_name?: string; avatar_url?: string; is_public?: boolean; profile_settings?: ProfileSettings }) => client.patch<User>('/users/me', data),
  changePassword: (data: { current_password: string; new_password: string }) => client.put('/users/me/password', data),
}
