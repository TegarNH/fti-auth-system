import React from 'react'
import { useAuth } from '../context/AuthContext'

interface NavbarProps {
  currentPath: string
  onNavigate: (path: string) => void
}

export const Navbar: React.FC<NavbarProps> = ({ currentPath, onNavigate }) => {
  const { user, logout } = useAuth()

  const handleLinkClick = (e: React.MouseEvent, path: string) => {
    e.preventDefault()
    onNavigate(path)
  }

  return (
    <nav className="navbar">
      <div className="nav-brand" onClick={(e) => handleLinkClick(e, user ? '/profile' : '/login')}>
        <span className="brand-badge">FTI</span>
        <span className="brand-title">Auth System</span>
      </div>

      <div className="nav-links">
        {user ? (
          <>
            <a
              href="/profile"
              className={`nav-link ${currentPath === '/profile' ? 'active' : ''}`}
              onClick={(e) => handleLinkClick(e, '/profile')}
            >
              Profil
            </a>

            {user.role === 'admin' && (
              <a
                href="/admin/users"
                className={`nav-link ${currentPath === '/admin/users' ? 'active' : ''}`}
                onClick={(e) => handleLinkClick(e, '/admin/users')}
              >
                Admin Users
              </a>
            )}

            <div className="nav-user-info">
              <span className="user-badge">{user.role}</span>
              <span className="user-name">{user.username}</span>
            </div>

            <button type="button" className="btn-logout" onClick={logout}>
              Keluar
            </button>
          </>
        ) : (
          <>
            <a
              href="/login"
              className={`nav-link ${currentPath === '/login' ? 'active' : ''}`}
              onClick={(e) => handleLinkClick(e, '/login')}
            >
              Masuk
            </a>
            <a
              href="/register"
              className={`nav-link ${currentPath === '/register' ? 'active' : ''}`}
              onClick={(e) => handleLinkClick(e, '/register')}
            >
              Daftar
            </a>
          </>
        )}
      </div>
    </nav>
  )
}
