import React from 'react'
import { useAuth } from '../context/AuthContext'

interface ProfileProps {
  onNavigate: (path: string) => void
}

export const ProfilePage: React.FC<ProfileProps> = ({ onNavigate }) => {
  const { user, logout } = useAuth()

  if (!user) {
    return null
  }

  return (
    <div className="profile-container">
      <div className="profile-card">
        <div className="profile-header">
          <div className="avatar-circle">
            {user.username.charAt(0).toUpperCase()}
          </div>
          <div>
            <h2 className="profile-name">{user.username}</h2>
            <p className="profile-email">{user.email}</p>
          </div>
          <span className={`badge-role badge-${user.role}`}>
            {user.role.toUpperCase()}
          </span>
        </div>

        <div className="profile-details">

          <div className="detail-item">
            <span className="detail-label">Role</span>
            <span className="detail-value">
              {user.role === 'admin' ? 'Administrator' : 'User'}
            </span>
          </div>

          <div className="detail-item">
            <span className="detail-label">Status Akun</span>
            <span className="detail-value text-success">● Aktif</span>
          </div>
          
        </div>

        <div className="profile-actions">
          {user.role === 'admin' && (
            <button
              type="button"
              className="btn-secondary"
              onClick={() => onNavigate('/admin/users')}
            >
              Kelola Pengguna (Admin)
            </button>
          )}

          <button type="button" className="btn-danger" onClick={logout}>
            Keluar dari Sesi Ini
          </button>
        </div>
      </div>
    </div>
  )
}
