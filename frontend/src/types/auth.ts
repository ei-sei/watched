export interface ProfileSettings {
  visible_statuses: string[] | null
}

export interface User {
  id: number
  username: string
  display_name: string | null
  avatar_url: string | null
  is_admin: boolean
  is_premium: boolean
  is_public: boolean
  profile_settings: ProfileSettings
  created_at: string
  feature_flags: Record<string, boolean>
}

export interface TokenResponse {
  access_token: string
  refresh_token?: string
  token_type: string
  expires_in: number
}

export interface LoginRequest {
  username: string
  password: string
}

export interface RegisterRequest {
  username: string
  password: string
  invite_code: string
}
