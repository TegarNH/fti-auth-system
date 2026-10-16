import React, { useState } from 'react'
import { apiFetch, ApiFetchError } from '../api'

interface RegisterProps {
  onNavigate: (path: string) => void
}

export const RegisterPage: React.FC<RegisterProps> = ({ onNavigate }) => {
  const [email, setEmail] = useState('')
  const [username, setUsername] = useState('')
  const [password, setPassword] = useState('')
  const [confirmPassword, setConfirmPassword] = useState('')
  const [error, setError] = useState<string | null>(null)
  const [success, setSuccess] = useState<string | null>(null)
  const [isLoading, setIsLoading] = useState(false)

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setError(null)
    setSuccess(null)

    if (password !== confirmPassword) {
      setError('Konfirmasi kata sandi tidak cocok')
      return
    }

    if (password.length < 8) {
      setError('Kata sandi minimal 8 karakter')
      return
    }

    setIsLoading(true)

    try {
      await apiFetch('/api/auth/register', {
        method: 'POST',
        body: JSON.stringify({ email, username, password }),
      })
      setSuccess('Pendaftaran berhasil! Mengalihkan ke halaman masuk...')
      setTimeout(() => {
        onNavigate('/login')
      }, 1500)
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
        <h2 className="card-title">Daftar Akun Baru</h2>
        <p className="card-subtitle">Buat akun untuk mengakses portal internal</p>
      </div>

      {error && <div className="alert alert-danger">{error}</div>}
      {success && <div className="alert alert-success">{success}</div>}

      <form onSubmit={handleSubmit} className="auth-form">
        <div className="form-group">
          <label htmlFor="reg-email">Alamat Email</label>
          <input
            id="reg-email"
            type="email"
            required
            autoComplete="email"
            placeholder="nama@fti.com"
            value={email}
            onChange={(e) => setEmail(e.target.value)}
            disabled={isLoading}
          />
        </div>

        <div className="form-group">
          <label htmlFor="reg-username">Username</label>
          <input
            id="reg-username"
            type="text"
            required
            autoComplete="username"
            placeholder="3-30 karakter alfanumerik"
            value={username}
            onChange={(e) => setUsername(e.target.value)}
            disabled={isLoading}
          />
        </div>

        <div className="form-group">
          <label htmlFor="reg-password">Kata Sandi</label>
          <input
            id="reg-password"
            type="password"
            required
            autoComplete="new-password"
            placeholder="Minimal 8 karakter"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            disabled={isLoading}
          />
        </div>

        <div className="form-group">
          <label htmlFor="reg-confirm">Konfirmasi Kata Sandi</label>
          <input
            id="reg-confirm"
            type="password"
            required
            autoComplete="new-password"
            placeholder="Ulangi kata sandi"
            value={confirmPassword}
            onChange={(e) => setConfirmPassword(e.target.value)}
            disabled={isLoading}
          />
        </div>

        <button type="submit" className="btn-primary" disabled={isLoading}>
          {isLoading ? 'Mendaftarkan...' : 'Daftar Sekarang'}
        </button>
      </form>

      <div className="card-footer">
        Sudah memiliki akun?{' '}
        <button
          type="button"
          className="link-btn"
          onClick={() => onNavigate('/login')}
        >
          Masuk di sini
        </button>
      </div>
    </div>
  )
}
