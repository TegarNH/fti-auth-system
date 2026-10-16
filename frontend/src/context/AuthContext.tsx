import React, { createContext, useContext, useEffect, useState, useCallback } from 'react'
import type { User } from '../types'
import { apiFetch, ApiFetchError } from '../api'

interface AuthContextType {
  user: User | null
  isLoading: boolean
  errorNotice: string | null
  setErrorNotice: (msg: string | null) => void
  refreshUser: () => Promise<void>
  loginSuccess: (user: User) => void
  logout: () => Promise<void>
}

const AuthContext = createContext<AuthContextType | undefined>(undefined)

export const AuthProvider: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  const [user, setUser] = useState<User | null>(null)
  const [isLoading, setIsLoading] = useState<boolean>(true)
  const [errorNotice, setErrorNotice] = useState<string | null>(null)

  const refreshUser = useCallback(async () => {
    try {
      const data = await apiFetch<{ user: User }>('/api/me')
      setUser(data.user)
    } catch (err) {
      setUser(null)
      if (err instanceof ApiFetchError && err.status === 401) {
        // If user was previously logged in or checking on protected route
        if (window.location.pathname !== '/login' && window.location.pathname !== '/register') {
          setErrorNotice('Sesi berakhir, silakan masuk lagi')
        }
      }
    } finally {
      setIsLoading(false)
    }
  }, [])

  useEffect(() => {
    refreshUser()
  }, [refreshUser])

  const loginSuccess = (newUser: User) => {
    setUser(newUser)
    setErrorNotice(null)
  }

  const logout = async () => {
    try {
      await apiFetch('/api/auth/logout', { method: 'POST' })
    } catch {
      // ignore
    } finally {
      setUser(null)
      window.history.pushState({}, '', '/login')
      window.dispatchEvent(new PopStateEvent('popstate'))
    }
  }

  return (
    <AuthContext.Provider
      value={{
        user,
        isLoading,
        errorNotice,
        setErrorNotice,
        refreshUser,
        loginSuccess,
        logout,
      }}
    >
      {children}
    </AuthContext.Provider>
  )
}

export const useAuth = (): AuthContextType => {
  const context = useContext(AuthContext)
  if (!context) {
    throw new Error('useAuth must be used within an AuthProvider')
  }
  return context
}
