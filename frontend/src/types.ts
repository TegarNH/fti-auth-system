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

export interface ApiError {
  error?: string
  message?: string
}
