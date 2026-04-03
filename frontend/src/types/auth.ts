export interface User {
  id: number
  username: string
  display_name: string | null
  avatar_url: string | null
  is_admin: boolean
  is_premium: boolean
  is_public: boolean
  created_at: string
}

export interface TokenResponse {
  access_token: string
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
