export interface User {
  id: string
  email: string
  username: string
  role: 'user' | 'admin'
}

export interface UserWithSessions extends User {
  is_active: boolean
  created_at: string
  updated_at: string
  active_sessions: number
}

export interface UserSession {
  id: string
  user_agent: string
  ip_address?: string
  created_at: string
  last_seen_at: string
  expires_at: string
  is_current: boolean
}

export interface ApiError {
  error?: string
  message?: string
}
