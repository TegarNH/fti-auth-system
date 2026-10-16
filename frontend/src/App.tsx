import { useState, useEffect } from 'react'
import { AuthProvider, useAuth } from './context/AuthContext'
import { Navbar } from './components/Navbar'
import { RouteGuard } from './components/RouteGuard'
import { LoginPage } from './pages/LoginPage'
import { RegisterPage } from './pages/RegisterPage'
import { ProfilePage } from './pages/ProfilePage'
import { AdminUsersPage } from './pages/AdminUsersPage'
import './App.css'

function AppContent() {
  const [currentPath, setCurrentPath] = useState<string>(window.location.pathname)
  const { user } = useAuth()

  useEffect(() => {
    const handlePopState = () => {
      setCurrentPath(window.location.pathname)
    }
    window.addEventListener('popstate', handlePopState)
    return () => window.removeEventListener('popstate', handlePopState)
  }, [])

  const navigate = (path: string) => {
    window.history.pushState({}, '', path)
    setCurrentPath(path)
  }

  // Redirect root path appropriately
  useEffect(() => {
    if (currentPath === '/') {
      navigate(user ? '/profile' : '/login')
    }
  }, [currentPath, user])

  const renderContent = () => {
    switch (currentPath) {
      case '/login':
        return (
          <RouteGuard onNavigate={navigate}>
            <LoginPage onNavigate={navigate} />
          </RouteGuard>
        )
      case '/register':
        return (
          <RouteGuard onNavigate={navigate}>
            <RegisterPage onNavigate={navigate} />
          </RouteGuard>
        )
      case '/profile':
        return (
          <RouteGuard requireAuth onNavigate={navigate}>
            <ProfilePage onNavigate={navigate} />
          </RouteGuard>
        )
      case '/admin/users':
        return (
          <RouteGuard requireAuth requireAdmin onNavigate={navigate}>
            <AdminUsersPage />
          </RouteGuard>
        )
      default:
        return (
          <div className="auth-card text-center">
            <h2 className="card-title">404 - Halaman Tidak Ditemukan</h2>
            <p className="card-subtitle mt-2">Alamat yang Anda tuju tidak tersedia.</p>
            <button
              type="button"
              className="btn-primary mt-4"
              onClick={() => navigate(user ? '/profile' : '/login')}
            >
              Kembali ke Beranda
            </button>
          </div>
        )
    }
  }

  return (
    <div className="app-container">
      <Navbar currentPath={currentPath} onNavigate={navigate} />
      <main className="main-content">{renderContent()}</main>
      <footer className="app-footer">
        <span>PT Kuliner Kreasindo Inovasi &copy; 2026</span>
      </footer>
    </div>
  )
}

function App() {
  return (
    <AuthProvider>
      <AppContent />
    </AuthProvider>
  )
}

export default App
