import React, { useEffect, useState, useCallback } from 'react'
import { useAuth } from '../context/AuthContext'
import { apiFetch, ApiFetchError } from '../api'
import type { UserSession } from '../types'

interface ProfileProps {
  onNavigate: (path: string) => void
}

export const ProfilePage: React.FC<ProfileProps> = ({ onNavigate }) => {
  const { user, logout } = useAuth()
  const [sessions, setSessions] = useState<UserSession[]>([])
  const [isLoadingSessions, setIsLoadingSessions] = useState(true)
  const [sessionError, setSessionError] = useState<string | null>(null)
  const [actionSuccess, setActionSuccess] = useState<string | null>(null)
  const [actionLoading, setActionLoading] = useState<string | null>(null)
  const [targetSession, setTargetSession] = useState<UserSession | null>(null)

  const fetchSessions = useCallback(async () => {
    setIsLoadingSessions(true)
    setSessionError(null)
    try {
      const data = await apiFetch<{ sessions: UserSession[] }>('/api/me/sessions')
      setSessions(data.sessions || [])
    } catch (err) {
      if (err instanceof ApiFetchError) {
        setSessionError(err.message)
      } else {
        setSessionError('Gagal memuat sesi aktif')
      }
    } finally {
      setIsLoadingSessions(false)
    }
  }, [])

  useEffect(() => {
    fetchSessions()
  }, [fetchSessions])

  const handleRevokeSession = async () => {
    if (!targetSession) return

    setActionLoading(targetSession.id)
    setSessionError(null)
    setActionSuccess(null)

    try {
      await apiFetch(`/api/me/sessions/${targetSession.id}`, {
        method: 'DELETE',
      })

      // If user revoked the current session, log out immediately
      if (targetSession.is_current) {
        await logout()
        return
      }

      setActionSuccess('Sesi berhasil dicabut.')
      setTargetSession(null)
      await fetchSessions()
    } catch (err) {
      if (err instanceof ApiFetchError) {
        setSessionError(err.message)
      } else {
        setSessionError('Gagal mencabut sesi')
      }
    } finally {
      setActionLoading(null)
    }
  }

  const formatUserAgent = (ua: string) => {
    if (!ua) return 'Perangkat Tidak Dikenal'
    if (ua.length > 55) {
      return ua.substring(0, 52) + '...'
    }
    return ua
  }

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

        {/* Section: Sesi Login Sendiri */}
        <div className="sessions-section">
          <div className="sessions-header">
            <div>
              <h3 className="sessions-title">Sesi Aktif Anda ({sessions.length})</h3>
              <p className="sessions-subtitle">Daftar perangkat yang sedang login ke akun Anda (maksimal 2 sesi bersamaan).</p>
            </div>
            <button
              type="button"
              className="btn-refresh-sm"
              onClick={fetchSessions}
              disabled={isLoadingSessions}
            >
              {isLoadingSessions ? '...' : 'Segarkan'}
            </button>
          </div>

          {actionSuccess && (
            <div className="alert alert-success alert-dismissible mt-2">
              <span>{actionSuccess}</span>
              <button type="button" className="close-btn" onClick={() => setActionSuccess(null)}>
                ×
              </button>
            </div>
          )}

          {sessionError && <div className="alert alert-danger mt-2">{sessionError}</div>}

          <div className="sessions-list">
            {isLoadingSessions && sessions.length === 0 ? (
              <p className="text-muted text-sm py-2">Memuat sesi aktif...</p>
            ) : sessions.length === 0 ? (
              <p className="text-muted text-sm py-2">Tidak ada sesi aktif.</p>
            ) : (
              sessions.map((sess) => (
                <div key={sess.id} className={`session-item ${sess.is_current ? 'current-session' : ''}`}>
                  <div className="session-info">
                    <div className="session-device-row">
                      <span className="session-device" title={sess.user_agent}>
                        💻 {formatUserAgent(sess.user_agent)}
                      </span>
                      {sess.is_current && <span className="current-badge">Sesi Ini</span>}
                    </div>
                    <div className="session-meta">
                      <span>IP: {sess.ip_address || 'Tidak tercatat'}</span>
                      <span>•</span>
                      <span>
                        Masuk:{' '}
                        {new Date(sess.created_at).toLocaleDateString('id-ID', {
                          day: 'numeric',
                          month: 'short',
                          hour: '2-digit',
                          minute: '2-digit',
                        })}
                      </span>
                      <span>•</span>
                      <span>
                        Aktif:{' '}
                        {new Date(sess.last_seen_at).toLocaleTimeString('id-ID', {
                          hour: '2-digit',
                          minute: '2-digit',
                        })}
                      </span>
                    </div>
                  </div>
                  <div className="session-action">
                    <button
                      type="button"
                      className="btn-revoke-sm"
                      onClick={() => setTargetSession(sess)}
                      disabled={actionLoading === sess.id}
                    >
                      {actionLoading === sess.id ? '...' : sess.is_current ? 'Keluar' : 'Cabut'}
                    </button>
                  </div>
                </div>
              ))
            )}
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

      {/* Confirmation Modal for Session Revocation */}
      {targetSession && (
        <div className="modal-backdrop">
          <div className="modal-content">
            <h3 className="modal-title">Konfirmasi Cabut Sesi</h3>
            <p className="modal-message">
              Apakah Anda yakin ingin mencabut sesi pada perangkat:
            </p>
            <p className="code-font text-sm mt-2 mb-2" style={{ wordBreak: 'break-all' }}>
              {targetSession.user_agent}
            </p>
            {targetSession.is_current ? (
              <p className="modal-note text-warning">
                Ini adalah sesi Anda saat ini. Jika dicabut, Anda akan langsung dikeluarkan dari portal.
              </p>
            ) : (
              <p className="modal-note">
                Perangkat tersebut akan segera kehilangan akses dan harus masuk kembali.
              </p>
            )}
            <div className="modal-actions">
              <button
                type="button"
                className="btn-modal-cancel"
                onClick={() => setTargetSession(null)}
                disabled={actionLoading !== null}
              >
                Batal
              </button>
              <button
                type="button"
                className="btn-modal-confirm"
                onClick={handleRevokeSession}
                disabled={actionLoading !== null}
              >
                {actionLoading !== null ? 'Memproses...' : 'Ya, Cabut Sesi'}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}
