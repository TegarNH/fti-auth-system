import React, { useState } from 'react'
import { useAuth } from '../context/AuthContext'
import { apiFetch, ApiFetchError } from '../api'
import type { User } from '../types'

interface LoginProps {
  onNavigate: (path: string) => void
}

export const LoginPage: React.FC<LoginProps> = ({ onNavigate }) => {
  const { loginSuccess, errorNotice, setErrorNotice } = useAuth()
  const [identifier, setIdentifier] = useState('')
  const [password, setPassword] = useState('')
  const [error, setError] = useState<string | null>(null)
  const [isLoading, setIsLoading] = useState(false)

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setError(null)
    setErrorNotice(null)
    setIsLoading(true)

    try {
      const res = await apiFetch<{ message: string; user: User }>('/api/auth/login', {
        method: 'POST',
        body: JSON.stringify({ identifier, password }),
      })
      loginSuccess(res.user)
      onNavigate('/profile')
    } catch (err) {
      if (err instanceof ApiFetchError) {
        setError(err.message)
      } else {
        setError('Gagal menghubungi server')
      }
    } finally {
      setIsLoading(false)
    }
  }

  return (
    <div className="auth-card">
      <div className="card-header">
        <h2 className="card-title">Masuk ke Akun</h2>
        <p className="card-subtitle">Sistem Autentikasi Internal Terpusat FTI</p>
      </div>

      {errorNotice && <div className="alert alert-warning">{errorNotice}</div>}
      {error && <div className="alert alert-danger">{error}</div>}

      <form onSubmit={handleSubmit} className="auth-form">
        <div className="form-group">
          <label htmlFor="identifier">Email atau Username</label>
          <input
            id="identifier"
            type="text"
            required
            autoComplete="username"
            placeholder="nama@fti.com atau username"
            value={identifier}
            onChange={(e) => setIdentifier(e.target.value)}
            disabled={isLoading}
          />
        </div>

        <div className="form-group">
          <label htmlFor="password">Kata Sandi</label>
          <input
            id="password"
            type="password"
            required
            autoComplete="current-password"
            placeholder="Masukkan kata sandi"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            disabled={isLoading}
          />
        </div>

        <button type="submit" className="btn-primary" disabled={isLoading}>
          {isLoading ? 'Memproses...' : 'Masuk Sekarang'}
        </button>
      </form>

      <div className="card-footer">
        Belum punya akun?{' '}
        <button
          type="button"
          className="link-btn"
          onClick={() => onNavigate('/register')}
        >
          Daftar di sini
        </button>
      </div>
    </div>
  )
}
