import { useState, useEffect } from 'react'
import './App.css'

interface HealthResponse {
  status: string
}

function App() {
  const [backendStatus, setBackendStatus] = useState<string>('Pemeriksaan...')
  const [isLoading, setIsLoading] = useState<boolean>(false)
  const [lastChecked, setLastChecked] = useState<string>('-')

  const checkHealth = async () => {
    setIsLoading(true)
    try {
      const res = await fetch('http://localhost:8080/health')
      if (res.ok) {
        const data: HealthResponse = await res.json()
        setBackendStatus(`Terkoneksi (status: ${data.status})`)
      } else {
        setBackendStatus(`HTTP Error: ${res.status}`)
      }
    } catch {
      setBackendStatus('Gagal terhubung ke http://localhost:8080')
    } finally {
      setIsLoading(false)
      setLastChecked(new Date().toLocaleTimeString())
    }
  }

  useEffect(() => {
    checkHealth()
  }, [])

  return (
    <div className="container">
      <header className="hero-section">
        <h1 className="hero-title">Sistem Authentication Internal FTI</h1>
        <p className="hero-subtitle">
          Infrastruktur fondasi Docker Compose berjalan penuh untuk backend Go 1.27+, basis data PostgreSQL 18, dan frontend SPA React + Vite + TypeScript.
        </p>
      </header>

      <section className="interactive-panel">
        <div className="panel-header">
          <div>
            <h2 className="panel-title">Status Koneksi Backend Health</h2>
            <p style={{ fontSize: '0.825rem', color: '#94a3b8', marginTop: '0.25rem' }}>
              Pemeriksaan endpoint <code>GET http://localhost:8080/health</code>
            </p>
          </div>
          <button
            type="button"
            className="btn-check"
            onClick={checkHealth}
            disabled={isLoading}
          >
            {isLoading ? 'Memeriksa...' : 'Cek Ulang Health'}
          </button>
        </div>
        <div className="status-box">
          <span>{backendStatus}</span>
          <span style={{ fontSize: '0.75rem', color: '#64748b' }}>Waktu: {lastChecked}</span>
        </div>
      </section>

      <section className="card-grid">
        <div className="glass-card">
          <div className="card-icon">⚡</div>
          <h3 className="card-title">Backend Service</h3>
          <p className="card-desc">
            Go 1.27+ HTTP service dengan router chi, multi-stage Alpine build, dan endpoint diagnostik.
          </p>
          <span className="card-tag">Port 8080</span>
        </div>

        <div className="glass-card">
          <div className="card-icon">🐘</div>
          <h3 className="card-title">Database Storage</h3>
          <p className="card-desc">
            PostgreSQL 18 dengan volume data persisten dan healthcheck otomatis <code>pg_isready</code>.
          </p>
          <span className="card-tag">Port 5432</span>
        </div>

        <div className="glass-card">
          <div className="card-icon">🌐</div>
          <h3 className="card-title">Frontend Gateway</h3>
          <p className="card-desc">
            React 19 + TypeScript yang di-bundle dengan Vite dan di-serve via Nginx dengan SPA routing fallback.
          </p>
          <span className="card-tag">Port 3000</span>
        </div>
      </section>

      <footer className="footer-info">
        PT Kuliner Kreasindo Inovasi &copy; 2026
      </footer>
    </div>
  )
}

export default App
