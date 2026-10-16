import React, { useEffect, useState, useCallback } from 'react'
import { apiFetch, ApiFetchError } from '../api'
import type { UserWithSessions } from '../types'

export const AdminUsersPage: React.FC = () => {
  const [users, setUsers] = useState<UserWithSessions[]>([])
  const [isLoading, setIsLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [actionSuccess, setActionSuccess] = useState<string | null>(null)
  const [actionLoading, setActionLoading] = useState<string | null>(null)

  // Target confirmation dialog state
  const [targetUser, setTargetUser] = useState<UserWithSessions | null>(null)

  const fetchUsers = useCallback(async () => {
    setIsLoading(true)
    setError(null)
    try {
      const data = await apiFetch<{ users: UserWithSessions[] }>('/api/admin/users')
      setUsers(data.users || [])
    } catch (err) {
      if (err instanceof ApiFetchError) {
        setError(err.message)
      } else {
        setError('Gagal memuat daftar pengguna')
      }
    } finally {
      setIsLoading(false)
    }
  }, [])

  useEffect(() => {
    fetchUsers()
  }, [fetchUsers])

  const handleForceLogout = async () => {
    if (!targetUser) return

    setActionLoading(targetUser.id)
    setError(null)
    setActionSuccess(null)

    try {
      await apiFetch(`/api/admin/users/${targetUser.id}/force-logout`, {
        method: 'POST',
      })
      setActionSuccess(`Semua sesi untuk pengguna ${targetUser.username} (${targetUser.email}) berhasil dicabut.`)
      setTargetUser(null)
      await fetchUsers()
    } catch (err) {
      if (err instanceof ApiFetchError) {
        setError(err.message)
      } else {
        setError('Gagal melakukan force logout')
      }
    } finally {
      setActionLoading(null)
    }
  }

  return (
    <div className="admin-container">
      <div className="admin-header">
        <div>
          <h2 className="admin-title">Manajemen Pengguna</h2>
          <p className="admin-subtitle">
            Daftar seluruh akun terdaftar beserta jumlah sesi login aktif secara real-time.
          </p>
        </div>
        <button
          type="button"
          className="btn-refresh"
          onClick={fetchUsers}
          disabled={isLoading}
        >
          {isLoading ? 'Menyinkron...' : 'Segarkan Data'}
        </button>
      </div>

      {actionSuccess && (
        <div className="alert alert-success alert-dismissible">
          <span>{actionSuccess}</span>
          <button type="button" className="close-btn" onClick={() => setActionSuccess(null)}>
            ×
          </button>
        </div>
      )}

      {error && <div className="alert alert-danger">{error}</div>}

      <div className="table-responsive">
        <table className="users-table">
          <thead>
            <tr>
              <th>Pengguna</th>
              <th>Peran</th>
              <th>Status</th>
              <th>Sesi Aktif</th>
              <th>Terdaftar</th>
              <th>Aksi</th>
            </tr>
          </thead>
          <tbody>
            {isLoading && users.length === 0 ? (
              <tr>
                <td colSpan={6} className="text-center py-4 text-muted">
                  Memuat data pengguna...
                </td>
              </tr>
            ) : users.length === 0 ? (
              <tr>
                <td colSpan={6} className="text-center py-4 text-muted">
                  Belum ada data pengguna.
                </td>
              </tr>
            ) : (
              users.map((u) => (
                <tr key={u.id}>
                  <td>
                    <div className="user-cell">
                      <span className="user-cell-name">{u.username}</span>
                      <span className="user-cell-email">{u.email}</span>
                    </div>
                  </td>
                  <td>
                    <span className={`badge-role badge-${u.role}`}>
                      {u.role.toUpperCase()}
                    </span>
                  </td>
                  <td>
                    <span className={`status-pill ${u.is_active ? 'active' : 'inactive'}`}>
                      {u.is_active ? 'Aktif' : 'Nonaktif'}
                    </span>
                  </td>
                  <td>
                    <span className={`session-count ${u.active_sessions > 0 ? 'has-sessions' : 'no-sessions'}`}>
                      {u.active_sessions} aktif
                    </span>
                  </td>
                  <td className="text-muted text-sm">
                    {new Date(u.created_at).toLocaleDateString('id-ID', {
                      day: 'numeric',
                      month: 'short',
                      year: 'numeric',
                    })}
                  </td>
                  <td>
                    <button
                      type="button"
                      className="btn-force-logout"
                      onClick={() => setTargetUser(u)}
                      disabled={actionLoading === u.id || u.active_sessions === 0}
                      title={u.active_sessions === 0 ? 'Pengguna tidak memiliki sesi aktif' : 'Cabut semua sesi pengguna'}
                    >
                      {actionLoading === u.id ? 'Memproses...' : 'Force Logout'}
                    </button>
                  </td>
                </tr>
              ))
            )}
          </tbody>
        </table>
      </div>

      {/* Confirmation Modal */}
      {targetUser && (
        <div className="modal-backdrop">
          <div className="modal-content">
            <h3 className="modal-title">Konfirmasi Force Logout</h3>
            <p className="modal-message">
              Apakah Anda yakin ingin mencabut <strong>semua sesi aktif ({targetUser.active_sessions} sesi)</strong> milik pengguna <strong>{targetUser.username}</strong> ({targetUser.email})?
            </p>
            <p className="modal-note">
              Pengguna akan langsung dikeluarkan dari seluruh perangkat yang sedang login dan harus masuk kembali.
            </p>
            <div className="modal-actions">
              <button
                type="button"
                className="btn-modal-cancel"
                onClick={() => setTargetUser(null)}
                disabled={actionLoading !== null}
              >
                Batal
              </button>
              <button
                type="button"
                className="btn-modal-confirm"
                onClick={handleForceLogout}
                disabled={actionLoading !== null}
              >
                {actionLoading !== null ? 'Mencabut Sesi...' : 'Ya, Cabut Semua Sesi'}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}
