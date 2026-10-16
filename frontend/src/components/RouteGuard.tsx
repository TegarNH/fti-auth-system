import React from 'react'
import { useAuth } from '../context/AuthContext'

interface RouteGuardProps {
  children: React.ReactNode
  requireAuth?: boolean
  requireAdmin?: boolean
  onNavigate: (path: string) => void
}

export const RouteGuard: React.FC<RouteGuardProps> = ({
  children,
  requireAuth = false,
  requireAdmin = false,
  onNavigate,
}) => {
  const { user, isLoading } = useAuth()

  if (isLoading) {
    return (
      <div className="loading-screen">
        <div className="spinner"></div>
        <p className="loading-text">Memeriksa status sesi autentikasi...</p>
      </div>
    )
  }

  // If requires login but user not logged in
  if (requireAuth && !user) {
    onNavigate('/login')
    return null
  }

  // If requires admin but user is not admin
  if (requireAdmin && (!user || user.role !== 'admin')) {
    return (
      <div className="auth-card text-center">
        <h2 className="card-title text-danger">403 - Akses Ditolak</h2>
        <p className="card-subtitle mt-2">
          Anda tidak memiliki izin administrator untuk mengakses halaman ini.
        </p>
        <button
          type="button"
          className="btn-primary mt-4"
          onClick={() => onNavigate('/profile')}
        >
          Kembali ke Profil
        </button>
      </div>
    )
  }

  // If guest route (e.g. login/register) but user already logged in
  if (!requireAuth && user && (window.location.pathname === '/login' || window.location.pathname === '/register')) {
    onNavigate('/profile')
    return null
  }

  return <>{children}</>
}
